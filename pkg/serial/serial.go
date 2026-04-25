package serial

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/danlindow/emu2mqtt/pkg/config"
	goserial "go.bug.st/serial"
)

// instantaneousDemand is a real-time power reading from the EMU-2.
// All numeric fields are hex strings (e.g. "0x000254").
type instantaneousDemand struct {
	XMLName     xml.Name `xml:"InstantaneousDemand"`
	DeviceMacId string   `xml:"DeviceMacId"`
	Demand      string   `xml:"Demand"`
	Multiplier  string   `xml:"Multiplier"`
	Divisor     string   `xml:"Divisor"`
	DigitsRight string   `xml:"DigitsRight"`
}

// currentSummationDelivered is a cumulative energy reading from the EMU-2.
// All numeric fields are hex strings.
type currentSummationDelivered struct {
	XMLName            xml.Name `xml:"CurrentSummationDelivered"`
	DeviceMacId        string   `xml:"DeviceMacId"`
	SummationDelivered string   `xml:"SummationDelivered"`
	Multiplier         string   `xml:"Multiplier"`
	Divisor            string   `xml:"Divisor"`
	DigitsRight        string   `xml:"DigitsRight"`
}

// Metric is a parsed, computed reading ready for MQTT publication.
type Metric struct {
	SensorName  string
	Value       float64
	DeviceMacID string // raw hex from DeviceMacId field, e.g. "0xd8d5b9000000c28c"
}

// parseHex converts a "0x..."-prefixed hex string to int64.
func parseHex(s string) (int64, error) {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if s == "" {
		return 0, fmt.Errorf("empty hex string")
	}
	return strconv.ParseInt(s, 16, 64)
}

// computeValue calculates round((value * multiplier) / divisor, digitsRight).
func computeValue(rawValue, rawMultiplier, rawDivisor, rawDigitsRight string) (float64, error) {
	val, err := parseHex(rawValue)
	if err != nil {
		return 0, fmt.Errorf("parsing value %q: %w", rawValue, err)
	}
	mul, err := parseHex(rawMultiplier)
	if err != nil {
		return 0, fmt.Errorf("parsing multiplier %q: %w", rawMultiplier, err)
	}
	div, err := parseHex(rawDivisor)
	if err != nil {
		return 0, fmt.Errorf("parsing divisor %q: %w", rawDivisor, err)
	}
	dig, err := parseHex(rawDigitsRight)
	if err != nil {
		return 0, fmt.Errorf("parsing digitsRight %q: %w", rawDigitsRight, err)
	}
	if div == 0 {
		return 0, fmt.Errorf("divisor is zero")
	}
	result := (float64(val) * float64(mul)) / float64(div)
	factor := math.Pow(10, float64(dig))
	return math.Round(result*factor) / factor, nil
}

// parseMessage interprets buf as one of the two supported XML message types.
// Returns nil, nil if the message type is not supported.
func parseMessage(buf string) (*Metric, error) {
	buf = strings.TrimSpace(buf)
	if strings.Contains(buf, "<InstantaneousDemand>") {
		var msg instantaneousDemand
		if err := xml.Unmarshal([]byte(buf), &msg); err != nil {
			return nil, fmt.Errorf("unmarshal InstantaneousDemand: %w", err)
		}
		val, err := computeValue(msg.Demand, msg.Multiplier, msg.Divisor, msg.DigitsRight)
		if err != nil {
			return nil, fmt.Errorf("compute InstantaneousDemand: %w", err)
		}
		return &Metric{SensorName: "HomeCurrentDemand", Value: val, DeviceMacID: msg.DeviceMacId}, nil
	}
	if strings.Contains(buf, "<CurrentSummationDelivered>") {
		var msg currentSummationDelivered
		if err := xml.Unmarshal([]byte(buf), &msg); err != nil {
			return nil, fmt.Errorf("unmarshal CurrentSummationDelivered: %w", err)
		}
		val, err := computeValue(msg.SummationDelivered, msg.Multiplier, msg.Divisor, msg.DigitsRight)
		if err != nil {
			return nil, fmt.Errorf("compute CurrentSummationDelivered: %w", err)
		}
		return &Metric{SensorName: "HomeEnergyUsage", Value: val, DeviceMacID: msg.DeviceMacId}, nil
	}
	return nil, nil
}

// Reader manages the serial port connection and XML parsing.
type Reader struct {
	cfg     *config.Config
	metrics chan<- Metric
	logger  *slog.Logger
}

// NewReader creates a Reader that sends parsed metrics to the provided channel.
func NewReader(cfg *config.Config, metrics chan<- Metric, logger *slog.Logger) *Reader {
	return &Reader{cfg: cfg, metrics: metrics, logger: logger}
}

// Run opens the serial port and reads forever, reconnecting on error.
// Blocks until ctx is cancelled. Reconnect backoff is handled by openPort.
func (r *Reader) Run(ctx context.Context) {
	for {
		port, err := r.openPort(ctx)
		if err != nil {
			return // ctx cancelled
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			r.readLoop(ctx, port)
		}()

		select {
		case <-ctx.Done():
			port.Close()
			<-done
			return
		case <-done:
			port.Close()
		}

		r.logger.Warn("serial disconnected, reconnecting")
	}
}

func (r *Reader) openPort(ctx context.Context) (goserial.Port, error) {
	delay := time.Second
	for {
		port, err := goserial.Open(r.cfg.SerialDevice, &goserial.Mode{
			BaudRate: 115200,
			DataBits: 8,
			Parity:   goserial.NoParity,
			StopBits: goserial.OneStopBit,
		})
		if err == nil {
			r.logger.Info("serial port opened", "device", r.cfg.SerialDevice)
			return port, nil
		}
		r.logger.Warn("serial open failed, retrying", "err", err, "delay", delay)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			if delay < 30*time.Second {
				delay *= 2
			}
		}
	}
}

func (r *Reader) readLoop(ctx context.Context, port goserial.Port) {
	scanner := bufio.NewScanner(port)
	var buf strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		r.logger.Debug("serial line", "line", line)

		// A root opening tag (e.g. <InstantaneousDemand>) while the buffer is
		// non-empty means we connected mid-stream and the previous message was
		// truncated. Discard the partial buffer and start fresh.
		if buf.Len() > 0 && strings.HasPrefix(line, "<") &&
			!strings.HasPrefix(line, "</") && !strings.Contains(line, "</") {
			r.logger.Debug("discarding truncated message, restarting")
			buf.Reset()
		}

		buf.WriteString(line)
		buf.WriteByte('\n')
		if strings.HasPrefix(line, "</") {
			raw := buf.String()
			buf.Reset()
			metric, err := parseMessage(raw)
			if err != nil {
				r.logger.Warn("parse error", "err", err, "raw", raw)
				continue
			}
			if metric != nil {
				select {
				case r.metrics <- *metric:
				case <-ctx.Done():
					return
				}
			} else {
				r.logger.Debug("unsupported message type", "raw", raw)
			}
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		r.logger.Error("serial read error", "err", err)
	}
}
