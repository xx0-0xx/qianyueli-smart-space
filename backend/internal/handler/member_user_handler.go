package handler

import (
	"strconv"
	"time"

	"smart-space-management/backend/internal/model/resource"
	"smart-space-management/backend/internal/service"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func (h *MemberHandler) GetMyMembership(c *gin.Context) {
	storeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || storeID == 0 {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	userID := GetUserID(c)
	now := time.Now().UnixMilli()
	active, err := service.FindActiveMember(h.ctx.DB, storeID, userID, now)
	if err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if active == nil {
		response.OK(c, gin.H{"is_member": false})
		return
	}
	response.OK(c, gin.H{
		"is_member":   true,
		"level_id":    active.Level.ID,
		"level_name":  active.Level.Name,
		"discount_bp": active.Level.EffectiveDiscountBP(),
		"expires_at":  active.Member.ExpiresAt,
	})
}

func (h *MemberHandler) GetMyPrice(c *gin.Context) {
	storeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	typeID, typeErr := strconv.ParseUint(c.Query("resource_type_id"), 10, 64)
	if err != nil || typeErr != nil || storeID == 0 || typeID == 0 {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	var typeCount int64
	if err := h.ctx.DB.Model(&resource.ResourceType{}).
		Where("id = ? AND (store_id = ? OR store_id = 0)", typeID, storeID).
		Count(&typeCount).Error; err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if typeCount == 0 {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	original := service.BasePrice(h.ctx.DB, storeID, typeID)
	now := time.Now().UnixMilli()
	active, err := service.FindActiveMember(h.ctx.DB, storeID, GetUserID(c), now)
	if err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	memberPrice := original
	discountBP := 10000
	levelName := ""
	if active != nil {
		discountBP = active.Level.EffectiveDiscountBP()
		memberPrice, _ = service.ApplyMemberDiscount(original, discountBP)
		levelName = active.Level.Name
	}
	response.OK(c, gin.H{
		"original_price": original,
		"member_price":   memberPrice,
		"discount_bp":    discountBP,
		"level_name":     levelName,
		"is_member":      active != nil,
	})
}
