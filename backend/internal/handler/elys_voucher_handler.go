package handler

import (
	"strconv"

	userModel "smart-space-management/backend/internal/model/user"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type ElysVoucherHandler struct {
	ctx *Context
}

func NewElysVoucherHandler(ctx *Context) *ElysVoucherHandler {
	return &ElysVoucherHandler{ctx: ctx}
}

func (h *ElysVoucherHandler) Config(c *gin.Context) {
	storeID, _ := strconv.ParseUint(c.Query("store_id"), 10, 64)
	if storeID == 0 {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	binding, err := h.ctx.ElysBindingSvc.Get(storeID)
	if err != nil {
		response.OK(c, gin.H{"enabled": false, "platforms": []any{}})
		return
	}
	response.OK(c, gin.H{
		"enabled": binding.Status == 1,
		"platforms": []gin.H{
			{"platform": "meituan", "label": "美团", "authorized": binding.MeituanAuthorized},
			{"platform": "douyin", "label": "抖音", "authorized": binding.DouyinAuthorized},
		},
	})
}

func (h *ElysVoucherHandler) Prepare(c *gin.Context) {
	role, _ := c.Get("user_role")
	if role != string(userModel.RoleUser) {
		response.FailCode(c, errcode.ErrPermissionDenied)
		return
	}
	var req struct {
		StoreID  uint64 `json:"store_id"`
		Platform string `json:"platform"`
		Code     string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.StoreID == 0 || req.Code == "" {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	view, err := h.ctx.ElysVoucherSvc.Prepare(c.Request.Context(), GetUserID(c), req.StoreID, req.Platform, req.Code)
	if err != nil {
		failFromErr(c, err)
		return
	}
	response.OK(c, view)
}

func (h *ElysVoucherHandler) Consume(c *gin.Context) {
	role, _ := c.Get("user_role")
	if role != string(userModel.RoleUser) {
		response.FailCode(c, errcode.ErrPermissionDenied)
		return
	}
	var req struct {
		PrepareToken string `json:"prepare_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PrepareToken == "" {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	view, err := h.ctx.ElysVoucherSvc.Consume(c.Request.Context(), GetUserID(c), req.PrepareToken)
	if err != nil {
		failFromErr(c, err)
		return
	}
	response.OK(c, view)
}

func (h *ElysVoucherHandler) History(c *gin.Context) {
	rows, err := h.ctx.ElysVoucherSvc.ListUserVerifications(GetUserID(c), 1, 20)
	if err != nil {
		failFromErr(c, err)
		return
	}
	response.OK(c, rows)
}
