package member

import (
	"smart-space-management/backend/internal/model/base"
)

type Level struct {
	base.Model
	StoreID    uint64 `gorm:"column:store_id;index;not null" json:"store_id,string"`
	Name       string `gorm:"column:name;type:varchar(64);not null" json:"name"`
	DiscountBP int    `gorm:"column:discount_bp;not null;default:10000" json:"discount_bp"`
	Sort       int    `gorm:"column:sort;not null;default:0" json:"sort"`
	Status     int    `gorm:"column:status;not null" json:"status"`
}

func (Level) TableName() string { return "member_levels" }

func (l Level) EffectiveDiscountBP() int {
	if l.DiscountBP <= 0 || l.DiscountBP > 10000 {
		return 10000
	}
	return l.DiscountBP
}

type StoreMember struct {
	base.Model
	StoreID   uint64 `gorm:"column:store_id;index;not null" json:"store_id,string"`
	UserID    uint64 `gorm:"column:user_id;index;not null" json:"user_id,string"`
	LevelID   uint64 `gorm:"column:level_id;index;not null" json:"level_id,string"`
	ExpiresAt *int64 `gorm:"column:expires_at" json:"expires_at,omitempty"`
	Status    int    `gorm:"column:status;not null;default:1" json:"status"`
	Remark    string `gorm:"column:remark;type:varchar(255)" json:"remark"`
	CreatedBy uint64 `gorm:"column:created_by" json:"created_by,string"`
}

func (StoreMember) TableName() string { return "store_members" }

func (m StoreMember) IsActive(nowMs int64) bool {
	if m.Status != 1 {
		return false
	}
	return m.ExpiresAt == nil || *m.ExpiresAt > nowMs
}
