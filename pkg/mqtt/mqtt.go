package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/danlindow/emu2mqtt/pkg/config"
	paho "github.com/eclipse/paho.mqtt.golang"
)

type haDevice struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
}

type haDiscoveryPayload struct {
	Name              string   `json:"name"`
	UniqueID          string   `json:"unique_id"`
	StateTopic        string   `json:"state_topic"`
	UnitOfMeasurement string   `json:"unit_of_measurement"`
	DeviceClass       string   `json:"device_class"`
	Device            haDevice `json:"device"`
}

type sensorDef struct {
	SensorName   string
	FriendlyName string
	Unit         string
	DeviceClass  string
}

var sensors = []sensorDef{
	{
		SensorName:   "HomeCurrentDemand",
		FriendlyName: "Home Current Demand",
		Unit:         "kW",
		DeviceClass:  "power",
	},
	{
		SensorName:   "HomeEnergyUsage",
		FriendlyName: "Home Energy Usage",
		Unit:         "kWh",
		DeviceClass:  "energy",
	},
}

// Publisher manages the MQTT connection and publishes HA discovery and state messages.
type Publisher struct {
	cfg    *config.Config
	client paho.Client
	logger *slog.Logger
}

// NewPublisher creates a Publisher. Call Connect before publishing.
func NewPublisher(cfg *config.Config, logger *slog.Logger) *Publisher {
	return &Publisher{cfg: cfg, logger: logger}
}

// Connect establishes the MQTT connection, retrying until successful or ctx is cancelled.
// On each (re)connect, HA discovery payloads are republished automatically.
func (p *Publisher) Connect(ctx context.Context) error {
	opts := paho.NewClientOptions().
		AddBroker(fmt.Sprintf("tcp://%s:1883", p.cfg.MQTTHost)).
		SetClientID("emu2mqtt").
		SetUsername(p.cfg.MQTTUser).
		SetPassword(p.cfg.MQTTPass).
		SetAutoReconnect(true).
		SetConnectionLostHandler(func(_ paho.Client, err error) {
			p.logger.Warn("MQTT connection lost", "err", err)
		}).
		SetOnConnectHandler(func(_ paho.Client) {
			p.logger.Info("MQTT connected, publishing discovery")
			if err := p.PublishDiscovery(); err != nil {
				p.logger.Warn("discovery publish failed", "err", err)
			}
		})

	p.client = paho.NewClient(opts)

	delay := time.Second
	for {
		token := p.client.Connect()
		if token.WaitTimeout(10 * time.Second) {
			if err := token.Error(); err == nil {
				return nil
			}
			p.logger.Warn("MQTT connect failed", "err", token.Error())
		} else {
			p.logger.Warn("MQTT connect timeout")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			if delay < 30*time.Second {
				delay *= 2
			}
		}
	}
}

// PublishDiscovery sends retained HA auto-discovery config messages for all sensors.
func (p *Publisher) PublishDiscovery() error {
	device := haDevice{
		Identifiers:  []string{"device_id"},
		Name:         "Rainforest-EMU-2",
		Manufacturer: "Rainforest Automation",
		Model:        "EMU-2",
	}
	for _, s := range sensors {
		payload := haDiscoveryPayload{
			Name:              s.FriendlyName,
			UniqueID:          s.SensorName,
			StateTopic:        p.stateTopic(s.SensorName),
			UnitOfMeasurement: s.Unit,
			DeviceClass:       s.DeviceClass,
			Device:            device,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal discovery for %s: %w", s.SensorName, err)
		}
		token := p.client.Publish(p.configTopic(s.SensorName), 1, true, data)
		token.Wait()
		if err := token.Error(); err != nil {
			return fmt.Errorf("publish discovery for %s: %w", s.SensorName, err)
		}
		p.logger.Info("published discovery", "sensor", s.SensorName)
	}
	return nil
}

// PublishState sends a state update for the named sensor.
func (p *Publisher) PublishState(sensorName string, value float64) error {
	payload := strconv.FormatFloat(value, 'f', -1, 64)
	token := p.client.Publish(p.stateTopic(sensorName), 0, false, payload)
	token.Wait()
	if err := token.Error(); err != nil {
		return fmt.Errorf("publish state for %s: %w", sensorName, err)
	}
	p.logger.Debug("published state", "sensor", sensorName, "value", payload)
	return nil
}

func (p *Publisher) configTopic(sensorName string) string {
	return fmt.Sprintf("%s/sensor/emu2/%s/config", p.cfg.DiscoveryPrefix, sensorName)
}

func (p *Publisher) stateTopic(sensorName string) string {
	return fmt.Sprintf("%s/sensor/emu2/%s/state", p.cfg.DiscoveryPrefix, sensorName)
}
