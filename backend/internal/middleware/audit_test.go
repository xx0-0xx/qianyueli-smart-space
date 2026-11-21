package middleware

import "testing"

func TestIsAuditablePath(t *testing.T) {
	cases := map[string]bool{
		"/api/v1/admin/stores":                       true,
		"/api/v1/admin/stores/:id":                   true,
		"/api/v1/merchant/orders/:id/refund":         true,
		"/api/v1/merchant/orders/:id/cancel":         true,
		"/api/v1/merchant/recharges/:id/confirm":     true,
		"/api/v1/merchant/coupons":                   true,
		"/api/v1/merchant/config/price-rules":        true,
		"/api/v1/merchant/refund-requests/:id/approve": true,
		"/api/v1/devices/:id/control":                false,
		"/api/v1/orders":                             false,
		"/api/v1/merchant":                           true,
		"":                                           false,
	}
	for path, want := range cases {
		if got := isAuditablePath(path); got != want {
			t.Errorf("isAuditablePath(%q) = %v, want %v", path, got, want)
		}
	}
}
