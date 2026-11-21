package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"smart-space-management/backend/config"
	devicePkg "smart-space-management/backend/internal/device"
	"smart-space-management/backend/internal/handler"
	"smart-space-management/backend/internal/model/audit"
	balanceModel "smart-space-management/backend/internal/model/balance"
	cfgm "smart-space-management/backend/internal/model/config"
	"smart-space-management/backend/internal/model/device"
	"smart-space-management/backend/internal/model/order"
	"smart-space-management/backend/internal/model/payment"
	"smart-space-management/backend/internal/model/resource"
	"smart-space-management/backend/internal/model/store"
	"smart-space-management/backend/internal/model/user"
	"smart-space-management/backend/internal/service"
	"smart-space-management/backend/pkg/alert"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/imageutil"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"
	"smart-space-management/backend/pkg/wechat"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminHandler 超级管理端处理器
type AdminHandler struct {
	ctx *handler.Context
}

func NewAdminHandler(ctx *handler.Context) *AdminHandler {
	return &AdminHandler{ctx: ctx}
}

// ────────── 门店管理 ──────────

type createStoreReq struct {
	Name          string    `json:"name" binding:"required"`
	Address       string    `json:"address" binding:"required"`
	Phone         string    `json:"phone"`
	Latitude      flexFloat `json:"latitude"`
	Longitude     flexFloat `json:"longitude"`
	OpenTime      string    `json:"open_time"`
	CloseTime     string    `json:"close_time"`
	Description   string    `json:"description"`
	Cover         string    `json:"cover"`
	Tags          string    `json:"tags"`
	NotifyWebhook string    `json:"notify_webhook"` // 门店群机器人 webhook
}

func (h *AdminHandler) CreateStore(c *gin.Context) {
	var req createStoreReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[ADMIN] CreateStore 参数绑定失败: %v", err)
		response.FailMsg(c, errcode.ErrBadRequest, "参数错误："+err.Error())
		return
	}
	s := store.Store{
		Name:          req.Name,
		Address:       req.Address,
		Phone:         req.Phone,
		Latitude:      float64(req.Latitude),
		Longitude:     float64(req.Longitude),
		OpenTime:      req.OpenTime,
		CloseTime:     req.CloseTime,
		Description:   req.Description,
		Cover:         req.Cover,
		Tags:          req.Tags,
		NotifyWebhook: req.NotifyWebhook,
	}
	s.ID = utils.ShortID()
	s.Status = store.StatusOpen
	s.InviteCode = h.genUniqueInviteCode()

	// 每个门店默认拥有一个独立余额组（组名 = 门店名）
	g := balanceModel.BalanceGroup{Name: s.Name}
	g.ID = utils.ShortID()
	if err := h.ctx.DB.Create(&g).Error; err != nil {
		response.FailMsg(c, errcode.ErrInternal, "创建余额组失败")
		return
	}
	s.BalanceGroupID = g.ID

	if err := h.ctx.DB.Create(&s).Error; err != nil {
		response.FailMsg(c, errcode.ErrInternal, "创建门店失败")
		return
	}
	response.Created(c, s)
}

func (h *AdminHandler) UpdateStore(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var s store.Store
	if h.ctx.DB.First(&s, id).Error != nil {
		response.FailCode(c, errcode.ErrStoreNotFound)
		return
	}
	var req store.Store
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	// balance_group_id 只能通过 SetStoreBalanceGroup 修改（需合并/拆分校验）
	h.ctx.DB.Model(&s).Omit("balance_group_id").Updates(req)
	response.OKMsg(c, "updated")
}

type storeDeleteAssociation struct {
	table string
	label string
}

func storeDeleteAssociations() []storeDeleteAssociation {
	return []storeDeleteAssociation{
		{table: "orders", label: "订单"},
		{table: "resources", label: "房间"},
		{table: "resource_types", label: "房型"},
		{table: "devices", label: "设备"},
		{table: "merchant_stores", label: "商家绑定"},
		{table: "payments", label: "支付记录"},
		{table: "refund_requests", label: "退款记录"},
		{table: "recharges", label: "充值记录"},
		{table: "coupons", label: "优惠券"},
		{table: "price_rules", label: "计费规则"},
		{table: "booking_rules", label: "预约规则"},
		{table: "ui_configs", label: "门店配置"},
		{table: "config_histories", label: "配置历史"},
		{table: "audit_logs", label: "审计记录"},
		{table: "product_templates", label: "门店产品模板"},
	}
}

// storeDeleteBlockedByBalance 该店是否会"孤立"一个有余额的余额池
func storeDeleteBlockedByBalance(isOnlyStoreInGroup, groupHasPositiveBalance bool) bool {
	return isOnlyStoreInGroup && groupHasPositiveBalance
}

// flexID 允许 JSON 中数字或字符串形式的 ID（前端 id 序列化为字符串 "id,string"）
type flexID uint64

func (f *flexID) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	*f = flexID(n)
	return nil
}

// flexFloat 允许 JSON 中数字或字符串形式的浮点（前端可能传 "" 或字符串坐标）
type flexFloat float64

func (f *flexFloat) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*f = flexFloat(v)
	return nil
}

func (h *AdminHandler) DeleteStore(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	tx := h.ctx.DB.Begin()
	if tx.Error != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	defer tx.Rollback()

	var current store.Store
	if err := tx.First(&current, id).Error; err != nil {
		response.FailCode(c, errcode.ErrStoreNotFound)
		return
	}

	for _, association := range storeDeleteAssociations() {
		query := tx.Table(association.table).Where("store_id = ?", id)
		if association.table == "product_templates" {
			query = query.Where("store_id <> 0")
		}
		var count int64
		if err := query.Count(&count).Error; err != nil {
			response.FailMsg(c, errcode.ErrInternal, "检查门店关联数据失败")
			return
		}
		if count > 0 {
			response.FailMsg(c, errcode.ErrConflict, "门店仍关联"+association.label+"，请先处理关联数据")
			return
		}
	}

	// 余额组孤立校验：若该店是所在组唯一门店且组内仍有用户余额 → 拒绝
	if current.BalanceGroupID != 0 {
		var groupStoreCount int64
		tx.Model(&store.Store{}).Where("balance_group_id = ?", current.BalanceGroupID).Count(&groupStoreCount)
		var posCount int64
		tx.Model(&balanceModel.Account{}).Where("group_id = ? AND balance <> 0", current.BalanceGroupID).Count(&posCount)
		if storeDeleteBlockedByBalance(groupStoreCount <= 1, posCount > 0) {
			response.FailMsg(c, errcode.ErrConflict, "该门店是余额组的最后门店且组内仍有用户余额，请先处理余额")
			return
		}
	}

	result := tx.Delete(&current)
	if result.Error != nil {
		response.FailMsg(c, errcode.ErrInternal, "删除门店失败")
		return
	}
	if result.RowsAffected == 0 {
		response.FailCode(c, errcode.ErrStoreNotFound)
		return
	}
	// 组内已无活跃门店则清理该余额组
	if current.BalanceGroupID != 0 {
		var cnt int64
		tx.Model(&store.Store{}).Where("balance_group_id = ?", current.BalanceGroupID).Count(&cnt)
		if cnt == 0 {
			tx.Delete(&balanceModel.BalanceGroup{}, current.BalanceGroupID)
		}
	}
	if err := tx.Commit().Error; err != nil {
		response.FailMsg(c, errcode.ErrInternal, "删除门店失败")
		return
	}
	response.OKMsg(c, "门店已删除")
}

// RegenerateInviteCode 重新生成门店邀请码
func (h *AdminHandler) RegenerateInviteCode(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var s store.Store
	if h.ctx.DB.First(&s, id).Error != nil {
		response.FailCode(c, errcode.ErrStoreNotFound)
		return
	}
	newCode := h.genUniqueInviteCode()
	if err := h.ctx.DB.Model(&s).Update("invite_code", newCode).Error; err != nil {
		response.FailMsg(c, errcode.ErrInternal, "邀请码更新失败: "+err.Error())
		return
	}
	response.OK(c, gin.H{"store_id": s.ID, "invite_code": newCode})
}

func (h *AdminHandler) ListAllStores(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	var stores []store.Store
	var total int64
	h.ctx.DB.Model(&store.Store{}).Count(&total)
	h.ctx.DB.Offset((page - 1) * size).Limit(size).Order("created_at DESC").Find(&stores)
	response.PageOK(c, response.PageData{
		List: stores, Total: total, Page: page, PageSize: size,
		TotalPages: calcTotalPages(total, size),
	})
}

// ────────── 商家账号管理 ──────────

func (h *AdminHandler) CreateMerchant(c *gin.Context) {
	var req struct {
		Nickname string `json:"nickname"`
		Phone    string `json:"phone"`
		StoreID  uint64 `json:"store_id,string" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	u := user.User{
		Nickname: req.Nickname,
		Phone:    req.Phone,
		Role:     user.RoleMerchant,
		StoreID:  req.StoreID,
		Status:   user.StatusNormal,
	}
	u.ID = utils.ShortID()
	h.ctx.DB.Create(&u)
	response.Created(c, u)
}

func (h *AdminHandler) DeleteMerchant(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	result := h.ctx.DB.Where("id = ? AND role IN ?", id, []string{"merchant", "admin"}).Delete(&user.User{})
	if result.RowsAffected == 0 {
		response.FailMsg(c, errcode.ErrNotFound, "商家不存在或不可删除")
		return
	}
	response.OKMsg(c, "已删除")
}

func (h *AdminHandler) UpdateMerchant(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Nickname string `json:"nickname"`
		Phone    string `json:"phone"`
		StoreID  uint64 `json:"store_id,string"`
		Status   int    `json:"status"`
	}
	c.ShouldBindJSON(&req)
	updates := map[string]interface{}{}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.StoreID > 0 {
		updates["store_id"] = req.StoreID
	}
	if req.Status > 0 {
		updates["status"] = req.Status
	}
	h.ctx.DB.Model(&user.User{}).Where("id = ?", id).Updates(updates)
	response.OKMsg(c, "updated")
}

func (h *AdminHandler) ListMerchants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	var merchants []user.User
	var total int64
	h.ctx.DB.Model(&user.User{}).Where("role = ?", user.RoleMerchant).Count(&total)
	h.ctx.DB.Where("role = ?", user.RoleMerchant).
		Offset((page - 1) * size).Limit(size).Order("created_at DESC").Find(&merchants)
	response.PageOK(c, response.PageData{
		List: merchants, Total: total, Page: page, PageSize: size,
		TotalPages: calcTotalPages(total, size),
	})
}

// ────────── 资源模板 ──────────

func (h *AdminHandler) CreateResourceTemplate(c *gin.Context) {
	var req resource.ResourceType
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	req.ID = utils.ShortID()
	h.ctx.DB.Create(&req)
	response.Created(c, req)
}

func (h *AdminHandler) UpdateResourceTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req resource.ResourceType
	c.ShouldBindJSON(&req)
	h.ctx.DB.Model(&resource.ResourceType{}).Where("id = ?", id).Updates(req)
	response.OKMsg(c, "updated")
}

func (h *AdminHandler) ListResourceTemplates(c *gin.Context) {
	var types []resource.ResourceType
	h.ctx.DB.Find(&types)
	response.OK(c, types)
}

// ────────── 订单审核 ──────────

func (h *AdminHandler) ListAllOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	var orders []order.Order
	var total int64
	h.ctx.DB.Model(&order.Order{}).Count(&total)
	h.ctx.DB.Offset((page - 1) * size).Limit(size).Order("created_at DESC").Find(&orders)
	response.PageOK(c, response.PageData{
		List: orders, Total: total, Page: page, PageSize: size,
		TotalPages: calcTotalPages(total, size),
	})
}

// ApproveRefund 审批退款
func (h *AdminHandler) ApproveRefund(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var o order.Order
	if h.ctx.DB.First(&o, id).Error != nil {
		response.FailCode(c, errcode.ErrOrderNotFound)
		return
	}
	if o.Status != order.StatusRefunding {
		response.FailMsg(c, errcode.ErrOrderStatusInvalid, "仅退款中订单可审批")
		return
	}

	var req struct {
		Approved bool   `json:"approved"`
		Reason   string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	if req.Approved {
		// 退款成功
		o.Status = order.StatusRefunded
		o.RefundAmount = o.TotalAmount
		h.ctx.DB.Save(&o)

		// 更新支付记录
		h.ctx.DB.Model(&payment.Payment{}).Where("order_id = ?", o.ID).
			Update("status", payment.StatusRefunded)

		// 释放资源
		h.ctx.DB.Model(&resource.Resource{}).Where("id = ?", o.ResourceID).
			Update("status", resource.StatusFree)
	} else {
		// 驳回退款
		o.Status = order.StatusCompleted
		o.CancelReason = req.Reason
		h.ctx.DB.Save(&o)
	}

	response.OKMsg(c, "processed")
}

// ────────── 数据统计 ──────────

func (h *AdminHandler) GetOverviewStats(c *gin.Context) {
	var totalStores, totalUsers, totalOrders int64
	var totalRevenue float64

	h.ctx.DB.Model(&store.Store{}).Count(&totalStores)
	h.ctx.DB.Model(&user.User{}).Count(&totalUsers)
	h.ctx.DB.Model(&order.Order{}).Count(&totalOrders)
	h.ctx.DB.Model(&order.Order{}).
		Where("status IN ?", []int{1, 2, 3}).
		Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenue)

	response.OK(c, gin.H{
		"total_stores":  totalStores,
		"total_users":   totalUsers,
		"total_orders":  totalOrders,
		"total_revenue": totalRevenue / 100, // 转元
	})
}

func (h *AdminHandler) GetStoreStats(c *gin.Context) {
	storeID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	todayStart := time.Now().Truncate(24 * time.Hour).UnixMilli()

	var orderCount int64
	var revenue float64
	h.ctx.DB.Model(&order.Order{}).
		Where("store_id = ? AND status IN ? AND created_at >= ?",
			storeID, []int{1, 2, 3}, time.UnixMilli(todayStart)).
		Count(&orderCount)
	h.ctx.DB.Model(&order.Order{}).
		Where("store_id = ? AND status IN ? AND created_at >= ?",
			storeID, []int{1, 2, 3}, time.UnixMilli(todayStart)).
		Select("COALESCE(SUM(total_amount), 0)").Scan(&revenue)

	response.OK(c, gin.H{
		"today_orders":  orderCount,
		"today_revenue": revenue / 100,
	})
}

// ────────── 系统配置 ──────────

func (h *AdminHandler) GetSystemSettings(c *gin.Context) {
	// 从配置文件读取，可扩展为从 DB 读取
	response.OK(c, gin.H{
		"server": gin.H{"port": h.ctx.Cfg.Server.Port},
		"mqtt":   gin.H{"broker": h.ctx.Cfg.MQTT.Broker},
	})
}

// ────────── 图片配置 ──────────

type imageConfigReq struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// GetImageConfigs 获取所有图片配置
func (h *AdminHandler) GetImageConfigs(c *gin.Context) {
	var configs []cfgm.UIConfig
	h.ctx.DB.Where("`key` LIKE 'img_%'").Find(&configs)
	result := make(map[string]string)
	for _, c := range configs {
		result[c.Key] = c.Value
	}
	response.OK(c, result)
}

// SaveImageConfigs 批量保存图片配置
func (h *AdminHandler) SaveImageConfigs(c *gin.Context) {
	var req []imageConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	for _, row := range req {
		if row.Key == "" || row.Value == "" {
			continue
		}
		var existing cfgm.UIConfig
		if h.ctx.DB.Where("`key` = ?", row.Key).First(&existing).Error == nil {
			existing.Value = row.Value
			h.ctx.DB.Save(&existing)
		} else {
			h.ctx.DB.Create(&cfgm.UIConfig{
				Key:   row.Key,
				Value: row.Value,
			})
		}
	}
	response.OKMsg(c, "保存成功")
}

func (h *AdminHandler) UpdateSystemSettings(c *gin.Context) {
	response.OKMsg(c, "settings saved (config file)")
}

// TriggerDailyReport 手动触发「今日营收」推送（测试/补推）
// POST /api/v1/admin/reports/daily
func (h *AdminHandler) TriggerDailyReport(c *gin.Context) {
	notifier := alert.New(h.ctx.Cfg.Alert)
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).UnixMilli()
	dayEnd := dayStart + 86400000
	dateStr := now.Format("2006-01-02")

	var stores []store.Store
	h.ctx.DB.Where("status = ?", store.StatusOpen).Find(&stores)

	var summary []string
	totalRevenue, totalCnt := 0, 0
	for _, st := range stores {
		var orders []order.Order
		h.ctx.DB.Where("store_id = ? AND start_time >= ? AND start_time < ?", st.ID, dayStart, dayEnd).Find(&orders)
		revenue, cnt := 0, 0
		for _, o := range orders {
			if o.Status == order.StatusPaid || o.Status == order.StatusInProgress || o.Status == order.StatusCompleted {
				revenue += o.TotalAmount
				cnt++
			}
		}
		if st.NotifyWebhook != "" {
			msg := fmt.Sprintf("📊【今日营收】%s\n日期：%s\n订单：%d 单\n营业额：¥%.2f",
				st.Name, dateStr, cnt, float64(revenue)/100)
			if err := notifier.SendTo(st.NotifyWebhook, msg); err != nil {
				log.Printf("[REPORT] store send failed: %v", err)
			}
		} else {
			summary = append(summary, fmt.Sprintf("· %s：%d单 ¥%.2f", st.Name, cnt, float64(revenue)/100))
			totalRevenue += revenue
			totalCnt += cnt
		}
	}
	if len(summary) > 0 {
		msg := fmt.Sprintf("📊【今日营收汇总】%s\n%s\n──────────\n总计：%d 单  ¥%.2f",
			dateStr, strings.Join(summary, "\n"), totalCnt, float64(totalRevenue)/100)
		if err := notifier.Send(msg); err != nil {
			log.Printf("[REPORT] summary send failed: %v", err)
		}
	}
	response.OKMsg(c, "已推送今日营收")
}

// GetVoiceConfig 读语音播报配置（DB 优先，回落配置文件）
// GET /api/v1/admin/voice-config
func (h *AdminHandler) GetVoiceConfig(c *gin.Context) {
	var result config.VoiceConfig
	var uc cfgm.UIConfig
	err := h.ctx.DB.Where("store_id = 0 AND `key` = ?", "voice_config").First(&uc).Error
	if err == nil && uc.Value != "" {
		if json.Unmarshal([]byte(uc.Value), &result) != nil || result.Devices == nil {
			result = config.VoiceConfig{Devices: []config.DeviceVoiceConfig{}}
		}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("[VOICE] 配置读取失败: %v", err)
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if result.Devices == nil {
		result.Devices = []config.DeviceVoiceConfig{}
	}
	response.OK(c, result)
}

// SaveVoiceConfig 保存设备 SN 专属语音配置
// PUT /api/v1/admin/voice-config
func (h *AdminHandler) SaveVoiceConfig(c *gin.Context) {
	var vc config.VoiceConfig
	if err := c.ShouldBindJSON(&vc); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	if err := config.ValidateVoiceConfig(&vc); err != nil {
		response.FailMsg(c, errcode.ErrBadRequest, err.Error())
		return
	}

	if len(vc.Devices) > 0 {
		sns := make([]string, 0, len(vc.Devices))
		for _, d := range vc.Devices {
			sns = append(sns, d.DeviceSN)
		}
		var rows []device.Device
		if err := h.ctx.DB.Where("device_sn IN ? AND protocol = ?", sns, device.ProtocolASCII).Find(&rows).Error; err != nil {
			response.FailMsg(c, errcode.ErrInternal, "检查设备失败")
			return
		}
		for _, d := range vc.Devices {
			if !voiceSNExistsAsASCII(rows, d.DeviceSN) {
				response.FailMsg(c, errcode.ErrBadRequest, "设备 SN 不存在或不是 ASCII 控制器: "+d.DeviceSN)
				return
			}
		}
	}

	b, err := json.Marshal(vc)
	if err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	err = h.ctx.DB.Transaction(func(tx *gorm.DB) error {
		var existing cfgm.UIConfig
		findErr := tx.Where("store_id = 0 AND `key` = ?", "voice_config").First(&existing).Error
		if findErr == nil {
			existing.Value = string(b)
			return tx.Save(&existing).Error
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}
		existing = cfgm.UIConfig{StoreID: 0, Key: "voice_config", Value: string(b), Remark: "设备语音播报配置"}
		existing.ID = utils.ShortID()
		return tx.Create(&existing).Error
	})
	if err != nil {
		log.Printf("[VOICE] 配置保存失败: %v", err)
		response.FailMsg(c, errcode.ErrInternal, "保存语音配置失败")
		return
	}
	response.OKMsg(c, "已保存")
}

func voiceSNExistsAsASCII(rows []device.Device, sn string) bool {
	for _, row := range rows {
		if row.DeviceSN == sn && row.Protocol == device.ProtocolASCII {
			return true
		}
	}
	return false
}

// ────────── 审计日志 ──────────

func (h *AdminHandler) ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	var logs []audit.AuditLog
	var total int64
	query := h.ctx.DB.Model(&audit.AuditLog{})
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	query.Count(&total)
	query.Offset((page - 1) * size).Limit(size).Order("created_at DESC").Find(&logs)
	response.PageOK(c, response.PageData{
		List: logs, Total: total, Page: page, PageSize: size,
		TotalPages: calcTotalPages(total, size),
	})
}

// ────────── 设备管理 ──────────

func (h *AdminHandler) ListAllDevices(c *gin.Context) {
	var devices []device.Device
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	var total int64
	h.ctx.DB.Model(&device.Device{}).Count(&total)
	h.ctx.DB.Offset((page - 1) * size).Limit(size).Order("created_at DESC").Find(&devices)
	response.PageOK(c, response.PageData{
		List: devices, Total: total, Page: page, PageSize: size,
		TotalPages: calcTotalPages(total, size),
	})
}

// ProbeDevice 检测设备路数（发 qa 指令 → 数返回里的 n/f 个数）
// POST /api/v1/admin/devices/probe
func (h *AdminHandler) ProbeDevice(c *gin.Context) {
	var req struct {
		DeviceSN string `json:"device_sn" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	cli := devicePkg.GetGlobalClient()
	if cli == nil {
		response.FailMsg(c, errcode.ErrInternal, "MQTT 未就绪")
		return
	}
	key := "ssm:device:status:" + req.DeviceSN
	ctx := c.Request.Context()

	// 查全部状态（qa 精简版；qc 详细版，两者设备均支持）
	var raw string
	for _, q := range []string{"qa", "qc"} {
		h.ctx.Redis.Del(ctx, key)
		if err := cli.PublishRaw(req.DeviceSN, q); err != nil {
			response.FailMsg(c, errcode.ErrInternal, "指令发送失败: "+err.Error())
			return
		}
		// 等设备回复（最多 2 秒）
		for i := 0; i < 4; i++ {
			time.Sleep(500 * time.Millisecond)
			if v, err := h.ctx.Redis.Get(ctx, key).Result(); err == nil && v != "" {
				raw = v
				break
			}
		}
		if raw != "" {
			break
		}
	}

	// 数 n/f 个数 = 路数（如 ztf1f2f3f4 或 s!-60,n1,n2,n3,n4 → 4）
	re := regexp.MustCompile(`[nf]\d+`)
	channels := len(re.FindAllString(raw, -1))

	response.OK(c, gin.H{
		"device_sn": req.DeviceSN,
		"channels":  channels,
		"raw":       raw,
	})
}

func (h *AdminHandler) CreateDeviceAdmin(c *gin.Context) {
	var d device.Device
	if err := c.ShouldBindJSON(&d); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	d.ID = utils.ShortID()
	if d.StoreID == 0 {
		d.StoreID = 100
	}
	d.UserControlEnabled = d.Type == device.TypeDoorLock
	if err := h.ctx.DB.Create(&d).Error; err != nil {
		response.FailMsg(c, errcode.ErrConflict, "SN重复或创建失败")
		return
	}
	response.Created(c, d)
}

func (h *AdminHandler) UpdateDeviceAdmin(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var d device.Device
	if h.ctx.DB.First(&d, id).Error != nil {
		response.FailCode(c, errcode.ErrNotFound)
		return
	}
	var req device.Device
	c.ShouldBindJSON(&req)
	h.ctx.DB.Model(&d).Updates(map[string]interface{}{
		"name": req.Name, "device_sn": req.DeviceSN, "type": req.Type, "store_id": req.StoreID,
	})
	response.OKMsg(c, "已更新")
}

func (h *AdminHandler) DeleteDeviceAdmin(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	h.ctx.DB.Unscoped().Delete(&device.Device{}, id)
	response.OKMsg(c, "已删除")
}

// ── helpers ──

// genUniqueInviteCode 生成一个数据库中尚不存在的邀请码（最多重试 10 次）
func (h *AdminHandler) genUniqueInviteCode() string {
	for i := 0; i < 10; i++ {
		code := utils.GenInviteCode()
		var count int64
		h.ctx.DB.Model(&store.Store{}).Where("invite_code = ?", code).Count(&count)
		if count == 0 {
			return code
		}
	}
	// 兜底：用 snowflake 后缀保证唯一
	return utils.GenInviteCode() + "-" + strconv.FormatUint(utils.ShortID(), 36)
}

func calcTotalPages(total int64, size int) int {
	if size <= 0 {
		return 0
	}
	tp := int(total) / size
	if int(total)%size > 0 {
		tp++
	}
	return tp
}

// ── 音频管理 ──

// UploadAudio 上传 MP3
func (h *AdminHandler) UploadAudio(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	ext := filepath.Ext(file.Filename)
	allowed := map[string]bool{".mp3": true, ".png": true, ".jpg": true, ".jpeg": true, ".gif": true}
	if !allowed[strings.ToLower(ext)] {
		response.FailMsg(c, errcode.ErrBadRequest, "仅支持 MP3/图片")
		return
	}
	src, err := file.Open()
	if err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	defer src.Close()

	dst, err := os.Create(fmt.Sprintf("./audio/%s", file.Filename))
	if err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	io.Copy(dst, src)
	dst.Close()

	filePath := fmt.Sprintf("./audio/%s", file.Filename)
	lowExt := strings.ToLower(ext)
	finalName := file.Filename
	// 图片自动压缩（宽度限 1200px；PNG 无透明转 JPEG）
	if lowExt == ".png" || lowExt == ".jpg" || lowExt == ".jpeg" {
		if n, err := imageutil.CompressImage(filePath, 1200, 82); err != nil {
			log.Printf("[UPLOAD] 图片压缩失败 %s: %v", filePath, err)
		} else {
			finalName = n
		}
	}

	response.OK(c, gin.H{"name": finalName, "filename": finalName})
}

// ListAudio 音频文件列表
func (h *AdminHandler) ListAudio(c *gin.Context) {
	entries, _ := os.ReadDir("./audio")
	type AudioFile struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	}
	var files []AudioFile
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".mp3" {
			info, _ := e.Info()
			files = append(files, AudioFile{Name: e.Name(), Size: info.Size()})
		}
	}
	response.OK(c, files)
}

// DeleteAudio 删除音频
func (h *AdminHandler) DeleteAudio(c *gin.Context) {
	file := c.Param("file")
	path := fmt.Sprintf("./audio/%s", filepath.Base(file))
	if err := os.Remove(path); err != nil {
		response.FailMsg(c, errcode.ErrBadRequest, "文件不存在")
		return
	}
	response.OKMsg(c, "已删除")
}

// StoreQRCode 生成门店小程序码
// GET /api/v1/admin/stores/:id/qrcode
func (h *AdminHandler) StoreQRCode(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var s store.Store
	if h.ctx.DB.First(&s, id).Error != nil {
		response.FailCode(c, errcode.ErrStoreNotFound)
		return
	}

	wx := wechat.NewMiniProgram(h.ctx.Cfg.Wechat.Miniprogram.AppID, h.ctx.Cfg.Wechat.Miniprogram.AppSecret).WithRedis(h.ctx.Redis)
	if wx.AppID == "" || wx.AppSecret == "" {
		response.FailMsg(c, errcode.ErrBadRequest, "未配置小程序 AppID/AppSecret")
		return
	}

	scene := fmt.Sprintf("store=%d", id)
	img, err := wx.GetMiniProgramCode(scene, "pages/index/index")
	if err != nil {
		log.Printf("[QRCODE] get mini program code failed store=%d", id)
		response.FailMsg(c, errcode.ErrInternal, "生成小程序码失败，请稍后重试")
		return
	}
	c.Data(http.StatusOK, "image/png", img)
}

// ────────── 产品模板管理 ──────────

type productTemplateReq struct {
	Name     string                  `json:"name" binding:"required"`
	Devices  []device.TemplateDevice `json:"devices"`
	Status   int                     `json:"status"`
	Protocol string                  `json:"protocol"`
	Type     string                  `json:"type"`
}

// ListProductTemplates 产品模板列表
// GET /api/v1/admin/product-templates
func (h *AdminHandler) ListProductTemplates(c *gin.Context) {
	var list []device.ProductTemplate
	h.ctx.DB.Where("store_id = 0").Order("sort ASC, id ASC").Find(&list)
	// 附加解析后的 devices
	type row struct {
		device.ProductTemplate
		Devices []device.TemplateDevice `json:"devices"`
	}
	var rows []row
	for _, t := range list {
		rows = append(rows, row{ProductTemplate: t, Devices: t.Devices()})
	}
	response.OK(c, rows)
}

// CreateProductTemplate 新增产品模板
// POST /api/v1/admin/product-templates
func (h *AdminHandler) CreateProductTemplate(c *gin.Context) {
	var req productTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	t := device.ProductTemplate{Name: req.Name, StoreID: 0, Status: 1, Type: "other"}
	if req.Status != 0 {
		t.Status = req.Status
	}
	if req.Type != "" {
		t.Type = req.Type
	}
	if req.Protocol != "" {
		t.Protocol = req.Protocol
	} else {
		t.Protocol = device.ProtocolJSON
	}
	if err := t.SetDevices(req.Devices); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	var maxSort int
	h.ctx.DB.Model(&device.ProductTemplate{}).Where("store_id = 0").
		Select("COALESCE(MAX(sort), 0)").Scan(&maxSort)
	t.Sort = maxSort + 10
	t.ID = utils.ShortID()
	h.ctx.DB.Create(&t)
	response.Created(c, t)
}

// UpdateProductTemplate 更新产品模板
// PUT /api/v1/admin/product-templates/:id
func (h *AdminHandler) UpdateProductTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var t device.ProductTemplate
	if h.ctx.DB.First(&t, id).Error != nil {
		response.FailCode(c, errcode.ErrNotFound)
		return
	}
	var req productTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	updates := map[string]interface{}{"name": req.Name, "status": req.Status}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.Protocol != "" {
		updates["protocol"] = req.Protocol
	}
	if req.Devices != nil {
		b, _ := json.Marshal(req.Devices)
		updates["devices_json"] = string(b)
	}
	h.ctx.DB.Model(&t).Updates(updates)
	response.OKMsg(c, "已更新")
}

// DeleteProductTemplate 删除产品模板
// DELETE /api/v1/admin/product-templates/:id
func (h *AdminHandler) DeleteProductTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	h.ctx.DB.Delete(&device.ProductTemplate{}, id)
	response.OKMsg(c, "已删除")
}

// MoveProductTemplate 调整产品模板展示顺序（上移/下移）
// POST /api/v1/admin/product-templates/:id/move?direction=up|down
func (h *AdminHandler) MoveProductTemplate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	direction := c.Query("direction")
	if direction != "up" && direction != "down" {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	var list []device.ProductTemplate
	if err := h.ctx.DB.Where("store_id = 0").Order("sort ASC, id ASC").Find(&list).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}

	index := -1
	for i := range list {
		if list[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		response.FailCode(c, errcode.ErrNotFound)
		return
	}

	target := index - 1
	if direction == "down" {
		target = index + 1
	}
	if target < 0 || target >= len(list) {
		response.OKMsg(c, "已经在边界")
		return
	}

	list[index], list[target] = list[target], list[index]
	tx := h.ctx.DB.Begin()
	for i := range list {
		if err := tx.Model(&device.ProductTemplate{}).Where("id = ?", list[i].ID).
			Update("sort", i*10).Error; err != nil {
			tx.Rollback()
			response.FailCode(c, errcode.ErrInternal)
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	response.OKMsg(c, "顺序已调整")
}

// ────────── 余额组管理 ──────────

// ListBalanceGroups 余额组列表（含所属门店）
// GET /api/v1/admin/balance-groups
func (h *AdminHandler) ListBalanceGroups(c *gin.Context) {
	var groups []balanceModel.BalanceGroup
	h.ctx.DB.Order("id ASC").Find(&groups)
	type storeBrief struct {
		ID   uint64 `json:"id"`
		Name string `json:"name"`
	}
	type groupRow struct {
		balanceModel.BalanceGroup
		Stores []storeBrief `json:"stores"`
	}
	rows := make([]groupRow, 0, len(groups))
	for _, g := range groups {
		var ss []storeBrief
		h.ctx.DB.Table("stores").Select("id, name").
			Where("balance_group_id = ? AND deleted_at IS NULL", g.ID).Scan(&ss)
		if len(ss) == 0 {
			// 无活跃门店的组不展示（残留组）
			continue
		}
		rows = append(rows, groupRow{BalanceGroup: g, Stores: ss})
	}
	response.OK(c, rows)
}

// CreateBalanceGroup 新建空余额组
// POST /api/v1/admin/balance-groups
func (h *AdminHandler) CreateBalanceGroup(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	g := balanceModel.BalanceGroup{Name: req.Name}
	g.ID = utils.ShortID()
	if err := h.ctx.DB.Create(&g).Error; err != nil {
		response.FailMsg(c, errcode.ErrInternal, "创建余额组失败")
		return
	}
	response.Created(c, g)
}

// SetStoreBalanceGroup 设置门店所属余额组（group_id=0 表示改为独立新组）
// PUT /api/v1/admin/stores/:id/balance-group
func (h *AdminHandler) SetStoreBalanceGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		GroupID flexID `json:"group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	tx := h.ctx.DB.Begin()
	if tx.Error != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	defer tx.Rollback()

	var s store.Store
	if err := tx.First(&s, id).Error; err != nil {
		response.FailCode(c, errcode.ErrStoreNotFound)
		return
	}
	target := uint64(req.GroupID)
	if target != 0 {
		var g balanceModel.BalanceGroup
		if tx.First(&g, target).Error != nil {
			response.FailMsg(c, errcode.ErrNotFound, "目标余额组不存在")
			return
		}
	}
	if target == s.BalanceGroupID {
		response.OKMsg(c, "未变化")
		return
	}

	var otherCount int64
	tx.Model(&store.Store{}).Where("balance_group_id = ? AND id <> ?", s.BalanceGroupID, s.ID).Count(&otherCount)
	var posCount int64
	tx.Model(&balanceModel.Account{}).Where("group_id = ? AND balance <> 0", s.BalanceGroupID).Count(&posCount)

	action, derr := service.DecideGroupChange(false, otherCount > 0, posCount > 0)
	if derr != nil {
		response.FailMsg(c, errcode.ErrConflict, derr.Error())
		return
	}

	if target == 0 {
		g := balanceModel.BalanceGroup{Name: s.Name}
		g.ID = utils.ShortID()
		if err := tx.Create(&g).Error; err != nil {
			response.FailCode(c, errcode.ErrInternal)
			return
		}
		target = g.ID
	}

	if action == service.GroupChangeMoveAccounts {
		// 源组仅此门店：把源组账户求和迁入目标组
		var srcAccounts []balanceModel.Account
		tx.Where("group_id = ?", s.BalanceGroupID).Find(&srcAccounts)
		for _, a := range srcAccounts {
			var dst balanceModel.Account
			if tx.Where("user_id = ? AND group_id = ?", a.UserID, target).First(&dst).Error == nil {
				tx.Model(&balanceModel.Account{}).Where("id = ?", dst.ID).
					Update("balance", gorm.Expr("balance + ?", a.Balance))
			} else {
				na := balanceModel.Account{UserID: a.UserID, GroupID: target, Balance: a.Balance}
				na.ID = utils.ShortID()
				tx.Create(&na)
			}
		}
		tx.Where("group_id = ?", s.BalanceGroupID).Delete(&balanceModel.Account{})
	}

	if err := tx.Model(&store.Store{}).Where("id = ?", s.ID).Update("balance_group_id", target).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	// 源组若已无活跃门店则清理
	if s.BalanceGroupID != 0 && s.BalanceGroupID != target {
		var cnt int64
		tx.Model(&store.Store{}).Where("balance_group_id = ?", s.BalanceGroupID).Count(&cnt)
		if cnt == 0 {
			tx.Delete(&balanceModel.BalanceGroup{}, s.BalanceGroupID)
		}
	}
	if err := tx.Commit().Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	response.OK(c, gin.H{"store_id": s.ID, "balance_group_id": target})
}

// ForceDeleteStore 强制删除门店及其全部关联数据（含资金记录），不做关联拦截；保留余额组。
// DELETE /api/v1/admin/stores/:id/force
func (h *AdminHandler) ForceDeleteStore(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	tx := h.ctx.DB.Begin()
	if tx.Error != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	defer tx.Rollback()

	var s store.Store
	if err := tx.First(&s, id).Error; err != nil {
		response.FailCode(c, errcode.ErrStoreNotFound)
		return
	}

	// 物理删除该店全部关联数据（余额组/账户保留）
	models := []interface{}{
		&order.Order{}, &payment.Payment{}, &payment.RefundRequest{}, &payment.Recharge{},
		&payment.BalanceLog{}, &payment.Coupon{}, &cfgm.PriceRule{}, &cfgm.BookingRule{},
		&cfgm.UIConfig{}, &cfgm.ConfigHistory{}, &resource.Resource{}, &resource.ResourceType{},
		&device.Device{}, &user.MerchantStore{}, &audit.AuditLog{},
	}
	for _, m := range models {
		if err := tx.Unscoped().Where("store_id = ?", id).Delete(m).Error; err != nil {
			response.FailMsg(c, errcode.ErrInternal, "删除关联数据失败: "+err.Error())
			return
		}
	}
	// 门店级产品模板（store_id=0 为全局，勿删）
	if err := tx.Unscoped().Where("store_id = ? AND store_id <> 0", id).Delete(&device.ProductTemplate{}).Error; err != nil {
		response.FailMsg(c, errcode.ErrInternal, "删除产品模板失败")
		return
	}
	// 物理删除门店本体
	if err := tx.Unscoped().Delete(&s).Error; err != nil {
		response.FailMsg(c, errcode.ErrInternal, "删除门店失败")
		return
	}
	if err := tx.Commit().Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	response.OKMsg(c, "门店及关联数据已强制删除")
}
