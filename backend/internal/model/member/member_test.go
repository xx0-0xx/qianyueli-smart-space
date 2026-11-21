package member

import (
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestStoreMemberIsActive(t *testing.T) {
	future := int64(2000)
	past := int64(1000)
	now := int64(1500)
	cases := []struct {
		name string
		m    StoreMember
		now  int64
		want bool
	}{
		{"permanent", StoreMember{Status: 1}, 1500, true},
		{"not expired", StoreMember{Status: 1, ExpiresAt: &future}, 1500, true},
		{"expired", StoreMember{Status: 1, ExpiresAt: &past}, 1500, false},
		{"expires exactly now", StoreMember{Status: 1, ExpiresAt: &now}, 1500, false},
		{"disabled", StoreMember{Status: 0}, 1500, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.m.IsActive(tc.now); got != tc.want {
				t.Fatalf("IsActive()=%v want %v", got, tc.want)
			}
		})
	}
}

func TestLevelEffectiveDiscountBP(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{8500, 8500}, {10000, 10000}, {0, 10000}, {-1, 10000}, {10001, 10000},
	}
	for _, tc := range cases {
		if got := (Level{DiscountBP: tc.in}).EffectiveDiscountBP(); got != tc.want {
			t.Fatalf("discount %d => %d want %d", tc.in, got, tc.want)
		}
	}
}

func TestCreateLevelPersistsStatusZero(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "user:pass@tcp(127.0.0.1:3306)/member_test?charset=utf8mb4&parseTime=True&loc=Local",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("open dry-run db: %v", err)
	}

	level := Level{
		StoreID:    456,
		Name:       "停用等级",
		DiscountBP: 9000,
		Sort:       2,
		Status:     0,
	}
	level.ID = 123

	tx := db.Session(&gorm.Session{DryRun: true}).Create(&level)
	if tx.Error != nil {
		t.Fatalf("dry-run create: %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	start := strings.Index(sql, "(")
	end := strings.Index(sql, ")")
	if start < 0 || end <= start {
		t.Fatalf("unexpected insert SQL: %s", sql)
	}

	columns := strings.Split(sql[start+1:end], ",")
	statusIndex := -1
	for i, column := range columns {
		if strings.Trim(strings.TrimSpace(column), "`") == "status" {
			statusIndex = i
			break
		}
	}
	if statusIndex < 0 {
		t.Fatalf("status column omitted from INSERT, zero replaced by DB default: %s", sql)
	}
	if statusIndex >= len(tx.Statement.Vars) {
		t.Fatalf("status index %d out of vars range %d", statusIndex, len(tx.Statement.Vars))
	}
	if got := fmt.Sprint(tx.Statement.Vars[statusIndex]); got != "0" {
		t.Fatalf("status insert value=%s want 0; sql=%s vars=%v", got, sql, tx.Statement.Vars)
	}
}
