package store

import (
	"smart-space-management/backend/internal/model/base"
)

// StoreStatus 门店状态
type StoreStatus int

const (
	StatusOpen   StoreStatus = 1
	StatusClose  StoreStatus = 2
	StatusMaint  StoreStatus = 3
)

// Store 门店/店铺模型
type Store struct {
	base.Model
	Name        string      `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Logo        string      `gorm:"column:logo;type:varchar(512)" json:"logo"`
	Cover       string      `gorm:"column:cover;type:varchar(512)" json:"cover"`       // 封面图
	Phone       string      `gorm:"column:phone;type:varchar(20)" json:"phone"`
	Address     string      `gorm:"column:address;type:varchar(512);not null" json:"address"`
	Province    string      `gorm:"column:province;type:varchar(32)" json:"province"`
	City        string      `gorm:"column:city;type:varchar(32)" json:"city"`
	District    string      `gorm:"column:district;type:varchar(32)" json:"district"`
	Latitude    float64     `gorm:"column:latitude;type:decimal(10,7)" json:"latitude"`
	Longitude   float64     `gorm:"column:longitude;type:decimal(10,7)" json:"longitude"`
	Status      StoreStatus `gorm:"column:status;type:tinyint;not null;default:1" json:"status"`
	Description string      `gorm:"column:description;type:text" json:"description"`
	OpenTime    string      `gorm:"column:open_time;type:varchar(8);default:'00:00'" json:"open_time"`     // HH:MM
	CloseTime   string      `gorm:"column:close_time;type:varchar(8);default:'24:00'" json:"close_time"`
	Tags        string      `gorm:"column:tags;type:varchar(512)" json:"tags"` // JSON 数组字符串
	InviteCode  string      `gorm:"column:invite_code;type:varchar(32);uniqueIndex" json:"invite_code,omitempty"` // 商家入驻邀请码
	ConfigJSON  string      `gorm:"column:config_json;type:text" json:"config_json"` // {wifi_ssid,wifi_pass,phone}
	NotifyWebhook string    `gorm:"column:notify_webhook;type:varchar(512)" json:"notify_webhook"` // 门店群机器人 webhook（老板群）
	BalanceGroupID uint64   `gorm:"column:balance_group_id;not null;default:0;index" json:"balance_group_id"` // 所属余额组
	AutoApproveRefund bool  `gorm:"column:auto_approve_refund;not null;default:1" json:"auto_approve_refund"`  // 用户申请退款自动同意（默认开）
}

func (Store) TableName() string { return "stores" }
