package handler

import "testing"

func TestMerchantMayControlDeviceOnlyInCurrentStore(t *testing.T) {
	cases := []struct {
		name, role      string
		current, device uint64
		want            bool
	}{
		{"same authorized store", "merchant", 10, 10, true},
		{"different store", "merchant", 10, 20, false},
		{"missing current store", "merchant", 0, 10, false},
		{"admin keeps access", "admin", 0, 10, true},
		{"super admin keeps access", "super_admin", 0, 10, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := merchantDeviceStoreAllowed(tc.role, tc.current, tc.device); got != tc.want {
				t.Fatalf("merchantDeviceStoreAllowed(%q, %d, %d) = %v, want %v", tc.role, tc.current, tc.device, got, tc.want)
			}
		})
	}
}
