// Package device 设备通信协议编解码
package device

import (
	"encoding/json"
	"fmt"
)

// Command 标准设备指令
type Command struct {
	Cmd    string                 `json:"cmd"`
	Params map[string]interface{} `json:"params,omitempty"`
	Seq    uint32                 `json:"seq"`
	TS     int64                  `json:"ts"`
}

// DeviceStatus 设备状态上报
type DeviceStatus struct {
	DeviceSN string                 `json:"device_sn"`
	Online   bool                   `json:"online"`
	States   map[string]interface{} `json:"states"` // door:locked, light:off, ac:25
	Version  string                 `json:"version"`
	RSSI     int                    `json:"rssi"`
	FreeHeap int                    `json:"free_heap"`
	Uptime   int64                  `json:"uptime"`
	TS       int64                  `json:"ts"`
}

// EncodeCommand 编码控制指令为 JSON
func EncodeCommand(cmd Command) ([]byte, error) {
	return json.Marshal(cmd)
}

// DecodeStatus 解码设备状态上报
func DecodeStatus(raw []byte) (*DeviceStatus, error) {
	var s DeviceStatus
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("decode status: %w", err)
	}
	return &s, nil
}

// AvailableCommands 支持的设备控制指令集
var AvailableCommands = map[string]string{
	"door_open":     "开门",
	"door_close":    "关门",
	"light_on":      "开灯",
	"light_off":     "关灯",
	"ac_on":         "空调开",
	"ac_off":        "空调关",
	"ac_set_temp":   "空调设温",
	"exhaust_on":    "排风开",
	"exhaust_off":   "排风关",
	"reboot":        "重启设备",
	"factory_reset": "恢复出厂",
}
