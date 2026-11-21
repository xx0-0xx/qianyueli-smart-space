package handler

import (
	"strconv"

	"smart-space-management/backend/internal/service"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type ElysAdminHandler struct {
	ctx *Context
}

func NewElysAdminHandler(ctx *Context) *ElysAdminHandler {
	return &ElysAdminHandler{ctx: ctx}
}

func (h *ElysAdminHandler) ListBindings(c *gin.Context) {
	list, err := h.ctx.ElysBindingSvc.List()
	if err != nil {
		failFromErr(c, err)
		return
	}
	response.OK(c, list)
}

func (h *ElysAdminHandler) UpsertBinding(c *gin.Context) {
	var req struct {
		StoreID          uint64 `json:"store_id"`
		ElysShopID       uint64 `json:"elys_shop_id"`
		ElysMasterShopID uint64 `json:"elys_master_shop_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	if err := h.ctx.ElysBindingSvc.Upsert(req.StoreID, req.ElysShopID, req.ElysMasterShopID); err != nil {
		failFromErr(c, err)
		return
	}
	response.OKMsg(c, "已保存")
}

func (h *ElysAdminHandler) DeleteBinding(c *gin.Context) {
	storeID, _ := strconv.ParseUint(c.Param("store_id"), 10, 64)
	if storeID == 0 {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	if err := h.ctx.ElysBindingSvc.Delete(storeID); err != nil {
		failFromErr(c, err)
		return
	}
	response.OKMsg(c, "已删除")
}

func (h *ElysAdminHandler) AuthURL(c *gin.Context) {
	storeID, _ := strconv.ParseUint(c.Param("store_id"), 10, 64)
	var req struct {
		Platform string `json:"platform"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	platform := 0
	if req.Platform == "meituan" {
		platform = 1
	}
	if req.Platform == "douyin" {
		platform = 2
	}
	if storeID == 0 || platform == 0 {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	url, err := h.ctx.ElysBindingSvc.AuthURL(c.Request.Context(), storeID, platform)
	if err != nil {
		failFromErr(c, err)
		return
	}
	response.OK(c, gin.H{"url": url, "platform": req.Platform})
}

func (h *ElysAdminHandler) ListVerifications(c *gin.Context) {
	storeID, _ := strconv.ParseUint(c.Query("store_id"), 10, 64)
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("page_size"))
	status, _ := strconv.Atoi(c.Query("status"))
	f := service.ElysVerificationFilter{
		StoreID: storeID, Platform: c.Query("platform"), Status: status,
		Page: page, PageSize: size,
	}
	rows, total, err := h.ctx.ElysVoucherSvc.ListVerifications(f)
	if err != nil {
		failFromErr(c, err)
		return
	}
	response.OK(c, gin.H{"list": rows, "total": total, "page": f.Page, "page_size": f.PageSize})
}

func (h *ElysAdminHandler) RetryCredit(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if id == 0 {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	if err := h.ctx.ElysVoucherSvc.RetryCreditByID(c.Request.Context(), id); err != nil {
		failFromErr(c, err)
		return
	}
	response.OKMsg(c, "已重试入账")
}
