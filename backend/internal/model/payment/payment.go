package payment

import (
	"smart-space-management/backend/internal/model/base"
)

// PaymentStatus 支付状态
type PaymentStatus int

const (
	StatusPending  PaymentStatus = 0 // 待支付
	StatusSuccess  PaymentStatus = 1 // 支付成功
	StatusFailed   PaymentStatus = 2 // 支付失败
	StatusRefunded PaymentStatus = 3 // 已退款
	StatusClosed   PaymentStatus = 4 // 已关闭
)

// Payment 支付流水 — out_trade_no 作为幂等键
type Payment struct {
	base.Model
	OutTradeNo     string        `gorm:"column:out_trade_no;type:varchar(64);uniqueIndex;not null" json:"out_trade_no"`
	OrderID        uint64        `gorm:"column:order_id;index;not null" json:"order_id"`
	OrderNo        string        `gorm:"column:order_no;type:varchar(32);index;not null" json:"order_no"`
	UserID         uint64        `gorm:"column:user_id;index;not null" json:"user_id"`
	StoreID        uint64        `gorm:"column:store_id;index;not null" json:"store_id"`
	Amount         int           `gorm:"column:amount;not null" json:"amount"`         // 支付金额 (分)
	TransactionID  string        `gorm:"column:transaction_id;type:varchar(64);index" json:"transaction_id"` // 微信支付单号
	Status         PaymentStatus `gorm:"column:status;type:tinyint;not null;default:0" json:"status"`
	PayTime        *int64        `gorm:"column:pay_time" json:"pay_time,omitempty"`
	RefundAmount   int           `gorm:"column:refund_amount;default:0" json:"refund_amount"`
	RefundID       string        `gorm:"column:refund_id;type:varchar(64)" json:"refund_id,omitempty"`
	CallbackRaw    string        `gorm:"column:callback_raw;type:text" json:"callback_raw,omitempty"` // 回调原始 JSON
}

func (Payment) TableName() string { return "payments" }
