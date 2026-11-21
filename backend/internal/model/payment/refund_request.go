package payment

import "smart-space-management/backend/internal/model/base"

// RefundStatus 退款申请状态
type RefundStatus int

const (
	RefundPending  RefundStatus = 0 // 待处理
	RefundApproved RefundStatus = 1 // 已同意
	RefundRejected RefundStatus = 2 // 已拒绝
)

// RefundRequest 退款申请表
type RefundRequest struct {
	base.Model
	OrderID  uint64       `gorm:"column:order_id;index;not null" json:"order_id"`
	UserID   uint64       `gorm:"column:user_id;index;not null" json:"user_id"`
	StoreID  uint64       `gorm:"column:store_id;index;not null" json:"store_id"`
	Amount   int          `gorm:"column:amount;not null" json:"amount"`
	Reason   string       `gorm:"column:reason;type:varchar(512)" json:"reason"`
	Status   RefundStatus `gorm:"column:status;default:0" json:"status"`
}

func (RefundRequest) TableName() string { return "refund_requests" }
