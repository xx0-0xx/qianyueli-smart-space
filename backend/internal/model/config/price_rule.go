package config

import (
	"smart-space-management/backend/internal/model/base"
)

// PriceRule 计费规则
type PriceRule struct {
	base.Model
	StoreID        uint64 `gorm:"column:store_id;index;not null" json:"store_id"`
	ResourceTypeID uint64 `gorm:"column:resource_type_id;index;not null" json:"resource_type_id"`
	Name           string `gorm:"column:name;type:varchar(64);not null" json:"name"`
	TimeUnit       int    `gorm:"column:time_unit;not null;default:60" json:"time_unit"`     // 时长单位 分钟
	BasePrice      int    `gorm:"column:base_price;not null" json:"base_price"`              // 基础单价 (分)
	// 高峰时段加价 JSON: [{"start":"18:00","end":"22:00","extra":500},{"start":"08:00","end":"12:00","extra":300}]
	PeakSurcharge string `gorm:"column:peak_surcharge;type:text" json:"peak_surcharge"`
	// 包时段优惠 JSON: [{"hours":2,"discount":90},{"hours":4,"discount":80}]
	PackageDiscount string `gorm:"column:package_discount;type:text" json:"package_discount"`
	IsDefault       bool   `gorm:"column:is_default;default:false" json:"is_default"`
	Status          int    `gorm:"column:status;type:tinyint;default:1" json:"status"` // 1 启用 0 停用
}

func (PriceRule) TableName() string { return "price_rules" }

// BookingRule 预约规则
type BookingRule struct {
	base.Model
	StoreID        uint64 `gorm:"column:store_id;index;not null" json:"store_id"`
	ResourceTypeID uint64 `gorm:"column:resource_type_id;index;not null" json:"resource_type_id"`
	AdvanceMinutes int    `gorm:"column:advance_minutes;default:30" json:"advance_minutes"`     // 最少提前 N 分钟
	MaxDuration    int    `gorm:"column:max_duration;default:240" json:"max_duration"`          // 最长预约 N 分钟
	MinDuration    int    `gorm:"column:min_duration;default:30" json:"min_duration"`           // 最短预约 N 分钟
	CancelBefore   int    `gorm:"column:cancel_before;default:30" json:"cancel_before"`         // 可取消时间 (开始前 N 分钟)
	MaxAdvanceDays int    `gorm:"column:max_advance_days;default:7" json:"max_advance_days"`    // 最多提前 N 天
}

func (BookingRule) TableName() string { return "booking_rules" }

// UIConfig UI 配置 (控制小程序展示)
type UIConfig struct {
	base.Model
	StoreID uint64 `gorm:"column:store_id;index;not null" json:"store_id"`
	Key     string `gorm:"column:key;type:varchar(64);not null" json:"key"`   // 配置键
	Value   string `gorm:"column:value;type:text" json:"value"`               // 配置值 (JSON)
	Remark  string `gorm:"column:remark;type:varchar(255)" json:"remark"`
}

func (UIConfig) TableName() string { return "ui_configs" }

// ConfigHistory 配置变更历史 (用于回滚)
type ConfigHistory struct {
	base.Model
	ConfigType string `gorm:"column:config_type;type:varchar(32);not null;index" json:"config_type"` // price_rule / booking_rule / ui_config
	ConfigID   uint64 `gorm:"column:config_id;not null;index" json:"config_id"`
	StoreID    uint64 `gorm:"column:store_id;index" json:"store_id"`
	OldValue   string `gorm:"column:old_value;type:text" json:"old_value"`        // 变更前 JSON
	NewValue   string `gorm:"column:new_value;type:text" json:"new_value"`        // 变更后 JSON
	OperatorID uint64 `gorm:"column:operator_id" json:"operator_id"`
	Version    int    `gorm:"column:version;not null;default:1" json:"version"`
}

func (ConfigHistory) TableName() string { return "config_histories" }
