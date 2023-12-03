# Setup and Usage

## background
The [Rainforest Automation EMU-2](https://www.rainforestautomation.com/rfa-z105-2-emu-2-2/) is used by some utility companies to publish utility metrics in near real time. In my use case I wanted to use this within Home Assistant.  There are existing Home Assistant plugins that are available which will utilize the serial communication over USB locally within the USB install. This is fine for some use cases. In my case though I wanted to separate the Home Assistant install from any local USB devices. This ensures that my Home Assistant install is able to move around within a small cluster of nodes.  This package is my take on a method to feed Home Assistant the utility data over the network rather than over serial.  You will need an MQTT server setup and integrated for Home Assistant to make use of this. MQTT Auto Discovery needs to be enabled in the MQTT plugin in HA.

# usage
This package is not published to docker directly and is intended to be build and executed locally.  The only parameters you should need to change are going to be ENV variables set within the container to help control how you will connect to the serial port as well as MQTT.

## building image
First, you will want to build the image.  A command like this will work:
`docker build . -t emu2mqtt:latest`

## Docker compose example
```
version: "3.3"
services:
  emu2mqtt:
    image: emu2mqtt:latest
    container_name: emu2mqtt
    restart: unless-stopped
    environment:
      - EMU2_DEV="<LOCAL_USB_DEV_FOR_THE_EMU2>"
      - MQTT_HOST=<MQTT_IP_OR_HOSTNAME>
      - MQTT_USER=<MQTT_USERNAME>
      - MQTT_PASS=<MQTT_PASSWORD>
      - HA_DISCOVERY_PREFIX=<MQTT_DISCOVERY_PREFIX> # this is optional and defaults to homeassistant 
    devices:
      - "<LOCAL_USB_DEV_FOR_THE_EMU2>"
```

## example output
Once the container is up and running you should be able to see events being pushed to MQTT:
```
2023-12-03 02:12:27,825 - ha_mqtt_discoverable - DEBUG - Writing '0.879' to hmd/sensor/Rainforest-EMU-2/HomeCurrentDemand/state
2023-12-03 02:12:27,825 - ha_mqtt_discoverable - DEBUG - Publish result: (0, 10722)
2023-12-03 02:12:42,829 - root - INFO - publishing: InstantaneousDemand-0.879
2023-12-03 02:12:42,829 - ha_mqtt_discoverable.sensors - INFO - Setting HomeCurrentDemand to 0.879 using hmd/sensor/Rainforest-EMU-2/HomeCurrentDemand/state
```

## Home Assistant
The device and entities should correctly show up in your HA installation if the auto discovery is setup correctly.