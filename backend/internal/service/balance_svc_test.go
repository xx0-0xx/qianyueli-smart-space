package service

import "testing"

func TestDecideGroupChange(t *testing.T) {
	cases := []struct {
		name          string
		same          bool
		srcHasOthers  bool
		srcBalancePos bool
		wantAction    GroupChangeAction
		wantErr       bool
	}{
		{"同组不改", true, false, true, GroupChangeNone, false},
		{"独立店并入他组→迁移账户", false, false, true, GroupChangeMoveAccounts, false},
		{"共享组有余额→拒绝拆分", false, true, true, GroupChangeRejectSplitBalance, true},
		{"共享组无余额→仅改归属", false, true, false, GroupChangeLeaveEmpty, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := DecideGroupChange(c.same, c.srcHasOthers, c.srcBalancePos)
			if (err != nil) != c.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, c.wantErr)
			}
			if got != c.wantAction {
				t.Fatalf("action=%v want=%v", got, c.wantAction)
			}
		})
	}
}
