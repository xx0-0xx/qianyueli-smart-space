package router

import (
	"smart-space-management/backend/internal/handler"
	"smart-space-management/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// APIRouter 用户 + 商家 API 路由
type APIRouter struct {
	ctx *handler.Context
}

func NewAPIRouter(ctx *handler.Context) *APIRouter {
	return &APIRouter{ctx: ctx}
}

func (r *APIRouter) Register(group *gin.RouterGroup) {
	// ── 公开接口 (无需登录) ──
	{
		userH := handler.NewUserHandler(r.ctx)
		group.POST("/login/wechat", userH.WechatLogin)
		group.POST("/login", userH.MockLogin) // 模拟登录（测试用）
		group.POST("/refresh", userH.RefreshToken)
		// 图片配置（公开，小程序无需登录即可获取）
		group.GET("/config/images", handler.NewConfigHandler(r.ctx).GetImageConfigsPublic)
		storeH := handler.NewStoreHandler(r.ctx)
		group.GET("/stores", storeH.ListStores)
		group.GET("/stores/:id", storeH.GetStore)
		group.GET("/stores/:id/resources", storeH.GetStoreResources)
		group.GET("/stores/:id/slots", storeH.GetTimeSlots)
		group.GET("/stores/:id/slots/resources", storeH.GetSlotResources)

		// 音频文件（公开）
		audioH := handler.NewAudioHandler(r.ctx)
		group.GET("/stores/:id/price", storeH.GetStorePrice)

		// 平台回调（美团/抖音）
		platH := handler.NewPlatformHandler(r.ctx)
		group.POST("/callback/meituan/verify", platH.MeituanVerify)
		group.POST("/callback/douyin/verify", platH.DouyinVerify)

		// 商家邀请码登录（Web 端，无需微信授权）
		group.POST("/merchant/invite-login", userH.InviteLogin)
		group.POST("/merchant/invite-login/web", userH.InviteLoginWeb)

		// 官网「预约演示」留资（公开，免登录）
		group.POST("/demo-lead", handler.NewLeadHandler(r.ctx).CreateDemoLead)

		// 无限聚核授权 Webhook（公开，HMAC 验签）
		group.POST("/webhook/elys", handler.NewElysWebhookHandler(r.ctx).Handle)

		group.GET("/audios", audioH.ListAudioFiles)
		group.Static("/audio", "./audio")
	}

	// ── 需登录接口 ──
	auth := group.Group("")
	auth.Use(middleware.Auth(r.ctx.JWT), middleware.Tenant(r.ctx.DB))
	{
		userH := handler.NewUserHandler(r.ctx)
		auth.GET("/user/info", userH.GetUserInfo)
		auth.PUT("/user/info", userH.UpdateUserInfo)
		auth.POST("/user/wechat-phone", userH.BindWechatPhone)

		orderH := handler.NewOrderHandler(r.ctx)
		auth.POST("/orders", orderH.CreateOrder)
		auth.GET("/orders", orderH.ListMyOrders)
		auth.GET("/orders/:id", orderH.GetOrder)
		auth.POST("/orders/:id/cancel", orderH.CancelOrder)

		storeH := handler.NewStoreHandler(r.ctx)
		auth.GET("/stores/:id/occupied", storeH.GetStoreOccupied)

		// 用户会员身份与会员价
		memberH := handler.NewMemberHandler(r.ctx)
		auth.GET("/stores/:id/my-membership", memberH.GetMyMembership)
		auth.GET("/stores/:id/my-price", memberH.GetMyPrice)

		payH := handler.NewPaymentHandler(r.ctx)
		// 微信支付 V3 尚未实现（验签/回调均为占位），故不注册在线支付路由，避免暴露半成品接口。
		// 接入真实 V3 后再放开：auth.POST("/orders/:id/pay", payH.CreatePayment)
		auth.POST("/orders/:id/mock-pay", payH.MockPay)
		auth.POST("/orders/:id/refund-request", payH.RequestRefund)

		// 券码（核销充值余额）
		couponH := handler.NewCouponHandler(r.ctx)
		auth.GET("/coupons/:code/info", couponH.GetCouponInfo)
		auth.POST("/coupons/redeem", couponH.RedeemCoupon)

		elysVoucherH := handler.NewElysVoucherHandler(r.ctx)
		auth.GET("/user/elys-vouchers/config", elysVoucherH.Config)
		auth.POST("/user/elys-vouchers/prepare", elysVoucherH.Prepare)
		auth.POST("/user/elys-vouchers/consume", elysVoucherH.Consume)
		auth.GET("/user/elys-vouchers", elysVoucherH.History)

		// 充值（用户侧）
		rechargeH := handler.NewRechargeHandler(r.ctx)
		auth.POST("/recharges", rechargeH.CreateRecharge)
		auth.GET("/recharges", rechargeH.ListMyRecharges)
		auth.GET("/balance", rechargeH.MyBalance)

		deviceH := handler.NewDeviceHandler(r.ctx)
		auth.POST("/devices/:id/control", deviceH.ControlDevice)
		auth.GET("/devices/:id/status", deviceH.GetDeviceStatus)

		configH := handler.NewConfigHandler(r.ctx)
		auth.GET("/config/ui", configH.GetUIConfig)

		// 商家绑定门店 — 需要登录但不限角色（新用户还不是merchant）
		auth.POST("/merchant/bind-store", userH.BindStore)
	}

	// ── 商家接口 ──
	merchant := group.Group("/merchant")
	merchant.Use(middleware.Auth(r.ctx.JWT), middleware.MerchantOnly(), middleware.Tenant(r.ctx.DB))
	{
		userH := handler.NewUserHandler(r.ctx)
		merchant.GET("/stores", userH.ListMerchantStores)
		merchant.POST("/stores/bind", userH.BindMerchantStore)

		// 成品控制器语音播报
		merchant.GET("/voice-config", userH.GetMerchantVoiceConfig)
		merchant.PUT("/voice-config", userH.SaveMerchantVoiceConfig)
		merchant.GET("/voice-devices", userH.ListMerchantVoiceDevices)

		resH := handler.NewResourceHandler(r.ctx)
		merchant.GET("/resources", resH.ListStoreResources)
		merchant.POST("/resources", resH.CreateResource)
		merchant.PUT("/resources/:id", resH.UpdateResource)
		merchant.PUT("/resources/:id/devices", resH.LinkDevices)
		merchant.DELETE("/resources/:id", resH.DeleteResource)

		// 房型管理
		merchant.GET("/resource-types", resH.ListResourceTypes)
		merchant.POST("/resource-types", resH.CreateResourceType)
		merchant.DELETE("/resource-types/:id", resH.DeleteResourceType)

		orderH := handler.NewOrderHandler(r.ctx)
		merchant.GET("/orders", orderH.ListStoreOrders)

		configH := handler.NewConfigHandler(r.ctx)
		merchant.GET("/config/price-rules", configH.ListPriceRules)
		merchant.POST("/config/price-rules", configH.CreatePriceRule)
		merchant.POST("/config/price-rules/bulk", configH.BulkReplacePriceRules)
		merchant.PUT("/config/price-rules/:id", configH.UpdatePriceRule)
		merchant.POST("/config/booking-rules", configH.CreateBookingRule)
		merchant.PUT("/config/booking-rules/:id", configH.UpdateBookingRule)

		// 充值（商家确认到账）
		rechargeH := handler.NewRechargeHandler(r.ctx)
		merchant.GET("/recharges", rechargeH.ListStoreRecharges)
		merchant.POST("/recharges/:id/confirm", rechargeH.ConfirmRecharge)
		merchant.POST("/recharges/:id/cancel", rechargeH.CancelRecharge)

		deviceH := handler.NewDeviceHandler(r.ctx)
		merchant.GET("/devices", deviceH.ListMerchantDevices)
		merchant.POST("/devices", deviceH.CreateDevice)
		merchant.GET("/product-templates", deviceH.ListProductTemplates)
		merchant.PUT("/devices/:id", deviceH.UpdateDevice)
		merchant.DELETE("/devices/:id", deviceH.DeleteDevice)
		merchant.GET("/audio-configs", deviceH.ListAudioConfigs)
		merchant.PUT("/audio-config/:sn", deviceH.UpdateDeviceAudio)

		// 门店设置
		storeH := handler.NewStoreHandler(r.ctx)
		merchant.GET("/store-config", storeH.GetStoreConfig)
		merchant.PUT("/store-config", storeH.UpdateStoreConfig)

		// 退款审批
		payH := handler.NewPaymentHandler(r.ctx)
		merchant.GET("/refund-requests", payH.ListRefundRequests)
		merchant.POST("/refund-requests/:id/approve", payH.ApproveRefund)
		merchant.POST("/refund-requests/:id/reject", payH.RejectRefund)
		merchant.POST("/orders/:id/cancel", payH.MerchantCancelOrder)
		merchant.POST("/orders/:id/refund", payH.MerchantDirectRefund)

		// 团购券管理
		couponH := handler.NewCouponHandler(r.ctx)
		merchant.GET("/coupons", couponH.ListCoupons)
		merchant.POST("/coupons", couponH.CreateCoupon)
		merchant.DELETE("/coupons/:id", couponH.DeleteCoupon)

		// 会员管理
		memberH := handler.NewMemberHandler(r.ctx)
		merchant.GET("/member-levels", memberH.ListMemberLevels)
		merchant.POST("/member-levels", memberH.CreateMemberLevel)
		merchant.PUT("/member-levels/:id", memberH.UpdateMemberLevel)
		merchant.DELETE("/member-levels/:id", memberH.DeleteMemberLevel)
		merchant.GET("/customers", memberH.ListCustomers)
		merchant.GET("/customers/:user_id", memberH.GetCustomer)
		merchant.PUT("/customers/:user_id/membership", memberH.SetCustomerMembership)
		merchant.DELETE("/customers/:user_id/membership", memberH.RemoveCustomerMembership)
	}
}

// 这个 import 在 app.go 使用，所以这里加上编译期引用
var _ = NewAdminRouter
