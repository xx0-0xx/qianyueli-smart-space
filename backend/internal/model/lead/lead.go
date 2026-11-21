// Package lead 官网「预约演示」意向客户线索
package lead

import "smart-space-management/backend/internal/model/base"

// 线索状态
const (
	StatusNew       = 0 // 新线索
	StatusContacted = 1 // 已联系
	StatusDeal      = 2 // 已成交
	StatusInvalid   = 3 // 无效
)

// Lead 官网留资线索（demo_leads 表）
type Lead struct {
	base.Model
	Name      string `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Phone     string `gorm:"column:phone;type:varchar(20);not null;index" json:"phone"`
	Company   string `gorm:"column:company;type:varchar(128)" json:"company"`
	City      string `gorm:"column:city;type:varchar(64)" json:"city"`
	StoreType string `gorm:"column:store_type;type:varchar(64)" json:"store_type"`
	Source    string `gorm:"column:source;type:varchar(32);default:'website'" json:"source"`
	Remark    string `gorm:"column:remark;type:varchar(255)" json:"remark"`
	IP        string `gorm:"column:ip;type:varchar(64)" json:"ip"`
	UserAgent string `gorm:"column:user_agent;type:varchar(255)" json:"user_agent"`
	Status    int    `gorm:"column:status;type:tinyint;default:0" json:"status"`
}

func (Lead) TableName() string { return "demo_leads" }
