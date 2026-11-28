package main

import "testing"

func TestValidDeviceID(t *testing.T) {
	for _, value := range []string{"lab-pc-001", "ROOM_2.PC"} {
		if !validDeviceID(value) { t.Errorf("expected %q to be valid", value) }
	}
	for _, value := range []string{"", "ab", "bad/id", "has space", "x"+string(make([]byte, 65))} {
		if validDeviceID(value) { t.Errorf("expected %q to be invalid", value) }
	}
}

func TestRoundPct(t *testing.T) {
	if got:=roundPct(52.345); got!=52.3 { t.Fatalf("roundPct()=%v, want 52.3",got) }
}
