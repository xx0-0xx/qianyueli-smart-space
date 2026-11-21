package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"smart-space-management/backend/internal/model/payment"
	"smart-space-management/backend/internal/model/user"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// RechargeHandler 充值管理
type RechargeHandler struct {
	ctx *Context
}

func NewRechargeHandler(ctx *Context) *RechargeHandler { return &RechargeHandler{ctx: ctx} }

// CreateRecharge 用户发起充值（线下收款，待商家确认）
// POST /api/v1/recharges
func (h *RechargeHandler) CreateRecharge(c *gin.Context) {
	userID := GetUserID(c)
	var u user.User
	if h.ctx.DB.First(&u, userID).Error != nil {
		response.FailCode(c, errcode.ErrUserNotFound)
		return
	}
	if strings.TrimSpace(u.Phone) == "" {
		response.FailCode(c, errcode.ErrPhoneRequired)
		return
	}
	var req struct {
		StoreID uint64 `json:"store_id" binding:"required"`
		Amount  int64  `json:"amount" binding:"required"`
		Note    string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	if req.Amount < 100 {
		response.FailMsg(c, errcode.ErrBadRequest, "充值金额至少1元")
		return
	}
	r := payment.Recharge{
		UserID: userID, StoreID: req.StoreID, Amount: req.Amount,
		Note: req.Note, Status: payment.RechargePending,
	}
	r.ID = utils.ShortID()
	h.ctx.DB.Create(&r)
	response.Created(c, r)
}

// ListMyRecharges 用户查看自己的充值记录
// GET /api/v1/recharges
func (h *RechargeHandler) ListMyRecharges(c *gin.Context) {
	userID := GetUserID(c)
	var list []payment.Recharge
	h.ctx.DB.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&list)
	response.OK(c, list)
}

// MyBalance 用户余额 + 流水（按门店所属余额组）
// GET /api/v1/balance?store_id=
func (h *RechargeHandler) MyBalance(c *gin.Context) {
	userID := GetUserID(c)
	storeID, _ := strconv.ParseUint(c.Query("store_id"), 10, 64)
	if storeID == 0 {
		response.FailMsg(c, errcode.ErrBadRequest, "缺少 store_id")
		return
	}
	bal, gid, err := h.ctx.BalanceSvc.BalanceOf(userID, storeID)
	if err != nil {
		response.FailMsg(c, errcode.ErrBadRequest, err.Error())
		return
	}
	var logs []payment.BalanceLog
	h.ctx.DB.Where("user_id = ? AND group_id = ?", userID, gid).
		Order("created_at DESC").Limit(30).Find(&logs)
	response.OK(c, gin.H{"balance": bal, "logs": logs, "group_id": gid})
}

// ListStoreRecharges 商家查看本店待确认充值单
// GET /api/v1/merchant/recharges?status=0
func (h *RechargeHandler) ListStoreRecharges(c *gin.Context) {
	storeID := GetStoreID(c)
	status := c.DefaultQuery("status", "0")
	var list []payment.Recharge
	h.ctx.DB.Where("store_id = ? AND status = ?", storeID, status).Order("created_at DESC").Limit(50).Find(&list)

	// 附加用户昵称
	type row struct {
		payment.Recharge
		Nickname string `json:"nickname"`
	}
	var rows []row
	for _, r := range list {
		var u user.User
		nick := ""
		if h.ctx.DB.First(&u, r.UserID).Error == nil {
			nick = u.Nickname
		}
		rows = append(rows, row{Recharge: r, Nickname: nick})
	}
	response.OK(c, rows)
}

// ConfirmRecharge 商家确认充值到账
// POST /api/v1/merchant/recharges/:id/confirm
func (h *RechargeHandler) ConfirmRecharge(c *gin.Context) {
	storeID := GetStoreID(c)
	merchantID := GetUserID(c)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var r payment.Recharge
	if h.ctx.DB.Where("id = ? AND store_id = ?", id, storeID).First(&r).Error != nil {
		response.FailMsg(c, errcode.ErrNotFound, "充值单不存在或已处理")
		return
	}

	tx := h.ctx.DB.Begin()
	if tx.Error != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	now := time.Now()
	claimed := tx.Model(&payment.Recharge{}).Where("id = ? AND store_id = ? AND status = ?", id, storeID, payment.RechargePending).Updates(map[string]interface{}{
		"status": payment.RechargeDone, "confirm_at": now, "confirm_by": merchantID,
	})
	if claimed.Error != nil || claimed.RowsAffected != 1 {
		tx.Rollback()
		response.FailMsg(c, errcode.ErrNotFound, "充值单不存在或已处理")
		return
	}
	newBal, gid, cErr := h.ctx.BalanceSvc.Credit(tx, r.UserID, r.StoreID, r.Amount)
	if cErr != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	log := payment.BalanceLog{
		UserID: r.UserID, Amount: r.Amount, Balance: newBal,
		Type: "recharge", RefID: fmt.Sprintf("%d", r.ID), Desc: "充值到账",
		StoreID: r.StoreID, GroupID: gid,
	}
	log.ID = utils.ShortID()
	if err := tx.Create(&log).Error; err != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	response.OKMsg(c, "已到账")
}

// CancelRecharge 商家取消充值单
// POST /api/v1/merchant/recharges/:id/cancel
func (h *RechargeHandler) CancelRecharge(c *gin.Context) {
	storeID := GetStoreID(c)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	h.ctx.DB.Model(&payment.Recharge{}).
		Where("id = ? AND store_id = ? AND status = ?", id, storeID, payment.RechargePending).
		Update("status", payment.RechargeCancel)
	response.OKMsg(c, "已取消")
}
