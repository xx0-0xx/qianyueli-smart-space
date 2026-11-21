package device

import "testing"

func TestIsStatusTopicSupportsStatusAndReport(t *testing.T) {
	for _, topic := range []string{
		"ssm/device/ABC123/status",
		"ssm/device/ABC123/report",
	} {
		if !isStatusTopic(topic) {
			t.Errorf("isStatusTopic(%q) = false, want true", topic)
		}
	}
	if isStatusTopic("ssm/device/ABC123/event") {
		t.Fatal("event topic must not be treated as status")
	}
}

func TestParseClientStateTopic(t *testing.T) {
	cases := []struct {
		topic  string
		client string
		online bool
		ok     bool
	}{
		{"$SYS/brokers/emqx@node/clients/ESP_ABC123/connected", "ESP_ABC123", true, true},
		{"$SYS/brokers/emqx@node/clients/ESP_ABC123/disconnected", "ESP_ABC123", false, true},
		{"ssm/device/ABC123/status", "", false, false},
	}
	for _, tc := range cases {
		client, online, ok := parseClientStateTopic(tc.topic)
		if client != tc.client || online != tc.online || ok != tc.ok {
			t.Errorf("parseClientStateTopic(%q) = (%q,%v,%v), want (%q,%v,%v)", tc.topic, client, online, ok, tc.client, tc.online, tc.ok)
		}
	}
}

func TestDeviceSNFromMQTTClientID(t *testing.T) {
	cases := []struct{ clientID, want string }{
		{"ESP_d8bfc01acfb3", "d8bfc01acfb3"},
		{"r4_d8bfc01acfb3_987", "d8bfc01acfb3"},
		{"s3_d8bfc01acfb3_45", "d8bfc01acfb3"},
		{"-8382", ""},
		{"ESP_", ""},
	}
	for _, tc := range cases {
		if got := DeviceSNFromMQTTClientID(tc.clientID); got != tc.want {
			t.Errorf("deviceSNFromMQTTClientID(%q) = %q, want %q", tc.clientID, got, tc.want)
		}
	}
}
