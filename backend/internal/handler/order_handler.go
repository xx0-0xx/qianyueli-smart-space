package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"smart-space-management/backend/internal/model/order"
	"smart-space-management/backend/internal/model/payment"
	"smart-space-management/backend/internal/model/resource"
	"smart-space-management/backend/internal/model/store"
	"smart-space-management/backend/internal/model/user"
	"smart-space-management/backend/internal/service"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// FlexID 柔性 ID：JSON 解析时同时接受数字和字符串
type FlexID uint64

func (f *FlexID) UnmarshalJSON(b []byte) error {
	var n uint64
	if err := json.Unmarshal(b, &n); err == nil {
		*f = FlexID(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		*f = FlexID(n)
		return nil
	}
	return fmt.Errorf("FlexID: invalid value %s", string(b))
}

type OrderHandler struct {
	ctx *Context
}

func NewOrderHandler(ctx *Context) *OrderHandler {
	return &OrderHandler{ctx: ctx}
}

// CreateOrderReq 创建订单请求
type CreateOrderReq struct {
	ResourceID FlexID `json:"resource_id" binding:"required"`
	StoreID    FlexID `json:"store_id" binding:"required"`
	StartTime  int64  `json:"start_time" binding:"required"` // unix ms
	EndTime    int64  `json:"end_time" binding:"required"`   // unix ms
	PayMethod  string `json:"pay_method"`                    // wechat=微信(默认) balance=余额
}

// CreateOrder 创建订单 — 含 Redis 分布式锁防止并发预约
// POST /api/v1/orders
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	userID := GetUserID(c)
	if userID == 0 {
		response.FailCode(c, errcode.ErrUnauthorized)
		return
	}

	var currentUser user.User
	if err := h.ctx.DB.First(&currentUser, userID).Error; err != nil {
		response.FailCode(c, errcode.ErrUserNotFound)
		return
	}
	if strings.TrimSpace(currentUser.Phone) == "" {
		response.FailCode(c, errcode.ErrPhoneRequired)
		return
	}

	now := time.Now().UnixMilli()

	// 校验时间：允许当前时段（开始时间不早于当前时间 5 分钟）
	if req.StartTime < now-5*60*1000 {
		response.FailMsg(c, errcode.ErrBadRequest, "预约时间必须晚于当前时间")
		return
	}
	if req.EndTime <= req.StartTime {
		response.FailMsg(c, errcode.ErrBadRequest, "结束时间必须晚于开始时间")
		return
	}

	// 计算时长(分钟)和费用
	durationMinutes := int((req.EndTime - req.StartTime) / 60000)

	rid := uint64(req.ResourceID)
	sid := uint64(req.StoreID)

	// ── Redis 分布式锁：锁定资源 ──
	lockKey := fmt.Sprintf("ssm:resource:lock:%d", rid)
	lockTTL := h.ctx.Cfg.Lock.BookingTTL
	lockVal := fmt.Sprintf("%d:%d", userID, now)

	ok, err := h.ctx.Redis.SetNX(c.Request.Context(), lockKey, lockVal, lockTTL).Result()
	if err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if !ok {
		response.FailCode(c, errcode.ErrResourceLockFail)
		return
	}
	defer func() {
		// 释放锁 — 仅在值匹配时释放
		script := `
			if redis.call("GET", KEYS[1]) == ARGV[1] then
				return redis.call("DEL", KEYS[1])
			else
				return 0
			end
		`
		h.ctx.Redis.Eval(c.Request.Context(), script, []string{lockKey}, lockVal)
	}()

	// ── 校验资源状态 ──
	var res resource.Resource
	if err := h.ctx.DB.First(&res, rid).Error; err != nil {
		response.FailCode(c, errcode.ErrResourceNotFound)
		return
	}
	if res.Status == resource.StatusMaint {
		response.FailCode(c, errcode.ErrResourceInMaint)
		return
	}
	if res.StoreID != sid {
		response.FailMsg(c, errcode.ErrBadRequest, "资源不属于该门店")
		return
	}
	// ── 校验门店是否已下架 ──
	var st store.Store
	if h.ctx.DB.First(&st, sid).Error != nil || st.Status != store.StatusOpen {
		response.FailMsg(c, errcode.ErrBadRequest, "店铺暂停营业")
		return
	}

	// ── 时间冲突校验 ──
	var conflictCount int64
	h.ctx.DB.Model(&order.Order{}).
		Where("resource_id = ? AND status IN (0,1,2) AND start_time < ? AND end_time > ?", rid, req.EndTime, req.StartTime).
		Count(&conflictCount)
	if conflictCount > 0 {
		response.FailCode(c, errcode.ErrOrderTimeConflict)
		return
	}

	// ── 查询价格 ──
	unitPrice := h.getUnitPrice(sid, res.ResourceTypeID)
	timeUnit := h.getTimeUnit(c.Request.Context(), sid, res.ResourceTypeID)
	if timeUnit <= 0 {
		timeUnit = 60
	}
	totalAmount := unitPrice * durationMinutes / timeUnit
	if totalAmount <= 0 {
		totalAmount = unitPrice // 保底
	}

	// ── 会员折扣 ──
	originalAmount := totalAmount
	memberDiscount := 0
	var memberLevelID *uint64
	var memberLevelName string
	active, err := service.FindActiveMember(h.ctx.DB, sid, userID, now)
	if err != nil {
		log.Printf("[ORDER] find active member failed store=%d user=%d: %v", sid, userID, err)
		response.FailMsg(c, errcode.ErrInternal, "会员信息读取失败，请稍后重试")
		return
	}
	if active != nil {
		totalAmount, memberDiscount = service.ApplyMemberDiscount(originalAmount, active.Level.EffectiveDiscountBP())
		id := active.Level.ID
		memberLevelID = &id
		memberLevelName = active.Level.Name
	}

	// ── 创建订单 ──
	orderNo := newOrderNo()
	payByBalance := req.PayMethod == "balance"
	status := order.StatusPendingPay
	var paidAt *int64
	if payByBalance {
		status = order.StatusPaid
		nowMs := time.Now().UnixMilli()
		paidAt = &nowMs
	}
	o := order.Order{
		OrderNo:         orderNo,
		UserID:          userID,
		StoreID:         sid,
		ResourceID:      rid,
		ResourceName:    res.Name,
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
		Duration:        durationMinutes,
		UnitPrice:       unitPrice,
		TotalAmount:     totalAmount,
		OriginalAmount:  originalAmount,
		MemberDiscount:  memberDiscount,
		MemberLevelID:   memberLevelID,
		MemberLevelName: memberLevelName,
		Status:          status,
		PaidAt:          paidAt,
	}
	o.ID = utils.ShortID()

	tx := h.ctx.DB.Begin()
	if tx.Create(&o).Error != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	// 余额扣款 + 流水（同一事务，按门店所属余额组）
	if payByBalance {
		newBal, gid, derr := h.ctx.BalanceSvc.Debit(tx, userID, sid, int64(totalAmount))
		if derr != nil {
			tx.Rollback()
			if errors.Is(derr, service.ErrInsufficientBalance) {
				response.FailMsg(c, errcode.ErrBadRequest, "余额不足")
			} else {
				response.FailCode(c, errcode.ErrInternal)
			}
			return
		}
		balLog := payment.BalanceLog{
			UserID: userID, Amount: -int64(totalAmount), Balance: newBal,
			Type: "order", RefID: orderNo, Desc: "预约订单扣款",
			StoreID: sid, GroupID: gid,
		}
		balLog.ID = utils.ShortID()
		if err := tx.Create(&balLog).Error; err != nil {
			tx.Rollback()
			response.FailCode(c, errcode.ErrInternal)
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}

	// 余额支付即视为已开始，自动下发开电 + 语音
	if payByBalance && h.ctx.DeviceSvc != nil {
		go h.ctx.DeviceSvc.SendSessionCmd(o.ResourceID, "session_start", h.ctx.Cfg.Server.PublicURL)
	}

	// 不锁资源状态，依赖 Redis 锁 + 订单时间冲突校验
	// 资源只在支付成功后变为"占用中"

	// ── 发布订单事件 ──
	// ws.PublishOrderEvent(h.ctx.Redis, o.ID, o.StoreID, "created")

	response.Created(c, gin.H{
		"order_id":     fmt.Sprintf("%d", o.ID),
		"order_no":     o.OrderNo,
		"total_amount": o.TotalAmount,
		"start_time":   o.StartTime,
		"end_time":     o.EndTime,
	})
}

// ListMyOrders 我的订单列表
func (h *OrderHandler) ListMyOrders(c *gin.Context) {
	userID := GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 20
	}

	var orders []order.Order
	var total int64

	query := h.ctx.DB.Model(&order.Order{}).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)
	query.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&orders)

	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}

	response.PageOK(c, response.PageData{
		List:       orders,
		Total:      total,
		Page:       page,
		PageSize:   size,
		TotalPages: totalPages,
	})
}

// GetOrder 订单详情
func (h *OrderHandler) GetOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	var o order.Order
	if h.ctx.DB.First(&o, id).Error != nil {
		response.FailCode(c, errcode.ErrOrderNotFound)
		return
	}

	// 权限：仅有本人或商家/管理员可查看
	userID := GetUserID(c)
	if o.UserID != userID {
		role, _ := c.Get("user_role")
		if role == "merchant" {
			// 商家仅可查看本店订单
			if o.StoreID != GetStoreID(c) {
				response.FailCode(c, errcode.ErrPermissionDenied)
				return
			}
		} else if role != "admin" && role != "super_admin" {
			response.FailCode(c, errcode.ErrPermissionDenied)
			return
		}
	}
	response.OK(c, o)
}

// CancelOrder 取消订单
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	var o order.Order
	if h.ctx.DB.First(&o, id).Error != nil {
		response.FailCode(c, errcode.ErrOrderNotFound)
		return
	}

	// 权限：仅本人或管理员可取消（商家取消走商家路由）
	if o.UserID != GetUserID(c) {
		role, _ := c.Get("user_role")
		if role != "admin" && role != "super_admin" {
			response.FailCode(c, errcode.ErrForbidden)
			return
		}
	}

	// 仅待支付订单可由用户自行取消；已支付订单走退款申请
	if o.Status != order.StatusPendingPay {
		if o.Status == order.StatusPaid || o.Status == order.StatusInProgress {
			response.FailMsg(c, errcode.ErrOrderStatusInvalid, "已支付订单请发起退款申请")
		} else {
			response.FailCode(c, errcode.ErrOrderStatusInvalid)
		}
		return
	}

	tx := h.ctx.DB.Begin()
	if err := tx.Model(&resource.Resource{}).Where("id = ?", o.ResourceID).
		Update("status", resource.StatusFree).Error; err != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if err := tx.Model(&order.Order{}).Where("id = ?", o.ID).
		Update("status", order.StatusCanceled).Error; err != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}

	// 自动下发 session_end（关门+断电）
	go h.ctx.DeviceSvc.SendSessionCmd(o.ResourceID, "session_end", h.ctx.Cfg.Server.PublicURL)

	response.OKMsg(c, "订单已取消")
}

// ListStoreOrders 商家查看本店订单
func (h *OrderHandler) ListStoreOrders(c *gin.Context) {
	storeID := GetStoreID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	role, _ := c.Get("user_role")

	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}

	query := h.ctx.DB.Model(&order.Order{})
	if role != "super_admin" && role != "admin" {
		query = query.Where("store_id = ?", storeID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var orders []order.Order
	var total int64
	query.Count(&total)
	query.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&orders)

	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}

	response.PageOK(c, response.PageData{
		List:       orders,
		Total:      total,
		Page:       page,
		PageSize:   size,
		TotalPages: totalPages,
	})
}

// ── helpers ──

// getUnitPrice 获取资源类型的单价 (分)
func (h *OrderHandler) getUnitPrice(storeID, resourceTypeID uint64) int {
	return service.BasePrice(h.ctx.DB, storeID, resourceTypeID)
}

// getTimeUnit 获取计费单位 (分钟)
func (h *OrderHandler) getTimeUnit(ctx context.Context, storeID, resourceTypeID uint64) int {
	var result struct{ TimeUnit int }
	h.ctx.DB.Table("price_rules").
		Select("time_unit").
		Where("store_id = ? AND resource_type_id IN ? AND status = 1", storeID, []uint64{resourceTypeID, 0}).
		Order("resource_type_id DESC, is_default DESC").
		Limit(1).
		Scan(&result)
	if result.TimeUnit > 0 {
		return result.TimeUnit
	}
	return 60
}

func newOrderNo() string {
	return fmt.Sprintf("SS%d%04d", time.Now().UnixMilli(), utils.ShortID()%10000)
}
