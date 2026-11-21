// Package handler HTTP 控制器公共类型
package handler

import (
	"smart-space-management/backend/config"
	"smart-space-management/backend/internal/service"
	"smart-space-management/backend/internal/ws"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// MQTTPublisher 用于向设备发送 MQTT 指令
type MQTTPublisher interface {
	PublishCommand(deviceSN, cmd string, params map[string]interface{}) error
}

// Context 处理器共享依赖
type Context struct {
	DB    *gorm.DB
	Redis *redis.Client
	JWT   *utils.JWTManager
	Hub   *ws.Hub
	Cfg   *config.Config
	MQTT  MQTTPublisher

	// Service 层
	OrderSvc       *service.OrderService
	DeviceSvc      *service.DeviceService
	StoreSvc       *service.StoreService
	BalanceSvc     *service.BalanceService
	ElysBindingSvc *service.ElysBindingService
	ElysVoucherSvc *service.ElysVoucherService
}

// GetUserID 从 gin.Context 提取 user_id
func GetUserID(c interface {
	Get(string) (interface{}, bool)
}) uint64 {
	v, ok := c.Get("user_id")
	if !ok {
		return 0
	}
	id, _ := v.(uint64)
	return id
}

// GetStoreID 从 gin.Context 提取 ctx_store_id，fallback 到 JWT store_id
func GetStoreID(c *gin.Context) uint64 {
	if v, ok := c.Get("ctx_store_id"); ok {
		if id, ok2 := v.(uint64); ok2 && id > 0 {
			return id
		}
	}
	v2, _ := c.Get("store_id")
	id, _ := v2.(uint64)
	return id
}
