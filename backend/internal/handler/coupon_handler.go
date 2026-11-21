package handler

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"smart-space-management/backend/internal/model/order"
	"smart-space-management/backend/internal/model/payment"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type CouponHandler struct {
	ctx *Context
}

func NewCouponHandler(ctx *Context) *CouponHandler { return &CouponHandler{ctx: ctx} }

// CreateCoupon 商家创建券码（支持批量）
// POST /api/v1/merchant/coupons
func (h *CouponHandler) CreateCoupon(c *gin.Context) {
	storeID := GetStoreID(c)
	var req struct {
		ResourceTypeID uint64 `json:"resource_type_id"`
		Name           string `json:"name"`
		Hours          int    `json:"hours"`
		Price          int    `json:"price"`
		Type           int    `json:"type"`
		Discount       int    `json:"discount"`
		Count          int    `json:"count"` // 批量生成数量，默认1，上限100
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest); return
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 100 {
		req.Count = 100
	}

	var created []payment.Coupon
	for i := 0; i < req.Count; i++ {
		code := fmt.Sprintf("%06d%06d", time.Now().UnixMilli()%1000000, rand.Int63n(1000000))
		coupon := payment.Coupon{
			ResourceTypeID: req.ResourceTypeID, Code: code,
			Name: req.Name, Hours: req.Hours, Price: req.Price, StoreID: storeID,
			Type: req.Type, Discount: req.Discount,
		}
		coupon.ID = utils.ShortID()
		if err := h.ctx.DB.Create(&coupon).Error; err != nil {
			continue // 券码冲突则跳过
		}
		created = append(created, coupon)
	}

	if len(created) == 0 {
		response.FailMsg(c, errcode.ErrConflict, "生成失败"); return
	}
	response.Created(c, gin.H{"count": len(created), "codes": created})
}

// ListCoupons 商家查看本店团购券
// GET /api/v1/merchant/coupons
func (h *CouponHandler) ListCoupons(c *gin.Context) {
	storeID := GetStoreID(c)
	var list []payment.Coupon
	h.ctx.DB.Where("store_id = ?", storeID).Order("created_at DESC").Find(&list)
	response.OK(c, list)
}

// DeleteCoupon 商家删除券码
// DELETE /api/v1/merchant/coupons/:id
func (h *CouponHandler) DeleteCoupon(c *gin.Context) {
	storeID := GetStoreID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest); return
	}
	h.ctx.DB.Where("id = ? AND store_id = ?", id, storeID).Delete(&payment.Coupon{})
	response.OKMsg(c, "已删除")
}

// RedeemCoupon 用户核销代金券 — 券面额直接充入余额
// POST /api/v1/coupons/redeem
func (h *CouponHandler) RedeemCoupon(c *gin.Context) {
	var req struct{ Code string `json:"code"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest); return
	}
	userID := GetUserID(c)
	if userID == 0 {
		response.FailCode(c, errcode.ErrUnauthorized); return
	}

	var cp payment.Coupon
	if h.ctx.DB.Where("code = ? AND status = 0", req.Code).First(&cp).Error != nil {
		response.FailMsg(c, errcode.ErrNotFound, "券码无效或已使用"); return
	}

	// 券面额（代金券取 discount，兜底 price）
	amount := cp.Discount
	if amount <= 0 {
		amount = cp.Price
	}
	if amount <= 0 {
		response.FailMsg(c, errcode.ErrBadRequest, "券面额无效"); return
	}

	// 事务：加余额 + 流水 + 标记券已用（按门店所属余额组）
	tx := h.ctx.DB.Begin()
	newBal, gid, cErr := h.ctx.BalanceSvc.Credit(tx, userID, cp.StoreID, int64(amount))
	if cErr != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal); return
	}
	now := time.Now()
	log := payment.BalanceLog{
		UserID: userID, Amount: int64(amount), Balance: newBal,
		Type: "coupon", RefID: cp.Code, Desc: "券码核销充值",
		StoreID: cp.StoreID, GroupID: gid,
	}
	log.ID = utils.ShortID()
	tx.Create(&log)
	tx.Model(&cp).Updates(map[string]interface{}{"status": 1, "used_at": now, "user_id": userID})
	tx.Commit()

	response.OK(c, gin.H{
		"amount":  amount,
		"balance": newBal,
		"name":    cp.Name,
	})
}

// ApplyVoucher 使用代金券抵扣订单金额
// POST /api/v1/coupons/apply-voucher
func (h *CouponHandler) ApplyVoucher(c *gin.Context) {
	var req struct {
		Code    string `json:"code"`
		OrderID uint64 `json:"order_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest); return
	}
	userID := GetUserID(c)

	var cp payment.Coupon
	if h.ctx.DB.Where("code = ? AND type = ? AND status = 0", req.Code, 1).First(&cp).Error != nil {
		response.FailMsg(c, errcode.ErrNotFound, "代金券无效或已使用"); return
	}

	var o order.Order
	if h.ctx.DB.Where("id = ? AND user_id = ? AND status = 0", req.OrderID, userID).First(&o).Error != nil {
		response.FailMsg(c, errcode.ErrOrderStatusInvalid, "订单不存在或不可用券"); return
	}

	discount := cp.Discount
	if discount == 0 { discount = cp.Price }
	newTotal := o.TotalAmount - discount
	if newTotal < 0 { newTotal = 0 }

	now := time.Now()
	tx := h.ctx.DB.Begin()
	tx.Model(&o).Update("total_amount", newTotal)
	tx.Model(&cp).Updates(map[string]interface{}{"status": 1, "used_at": now, "user_id": userID})
	tx.Commit()

	response.OK(c, gin.H{"original": o.TotalAmount, "discount": discount, "final": newTotal})
}

// GetCouponInfo 查询券信息（核销前预览面额）
// GET /api/v1/coupons/:code/info
func (h *CouponHandler) GetCouponInfo(c *gin.Context) {
	code := c.Param("code")
	var cp payment.Coupon
	if h.ctx.DB.Where("code = ? AND status = 0", code).First(&cp).Error != nil {
		response.FailMsg(c, errcode.ErrNotFound, "券码无效或已使用"); return
	}
	amount := cp.Discount
	if amount <= 0 {
		amount = cp.Price
	}
	response.OK(c, gin.H{
		"code":   cp.Code,
		"name":   cp.Name,
		"amount": amount,
	})
}
