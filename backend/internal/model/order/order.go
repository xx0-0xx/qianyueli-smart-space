package order

import (
	"smart-space-management/backend/internal/model/base"
)

// OrderStatus 订单状态
type OrderStatus int

const (
	StatusPendingPay OrderStatus = 0 // 待支付
	StatusPaid       OrderStatus = 1 // 已支付 (预约生效)
	StatusInProgress OrderStatus = 2 // 进行中
	StatusCompleted  OrderStatus = 3 // 已完成
	StatusCanceled   OrderStatus = 4 // 已取消
	StatusRefunding  OrderStatus = 5 // 退款中
	StatusRefunded   OrderStatus = 6 // 已退款
)

// Order 订单模型
type Order struct {
	base.Model
	OrderNo         string      `gorm:"column:order_no;type:varchar(32);uniqueIndex;not null" json:"order_no"`
	UserID          uint64      `gorm:"column:user_id;index;not null" json:"user_id"`
	StoreID         uint64      `gorm:"column:store_id;index;not null" json:"store_id"`
	ResourceID      uint64      `gorm:"column:resource_id;index;not null" json:"resource_id"`
	ResourceName    string      `gorm:"column:resource_name;type:varchar(128)" json:"resource_name"`
	StartTime       int64       `gorm:"column:start_time;not null" json:"start_time"`     // 预约开始 (unix ms)
	EndTime         int64       `gorm:"column:end_time;not null" json:"end_time"`         // 预约结束 (unix ms)
	Duration        int         `gorm:"column:duration;not null" json:"duration"`         // 时长 (分钟)
	UnitPrice       int         `gorm:"column:unit_price;not null" json:"unit_price"`     // 单价 (分)
	TotalAmount     int         `gorm:"column:total_amount;not null" json:"total_amount"` // 总价 (分)
	Status          OrderStatus `gorm:"column:status;type:tinyint;not null;default:0" json:"status"`
	PaidAt          *int64      `gorm:"column:paid_at" json:"paid_at,omitempty"`
	CancelReason    string      `gorm:"column:cancel_reason;type:varchar(512)" json:"cancel_reason,omitempty"`
	RefundAmount    int         `gorm:"column:refund_amount;default:0" json:"refund_amount"`
	OriginalAmount  int         `gorm:"column:original_amount;not null;default:0" json:"original_amount"`
	MemberDiscount  int         `gorm:"column:member_discount;not null;default:0" json:"member_discount"`
	MemberLevelID   *uint64     `gorm:"column:member_level_id" json:"member_level_id,string,omitempty"`
	MemberLevelName string      `gorm:"column:member_level_name;type:varchar(64)" json:"member_level_name,omitempty"`
}

func (Order) TableName() string { return "orders" }
