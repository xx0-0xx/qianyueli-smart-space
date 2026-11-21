package service

import (
	"testing"

	"smart-space-management/backend/config"
)

func TestVoiceCommandForDeviceEvent(t *testing.T) {
	cfg := config.DeviceVoiceConfig{DeviceSN: "SN-A", Enabled: true, OnStart: "mp3=00005", OnEnd: "mp3=00012"}
	if got := voiceCommandForDeviceEvent(cfg, "session_start"); got != "mp3=00005" {
		t.Fatalf("start command = %q", got)
	}
	if got := voiceCommandForDeviceEvent(cfg, "session_end"); got != "mp3=00012" {
		t.Fatalf("end command = %q", got)
	}
	cfg.Enabled = false
	if got := voiceCommandForDeviceEvent(cfg, "session_start"); got != "" {
		t.Fatalf("disabled command = %q, want empty", got)
	}
	cfg.Enabled = true
	if got := voiceCommandForDeviceEvent(cfg, "unknown"); got != "" {
		t.Fatalf("unknown event command = %q, want empty", got)
	}
}
