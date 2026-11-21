package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateVoiceConfigAcceptsDeviceSpecificRules(t *testing.T) {
	vc := VoiceConfig{Devices: []DeviceVoiceConfig{{
		DeviceSN: "  ABC123  ", Enabled: true, OnStart: "mp3=00005", OnEnd: "mp3=00012",
		Reminders: []VoiceReminder{{MinutesBeforeEnd: 30, Command: "mp3=00009"}, {MinutesBeforeEnd: 8, Command: "mp3=00010"}},
	}}}
	if err := ValidateVoiceConfig(&vc); err != nil {
		t.Fatalf("ValidateVoiceConfig() error = %v", err)
	}
	if vc.Devices[0].DeviceSN != "ABC123" {
		t.Fatalf("SN was not trimmed: %q", vc.Devices[0].DeviceSN)
	}
}

func TestValidateVoiceConfigRejectsInvalidRules(t *testing.T) {
	valid := func() VoiceConfig {
		return VoiceConfig{Devices: []DeviceVoiceConfig{{
			DeviceSN: "ABC123", Enabled: true, OnStart: "mp3=00005", OnEnd: "mp3=00012",
			Reminders: []VoiceReminder{{MinutesBeforeEnd: 30, Command: "mp3=00009"}},
		}}}
	}
	cases := []struct {
		name   string
		mutate func(*VoiceConfig)
	}{
		{"empty SN", func(v *VoiceConfig) { v.Devices[0].DeviceSN = " " }},
		{"long SN", func(v *VoiceConfig) { v.Devices[0].DeviceSN = strings.Repeat("x", 65) }},
		{"duplicate SN", func(v *VoiceConfig) { v.Devices = append(v.Devices, v.Devices[0]) }},
		{"duplicate reminder", func(v *VoiceConfig) {
			v.Devices[0].Reminders = append(v.Devices[0].Reminders, v.Devices[0].Reminders[0])
		}},
		{"zero minute", func(v *VoiceConfig) { v.Devices[0].Reminders[0].MinutesBeforeEnd = 0 }},
		{"negative minute", func(v *VoiceConfig) { v.Devices[0].Reminders[0].MinutesBeforeEnd = -1 }},
		{"minute over one week", func(v *VoiceConfig) { v.Devices[0].Reminders[0].MinutesBeforeEnd = 10081 }},
		{"too many reminders", func(v *VoiceConfig) {
			for i := 0; i < 10; i++ {
				v.Devices[0].Reminders = append(v.Devices[0].Reminders, VoiceReminder{MinutesBeforeEnd: i + 31, Command: "mp3=00009"})
			}
		}},
		{"missing start command", func(v *VoiceConfig) { v.Devices[0].OnStart = "" }},
		{"missing reminder command", func(v *VoiceConfig) { v.Devices[0].Reminders[0].Command = " " }},
		{"control char command", func(v *VoiceConfig) { v.Devices[0].OnEnd = "mp3=00012\n" }},
		{"long command", func(v *VoiceConfig) { v.Devices[0].OnEnd = strings.Repeat("x", 65) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vc := valid()
			tc.mutate(&vc)
			if err := ValidateVoiceConfig(&vc); err == nil {
				t.Fatal("ValidateVoiceConfig() error = nil, want error")
			}
		})
	}
}

func TestValidateVoiceConfigAllowsDisabledDraftWithoutCommands(t *testing.T) {
	vc := VoiceConfig{Devices: []DeviceVoiceConfig{{DeviceSN: "ABC123", Enabled: false}}}
	if err := ValidateVoiceConfig(&vc); err != nil {
		t.Fatalf("ValidateVoiceConfig() error = %v", err)
	}
}

func TestVoiceConfigIgnoresLegacyGlobalJSON(t *testing.T) {
	var vc VoiceConfig
	if err := json.Unmarshal([]byte(`{"enabled":true,"on_start":"mp3=00005","on_30min":"mp3=00009"}`), &vc); err != nil {
		t.Fatalf("unmarshal legacy config: %v", err)
	}
	if len(vc.Devices) != 0 {
		t.Fatalf("legacy config unexpectedly populated device rules: %+v", vc.Devices)
	}
}
