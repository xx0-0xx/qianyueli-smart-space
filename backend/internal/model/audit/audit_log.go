package audit

import (
	"smart-space-management/backend/internal/model/base"
)

// AuditLog 审计日志
type AuditLog struct {
	base.Model
	UserID     uint64 `gorm:"column:user_id;index;not null" json:"user_id"`
	StoreID    uint64 `gorm:"column:store_id;index" json:"store_id"`
	Action     string `gorm:"column:action;type:varchar(64);not null;index" json:"action"`   // refund_approve / price_change / role_assign
	TargetType string `gorm:"column:target_type;type:varchar(32)" json:"target_type"`         // order / config / user
	TargetID   uint64 `gorm:"column:target_id" json:"target_id"`
	OldValue   string `gorm:"column:old_value;type:text" json:"old_value,omitempty"`
	NewValue   string `gorm:"column:new_value;type:text" json:"new_value,omitempty"`
	IP         string `gorm:"column:ip;type:varchar(64)" json:"ip"`
	UserAgent  string `gorm:"column:user_agent;type:varchar(512)" json:"user_agent"`
	Remark     string `gorm:"column:remark;type:varchar(512)" json:"remark"`
}

func (AuditLog) TableName() string { return "audit_logs" }
