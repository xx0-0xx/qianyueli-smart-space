package handler

import (
	"encoding/json"
	"fmt"
	"strconv"

	cfgm "smart-space-management/backend/internal/model/config"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	ctx *Context
}

func NewConfigHandler(ctx *Context) *ConfigHandler {
	return &ConfigHandler{ctx: ctx}
}

// ── 价格规则 ──

// ListPriceRules 商家查看本店价格规则
func (h *ConfigHandler) ListPriceRules(c *gin.Context) {
	storeID := GetStoreID(c)
	var rules []cfgm.PriceRule
	h.ctx.DB.Where("store_id = ?", storeID).Order("created_at DESC").Find(&rules)
	response.OK(c, rules)
}

func (h *ConfigHandler) CreatePriceRule(c *gin.Context) {
	storeID := GetStoreID(c)
	var req cfgm.PriceRule
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	req.ID = utils.ShortID()
	req.StoreID = storeID
	if h.ctx.DB.Create(&req).Error != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	h.saveConfigHistory("price_rule", req.ID, storeID, "", toJSON(req), GetUserID(c))
	h.invalidateConfigCache("price_rule", storeID)
	response.Created(c, req)
}

// BulkReplacePriceRules 一次性替换门店所有价格规则
func (h *ConfigHandler) BulkReplacePriceRules(c *gin.Context) {
	storeID := GetStoreID(c)
	role, _ := c.Get("user_role")
	if storeID == 0 && (role == "super_admin" || role == "admin") {
		if sid := c.Query("store_id"); sid != "" {
			if id, err := strconv.ParseUint(sid, 10, 64); err == nil {
				storeID = id
			}
		}
	}
	var req struct {
		Rules []cfgm.PriceRule `json:"rules"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	// 删旧建新
	h.ctx.DB.Unscoped().Where("store_id = ?", storeID).Delete(&cfgm.PriceRule{})
	for i := range req.Rules {
		req.Rules[i].ID = utils.ShortID()
		req.Rules[i].StoreID = storeID
		h.ctx.DB.Create(&req.Rules[i])
	}
	response.OKMsg(c, fmt.Sprintf("已更新 %d 条定价", len(req.Rules)))
}

func (h *ConfigHandler) UpdatePriceRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	storeID := GetStoreID(c)

	var old cfgm.PriceRule
	if h.ctx.DB.Where("id = ? AND store_id = ?", id, storeID).First(&old).Error != nil {
		response.FailCode(c, errcode.ErrConfigNotFound)
		return
	}

	var req cfgm.PriceRule
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	oldJSON := toJSON(old)
	if err := h.ctx.DB.Model(&old).Updates(req).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}

	h.saveConfigHistory("price_rule", id, storeID, oldJSON, toJSON(req), GetUserID(c))
	h.invalidateConfigCache("price_rule", storeID)
	response.OKMsg(c, "updated")
}

// ── 预约规则 ──

func (h *ConfigHandler) CreateBookingRule(c *gin.Context) {
	storeID := GetStoreID(c)
	var req cfgm.BookingRule
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	req.ID = utils.ShortID()
	req.StoreID = storeID
	if h.ctx.DB.Create(&req).Error != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	h.saveConfigHistory("booking_rule", req.ID, storeID, "", toJSON(req), GetUserID(c))
	response.Created(c, req)
}

func (h *ConfigHandler) UpdateBookingRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	storeID := GetStoreID(c)

	var old cfgm.BookingRule
	if h.ctx.DB.Where("id = ? AND store_id = ?", id, storeID).First(&old).Error != nil {
		response.FailCode(c, errcode.ErrConfigNotFound)
		return
	}

	var req cfgm.BookingRule
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	oldJSON := toJSON(old)
	h.ctx.DB.Model(&old).Updates(req)

	h.saveConfigHistory("booking_rule", id, storeID, oldJSON, toJSON(req), GetUserID(c))
	response.OKMsg(c, "updated")
}

// ── UI 配置 ──

// GetImageConfigsPublic 小程序公开获取图片配置（无需登录）
func (h *ConfigHandler) GetImageConfigsPublic(c *gin.Context) {
	var configs []cfgm.UIConfig
	h.ctx.DB.Where("`key` LIKE 'img_%'").Find(&configs)
	result := make(map[string]string)
	for _, c := range configs {
		result[c.Key] = c.Value
	}
	response.OK(c, result)
}

func (h *ConfigHandler) GetUIConfig(c *gin.Context) {
	storeID, _ := strconv.ParseUint(c.Query("store_id"), 10, 64)
	if storeID == 0 {
		storeID = GetStoreID(c)
	}

	var configs []cfgm.UIConfig
	h.ctx.DB.Where("store_id = ?", storeID).Find(&configs)

	result := make(map[string]string)
	for _, c := range configs {
		result[c.Key] = c.Value
	}
	response.OK(c, result)
}

// ── helpers ──

func (h *ConfigHandler) saveConfigHistory(configType string, configID, storeID uint64, oldVal, newVal string, operatorID uint64) {
	history := cfgm.ConfigHistory{
		ConfigType: configType,
		ConfigID:   configID,
		StoreID:    storeID,
		OldValue:   oldVal,
		NewValue:   newVal,
		OperatorID: operatorID,
		Version:    1,
	}
	history.ID = utils.ShortID()
	h.ctx.DB.Create(&history)
}

func (h *ConfigHandler) invalidateConfigCache(configType string, storeID uint64) {
	// Redis 缓存失效 — 使配置中心的缓存失效，下次读取从 DB 加载
	key := "ssm:config:" + configType + ":" + strconv.FormatUint(storeID, 10)
	h.ctx.Redis.Del(h.ctx.DB.Statement.Context, key)
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
