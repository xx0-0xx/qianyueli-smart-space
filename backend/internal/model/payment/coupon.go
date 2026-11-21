package payment

import "time"
import "smart-space-management/backend/internal/model/base"

// Coupon 团购券
type Coupon struct {
	base.Model
	StoreID        uint64     `gorm:"column:store_id;index;not null" json:"store_id"`
	ResourceTypeID uint64     `gorm:"column:resource_type_id;not null" json:"resource_type_id"`
	Code           string     `gorm:"column:code;type:varchar(32);uniqueIndex;not null" json:"code"`
	Name           string     `gorm:"column:name;type:varchar(128)" json:"name"`
	Hours          int        `gorm:"column:hours;default:0" json:"hours"`
	Price          int        `gorm:"column:price;default:0" json:"price"`
	Type           int        `gorm:"column:type;default:0" json:"type"`           // 0=时长券 1=代金券
	Discount       int        `gorm:"column:discount;default:0" json:"discount"`   // 代金券抵扣金额（分）
	Status         int        `gorm:"column:status;default:0" json:"status"`
	UsedAt         *time.Time `gorm:"column:used_at" json:"used_at,omitempty"`
	UserID         uint64     `gorm:"column:user_id;default:0" json:"user_id"`
}

func (Coupon) TableName() string { return "coupons" }
