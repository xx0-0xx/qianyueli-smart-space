package router

import (
	"smart-space-management/backend/internal/handler"
	"smart-space-management/backend/internal/handler/admin"

	"github.com/gin-gonic/gin"
)

// AdminRouter 超级管理端路由
type AdminRouter struct {
	ctx *handler.Context
}

func NewAdminRouter(ctx *handler.Context) *AdminRouter {
	return &AdminRouter{ctx: ctx}
}

// RegisterPublic 公开接口（无需鉴权）
func (r *AdminRouter) RegisterPublic(group *gin.RouterGroup) {
	admH := admin.NewAdminHandler(r.ctx)
	group.POST("/login", admH.AdminLogin)
	group.POST("/login/web", admH.AdminLoginWeb)
}

func (r *AdminRouter) Register(group *gin.RouterGroup) {
	admH := admin.NewAdminHandler(r.ctx)

	// 门店管理
	group.POST("/stores", admH.CreateStore)
	group.PUT("/stores/:id", admH.UpdateStore)
	group.DELETE("/stores/:id", admH.DeleteStore)
	group.DELETE("/stores/:id/force", admH.ForceDeleteStore)
	group.GET("/stores", admH.ListAllStores)
	group.POST("/stores/:id/invite-code", admH.RegenerateInviteCode)
	group.GET("/stores/:id/qrcode", admH.StoreQRCode)
	group.PUT("/stores/:id/balance-group", admH.SetStoreBalanceGroup)

	// 余额组
	group.GET("/balance-groups", admH.ListBalanceGroups)
	group.POST("/balance-groups", admH.CreateBalanceGroup)

	// 商家账号管理
	group.POST("/merchants", admH.CreateMerchant)
	group.PUT("/merchants/:id", admH.UpdateMerchant)
	group.DELETE("/merchants/:id", admH.DeleteMerchant)
	group.GET("/merchants", admH.ListMerchants)

	// 资源模板
	group.POST("/resource-templates", admH.CreateResourceTemplate)
	group.PUT("/resource-templates/:id", admH.UpdateResourceTemplate)
	group.GET("/resource-templates", admH.ListResourceTemplates)

	// 订单审核 (退款审批)
	group.GET("/orders", admH.ListAllOrders)
	group.POST("/orders/:id/approve-refund", admH.ApproveRefund)

	// 数据统计
	group.GET("/statistics/overview", admH.GetOverviewStats)
	group.GET("/statistics/store/:id", admH.GetStoreStats)

	// 系统配置
	group.GET("/system-settings", admH.GetSystemSettings)
	group.PUT("/system-settings", admH.UpdateSystemSettings)

	// 语音播报配置
	group.GET("/voice-config", admH.GetVoiceConfig)
	group.PUT("/voice-config", admH.SaveVoiceConfig)

	// 营收播报（手动触发）
	group.POST("/reports/daily", admH.TriggerDailyReport)

	// 图片配置
	group.GET("/image-configs", admH.GetImageConfigs)
	group.PUT("/image-configs", admH.SaveImageConfigs)

	// 审计日志
	group.GET("/audit-logs", admH.ListAuditLogs)

	// 设备管理
	group.GET("/devices", admH.ListAllDevices)
	group.POST("/devices/probe", admH.ProbeDevice)
	group.POST("/devices", admH.CreateDeviceAdmin)
	group.PUT("/devices/:id", admH.UpdateDeviceAdmin)
	group.DELETE("/devices/:id", admH.DeleteDeviceAdmin)

	// 音频管理
	deviceH := handler.NewDeviceHandler(r.ctx)
	group.POST("/devices/:sn/speaker-test", deviceH.SpeakerTest)
	group.POST("/audio/upload", admH.UploadAudio)
	group.GET("/audio", admH.ListAudio)
	group.DELETE("/audio/:file", admH.DeleteAudio)

	// 产品模板
	group.GET("/product-templates", admH.ListProductTemplates)
	group.POST("/product-templates", admH.CreateProductTemplate)
	group.POST("/product-templates/:id/move", admH.MoveProductTemplate)
	group.PUT("/product-templates/:id", admH.UpdateProductTemplate)
	group.DELETE("/product-templates/:id", admH.DeleteProductTemplate)

	// 官网留资线索
	leadH := handler.NewLeadHandler(r.ctx)
	group.GET("/leads", leadH.ListDemoLeads)
	group.PUT("/leads/:id", leadH.UpdateDemoLead)

	// 团购核销（无限聚核）
	elysAdminH := handler.NewElysAdminHandler(r.ctx)
	group.GET("/elys/bindings", elysAdminH.ListBindings)
	group.POST("/elys/bindings", elysAdminH.UpsertBinding)
	group.DELETE("/elys/bindings/:store_id", elysAdminH.DeleteBinding)
	group.POST("/elys/bindings/:store_id/auth-url", elysAdminH.AuthURL)
	group.GET("/elys/verifications", elysAdminH.ListVerifications)
	group.POST("/elys/verifications/:id/retry-credit", elysAdminH.RetryCredit)
}
