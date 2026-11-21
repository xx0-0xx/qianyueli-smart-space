package resource

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"smart-space-management/backend/internal/model/base"
)

// ResourceStatus 资源状态
type ResourceStatus int

const (
	StatusFree     ResourceStatus = 0 // 空闲
	StatusLocked   ResourceStatus = 1 // 锁定中 (下单未支付)
	StatusOccupied ResourceStatus = 2 // 使用中
	StatusMaint    ResourceStatus = 3 // 维护中
)

// ResourceType 资源类型定义 (乒乓球桌、棋牌室大包间等)
type ResourceType struct {
	base.Model
	Name        string `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Icon        string `gorm:"column:icon;type:varchar(512)" json:"icon"`
	Description string `gorm:"column:description;type:text" json:"description"`
	StoreID     uint64 `gorm:"column:store_id;index;not null" json:"store_id,string"`
	ConfigJSON  string `gorm:"column:config_json;type:text" json:"config_json"` // 自定义属性 JSON
	Sort        int    `gorm:"column:sort;default:0" json:"sort"`
}

func (ResourceType) TableName() string { return "resource_types" }

// Resource 资源实例 (乒乓球桌1号机、大包间A等)
type Resource struct {
	base.Model
	StoreID        uint64         `gorm:"column:store_id;index;not null" json:"store_id"`
	ResourceTypeID uint64         `gorm:"column:resource_type_id;index;not null" json:"resource_type_id"`
	Name           string         `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Image          string         `gorm:"column:image;type:varchar(512)" json:"image"`
	Status         ResourceStatus `gorm:"column:status;type:tinyint;not null;default:0" json:"status"`
	DeviceIDs      string         `gorm:"column:device_ids;type:varchar(1024)" json:"device_ids"` // 绑定的设备 ID 列表 (JSON)
	Sort           int            `gorm:"column:sort;default:0" json:"sort"`
	Remark         string         `gorm:"column:remark;type:varchar(512)" json:"remark"`

	// 关联
	ResourceType *ResourceType `gorm:"foreignKey:ResourceTypeID" json:"resource_type,omitempty"`
}

func (Resource) TableName() string { return "resources" }

// ParseDeviceIDs 兼容前端历史数据：device_ids 可能是 JSON 数字数组，
// 也可能是 JSON 字符串数组（如 ["3726401","3726802"]）。
func ParseDeviceIDs(raw string) ([]uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil, nil
	}

	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var values []interface{}
	if err := dec.Decode(&values); err != nil {
		return nil, err
	}

	ids := make([]uint64, 0, len(values))
	for _, value := range values {
		switch v := value.(type) {
		case string:
			id, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid device id %q: %w", v, err)
			}
			ids = append(ids, id)
		case json.Number:
			id, err := strconv.ParseUint(v.String(), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid device id %q: %w", v, err)
			}
			ids = append(ids, id)
		default:
			return nil, fmt.Errorf("unsupported device id type %T", value)
		}
	}
	return ids, nil
}
