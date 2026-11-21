package middleware

import (
	"log"
	"strings"
	"time"

	"smart-space-management/backend/internal/model/audit"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// auditableMethods 需要审计的 HTTP 方法
var auditableMethods = map[string]bool{
	"POST":   true,
	"PUT":    true,
	"DELETE": true,
	"PATCH":  true,
}

// auditablePaths 需要审计的路径前缀（仅记录管理端/商家端写操作，覆盖退款、取消、充值确认、券码、计价等敏感变更；
// 高频的设备控制走 /api/v1/devices/:id/control，不在这些前缀内，避免淹没审计表）
var auditablePaths = []string{
	"/api/v1/admin",
	"/api/v1/merchant",
	"/api/v1/config",
}

const auditQueueSize = 2048

// Audit 敏感操作审计中间件
// 仅审计白名单路径的写操作；通过有界 channel + 单 worker 异步落库，避免每请求起 goroutine 且高并发不受控。
func Audit(db *gorm.DB) gin.HandlerFunc {
	queue := make(chan audit.AuditLog, auditQueueSize)
	go func() {
		for l := range queue {
			if err := db.Create(&l).Error; err != nil {
				log.Printf("[AUDIT] 审计日志写入失败: action=%s user=%d err=%v", l.Action, l.UserID, err)
			}
		}
	}()

	return func(c *gin.Context) {
		c.Next() // 先执行业务，再记录

		if !auditableMethods[c.Request.Method] {
			return
		}
		if !isAuditablePath(c.FullPath()) {
			return
		}
		role, _ := c.Get("user_role")
		if role != "admin" && role != "super_admin" && role != "merchant" {
			return
		}

		userID, _ := c.Get("user_id")
		storeID, _ := c.Get("ctx_store_id")

		record := audit.AuditLog{
			UserID:    toUint64(userID),
			StoreID:   toUint64(storeID),
			Action:    c.Request.Method + " " + c.FullPath(),
			IP:        c.ClientIP(),
			UserAgent: c.GetHeader("User-Agent"),
		}
		record.ID = utils.ShortID()
		record.CreatedAt = time.Now()
		record.UpdatedAt = time.Now()

		select {
		case queue <- record:
		default:
			log.Printf("[AUDIT] 队列已满，丢弃审计记录: %s", record.Action)
		}
	}
}

func isAuditablePath(path string) bool {
	if path == "" {
		return false
	}
	for _, prefix := range auditablePaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func toUint64(v interface{}) uint64 {
	switch val := v.(type) {
	case uint64:
		return val
	case int64:
		return uint64(val)
	default:
		return 0
	}
}
