package handler

import (
	"testing"

	"smart-space-management/backend/internal/model/device"
)

func TestUserCanControlDeviceRequiresFlagAndReservation(t *testing.T) {
	cases := []struct {
		name        string
		dev         device.Device
		reservation bool
		want        bool
	}{
		{"enabled with reservation", device.Device{UserControlEnabled: true}, true, true},
		{"disabled with reservation", device.Device{UserControlEnabled: false}, true, false},
		{"enabled without reservation", device.Device{UserControlEnabled: true}, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := userCanControlDevice(tc.dev, tc.reservation); got != tc.want {
				t.Fatalf("userCanControlDevice() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUserControlCommandMustBeDeviceAllowedCommand(t *testing.T) {
	custom := device.Device{Type: device.TypeMachine, ConfigJSON: `{"cmd_on":"relay1_on","cmd_off":"relay1_off"}`}
	for _, cmd := range []string{"relay1_on", "relay1_off"} {
		if !userControlCommandAllowed(custom, cmd) {
			t.Errorf("custom command %q should be allowed", cmd)
		}
	}
	if userControlCommandAllowed(custom, "factory_reset") {
		t.Fatal("unconfigured destructive command must not be allowed")
	}
	if !userControlCommandAllowed(device.Device{Type: device.TypeDoorLock}, "door_close") {
		t.Fatal("door locks should allow close by default")
	}
}

func TestDefaultUserControlForDeviceType(t *testing.T) {
	if !defaultUserControlEnabled(device.TypeDoorLock) {
		t.Fatal("door lock should allow user control by default")
	}
	for _, typ := range []device.DeviceType{device.TypeLight, device.TypeAC, device.TypeExhaust, device.TypeMachine, device.TypeOther} {
		if defaultUserControlEnabled(typ) {
			t.Errorf("device type %q should disallow user control by default", typ)
		}
	}
}

func TestCanControlWithoutReservationRequiresMerchantClient(t *testing.T) {
	cases := []struct {
		name   string
		role   string
		client string
		want   bool
	}{
		{"admin", "admin", "admin", true},
		{"super admin", "super_admin", "admin", true},
		{"merchant app", "merchant", "merchant", true},
		{"merchant using user app", "merchant", "user", false},
		{"merchant legacy token without client", "merchant", "", false},
		{"normal user", "user", "user", false},
		{"normal user with forged merchant client", "user", "merchant", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canControlWithoutReservation(tc.role, tc.client); got != tc.want {
				t.Fatalf("canControlWithoutReservation(%q, %q) = %v, want %v", tc.role, tc.client, got, tc.want)
			}
		})
	}
}
