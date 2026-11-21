package handler

import (
	"testing"

	"smart-space-management/backend/internal/model/order"
)

func TestCanRefundFromBalanceRequiresPaidOrderAndBalanceDebit(t *testing.T) {
	cases := []struct {
		name       string
		status     order.OrderStatus
		debitFound bool
		want       bool
	}{
		{"paid balance order", order.StatusPaid, true, true},
		{"in progress balance order", order.StatusInProgress, true, true},
		{"paid without balance debit", order.StatusPaid, false, false},
		{"already refunding", order.StatusRefunding, true, false},
		{"already refunded", order.StatusRefunded, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canRefundFromBalance(tc.status, tc.debitFound); got != tc.want {
				t.Fatalf("canRefundFromBalance(%v, %v) = %v, want %v", tc.status, tc.debitFound, got, tc.want)
			}
		})
	}
}

func TestCanRefundSettleAllowsRefunding(t *testing.T) {
	cases := []struct {
		name       string
		status     order.OrderStatus
		debitFound bool
		want       bool
	}{
		{"paid", order.StatusPaid, true, true},
		{"in progress", order.StatusInProgress, true, true},
		{"refunding (用户已申请，待审批)", order.StatusRefunding, true, true},
		{"refunding without debit", order.StatusRefunding, false, false},
		{"already refunded", order.StatusRefunded, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canRefundSettle(tc.status, tc.debitFound); got != tc.want {
				t.Fatalf("canRefundSettle(%v, %v) = %v, want %v", tc.status, tc.debitFound, got, tc.want)
			}
		})
	}
}
