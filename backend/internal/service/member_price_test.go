package service

import "testing"

func TestApplyMemberDiscount(t *testing.T) {
	cases := []struct {
		name       string
		original   int
		discountBP int
		total      int
		discount   int
	}{
		{"no discount", 10000, 10000, 10000, 0},
		{"85 off", 10000, 8500, 8500, 1500},
		{"round down", 9999, 8500, 8499, 1500},
		{"invalid high", 10000, 12000, 10000, 0},
		{"invalid zero", 10000, 0, 10000, 0},
		{"minimum one fen", 1, 1, 1, 0},
		{"zero amount", 0, 8500, 0, 0},
		{"negative amount", -100, 8500, 0, 0},
		{"large amount overflow 32-bit", 1000000000, 8500, 850000000, 150000000},
		{"one basis point large amount", 1000000, 1, 100, 999900},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			total, discount := ApplyMemberDiscount(tc.original, tc.discountBP)
			if total != tc.total || discount != tc.discount {
				t.Fatalf("total=%d discount=%d want total=%d discount=%d", total, discount, tc.total, tc.discount)
			}
		})
	}
}
