package device

import (
	"encoding/json"

	"smart-space-management/backend/internal/model/base"
)

// ProductTemplate 产品模板（管理后台维护的硬件模板）
// 一个模板定义一组子设备（如 3路灯开关 = 3个灯子设备）
type ProductTemplate struct {
	base.Model
	Name        string `gorm:"column:name;type:varchar(128);not null" json:"name"`
	StoreID     uint64 `gorm:"column:store_id;not null;default:0" json:"store_id"`              // 0=全局模板
	DevicesJSON string `gorm:"column:devices_json;type:text" json:"devices_json"`               // 子设备定义 JSON
	Status      int    `gorm:"column:status;not null;default:1" json:"status"`                  // 1=启用 0=停用
	Protocol    string `gorm:"column:protocol;type:varchar(16);default:'json'" json:"protocol"` // 通信协议 json/ascii
	Type        string `gorm:"column:type;type:varchar(32);default:'other'" json:"type"`        // 无子设备模板生成的设备类型
	Sort        int    `gorm:"column:sort;not null;default:0" json:"sort"`                      // 展示顺序，越小越靠前
}

func (ProductTemplate) TableName() string { return "product_templates" }

// TemplateDevice 子设备定义
type TemplateDevice struct {
	Name     string `json:"name"`      // 设备名（灯1/灯2）
	GPIO     int    `json:"gpio"`      // GPIO 编号
	CmdOn    string `json:"cmd_on"`    // 开指令
	CmdOff   string `json:"cmd_off"`   // 关指令
	Type     string `json:"type"`      // 设备类型 light/door_lock/machine...
	IsToggle bool   `json:"is_toggle"` // 是否开关型（有开有关）；false=点动型（如门锁只有开门）
}

// Devices 解析子设备定义
func (t *ProductTemplate) Devices() []TemplateDevice {
	var list []TemplateDevice
	if t.DevicesJSON != "" {
		json.Unmarshal([]byte(t.DevicesJSON), &list)
	}
	return list
}

// SetDevices 序列化子设备定义
func (t *ProductTemplate) SetDevices(list []TemplateDevice) error {
	b, err := json.Marshal(list)
	if err != nil {
		return err
	}
	t.DevicesJSON = string(b)
	return nil
}
