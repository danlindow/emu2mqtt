package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/danlindow/emu2mqtt/pkg/config"
	"github.com/danlindow/emu2mqtt/pkg/mqtt"
	"github.com/danlindow/emu2mqtt/pkg/serial"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("configuration error", "err", err)
		os.Exit(1)
	}

	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	logger.Info("starting emu2mqtt", "device", cfg.SerialDevice, "mqtt_host", cfg.MQTTHost)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	publisher := mqtt.NewPublisher(cfg, logger)
	if err := publisher.Connect(ctx); err != nil {
		logger.Error("MQTT connect failed", "err", err)
		os.Exit(1)
	}

	metrics := make(chan serial.Metric, 10)
	reader := serial.NewReader(cfg, metrics, logger)
	go reader.Run(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutting down")
			return
		case m := <-metrics:
			logger.Info("metric received", "sensor", m.SensorName, "value", m.Value)
			if err := publisher.PublishState(m.SensorName, m.Value); err != nil {
				logger.Warn("publish state error", "err", err)
			}
		}
	}
}
