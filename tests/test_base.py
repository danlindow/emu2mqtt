from emu2mqtt.base import convert_data_to_metrics, convert_to_dict
import pytest

@pytest.fixture
def raw_instant_demand():
    return """
<InstantaneousDemand>
<DeviceMacId>0xf00</DeviceMacId>
<MeterMacId>0x001c6400135bce4c</MeterMacId>
<TimeStamp>0x2f4fb816</TimeStamp>
<Demand>0x000254</Demand>
<Multiplier>0x00000003</Multiplier>
<Divisor>0x000003e8</Divisor>
<DigitsRight>0x03</DigitsRight>
<DigitsLeft>0x05</DigitsLeft>
<SuppressLeadingZero>Y</SuppressLeadingZero>
</InstantaneousDemand>
"""


def test_convert_to_dict(raw_instant_demand):
    assert type(convert_to_dict(raw_instant_demand)) == dict

def test_convert_data_to_metrics(raw_instant_demand):
    assert convert_data_to_metrics(convert_to_dict(raw_instant_demand)) == ("InstantaneousDemand", 1.788)