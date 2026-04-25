package serial

import "testing"

const rawInstantDemand = `<InstantaneousDemand>
<DeviceMacId>0xf00</DeviceMacId>
<MeterMacId>0x001c6400135bce4c</MeterMacId>
<TimeStamp>0x2f4fb816</TimeStamp>
<Demand>0x000254</Demand>
<Multiplier>0x00000003</Multiplier>
<Divisor>0x000003e8</Divisor>
<DigitsRight>0x03</DigitsRight>
<DigitsLeft>0x05</DigitsLeft>
<SuppressLeadingZero>Y</SuppressLeadingZero>
</InstantaneousDemand>`

func TestParseHex(t *testing.T) {
	cases := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"0x000254", 596, false},
		{"0x00000003", 3, false},
		{"0x000003e8", 1000, false},
		{"0x03", 3, false},
		{"0x0", 0, false},
		{"", 0, true},
		{"0xgg", 0, true},
	}
	for _, c := range cases {
		got, err := parseHex(c.input)
		if c.wantErr && err == nil {
			t.Errorf("parseHex(%q): expected error", c.input)
		}
		if !c.wantErr && err != nil {
			t.Errorf("parseHex(%q): unexpected error: %v", c.input, err)
		}
		if !c.wantErr && got != c.want {
			t.Errorf("parseHex(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}

func TestComputeValue(t *testing.T) {
	// Matches the Python test: demand=0x254=596, mul=0x3=3, div=0x3e8=1000, digitsRight=3
	// (596 * 3) / 1000 = 1.788
	val, err := computeValue("0x000254", "0x00000003", "0x000003e8", "0x03")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 1.788 {
		t.Errorf("got %v, want 1.788", val)
	}
}

func TestComputeValue_DivisorZero(t *testing.T) {
	_, err := computeValue("0x01", "0x01", "0x00", "0x03")
	if err == nil {
		t.Error("expected error for zero divisor")
	}
}

func TestParseMessage_InstantaneousDemand(t *testing.T) {
	m, err := parseMessage(rawInstantDemand)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("got nil metric, want non-nil")
	}
	if m.SensorName != "HomeCurrentDemand" {
		t.Errorf("SensorName = %q, want HomeCurrentDemand", m.SensorName)
	}
	if m.Value != 1.788 {
		t.Errorf("Value = %v, want 1.788", m.Value)
	}
}

func TestParseMessage_Unsupported(t *testing.T) {
	m, err := parseMessage("<SomeOtherTag><Foo>bar</Foo></SomeOtherTag>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m != nil {
		t.Errorf("expected nil metric for unsupported message, got %+v", m)
	}
}

func TestParseMessage_InvalidHex(t *testing.T) {
	raw := `<InstantaneousDemand>
<Demand>not-hex</Demand>
<Multiplier>0x3</Multiplier>
<Divisor>0x3e8</Divisor>
<DigitsRight>0x3</DigitsRight>
</InstantaneousDemand>`
	_, err := parseMessage(raw)
	if err == nil {
		t.Error("expected error for invalid hex in demand field")
	}
}
