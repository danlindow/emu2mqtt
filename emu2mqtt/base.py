import serial
import xmltodict
from ha_mqtt_discoverable import Settings, DeviceInfo
from ha_mqtt_discoverable.sensors import Sensor, SensorInfo
import os
from typing import Tuple
from dataclasses import dataclass
import logging
import sys

def convert_to_dict(serial_blurb) -> dict:
    try:
        data_dict = xmltodict.parse(serial_blurb)
    except Exception:
        data_dict = {}
    return data_dict

def convert_data_to_metrics(blurb_dict) -> Tuple[str, float]:
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


@dataclass
class EMUConfig:
    emu_serial_dev: str
    mqtt_host: str
    mqtt_user: str
    mqtt_pass: str
    mqtt_discovery_prefix: str
    debug: bool = False


class EMUHandler:
    # supported read values from EMU-2
    SUPPORTED_READS = ['CurrentSummationDelivered', 'InstantaneousDemand']

    def __init__(self):
        self.config: EMUConfig = self._set_config_parms()
        self.logger = logging.getLogger()
        if self.config.debug:
            logging.basicConfig(level=logging.DEBUG, stream=sys.stdout)
        else:
            logging.basicConfig(level=logging.INFO, stream=sys.stdout)      
        self.logger.info(f'config set: {self.config}')
        self.ser_dev = serial.Serial(self.config.emu_serial_dev, 115200)
        self.logger.info(f'ser device set: {self.ser_dev}')
        self.mqtt_settings = self._set_mqtt_settings()
        self.logger.info(f'mqtt settings set set: {self.mqtt_settings}')
        self.energy_usage_sensor = self._create_sensor('HomeEnergyUsage', unique_id='HomeEnergyUsage', measurement_unit='kWh')
        self.logger.info(f'energy_usage_sensor: {self.energy_usage_sensor}')
        self.current_demand_sensor = self._create_sensor('HomeCurrentDemand', unique_id='HomeCurrentDemand', measurement_unit='kw')
        self.logger.info(f'current_demand_sensor: {self.current_demand_sensor}')

    def _set_config_parms(self) -> EMUConfig:
        return EMUConfig(
            emu_serial_dev=os.environ.get('EMU2_DEV'),
            mqtt_host=os.environ.get('MQTT_HOST'),
            mqtt_user=os.environ.get('MQTT_USER'),
            mqtt_pass=os.environ.get('MQTT_PASS'),
            mqtt_discovery_prefix=os.environ.get('HA_DISCOVERY_PREFIX', 'homeassistant'),
            debug=os.environ.get('DEBUG', False)
        )

    def _set_mqtt_settings(self) -> Settings.MQTT:
        return Settings.MQTT(
            host=self.config.mqtt_host, username=self.config.mqtt_user, password=self.config.mqtt_pass, discovery_prefix=self.config.mqtt_discovery_prefix
        )

    def _create_sensor(self, sensor_name: str, unique_id: str, measurement_unit: str) -> Sensor:
        dev_info = DeviceInfo(name="Rainforest-EMU-2", identifiers="device_id", manufacturer='Rainforest Automation', model='EMU-2')
        sensor_info = SensorInfo(name=sensor_name, device_class="energy", unique_id=unique_id, device=dev_info, unit_of_measurement=measurement_unit)
        settings = Settings(mqtt=self.mqtt_settings, entity=sensor_info)
        return Sensor(settings)
    
    def publish_message(self, metric_name, value) -> None:
        if metric_name == 'InstantaneousDemand':
            self.current_demand_sensor.set_state(value)
        elif metric_name == 'SummationDelivered':
            self.energy_usage_sensor.set_state(value)

    def start(self) -> None:
        serial_blurb = ''
        while True:
            line = self.ser_dev.readline().decode('utf-8').strip()
            self.logger.debug(f'raw line: {line}')
            serial_blurb += line
            if line.startswith('</'):
                self.logger.debug(f'found line: {line}')
                blurb_dict = convert_to_dict(serial_blurb)
                serial_blurb = ''
                if blurb_dict and list(blurb_dict.keys())[0] in self.SUPPORTED_READS:
                    metric_name, value = convert_data_to_metrics(blurb_dict)
                    self.publish_message(metric_name, value)

def run_app():
    emu_handler = EMUHandler()
    emu_handler.start()

if __name__ == '__main__':
    run_app()
