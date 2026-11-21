package device

import (
	"smart-space-management/backend/internal/model/base"
)

// DeviceStatus 设备在线状态
type DeviceStatus int

const (
	StatusOffline DeviceStatus = 0
	StatusOnline  DeviceStatus = 1
)

// DeviceType 设备类型
type DeviceType string

const (
	TypeDoorLock DeviceType = "door_lock"
	TypeLight    DeviceType = "light"
	TypeAC       DeviceType = "ac"
	TypeExhaust  DeviceType = "exhaust"
	TypeMachine  DeviceType = "machine"
	TypeSpeaker  DeviceType = "speaker"
	TypeOther    DeviceType = "other"
)

// Device 设备实例
type Device struct {
	base.Model
	StoreID            uint64       `gorm:"column:store_id;index;not null" json:"store_id"`
	DeviceSN           string       `gorm:"column:device_sn;type:varchar(64);index;not null" json:"device_sn"` // 设备序列号（同一SN可建多条）
	Name               string       `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Type               DeviceType   `gorm:"column:type;type:varchar(32);not null" json:"type"`
	Status             DeviceStatus `gorm:"column:status;type:tinyint;not null;default:0" json:"status"`
	MQTTTopic          string       `gorm:"column:mqtt_topic;type:varchar(256)" json:"mqtt_topic"`
	LastOnlineAt       *int64       `gorm:"column:last_online_at" json:"last_online_at,omitempty"`
	ConfigJSON         string       `gorm:"column:config_json;type:text" json:"config_json"`                 // 设备自定义配置
	Protocol           string       `gorm:"column:protocol;type:varchar(16);default:'json'" json:"protocol"` // json=自制固件 | ascii=成品控制器
	UserControlEnabled bool         `gorm:"column:user_control_enabled;type:tinyint(1);not null;default:false" json:"user_control_enabled"`
}

// 设备通信协议
const (
	ProtocolJSON  = "json"  // 自制固件：发 JSON {"cmd":"..."}
	ProtocolASCII = "ascii" // 成品控制器：发纯文本 "a1"
)

func (Device) TableName() string { return "devices" }
