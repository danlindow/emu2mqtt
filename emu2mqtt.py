import serial
import xmltodict
from ha_mqtt_discoverable import Settings, DeviceInfo
from ha_mqtt_discoverable.sensors import Sensor, SensorInfo
import os
import logging
import sys

# setup logging
root = logging.getLogger()
root.setLevel(logging.DEBUG)

handler = logging.StreamHandler(sys.stdout)
handler.setLevel(logging.DEBUG)
formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
handler.setFormatter(formatter)
root.addHandler(handler)

# PULL IN OS ENV TUNABLES
emu_serial_dev = os.environ.get('EMU2_DEV')
if not emu_serial_dev:
    raise ValueError('environment variable EMU2_DEV must be defined')
mqtt_host = os.environ.get('MQTT_HOST')
mqtt_user = os.environ.get('MQTT_USER')
mqtt_pass = os.environ.get('MQTT_PASS')
discovery_prefix = os.environ.get('HA_DISCOVERY_PREFIX')
if not discovery_prefix:
    discovery_prefix = 'homeassistant'

ser = serial.Serial(emu_serial_dev, 115200)

# MQTT SESSINGS
mqtt_settings = Settings.MQTT(host=mqtt_host, username=mqtt_user, password=mqtt_pass, discovery_prefix=discovery_prefix)
device_info = DeviceInfo(name="Rainforest-EMU-2", identifiers="device_id", manufacturer='Rainforest Automation', model='EMU-2')

# Home Energy Usage Sensor
home_energy_usage_sensor_info = SensorInfo(name="HomeEnergyUsage", device_class="energy",unique_id='HomeEnergyUsage', device=device_info, unit_of_measurement='kWh')
home_energy_usage_sensor_settings = Settings(mqtt=mqtt_settings, entity=home_energy_usage_sensor_info)
home_energy_usage_sensor = Sensor(home_energy_usage_sensor_settings)

# Home Current demand sensor
current_demand_sensor_info = SensorInfo(name="HomeCurrentDemand", device_class="energy", unique_id='HomeCurrentDemand', device=device_info, unit_of_measurement='kw')
current_demand_sensor_settings = Settings(mqtt=mqtt_settings, entity=current_demand_sensor_info)
current_demand_sensor_sensor = Sensor(current_demand_sensor_settings)

# Supported messages off serial dev
SUPPORTED_READS = ['CurrentSummationDelivered', 'InstantaneousDemand']

def convert_to_dict(serial_blurb):
    try:
        data_dict = xmltodict.parse(serial_blurb)
    except Exception:
        data_dict = {}
        root.info(f'not supported XML: {serial_blurb}')
    return data_dict

def convert_data_to_metrics(blurb_dict):
    if 'InstantaneousDemand' in blurb_dict:
        items = blurb_dict['InstantaneousDemand']
        demand = int(items.get('Demand'), 16)
        multiplier = int(items.get('Multiplier'),16)
        divisor = int(items.get('Divisor'), 16)
        digitsRight = int(items.get('DigitsRight'), 16)
        value = round(((demand * multiplier) / divisor), digitsRight)
        metric_name = 'InstantaneousDemand'
    elif 'CurrentSummationDelivered' in blurb_dict:
        items = blurb_dict['CurrentSummationDelivered']
        delivered = int(items.get('SummationDelivered'), 16)
        multiplier = int(items.get('Multiplier'),16)
        divisor = int(items.get('Divisor'), 16)
        digitsRight = int(items.get('DigitsRight'), 16)
        value = round(((delivered * multiplier) / divisor), digitsRight)
        metric_name = 'SummationDelivered'
    return metric_name, value


def publish_message(metric_name, value):
    if metric_name == 'InstantaneousDemand':
        current_demand_sensor_sensor.set_state(value)
    elif metric_name == 'SummationDelivered':
        home_energy_usage_sensor.set_state(value)

def monitor_serial():
    serial_blurb = ''
    while True:
        line = ser.readline().decode('utf-8').strip()
        serial_blurb += line
        if line.startswith('</'):
            blurb_dict = convert_to_dict(serial_blurb)
            serial_blurb = ''
            if blurb_dict and list(blurb_dict.keys())[0] in SUPPORTED_READS:
                metric_name, value = convert_data_to_metrics(blurb_dict)
                root.info(f'publishing: {metric_name}-{value}')
                publish_message(metric_name, value)



if __name__ == '__main__':
    monitor_serial()