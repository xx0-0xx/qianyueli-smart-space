package handler

import (
	"fmt"
	"strconv"
	"time"

	"smart-space-management/backend/internal/model/order"
	"smart-space-management/backend/internal/model/resource"
	"smart-space-management/backend/internal/model/store"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type StoreHandler struct {
	ctx *Context
}

func NewStoreHandler(ctx *Context) *StoreHandler {
	return &StoreHandler{ctx: ctx}
}

// ListStores 附近店铺列表 — 公开接口，按距离排序
// GET /api/v1/stores?lat=30.123&lng=120.456&page=1&size=20
func (h *StoreHandler) ListStores(c *gin.Context) {
	lat, _ := strconv.ParseFloat(c.DefaultQuery("lat", "0"), 64)
	lng, _ := strconv.ParseFloat(c.DefaultQuery("lng", "0"), 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 20
	}

	var stores []store.Store
	var total int64

	// 下架门店仍展示（客户端据 status 提示"暂停营业"），此处不再按状态过滤
	query := h.ctx.DB.Model(&store.Store{})
	query.Count(&total)
	query.Offset((page - 1) * size).Limit(size).Find(&stores)

	// 如果提供了坐标，后处理排序 (简化版 — 生产环境应用 SQL 排序)
	_ = lat
	_ = lng

	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}

	response.PageOK(c, response.PageData{
		List:       stores,
		Total:      total,
		Page:       page,
		PageSize:   size,
		TotalPages: totalPages,
	})
}

// GetStore 获取店铺详情
func (h *StoreHandler) GetStore(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	var s store.Store
	if err := h.ctx.DB.First(&s, id).Error; err != nil {
		response.FailCode(c, errcode.ErrStoreNotFound)
		return
	}
	response.OK(c, s)
}

// GetStoreResources 获取门店下所有资源及其状态
func (h *StoreHandler) GetStoreResources(c *gin.Context) {
	storeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	var resources []resource.Resource
	h.ctx.DB.Where("store_id = ? AND status != ?", storeID, resource.StatusMaint).
		Preload("ResourceType").
		Order("sort ASC").
		Find(&resources)

	// 构建按类型分组的结构
	type ResourceTypeGroup struct {
		Type      resource.ResourceType `json:"type"`
		Resources []resource.Resource   `json:"resources"`
	}

	groupMap := make(map[uint64]*ResourceTypeGroup)
	var groups []*ResourceTypeGroup

	for _, r := range resources {
		tID := r.ResourceTypeID
		if g, ok := groupMap[tID]; ok {
			g.Resources = append(g.Resources, r)
		} else {
			var rt resource.ResourceType
			if r.ResourceType != nil {
				rt = *r.ResourceType
			}
			g := &ResourceTypeGroup{Type: rt, Resources: []resource.Resource{r}}
			groupMap[tID] = g
			groups = append(groups, g)
		}
	}

	response.OK(c, groups)
}

// GetStoreOccupied 门店所有进行中/待支付订单的占用时段（不含用户信息）
// GET /api/v1/stores/:id/occupied
func (h *StoreHandler) GetStoreOccupied(c *gin.Context) {
	storeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	var orders []order.Order
	h.ctx.DB.Select("resource_id, start_time, end_time, status").
		Where("store_id = ? AND status IN (0,1,2) AND end_time > ?", storeID, time.Now().UnixMilli()).
		Find(&orders)

	type Occupied struct {
		ResourceID uint64 `json:"resource_id"`
		StartTime  int64  `json:"start_time"`
		EndTime    int64  `json:"end_time"`
		Status     int    `json:"status"`
	}
	var list []Occupied
	for _, o := range orders {
		list = append(list, Occupied{ResourceID: o.ResourceID, StartTime: o.StartTime, EndTime: o.EndTime, Status: int(o.Status)})
	}
	response.OK(c, list)
}

// TimeSlot 时间段信息
type TimeSlot struct {
	Time       string                `json:"time"`        // "08:00"
	StartMs    int64                 `json:"start_ms"`
	EndMs      int64                 `json:"end_ms"`
	PriceLabel string                `json:"price_label"` // "非高峰" / "高峰" / "优惠"
	UnitPrice  int                   `json:"unit_price"`  // 分
	TimeUnit   int                   `json:"time_unit"`   // 分钟
	Available  int                   `json:"available"`   // 该时段可预约资源数
	Total      int                   `json:"total"`       // 资源总数
	Types      []TypeAvailability    `json:"types"`       // 按类型细分
}

// TypeAvailability 资源类型可用性
type TypeAvailability struct {
	TypeName  string `json:"type_name"`
	Available int    `json:"available"`
	Total     int    `json:"total"`
}

// GetTimeSlots 获取某日时间段列表
// GET /api/v1/stores/:id/slots?date=2026-06-01
func (h *StoreHandler) GetTimeSlots(c *gin.Context) {
	storeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	dateStr := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		response.FailMsg(c, errcode.ErrBadRequest, "日期格式错误，请使用 YYYY-MM-DD")
		return
	}

	var st store.Store
	if h.ctx.DB.First(&st, storeID).Error != nil {
		response.FailCode(c, errcode.ErrStoreNotFound)
		return
	}

	// 解析营业时间
	openH, openM := parseTime(st.OpenTime)
	closeH, closeM := parseTime(st.CloseTime)

	openTs := date.UnixMilli() + int64(openH)*3600000 + int64(openM)*60000
	closeTs := date.UnixMilli() + int64(closeH)*3600000 + int64(closeM)*60000
	// 跨天营业（如 08:00 - 02:00）：closeTs 向后加 24h
	if closeTs <= openTs {
		closeTs += 24 * 3600000
	}

	// 加载该店所有资源（非维护中）
	var resources []resource.Resource
	h.ctx.DB.Where("store_id = ? AND status != ?", storeID, resource.StatusMaint).
		Preload("ResourceType").Find(&resources)

	// 按资源类型分组
	typeGroups := make(map[uint64][]resource.Resource)
	for _, r := range resources {
		typeGroups[r.ResourceTypeID] = append(typeGroups[r.ResourceTypeID], r)
	}

	// 加载当天所有已支付/进行中的订单
	dayStart := date.UnixMilli()
	dayEnd := date.Add(24 * time.Hour).UnixMilli()
	var orders []order.Order
	h.ctx.DB.Where("store_id = ? AND status IN ? AND start_time < ? AND end_time > ?",
		storeID, []int{1, 2}, dayEnd, dayStart).Find(&orders)

	// 预加载价格规则（只查一次，不在循环内重复查）
	var rule struct {
		BasePrice     int
		TimeUnit      int
		PeakSurcharge string
	}
	h.ctx.DB.Table("price_rules").
		Where("store_id = ? AND status = 1", storeID).Order("is_default DESC").Limit(1).
		Scan(&rule)
	if rule.BasePrice <= 0 {
		rule.BasePrice = 5000
	}
	if rule.TimeUnit <= 0 {
		rule.TimeUnit = 30
	}

	// 批量预加载所有资源类型的专属价格（避免循环内 N+1）
	typePrices := h.loadTypePrices(storeID)

	// 生成 30 分钟槽位
	var slots []TimeSlot
	slotMs := int64(30 * 60000)
	for ts := openTs; ts < closeTs; ts += slotMs {
		slotEnd := ts + slotMs

		// 检查该时段已占用的资源 ID
		occupiedIDs := make(map[uint64]bool)
		for _, o := range orders {
			if o.StartTime < slotEnd && o.EndTime > ts {
				occupiedIDs[o.ResourceID] = true
			}
		}

		// 统计可用，按类型查询专属价格
		totalAll := 0
		availAll := 0
		var types []TypeAvailability
		showPrice := rule.BasePrice  // 默认通用价格

		for tid, resList := range typeGroups {
			total := len(resList)
			avail := 0
			for _, r := range resList {
				if !occupiedIDs[r.ID] {
					avail++
				}
			}
			totalAll += total
			availAll += avail

			typeName := fmt.Sprintf("类型%d", tid)
			for _, r := range resList {
				if r.ResourceType != nil {
					typeName = r.ResourceType.Name
					break
				}
			}

			// 查该类型的专属价格（从预加载 map 中取）
			typePrice := rule.BasePrice
			if p, ok := typePrices[tid]; ok && p > 0 {
				typePrice = p
			}

			types = append(types, TypeAvailability{
				TypeName: typeName, Available: avail, Total: total,
			})

			// 用第一个类型的价格作为时段显示价
			if showPrice == rule.BasePrice && typePrice != rule.BasePrice {
				showPrice = typePrice
			}
		}

		// 价格标签
		priceLabel := checkPeak(ts, rule.PeakSurcharge)
		unitPrice := showPrice
		timeUnit := rule.TimeUnit

		// 格式化时间：跨天时按实际钟表显示（24h取模）
		slotHour := (openH + (int(ts-openTs) / 3600000)) % 24
		slotMin := (int(ts-openTs) / 60000) % 60
		timeStr := fmt.Sprintf("%02d:%02d", slotHour, slotMin)
		if openH+(int(ts-openTs)/3600000) >= 24 {
			timeStr += " (次日)"
		}

		slots = append(slots, TimeSlot{
			Time:       timeStr,
			StartMs:    ts,
			EndMs:      slotEnd,
			PriceLabel: priceLabel,
			UnitPrice:  unitPrice,
			TimeUnit:   timeUnit,
			Available:  availAll,
			Total:      totalAll,
			Types:      types,
		})
	}

	response.OK(c, gin.H{
		"date":  dateStr,
		"store": st.Name,
		"slots": slots,
	})
}

// SlotResource 时段可用的具体资源
type SlotResource struct {
	ResourceID     uint64 `json:"resource_id,string"`
	Name           string `json:"name"`
	TypeName       string `json:"type_name"`
	ResourceTypeID uint64 `json:"resource_type_id,string"`
	Price          int    `json:"price"`
	PriceLabel     string `json:"price_label"`
}

// GetSlotResources 获取某时段可预约的资源列表
// GET /api/v1/stores/:id/slots/resources?start=xxx&end=xxx
func (h *StoreHandler) GetSlotResources(c *gin.Context) {
	storeID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	startMs, _ := strconv.ParseInt(c.Query("start"), 10, 64)
	endMs, _ := strconv.ParseInt(c.Query("end"), 10, 64)

	if startMs == 0 || endMs == 0 {
		response.FailMsg(c, errcode.ErrBadRequest, "缺少 start/end 参数")
		return
	}

	// 加载该时段占用的资源
	var occupiedOrders []order.Order
	h.ctx.DB.Where("store_id = ? AND status IN ? AND start_time < ? AND end_time > ?",
		storeID, []int{1, 2}, endMs, startMs).Find(&occupiedOrders)
	occupied := make(map[uint64]bool)
	for _, o := range occupiedOrders {
		occupied[o.ResourceID] = true
	}

	// 加载所有非维护资源
	var resources []resource.Resource
	h.ctx.DB.Where("store_id = ? AND status != ?", storeID, resource.StatusMaint).
		Preload("ResourceType").Order("sort ASC").Find(&resources)

	// 获取价格
	var rule struct {
		BasePrice     int
		PeakSurcharge string
	}
	h.ctx.DB.Table("price_rules").
		Where("store_id = ? AND status = 1", storeID).Order("is_default DESC").Limit(1).Scan(&rule)
	if rule.BasePrice <= 0 {
		rule.BasePrice = 5000
	}
	label := checkPeak(startMs, rule.PeakSurcharge)

	// 批量预加载所有资源类型的专属价格（避免循环内 N+1）
	typePrices := h.loadTypePrices(storeID)

	var result []SlotResource
	for _, r := range resources {
		typeName := ""
		if r.ResourceType != nil {
			typeName = r.ResourceType.Name
		}

		// 查该类型的专属价格，没有则用通用价
		resPrice := rule.BasePrice
		if p, ok := typePrices[r.ResourceTypeID]; ok && p > 0 {
			resPrice = p
		}

		inUse := occupied[r.ID]
		result = append(result, SlotResource{
			ResourceID:     r.ID,
			Name:           r.Name,
			TypeName:       typeName,
			ResourceTypeID: r.ResourceTypeID,
			Price:          resPrice,
			PriceLabel:     label,
		})
		_ = inUse
	}

	// 按类型分组
	type GroupedResource struct {
		TypeName      string         `json:"type_name"`
		ResourceTypeID uint64        `json:"resource_type_id"`
		Price         int            `json:"price"`
		PriceLabel    string         `json:"price_label"`
		Items         []SlotResource `json:"items"`
	}

	groupMap := make(map[uint64]*GroupedResource)
	var groups []*GroupedResource
	for _, r := range result {
		g, ok := groupMap[r.ResourceTypeID]
		if !ok {
			g = &GroupedResource{
				TypeName:       r.TypeName,
				ResourceTypeID: r.ResourceTypeID,
				Price:          r.Price,
				PriceLabel:     r.PriceLabel,
			}
			groupMap[r.ResourceTypeID] = g
			groups = append(groups, g)
		}
		if !occupied[r.ResourceID] {
			g.Items = append(g.Items, r)
		}
	}

	response.OK(c, gin.H{
		"start_ms": startMs,
		"end_ms":   endMs,
		"duration": (endMs - startMs) / 60000,
		"groups":   groups,
	})
}

func parseTime(t string) (h, m int) {
	fmt.Sscanf(t, "%d:%d", &h, &m)
	if h < 0 || h > 24 {
		h = 0
	}
	if m < 0 || m >= 60 {
		m = 0
	}
	return
}

func checkPeak(ts int64, surchargeJSON string) string {
	// 简化：按小时判断，8-12 和 18-22 为高峰
	t := time.UnixMilli(ts)
	hour := t.Hour()
	if (hour >= 8 && hour < 12) || (hour >= 18 && hour < 22) {
		return "高峰"
	}
	if hour >= 12 && hour < 14 {
		return "优惠"
	}
	return "非高峰"
}


func (h *StoreHandler) GetStorePrice(c *gin.Context) {
	storeID,_:=strconv.ParseUint(c.Param("id"),10,64)
	type priceItem struct {
		ResourceTypeID uint64 `json:"resource_type_id"`
		BasePrice      int    `json:"unit_price"`
		TimeUnit       int    `json:"time_unit"`
	}
	var results []priceItem
	h.ctx.DB.Table("price_rules").Select("resource_type_id,base_price,time_unit").Where("store_id=? AND status=1",storeID).Find(&results)
	prices := make(map[string]interface{})
	for _, r := range results {
		prices[fmt.Sprintf("%d", r.ResourceTypeID)] = gin.H{"unit_price":r.BasePrice,"time_unit":r.TimeUnit,"display":fmt.Sprintf("%d元/%d分钟",r.BasePrice/100,r.TimeUnit)}
	}
	response.OK(c,gin.H{"prices":prices})
}

// GetStoreConfig 商家获取本店配置
func (h *StoreHandler) GetStoreConfig(c *gin.Context) {
	storeID := GetStoreID(c)
	if storeID == 0 {
		response.FailCode(c, errcode.ErrBadRequest); return
	}
	var s store.Store
	if h.ctx.DB.First(&s, storeID).Error != nil {
		response.FailCode(c, errcode.ErrNotFound); return
	}
	response.OK(c, gin.H{
		"cover":              s.Cover,
		"logo":               s.Logo,
		"phone":              s.Phone,
		"config_json":        s.ConfigJSON,
		"auto_approve_refund": s.AutoApproveRefund,
	})
}

// UpdateStoreConfig 商家修改本店配置
func (h *StoreHandler) UpdateStoreConfig(c *gin.Context) {
	storeID := GetStoreID(c)
	if storeID == 0 {
		response.FailCode(c, errcode.ErrBadRequest); return
	}
	var req struct {
		Phone             string `json:"phone"`
		ConfigJSON        string `json:"config_json"`
		AutoApproveRefund *bool  `json:"auto_approve_refund"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest); return
	}
	updates := map[string]interface{}{
		"phone": req.Phone, "config_json": req.ConfigJSON,
	}
	if req.AutoApproveRefund != nil {
		updates["auto_approve_refund"] = *req.AutoApproveRefund
	}
	h.ctx.DB.Model(&store.Store{}).Where("id = ?", storeID).Updates(updates)
	response.OKMsg(c, "已保存")
}

// loadTypePrices 批量加载门店所有资源类型的专属价格，返回 map[resource_type_id]base_price
func (h *StoreHandler) loadTypePrices(storeID uint64) map[uint64]int {
	type priceRow struct {
		ResourceTypeID uint64
		BasePrice      int
	}
	var rows []priceRow
	h.ctx.DB.Table("price_rules").
		Select("resource_type_id, base_price").
		Where("store_id = ? AND status = 1", storeID).
		Find(&rows)

	prices := make(map[uint64]int, len(rows))
	for _, r := range rows {
		prices[r.ResourceTypeID] = r.BasePrice
	}
	return prices
}
