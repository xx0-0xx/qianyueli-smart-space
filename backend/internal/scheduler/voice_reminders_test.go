package scheduler

import (
	"testing"
	"time"

	"smart-space-management/backend/config"
)

func TestVoiceReminderMatchWindow(t *testing.T) {
	minute := int64(time.Minute / time.Millisecond)
	tests := []struct {
		remain int64
		want   bool
	}{
		{29*minute + 1, true}, {30 * minute, true}, {31 * minute, true},
		{29 * minute, false}, {31*minute + 1, false},
	}
	for _, tc := range tests {
		if got := voiceReminderMatches(tc.remain, 30); got != tc.want {
			t.Errorf("voiceReminderMatches(%d, 30) = %v, want %v", tc.remain, got, tc.want)
		}
	}
}

func TestVoiceReminderDedupKeyIncludesDeviceAndMinute(t *testing.T) {
	a := voiceReminderKey(12, "SN-A", 10)
	if a == voiceReminderKey(12, "SN-B", 10) || a == voiceReminderKey(12, "SN-A", 5) || a != voiceReminderKey(12, "SN-A", 10) {
		t.Fatalf("unexpected reminder keys: %q", a)
	}
}

func TestMatchingVoiceRemindersForDevice(t *testing.T) {
	cfg := config.DeviceVoiceConfig{Enabled: true, Reminders: []config.VoiceReminder{{MinutesBeforeEnd: 30, Command: "a"}, {MinutesBeforeEnd: 10, Command: "b"}}}
	if got := matchingVoiceReminders(cfg, 10*60*1000); len(got) != 1 || got[0].Command != "b" {
		t.Fatalf("10m reminders = %#v", got)
	}
	cfg.Enabled = false
	if got := matchingVoiceReminders(cfg, 10*60*1000); len(got) != 0 {
		t.Fatalf("disabled reminders = %#v", got)
	}
}
