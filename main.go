package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := LoadConfig()
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

	publisher := NewMQTTPublisher(cfg, logger)
	if err := publisher.Connect(ctx); err != nil {
		logger.Error("MQTT connect failed", "err", err)
		os.Exit(1)
	}

	metrics := make(chan Metric, 10)
	reader := NewSerialReader(cfg, metrics, logger)
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
