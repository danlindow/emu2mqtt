package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config holds all runtime configuration for the service.
type Config struct {
	SerialDevice    string
	MQTTHost        string
	MQTTUser        string
	MQTTPass        string
	DiscoveryPrefix string
	Debug           bool
}

// LoadConfig parses CLI flags first, then falls back to environment variables.
// CLI flags always take precedence over env vars.
func LoadConfig() (*Config, error) {
	dev    := flag.String("dev",    "", "Serial device path (env: EMU2_DEV)")
	host   := flag.String("host",   "", "MQTT broker host (env: MQTT_HOST)")
	user   := flag.String("user",   "", "MQTT username (env: MQTT_USER)")
	pass   := flag.String("pass",   "", "MQTT password (env: MQTT_PASS)")
	prefix := flag.String("prefix", "", "HA discovery prefix (env: HA_DISCOVERY_PREFIX, default: homeassistant)")
	debug  := flag.Bool("debug", false, "Enable debug logging (env: DEBUG)")
	flag.Parse()

	cfg := &Config{
		SerialDevice:    firstNonEmpty(*dev, os.Getenv("EMU2_DEV")),
		MQTTHost:        firstNonEmpty(*host, os.Getenv("MQTT_HOST")),
		MQTTUser:        firstNonEmpty(*user, os.Getenv("MQTT_USER")),
		MQTTPass:        firstNonEmpty(*pass, os.Getenv("MQTT_PASS")),
		DiscoveryPrefix: firstNonEmpty(*prefix, os.Getenv("HA_DISCOVERY_PREFIX"), "homeassistant"),
		Debug:           *debug || os.Getenv("DEBUG") != "",
	}

	var missing []string
	if cfg.SerialDevice == "" {
		missing = append(missing, "EMU2_DEV / -dev")
	}
	if cfg.MQTTHost == "" {
		missing = append(missing, "MQTT_HOST / -host")
	}
	if cfg.MQTTUser == "" {
		missing = append(missing, "MQTT_USER / -user")
	}
	if cfg.MQTTPass == "" {
		missing = append(missing, "MQTT_PASS / -pass")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

// firstNonEmpty returns the first non-empty string from the candidates.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
