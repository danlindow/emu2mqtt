# emu2mqtt

Bridges a [Rainforest Automation EMU-2](https://www.rainforestautomation.com/rfa-z105-2-emu-2-2/) USB energy monitor to Home Assistant via MQTT auto-discovery.

The EMU-2 connects to your smart electricity meter over Zigbee and streams real-time energy data over USB serial. This service reads that stream and publishes it to MQTT so Home Assistant automatically creates two sensor entities — no manual MQTT configuration required.

The primary motivation is to decouple the USB device from the Home Assistant host, allowing HA to run anywhere in a cluster while receiving energy data over the network.

## Requirements

- An MQTT broker reachable from wherever you run this service
- Home Assistant with the MQTT integration installed and auto-discovery enabled

## Configuration

All configuration is accepted as CLI flags or environment variables. CLI flags take precedence over env vars.

| Flag | Env var | Required | Default | Description |
|---|---|---|---|---|
| `-dev` | `EMU2_DEV` | yes | | Path to the EMU-2 serial device |
| `-host` | `MQTT_HOST` | yes | | MQTT broker hostname or IP |
| `-user` | `MQTT_USER` | yes | | MQTT username |
| `-pass` | `MQTT_PASS` | yes | | MQTT password |
| `-prefix` | `HA_DISCOVERY_PREFIX` | no | `homeassistant` | HA MQTT discovery prefix |
| `-debug` | `DEBUG` | no | false | Enable debug logging |

## Standalone binary

Download the latest binary for your platform from the [Releases](../../releases) page.

```bash
# Linux
chmod +x emu2mqtt-linux-amd64
./emu2mqtt-linux-amd64 -dev /dev/ttyACM0 -host mqtt.example.com -user mqttuser -pass mqttpass

# macOS (Apple Silicon)
chmod +x emu2mqtt-darwin-arm64
./emu2mqtt-darwin-arm64 -dev /dev/tty.usbmodem1 -host mqtt.example.com -user mqttuser -pass mqttpass
```

Or with environment variables:

```bash
export EMU2_DEV=/dev/ttyACM0
export MQTT_HOST=mqtt.example.com
export MQTT_USER=mqttuser
export MQTT_PASS=mqttpass
./emu2mqtt-linux-amd64
```

On Linux the binary needs access to the serial device. Add your user to the `dialout` group if you see a permission error:

```bash
sudo usermod -a -G dialout $USER
# then log out and back in, or run: newgrp dialout
```

## Docker

The image is published to GitHub Container Registry on every release.

```bash
docker run --rm \
  --device=/dev/ttyACM0 \
  -e EMU2_DEV=/dev/ttyACM0 \
  -e MQTT_HOST=mqtt.example.com \
  -e MQTT_USER=mqttuser \
  -e MQTT_PASS=mqttpass \
  ghcr.io/danlindow/emu2mqtt:latest
```

### Docker Compose

```yaml
services:
  emu2mqtt:
    image: ghcr.io/danlindow/emu2mqtt:latest
    container_name: emu2mqtt
    restart: unless-stopped
    devices:
      - /dev/serial/by-id/usb-Rainforest_Automation__Inc._RFA-Z105-2_HW2.7.3_EMU-2-if00
    environment:
      - EMU2_DEV=/dev/serial/by-id/usb-Rainforest_Automation__Inc._RFA-Z105-2_HW2.7.3_EMU-2-if00
      - MQTT_HOST=mqtt.example.com
      - MQTT_USER=mqttuser
      - MQTT_PASS=mqttpass
      # - HA_DISCOVERY_PREFIX=homeassistant  # optional
```

Using the `by-id` path for the device is recommended — it stays stable across reboots regardless of which USB port you use.

## Home Assistant

On startup the service publishes MQTT auto-discovery messages so Home Assistant automatically creates a device called **Rainforest-EMU-2** with two sensors:

| Entity | Unit | Description |
|---|---|---|
| Home Current Demand | kW | Real-time power draw |
| Home Energy Usage | kWh | Cumulative energy consumed |

The device and entities appear under **Settings → Devices & Services → MQTT** once the first message is received.

## Example log output

```
time=2026-04-25T17:19:58.892+01:00 level=INFO msg="starting emu2mqtt" device=/dev/ttyACM0 mqtt_host=mqtt.example.com
time=2026-04-25T17:19:58.892+01:00 level=INFO msg="MQTT connected, publishing discovery"
time=2026-04-25T17:19:58.892+01:00 level=INFO msg="published discovery" sensor=HomeCurrentDemand
time=2026-04-25T17:19:58.892+01:00 level=INFO msg="published discovery" sensor=HomeEnergyUsage
time=2026-04-25T17:19:58.893+01:00 level=INFO msg="serial port opened" device=/dev/ttyACM0
time=2026-04-25T17:20:08.582+01:00 level=INFO msg="metric received" sensor=HomeCurrentDemand value=0.483
time=2026-04-25T17:20:18.901+01:00 level=INFO msg="metric received" sensor=HomeEnergyUsage value=14523.441
```
