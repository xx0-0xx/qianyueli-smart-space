package middleware

import (
	"strconv"

	"smart-space-management/backend/internal/model/user"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Tenant 多店数据隔离中间件
// 从请求头 X-Store-ID 或 JWT claims 中获取当前操作的门店 ID
// 商家：可切换到「自己名下」的任意门店；未指定则用主门店
func Tenant(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("user_role")
		userStoreID, _ := c.Get("store_id")
		userID, _ := c.Get("user_id")

		// 超级管理员不受门店限制
		if role == "admin" || role == "super_admin" {
			if sid := c.GetHeader("X-Store-ID"); sid != "" {
				if id, err := strconv.ParseUint(sid, 10, 64); err == nil {
					c.Set("ctx_store_id", id)
				}
			}
			c.Next()
			return
		}

		// 商家：可切换到自己名下的门店
		if role == "merchant" {
			// 1. 若 header 指定了门店，校验是否在商家名下
			if sid := c.GetHeader("X-Store-ID"); sid != "" {
				if id, err := strconv.ParseUint(sid, 10, 64); err == nil && id > 0 {
					uid, _ := userID.(uint64)
					var cnt int64
					db.Model(&user.MerchantStore{}).
						Where("merchant_id = ? AND store_id = ?", uid, id).Count(&cnt)
					if cnt > 0 {
						c.Set("ctx_store_id", id)
						c.Next()
						return
					}
				}
			}
			// 2. 默认使用主门店（JWT 中的 store_id）
			if userStoreID == nil || userStoreID.(uint64) == 0 {
				response.FailCode(c, errcode.ErrPermissionDenied)
				return
			}
			c.Set("ctx_store_id", userStoreID.(uint64))
			c.Next()
			return
		}

		// 普通用户：可以从 header 指定门店
		if sid := c.GetHeader("X-Store-ID"); sid != "" {
			if id, err := strconv.ParseUint(sid, 10, 64); err == nil {
				c.Set("ctx_store_id", id)
			}
		}
		c.Next()
	}
}
