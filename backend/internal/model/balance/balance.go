package balance

import "smart-space-management/backend/internal/model/base"

// BalanceGroup 余额组：组内门店共享用户余额
type BalanceGroup struct {
	base.Model
	Name string `gorm:"column:name;type:varchar(128);not null" json:"name"`
}

func (BalanceGroup) TableName() string { return "balance_groups" }

// Account 用户在某余额组的余额（分）
type Account struct {
	base.Model
	UserID  uint64 `gorm:"column:user_id;not null;uniqueIndex:uk_user_group" json:"user_id"`
	GroupID uint64 `gorm:"column:group_id;not null;uniqueIndex:uk_user_group;index" json:"group_id"`
	Balance int64  `gorm:"column:balance;not null;default:0" json:"balance"`
}

func (Account) TableName() string { return "balance_accounts" }
