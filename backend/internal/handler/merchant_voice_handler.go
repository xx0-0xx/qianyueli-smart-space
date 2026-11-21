package handler

import (
	"encoding/json"
	"errors"
	"log"

	"smart-space-management/backend/config"
	cfgm "smart-space-management/backend/internal/model/config"
	"smart-space-management/backend/internal/model/device"
	"smart-space-management/backend/internal/model/resource"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetMerchantVoiceConfig 商家读取本门店成品控制器的语音播报配置。
// GET /api/v1/merchant/voice-config
func (h *UserHandler) GetMerchantVoiceConfig(c *gin.Context) {
	storeID := GetStoreID(c)
	if storeID == 0 {
		response.FailCode(c, errcode.ErrForbidden)
		return
	}

	speakers, err := h.merchantSpeakerDevices(storeID)
	if err != nil {
		response.FailMsg(c, errcode.ErrInternal, "读取设备失败")
		return
	}

	global, err := loadVoiceConfig(h.ctx.DB)
	if err != nil {
		response.FailMsg(c, errcode.ErrInternal, "读取语音配置失败")
		return
	}
	bySN := make(map[string]config.DeviceVoiceConfig, len(global.Devices))
	for _, item := range global.Devices {
		bySN[item.DeviceSN] = item
	}

	devices := make([]config.DeviceVoiceConfig, 0, len(speakers))
	for _, speaker := range speakers {
		if item, ok := bySN[speaker.SN]; ok {
			item.Name = speaker.Name
			devices = append(devices, item)
			continue
		}
		devices = append(devices, config.DeviceVoiceConfig{
			DeviceSN:  speaker.SN,
			Name:      speaker.Name,
			Enabled:   false,
			Reminders: []config.VoiceReminder{},
		})
	}

	response.OK(c, config.VoiceConfig{Devices: devices})
}

// SaveMerchantVoiceConfig 商家保存本门店成品控制器的语音播报配置。
// 只更新本门店的 SN，其他门店的配置保留。
// PUT /api/v1/merchant/voice-config
func (h *UserHandler) SaveMerchantVoiceConfig(c *gin.Context) {
	var req config.VoiceConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	if err := config.ValidateVoiceConfig(&req); err != nil {
		response.FailMsg(c, errcode.ErrBadRequest, err.Error())
		return
	}

	storeID := GetStoreID(c)
	if storeID == 0 {
		response.FailCode(c, errcode.ErrForbidden)
		return
	}

	speakers, err := h.merchantSpeakerDevices(storeID)
	if err != nil {
		response.FailMsg(c, errcode.ErrInternal, "读取设备失败")
		return
	}
	allowed := make(map[string]string, len(speakers))
	for _, speaker := range speakers {
		allowed[speaker.SN] = speaker.Name
	}
	for i := range req.Devices {
		name, ok := allowed[req.Devices[i].DeviceSN]
		if !ok {
			response.FailMsg(c, errcode.ErrBadRequest, "设备不属于当前门店或不是云喇叭设备: "+req.Devices[i].DeviceSN)
			return
		}
		req.Devices[i].Name = name
	}

	global, err := loadVoiceConfig(h.ctx.DB)
	if err != nil {
		response.FailMsg(c, errcode.ErrInternal, "读取语音配置失败")
		return
	}

	// 先移除本门店的所有旧配置，再合并本次提交，实现“删除即停用/移除”。
	merged := make([]config.DeviceVoiceConfig, 0, len(global.Devices)+len(req.Devices))
	for _, item := range global.Devices {
		if _, ok := allowed[item.DeviceSN]; !ok {
			merged = append(merged, item)
		}
	}
	merged = append(merged, req.Devices...)

	if err := saveVoiceConfig(h.ctx.DB, config.VoiceConfig{Devices: merged}); err != nil {
		log.Printf("[VOICE] 商家语音配置保存失败 store=%d: %v", storeID, err)
		response.FailMsg(c, errcode.ErrInternal, "保存语音配置失败")
		return
	}
	response.OKMsg(c, "已保存")
}

// speakerDevice 语音播报页展示的云喇叭/音响设备。
type speakerDevice struct {
	SN   string
	Name string
}

// merchantSpeakerDevices 返回门店下所有云喇叭/音响设备，名称优先使用产品模板名。
// ListMerchantVoiceDevices 返回当前门店的语音设备及所在房间，供房间列表只读展示。
// GET /api/v1/merchant/voice-devices
func (h *UserHandler) ListMerchantVoiceDevices(c *gin.Context) {
	storeID := GetStoreID(c)
	if storeID == 0 {
		response.FailCode(c, errcode.ErrForbidden)
		return
	}

	var speakers []device.Device
	if err := h.ctx.DB.
		Where("store_id = ? AND protocol = ? AND type = ?", storeID, device.ProtocolASCII, device.TypeSpeaker).
		Order("device_sn ASC").
		Find(&speakers).Error; err != nil {
		response.FailMsg(c, errcode.ErrInternal, "获取语音设备失败")
		return
	}

	var rooms []resource.Resource
	if err := h.ctx.DB.Where("store_id = ?", storeID).Find(&rooms).Error; err != nil {
		response.FailMsg(c, errcode.ErrInternal, "获取房间失败")
		return
	}
	roomByDevice := make(map[uint64]resource.Resource, len(rooms))
	for _, room := range rooms {
		ids, err := resource.ParseDeviceIDs(room.DeviceIDs)
		if err != nil {
			continue
		}
		for _, id := range ids {
			if _, exists := roomByDevice[id]; !exists {
				roomByDevice[id] = room
			}
		}
	}

	type voiceDeviceView struct {
		ID       uint64 `json:"id"`
		DeviceSN string `json:"device_sn"`
		Name     string `json:"name"`
		Type     string `json:"type"`
		Status   int    `json:"status"`
		RoomID   uint64 `json:"room_id"`
		RoomName string `json:"room_name"`
	}
	result := make([]voiceDeviceView, 0, len(speakers))
	for _, speaker := range speakers {
		room := roomByDevice[speaker.ID]
		result = append(result, voiceDeviceView{
			ID: speaker.ID, DeviceSN: speaker.DeviceSN, Name: speaker.Name,
			Type: string(speaker.Type), Status: int(speaker.Status),
			RoomID: room.ID, RoomName: room.Name,
		})
	}
	response.OK(c, result)
}

func (h *UserHandler) merchantSpeakerDevices(storeID uint64) ([]speakerDevice, error) {
	var rows []device.Device
	if err := h.ctx.DB.
		Where("store_id = ? AND protocol = ? AND type = ?", storeID, device.ProtocolASCII, device.TypeSpeaker).
		Order("device_sn ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []speakerDevice{}, nil
	}

	type templateConfig struct {
		TemplateID uint64 `json:"template_id"`
	}
	templateIDs := make(map[uint64]bool)
	for _, row := range rows {
		var cfg templateConfig
		if row.ConfigJSON != "" && json.Unmarshal([]byte(row.ConfigJSON), &cfg) == nil && cfg.TemplateID > 0 {
			templateIDs[cfg.TemplateID] = true
		}
	}
	templateNames := make(map[uint64]string)
	if len(templateIDs) > 0 {
		ids := make([]uint64, 0, len(templateIDs))
		for id := range templateIDs {
			ids = append(ids, id)
		}
		var templates []device.ProductTemplate
		if err := h.ctx.DB.Where("id IN ?", ids).Find(&templates).Error; err != nil {
			return nil, err
		}
		for _, tpl := range templates {
			templateNames[tpl.ID] = tpl.Name
		}
	}

	seen := make(map[string]bool, len(rows))
	result := make([]speakerDevice, 0, len(rows))
	for _, row := range rows {
		if row.DeviceSN == "" || seen[row.DeviceSN] {
			continue
		}
		seen[row.DeviceSN] = true
		name := row.Name
		var cfg templateConfig
		if row.ConfigJSON != "" && json.Unmarshal([]byte(row.ConfigJSON), &cfg) == nil {
			if templateName := templateNames[cfg.TemplateID]; templateName != "" {
				name = templateName
			}
		}
		if name == "" {
			name = row.DeviceSN
		}
		result = append(result, speakerDevice{SN: row.DeviceSN, Name: name})
	}
	return result, nil
}

func loadVoiceConfig(db *gorm.DB) (config.VoiceConfig, error) {
	result := config.VoiceConfig{Devices: []config.DeviceVoiceConfig{}}
	var uc cfgm.UIConfig
	err := db.Where("store_id = 0 AND `key` = ?", "voice_config").First(&uc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	if uc.Value == "" {
		return result, nil
	}
	if err := json.Unmarshal([]byte(uc.Value), &result); err != nil || result.Devices == nil {
		return config.VoiceConfig{Devices: []config.DeviceVoiceConfig{}}, nil
	}
	return result, nil
}

func removeVoiceConfigSN(db *gorm.DB, sn string) error {
	global, err := loadVoiceConfig(db)
	if err != nil {
		return err
	}
	filtered := make([]config.DeviceVoiceConfig, 0, len(global.Devices))
	for _, item := range global.Devices {
		if item.DeviceSN != sn {
			filtered = append(filtered, item)
		}
	}
	return saveVoiceConfig(db, config.VoiceConfig{Devices: filtered})
}

func saveVoiceConfig(db *gorm.DB, vc config.VoiceConfig) error {
	b, err := json.Marshal(vc)
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var existing cfgm.UIConfig
		findErr := tx.Where("store_id = 0 AND `key` = ?", "voice_config").First(&existing).Error
		if findErr == nil {
			existing.Value = string(b)
			return tx.Save(&existing).Error
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		existing = cfgm.UIConfig{
			StoreID: 0,
			Key:     "voice_config",
			Value:   string(b),
			Remark:  "设备语音播报配置",
		}
		existing.ID = utils.ShortID()
		return tx.Create(&existing).Error
	})
}
