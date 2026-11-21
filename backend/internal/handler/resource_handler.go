package handler

import (
	"strconv"

	"smart-space-management/backend/internal/model/resource"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type ResourceHandler struct {
	ctx *Context
}

func NewResourceHandler(ctx *Context) *ResourceHandler {
	return &ResourceHandler{ctx: ctx}
}

// ListStoreResources 商家查看本店资源列表
func (h *ResourceHandler) ListStoreResources(c *gin.Context) {
	storeID := GetStoreID(c)
	var resources []resource.Resource
	h.ctx.DB.Where("store_id = ?", storeID).Preload("ResourceType").Order("sort ASC").Find(&resources)
	response.OK(c, resources)
}

// CreateResource 商家创建资源
func (h *ResourceHandler) CreateResource(c *gin.Context) {
	storeID := GetStoreID(c)
	var req struct {
		StoreID        uint64 `json:"store_id"`
		ResourceTypeID uint64 `json:"resource_type_id" binding:"required"`
		Name           string `json:"name" binding:"required"`
		Image          string `json:"image"`
		DeviceIDs      string `json:"device_ids"`
		Sort           int    `json:"sort"`
		Remark         string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	if storeID == 0 && req.StoreID > 0 { storeID = req.StoreID }

	r := resource.Resource{
		StoreID:        storeID,
		ResourceTypeID: req.ResourceTypeID,
		Name:           req.Name,
		Image:          req.Image,
		DeviceIDs:      req.DeviceIDs,
		Sort:           req.Sort,
		Remark:         req.Remark,
		Status:         resource.StatusFree,
	}
	r.ID = utils.ShortID()

	if err := h.ctx.DB.Create(&r).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}

	response.Created(c, r)
}

// UpdateResource 商家更新资源
func (h *ResourceHandler) UpdateResource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	storeID := GetStoreID(c)

	var r resource.Resource
	if h.ctx.DB.Where("id = ? AND store_id = ?", id, storeID).First(&r).Error != nil {
		response.FailCode(c, errcode.ErrResourceNotFound)
		return
	}

	var req struct {
		Name      string `json:"name"`
		Image     string `json:"image"`
		DeviceIDs string `json:"device_ids"`
		Sort      *int   `json:"sort"`
		Remark    string `json:"remark"`
		Status    *int   `json:"status"` // 可设为维护中
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Image != "" {
		updates["image"] = req.Image
	}
	if req.DeviceIDs != "" {
		updates["device_ids"] = req.DeviceIDs
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	h.ctx.DB.Model(&r).Updates(updates)
	response.OKMsg(c, "updated")
}

// ── 房型（资源类型）管理 ──

// ListResourceTypes 商家查看本店房型
func (h *ResourceHandler) ListResourceTypes(c *gin.Context) {
	storeID := GetStoreID(c)
	// Admin 可通过 query param 指定门店
	if storeID == 0 {
		if sid := c.Query("store_id"); sid != "" {
			if id, err := strconv.ParseUint(sid, 10, 64); err == nil { storeID = id }
		}
	}
	var types []resource.ResourceType
	if storeID == 0 { h.ctx.DB.Order("sort ASC").Find(&types) } else { h.ctx.DB.Where("store_id = ?", storeID).Order("sort ASC").Find(&types) }
	response.OK(c, types)
}

// CreateResourceType 商家新增房型
func (h *ResourceHandler) CreateResourceType(c *gin.Context) {
	storeID := GetStoreID(c)
	var req struct {
		Name       string `json:"name" binding:"required"`
		StoreID    uint64 `json:"store_id"`
		Description string `json:"description"`
		ConfigJSON string `json:"config_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	sid := storeID
	if sid == 0 && req.StoreID > 0 { sid = req.StoreID }
	rt := resource.ResourceType{
		Name:       req.Name,
		Description: req.Description,
		ConfigJSON: req.ConfigJSON,
		StoreID:    sid,
	}
	rt.ID = utils.ShortID()
	h.ctx.DB.Create(&rt)
	response.Created(c, rt)
}

// DeleteResourceType 商家删除房型
func (h *ResourceHandler) DeleteResourceType(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	storeID := GetStoreID(c)
	q := h.ctx.DB
	if storeID > 0 { q = q.Where("store_id = ?", storeID) }
	result := q.Where("id = ?", id).Delete(&resource.ResourceType{})
	if result.RowsAffected == 0 {
		response.FailCode(c, errcode.ErrNotFound)
		return
	}
	response.OKMsg(c, "已删除")
}

// LinkDevices 关联/取消关联设备到资源
// PUT /merchant/resources/:id/devices
func (h *ResourceHandler) LinkDevices(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	storeID := GetStoreID(c)

	var r resource.Resource
	q := h.ctx.DB
	if storeID > 0 { q = q.Where("store_id = ?", storeID) }
	if q.Where("id = ?", id).First(&r).Error != nil {
		response.FailCode(c, errcode.ErrResourceNotFound)
		return
	}

	var req struct {
		DeviceIDs string `json:"device_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	h.ctx.DB.Model(&r).Update("device_ids", req.DeviceIDs)

	response.OK(c, gin.H{"device_ids": req.DeviceIDs})
}

// DeleteResource 商家删除资源 (软删除)
func (h *ResourceHandler) DeleteResource(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	storeID := GetStoreID(c)

	q := h.ctx.DB
	if storeID > 0 { q = q.Where("store_id = ?", storeID) }
	result := q.Where("id = ?", id).Delete(&resource.Resource{})
	if result.RowsAffected == 0 {
		response.FailCode(c, errcode.ErrResourceNotFound)
		return
	}
	response.OKMsg(c, "deleted")
}
