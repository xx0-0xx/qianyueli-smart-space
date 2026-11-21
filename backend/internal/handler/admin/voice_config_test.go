package admin

import (
	"testing"

	"smart-space-management/backend/internal/model/device"
)

func TestVoiceSNExistsAsASCII(t *testing.T) {
	rows := []device.Device{
		{DeviceSN: "SN-JSON", Protocol: device.ProtocolJSON},
		{DeviceSN: "SN-ASCII", Protocol: device.ProtocolASCII},
		{DeviceSN: "SN-ASCII", Protocol: device.ProtocolASCII},
	}
	if !voiceSNExistsAsASCII(rows, "SN-ASCII") {
		t.Fatal("expected existing ASCII SN to be accepted")
	}
	for _, sn := range []string{"SN-JSON", "UNKNOWN"} {
		if voiceSNExistsAsASCII(rows, sn) {
			t.Errorf("SN %q should not be accepted as ASCII", sn)
		}
	}
}
