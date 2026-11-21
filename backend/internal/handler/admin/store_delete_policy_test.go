package admin

import "testing"

func TestStoreDeleteAssociationsIncludeStoreOwnedRecords(t *testing.T) {
	associations := storeDeleteAssociations()
	want := map[string]bool{
		"orders": true, "resources": true, "resource_types": true, "devices": true,
		"merchant_stores": true, "payments": true, "refund_requests": true, "recharges": true,
		"coupons": true, "price_rules": true, "booking_rules": true, "ui_configs": true,
		"config_histories": true, "audit_logs": true, "product_templates": true,
	}
	for _, association := range associations {
		delete(want, association.table)
	}
	if len(want) != 0 {
		t.Fatalf("store delete guard is missing association tables: %v", want)
	}
	for _, association := range associations {
		if association.label == "" {
			t.Errorf("association %q must have a user-facing label", association.table)
		}
	}
}
