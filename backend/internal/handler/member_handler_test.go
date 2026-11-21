package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"smart-space-management/backend/internal/model/member"
)

func TestValidateMemberLevel(t *testing.T) {
	cases := []struct {
		name       string
		levelName  string
		discountBP int
		wantErr    bool
	}{
		{"ok", "金卡", 8500, false},
		{"no name", "", 8500, true},
		{"blank name", "   ", 8500, true},
		{"too high", "金卡", 10001, true},
		{"zero discount", "金卡", 0, true},
		{"minimum discount", "金卡", 1, false},
		{"maximum discount", "金卡", 10000, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMemberLevel(tc.levelName, tc.discountBP)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateMemberLevel() err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestCustomerCanBeMember(t *testing.T) {
	cases := []struct {
		name  string
		phone string
		want  bool
	}{
		{"empty", "", false},
		{"blank", "   ", false},
		{"phone", "13800138000", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := customerCanBeMember(tc.phone); got != tc.want {
				t.Fatalf("customerCanBeMember(%q)=%v want %v", tc.phone, got, tc.want)
			}
		})
	}
}

func TestNetSpent(t *testing.T) {
	cases := []struct {
		name   string
		total  int64
		refund int64
		want   int64
	}{
		{"no refund", 10000, 0, 10000},
		{"partial refund", 10000, 1500, 8500},
		{"full refund", 10000, 10000, 0},
		{"zero", 0, 0, 0},
		{"refund exceeds total", 10000, 12000, -2000},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := netSpent(tc.total, tc.refund); got != tc.want {
				t.Fatalf("netSpent(%d, %d)=%d want %d", tc.total, tc.refund, got, tc.want)
			}
		})
	}
}

func TestNewLevelDTO(t *testing.T) {
	var level member.Level
	level.ID = 123
	level.StoreID = 456
	level.Name = "金卡"
	level.DiscountBP = 8500
	level.Sort = 3
	level.Status = 1

	dto := newLevelDTO(level, 7)
	if dto.ID != 123 || dto.StoreID != 456 || dto.Name != "金卡" ||
		dto.DiscountBP != 8500 || dto.Sort != 3 || dto.Status != 1 || dto.MemberCount != 7 {
		t.Fatalf("unexpected dto: %+v", dto)
	}

	raw, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal dto: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, `"id":123`) || !strings.Contains(body, `"store_id":456`) {
		t.Fatalf("level ids should be numeric: %s", body)
	}
	if strings.Contains(body, `"id":"123"`) || strings.Contains(body, `"store_id":"456"`) {
		t.Fatalf("level ids must not be strings: %s", body)
	}
}
