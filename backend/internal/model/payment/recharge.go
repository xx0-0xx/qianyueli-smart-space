package payment

import (
	"smart-space-management/backend/internal/model/base"
	"time"
)

// RechargeStatus 充值单状态
type RechargeStatus int

const (
	RechargePending RechargeStatus = 0 // 待确认（顾客已提交，线下付款后等商家确认）
	RechargeDone    RechargeStatus = 1 // 已到账
	RechargeCancel  RechargeStatus = 2 // 已取消
)

// Recharge 充值单（线下收款，商家确认到账）
type Recharge struct {
	base.Model
	UserID  uint64         `gorm:"column:user_id;index;not null" json:"user_id"`
	StoreID uint64         `gorm:"column:store_id;index;not null" json:"store_id"`
	Amount  int64          `gorm:"column:amount;not null" json:"amount"` // 充值金额（分）
	Status  RechargeStatus `gorm:"column:status;not null;default:0" json:"status"`
	Note    string         `gorm:"column:note;type:varchar(255)" json:"note"`
	ConfirmAt *time.Time   `gorm:"column:confirm_at" json:"confirm_at,omitempty"`
	ConfirmBy uint64       `gorm:"column:confirm_by;default:0" json:"confirm_by"`
}

func (Recharge) TableName() string { return "recharges" }

// BalanceLog 余额流水
type BalanceLog struct {
	base.Model
	UserID   uint64 `gorm:"column:user_id;index;not null" json:"user_id"`
	Amount   int64  `gorm:"column:amount;not null" json:"amount"`   // 变动金额（分，正=入账 负=扣款）
	Balance  int64  `gorm:"column:balance;not null" json:"balance"` // 变动后余额
	Type     string `gorm:"column:type;type:varchar(32);not null" json:"type"` // recharge=充值 order=订单扣款 refund=退款
	RefID    string `gorm:"column:ref_id;type:varchar(64)" json:"ref_id"`      // 关联单据号
	Desc     string `gorm:"column:desc;type:varchar(255)" json:"desc"`
	StoreID  uint64 `gorm:"column:store_id;index" json:"store_id"` // 归属门店
	GroupID  uint64 `gorm:"column:group_id;index" json:"group_id"` // 归属余额组
}

func (BalanceLog) TableName() string { return "balance_logs" }
