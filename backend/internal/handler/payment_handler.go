package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"smart-space-management/backend/internal/model/order"
	"smart-space-management/backend/internal/model/payment"
	"smart-space-management/backend/internal/model/resource"
	"smart-space-management/backend/internal/model/store"
	"smart-space-management/backend/internal/service"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PaymentHandler struct {
	ctx *Context
}

func canRefundFromBalance(status order.OrderStatus, balanceDebitFound bool) bool {
	return balanceDebitFound && (status == order.StatusPaid || status == order.StatusInProgress)
}

// canRefundSettle 审批/结算退款：用户发起退款申请后订单已进入 StatusRefunding，
// 因此结算时需同时接受 Paid / InProgress / Refunding 三种状态。
func canRefundSettle(status order.OrderStatus, balanceDebitFound bool) bool {
	return balanceDebitFound && (status == order.StatusPaid || status == order.StatusInProgress || status == order.StatusRefunding)
}

func (h *PaymentHandler) hasBalanceDebit(o order.Order) bool {
	var debit payment.BalanceLog
	return h.ctx.DB.Where("user_id = ? AND type = ? AND ref_id = ? AND amount = ?", o.UserID, "order", o.OrderNo, -int64(o.TotalAmount)).First(&debit).Error == nil
}

func refundBalance(tx *gorm.DB, balanceSvc *service.BalanceService, o order.Order, amount int, reason string) error {
	if amount <= 0 || amount != o.TotalAmount {
		return fmt.Errorf("refund must equal the order total")
	}
	newBal, gid, err := balanceSvc.Credit(tx, o.UserID, o.StoreID, int64(amount))
	if err != nil {
		return err
	}
	log := payment.BalanceLog{
		UserID: o.UserID, Amount: int64(amount), Balance: newBal,
		Type: "refund", RefID: o.OrderNo, Desc: reason,
		StoreID: o.StoreID, GroupID: gid,
	}
	log.ID = utils.ShortID()
	return tx.Create(&log).Error
}

// applyRefundTx 在同一事务内执行退款结算：
// 退款申请 → 已同意；订单 → 已退款；退回余额；支付流水 → 已退款；释放房间。
// 供「商家审批」与「自动同意退款」复用。
func (h *PaymentHandler) applyRefundTx(tx *gorm.DB, rr payment.RefundRequest, o order.Order) error {
	res := tx.Model(&payment.RefundRequest{}).
		Where("id = ? AND status = ?", rr.ID, payment.RefundPending).
		Update("status", payment.RefundApproved)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return fmt.Errorf("退款申请已处理")
	}
	res = tx.Model(&order.Order{}).
		Where("id = ? AND status IN ?", o.ID, []order.OrderStatus{order.StatusPaid, order.StatusInProgress, order.StatusRefunding}).
		Updates(map[string]interface{}{
			"status": order.StatusRefunded, "refund_amount": rr.Amount, "cancel_reason": rr.Reason,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return fmt.Errorf("订单状态已变化")
	}
	if err := refundBalance(tx, h.ctx.BalanceSvc, o, rr.Amount, rr.Reason); err != nil {
		return err
	}
	tx.Model(&payment.Payment{}).Where("order_id = ? AND status = ?", o.ID, payment.StatusSuccess).
		Updates(map[string]interface{}{"status": payment.StatusRefunded, "refund_amount": rr.Amount})
	if err := tx.Model(&resource.Resource{}).Where("id = ? AND store_id = ?", o.ResourceID, o.StoreID).
		Update("status", resource.StatusFree).Error; err != nil {
		return err
	}
	return nil
}

func NewPaymentHandler(ctx *Context) *PaymentHandler {
	return &PaymentHandler{ctx: ctx}
}

// CreatePayment 创建支付 — 生成 out_trade_no 并调用微信统一下单
// POST /api/v1/orders/:id/pay
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	userID := GetUserID(c)

	var o order.Order
	if h.ctx.DB.First(&o, orderID).Error != nil {
		response.FailCode(c, errcode.ErrOrderNotFound)
		return
	}
	if o.UserID != userID {
		response.FailCode(c, errcode.ErrPermissionDenied)
		return
	}
	if o.Status != order.StatusPendingPay {
		response.FailMsg(c, errcode.ErrOrderStatusInvalid, "仅待支付订单可以发起支付")
		return
	}

	// 微信支付未配置商户号时明确拒绝，避免返回伪造的 prepay_id 误导客户端
	if h.ctx.Cfg.Wechat.Pay.MchID == "" {
		response.FailMsg(c, errcode.ErrWechatPayFail, "在线支付未开通，请使用余额支付")
		return
	}

	// ── 生成幂等 out_trade_no ──
	outTradeNo := fmt.Sprintf("SSM%s%d", time.Now().Format("20060102150405"), orderID%100000)

	// ── 插入支付记录 (唯一约束确保幂等) ──
	p := payment.Payment{
		OutTradeNo: outTradeNo,
		OrderID:    o.ID,
		OrderNo:    o.OrderNo,
		UserID:     userID,
		StoreID:    o.StoreID,
		Amount:     o.TotalAmount,
		Status:     payment.StatusPending,
	}
	p.ID = utils.ShortID()

	if err := h.ctx.DB.Create(&p).Error; err != nil {
		// out_trade_no 冲突 = 重复请求
		response.FailCode(c, errcode.ErrPaymentDuplicate)
		return
	}

	// ── 调用微信统一下单生成 prepay_id ──
	prepayID, err := h.createWechatOrder(c.Request.Context(), outTradeNo, o)
	if err != nil {
		h.ctx.DB.Model(&p).Update("status", payment.StatusFailed)
		response.FailMsg(c, errcode.ErrWechatPayFail, err.Error())
		return
	}

	response.OK(c, gin.H{
		"out_trade_no": outTradeNo,
		"prepay_id":    prepayID,
		"amount":       o.TotalAmount,
		"order_no":     o.OrderNo,
	})
}

// WechatPayCallback 微信支付回调 — 幂等处理
// POST /api/v1/payment/callback
func (h *PaymentHandler) WechatPayCallback(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)

	// ── 解析微信回调 ──
	var cb struct {
		OutTradeNo    string `json:"out_trade_no"`
		TransactionID string `json:"transaction_id"`
		ResultCode    string `json:"result_code"`
		ReturnCode    string `json:"return_code"`
	}
	json.Unmarshal(body, &cb)

	if cb.ResultCode != "SUCCESS" {
		response.FailMsg(c, errcode.ErrWechatPayFail, "支付未成功")
		return
	}

	// ── 幂等：通过 out_trade_no 唯一约束保证 ──
	var p payment.Payment
	if err := h.ctx.DB.Where("out_trade_no = ?", cb.OutTradeNo).First(&p).Error; err != nil {
		response.FailCode(c, errcode.ErrPaymentNotFound)
		return
	}

	// 已处理过，直接返回成功
	if p.Status == payment.StatusSuccess {
		c.String(http.StatusOK, `<xml><return_code><![CDATA[SUCCESS]]></return_code></xml>`)
		return
	}

	// ── 事务：更新支付 + 订单 + 资源 ──
	now := time.Now().UnixMilli()
	tx := h.ctx.DB.Begin()

	// 更新支付状态
	tx.Model(&p).Updates(map[string]interface{}{
		"status":         payment.StatusSuccess,
		"transaction_id": cb.TransactionID,
		"pay_time":       now,
		"callback_raw":   string(body),
	})

	// 更新订单状态
	tx.Model(&order.Order{}).Where("id = ?", p.OrderID).
		Updates(map[string]interface{}{
			"status":  order.StatusPaid,
			"paid_at": now,
		})

	tx.Commit()

	// 支付成功即视为已开始，自动下发开电 + 语音
	var o order.Order
	if h.ctx.DeviceSvc != nil && h.ctx.DB.First(&o, p.OrderID).Error == nil {
		go h.ctx.DeviceSvc.SendSessionCmd(o.ResourceID, "session_start", h.ctx.Cfg.Server.PublicURL)
	}

	// ── 发布订单已支付事件 ──
	// ws.PublishOrderEvent(h.ctx.Redis, p.OrderID, p.StoreID, "paid")

	c.String(http.StatusOK, `<xml><return_code><![CDATA[SUCCESS]]></return_code></xml>`)
}

// RequestRefund 用户申请退款 — 需商家审核
// POST /api/v1/merchant/orders/:id/refund-request
func (h *PaymentHandler) RequestRefund(c *gin.Context) {
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	userID := GetUserID(c)

	var o order.Order
	if h.ctx.DB.Where("id = ? AND user_id = ?", orderID, userID).First(&o).Error != nil {
		response.FailCode(c, errcode.ErrOrderNotFound)
		return
	}
	if o.Status != order.StatusPaid && o.Status != order.StatusInProgress {
		response.FailMsg(c, errcode.ErrOrderStatusInvalid, "仅已支付或进行中的订单可申请退款")
		return
	}
	// 距开始不足 15 分钟（含已开始）不可退款
	if time.Now().UnixMilli() > o.StartTime-15*60*1000 {
		response.FailMsg(c, errcode.ErrBadRequest, "距开始不足15分钟，已不可退款")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	// 门店是否开启"自动同意退款"（缺省视为开启）
	autoApprove := true
	var st store.Store
	if h.ctx.DB.Select("auto_approve_refund").First(&st, o.StoreID).Error == nil {
		autoApprove = st.AutoApproveRefund
	}

	rr := payment.RefundRequest{
		OrderID: o.ID, UserID: userID, StoreID: o.StoreID,
		Amount: o.TotalAmount, Reason: req.Reason, Status: payment.RefundPending,
	}
	rr.ID = utils.ShortID()

	tx := h.ctx.DB.Begin()
	if tx.Error != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	res := tx.Model(&order.Order{}).Where("id = ? AND user_id = ? AND status IN ?", o.ID, userID, []order.OrderStatus{order.StatusPaid, order.StatusInProgress}).Update("status", order.StatusRefunding)
	if res.Error != nil || res.RowsAffected != 1 {
		tx.Rollback()
		response.FailMsg(c, errcode.ErrOrderStatusInvalid, "订单状态已变化")
		return
	}
	if err := tx.Create(&rr).Error; err != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if autoApprove {
		if err := h.applyRefundTx(tx, rr, o); err != nil {
			tx.Rollback()
			response.FailMsg(c, errcode.ErrInternal, err.Error())
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if autoApprove {
		if h.ctx.DeviceSvc != nil {
			go h.ctx.DeviceSvc.SendSessionCmd(o.ResourceID, "session_end", h.ctx.Cfg.Server.PublicURL)
		}
		response.OKMsg(c, "已退款并退回余额")
		return
	}
	response.OKMsg(c, "退款申请已提交")
}

// ListRefundRequests 商家查看退款申请
// GET /api/v1/merchant/refund-requests
func (h *PaymentHandler) ListRefundRequests(c *gin.Context) {
	storeID := GetStoreID(c)
	var list []payment.RefundRequest
	h.ctx.DB.Where("store_id = ?", storeID).Order("created_at DESC").Find(&list)
	response.OK(c, list)
}

// ApproveRefund 商家同意退款
// POST /api/v1/merchant/refund-requests/:id/approve
func (h *PaymentHandler) ApproveRefund(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	storeID := GetStoreID(c)

	var rr payment.RefundRequest
	if h.ctx.DB.Where("id = ? AND store_id = ?", id, storeID).First(&rr).Error != nil {
		response.FailCode(c, errcode.ErrNotFound)
		return
	}
	if rr.Status != payment.RefundPending {
		response.FailMsg(c, errcode.ErrOrderStatusInvalid, "已处理")
		return
	}

	var o order.Order
	if err := h.ctx.DB.Where("id = ? AND store_id = ?", rr.OrderID, storeID).First(&o).Error; err != nil {
		response.FailCode(c, errcode.ErrOrderNotFound)
		return
	}
	if !canRefundSettle(o.Status, h.hasBalanceDebit(o)) || rr.Amount != o.TotalAmount {
		response.FailMsg(c, errcode.ErrBadRequest, "仅支持已确认余额支付的订单退款")
		return
	}

	tx := h.ctx.DB.Begin()
	if tx.Error != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if err := h.applyRefundTx(tx, rr, o); err != nil {
		tx.Rollback()
		response.FailMsg(c, errcode.ErrOrderStatusInvalid, err.Error())
		return
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}

	if h.ctx.DeviceSvc != nil {
		go h.ctx.DeviceSvc.SendSessionCmd(o.ResourceID, "session_end", h.ctx.Cfg.Server.PublicURL)
	}
	response.OKMsg(c, "已退款并退回余额")
}

// RejectRefund 商家拒绝退款
// POST /api/v1/merchant/refund-requests/:id/reject
func (h *PaymentHandler) RejectRefund(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	storeID := GetStoreID(c)

	var rr payment.RefundRequest
	if h.ctx.DB.Where("id = ? AND store_id = ? AND status = ?", id, storeID, payment.RefundPending).First(&rr).Error != nil {
		response.FailMsg(c, errcode.ErrNotFound, "退款申请不存在或已处理")
		return
	}

	tx := h.ctx.DB.Begin()
	if tx.Error != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	claimed := tx.Model(&payment.RefundRequest{}).Where("id = ? AND store_id = ? AND status = ?", id, storeID, payment.RefundPending).Update("status", payment.RefundRejected)
	if claimed.Error != nil || claimed.RowsAffected != 1 {
		tx.Rollback()
		response.FailMsg(c, errcode.ErrOrderStatusInvalid, "退款申请已处理")
		return
	}
	changed := tx.Model(&order.Order{}).Where("id = ? AND store_id = ? AND status = ?", rr.OrderID, storeID, order.StatusRefunding).Update("status", order.StatusInProgress)
	if changed.Error != nil || changed.RowsAffected != 1 {
		tx.Rollback()
		response.FailCode(c, errcode.ErrOrderStatusInvalid)
		return
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	response.OKMsg(c, "已拒绝")
}

// MerchantCancelOrder 商家取消待支付订单
// POST /api/v1/merchant/orders/:id/cancel
func (h *PaymentHandler) MerchantCancelOrder(c *gin.Context) {
	orderID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	storeID := GetStoreID(c)

	var o order.Order
	if h.ctx.DB.Where("id = ? AND store_id = ?", orderID, storeID).First(&o).Error != nil {
		response.FailCode(c, errcode.ErrOrderNotFound)
		return
	}
	if o.Status != order.StatusPendingPay {
		response.FailMsg(c, errcode.ErrOrderStatusInvalid, "仅待支付订单可取消")
		return
	}
	tx := h.ctx.DB.Begin()
	tx.Model(&o).Updates(map[string]interface{}{"status": order.StatusCanceled, "cancel_reason": "商家取消"})
	tx.Model(&resource.Resource{}).Where("id = ?", o.ResourceID).Update("status", resource.StatusFree)
	tx.Commit()
	response.OKMsg(c, "已取消")
}

// MerchantDirectRefund 商家直接退款（已支付/进行中）
// POST /api/v1/merchant/orders/:id/refund
func (h *PaymentHandler) MerchantDirectRefund(c *gin.Context) {
	orderID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	storeID := GetStoreID(c)

	var o order.Order
	if h.ctx.DB.Where("id = ? AND store_id = ?", orderID, storeID).First(&o).Error != nil {
		response.FailCode(c, errcode.ErrOrderNotFound)
		return
	}
	if !canRefundSettle(o.Status, h.hasBalanceDebit(o)) {
		response.FailMsg(c, errcode.ErrBadRequest, "仅支持已确认余额支付的订单退款")
		return
	}
	tx := h.ctx.DB.Begin()
	if tx.Error != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	changed := tx.Model(&order.Order{}).Where("id = ? AND store_id = ? AND status IN ?", o.ID, storeID, []order.OrderStatus{order.StatusPaid, order.StatusInProgress, order.StatusRefunding}).Updates(map[string]interface{}{
		"status": order.StatusRefunded, "refund_amount": o.TotalAmount, "cancel_reason": "商家退款",
	})
	if changed.Error != nil || changed.RowsAffected != 1 {
		tx.Rollback()
		response.FailCode(c, errcode.ErrOrderStatusInvalid)
		return
	}
	if err := refundBalance(tx, h.ctx.BalanceSvc, o, o.TotalAmount, "商家退款"); err != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if err := tx.Model(&resource.Resource{}).Where("id = ? AND store_id = ?", o.ResourceID, storeID).Update("status", resource.StatusFree).Error; err != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if h.ctx.DeviceSvc != nil {
		go h.ctx.DeviceSvc.SendSessionCmd(o.ResourceID, "session_end", h.ctx.Cfg.Server.PublicURL)
	}
	response.OKMsg(c, "已退款并退回余额")
}

// MockPay 模拟支付成功（测试用，跳过微信支付）
// POST /api/v1/orders/:id/mock-pay
func (h *PaymentHandler) MockPay(c *gin.Context) {
	// 安全：模拟支付需显式开启 server.enable_mock（防止绕过真实收款）
	if !h.ctx.Cfg.Server.EnableMock {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
		return
	}
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	var o order.Order
	if h.ctx.DB.First(&o, orderID).Error != nil {
		response.FailCode(c, errcode.ErrOrderNotFound)
		return
	}
	if o.Status != order.StatusPendingPay {
		response.FailMsg(c, errcode.ErrOrderStatusInvalid, "仅待支付订单可模拟支付")
		return
	}
	if o.UserID != GetUserID(c) {
		response.FailCode(c, errcode.ErrPermissionDenied)
		return
	}

	now := time.Now().UnixMilli()
	tx := h.ctx.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 订单 → 已支付
	if err := tx.Model(&o).Updates(map[string]interface{}{
		"status":  order.StatusPaid,
		"paid_at": now,
	}).Error; err != nil {
		tx.Rollback()
		response.FailMsg(c, errcode.ErrInternal, "更新订单失败")
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.FailMsg(c, errcode.ErrInternal, "提交失败")
		return
	}

	// 自动下发 session_start（开门+开灯）
	go h.ctx.DeviceSvc.SendSessionCmd(o.ResourceID, "session_start", h.ctx.Cfg.Server.PublicURL)

	response.OKMsg(c, "模拟支付成功")
}

// GetPaymentStatus 查询支付状态 (供前端轮询)
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	orderID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var p payment.Payment
	if err := h.ctx.DB.Where("order_id = ?", orderID).Order("created_at DESC").First(&p).Error; err != nil {
		response.FailCode(c, errcode.ErrPaymentNotFound)
		return
	}

	var o order.Order
	h.ctx.DB.First(&o, p.OrderID)

	response.OK(c, gin.H{
		"payment_status": p.Status,
		"order_status":   o.Status,
		"transaction_id": p.TransactionID,
		"amount":         p.Amount,
	})
}

// createWechatOrder 调用微信统一下单
func (h *PaymentHandler) createWechatOrder(ctx context.Context, outTradeNo string, o order.Order) (string, error) {
	// 实际实现需要调用微信支付 API V3
	// 此处为占位 — 生产环境需：
	// 1. 构建请求体 (appid, mchid, description, out_trade_no, amount, notify_url, payer.openid)
	// 2. 签名 (RSA with merchant private key)
	// 3. POST https://api.mch.weixin.qq.com/v3/pay/transactions/jsapi
	_ = outTradeNo
	_ = o
	return fmt.Sprintf("prepay_%d", time.Now().UnixNano()), nil
}
