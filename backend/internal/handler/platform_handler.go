package handler

import (
	"fmt"
	"net/http"
	"time"

	"smart-space-management/backend/internal/model/order"
	"smart-space-management/backend/internal/model/resource"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type PlatformHandler struct {
	ctx *Context
}

func NewPlatformHandler(ctx *Context) *PlatformHandler { return &PlatformHandler{ctx: ctx} }

// MeituanVerify 美团验券回调
// POST /api/v1/callback/meituan/verify
func (h *PlatformHandler) MeituanVerify(c *gin.Context) {
	// 安全：暂无签名校验，需显式开启 server.enable_mock；接入正式验券前必须补签名后再放行
	if !h.ctx.Cfg.Server.EnableMock {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
		return
	}
	var req struct {
		CouponCode string `json:"coupon_code"`
		PoiID      string `json:"poi_id"`
		SerialNo   string `json:"serial_no"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest); return
	}

	// 幂等
	key := fmt.Sprintf("ssm:meituan:%s", req.SerialNo)
	ok, _ := h.ctx.Redis.SetNX(c.Request.Context(), key, "1", 24*time.Hour).Result()
	if !ok { response.OK(c, gin.H{"result": "SUCCESS"}); return }

	// 查门店（poi_id 存在 config_json 的 meituan_poi 里）
	var s struct{ ID uint64 }
	h.ctx.DB.Raw("SELECT id FROM stores WHERE JSON_EXTRACT(config_json,'$.meituan_poi') = ?", req.PoiID).Scan(&s)
	if s.ID == 0 { response.FailMsg(c, errcode.ErrNotFound, "未匹配门店"); return }

	// 自动找房建单
	result := h.autoCreateOrder(s.ID, fmt.Sprintf("美团验券-%s", req.CouponCode))
	response.OK(c, result)
}

// DouyinVerify 抖音验券回调
// POST /api/v1/callback/douyin/verify
func (h *PlatformHandler) DouyinVerify(c *gin.Context) {
	// 安全：暂无签名校验，需显式开启 server.enable_mock；接入正式验券前必须补签名后再放行
	if !h.ctx.Cfg.Server.EnableMock {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
		return
	}
	var req struct {
		CouponID string `json:"coupon_id"`
		ShopID   string `json:"shop_id"`
		VerifyID string `json:"verify_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest); return
	}

	key := fmt.Sprintf("ssm:douyin:%s", req.VerifyID)
	ok, _ := h.ctx.Redis.SetNX(c.Request.Context(), key, "1", 24*time.Hour).Result()
	if !ok { response.OK(c, gin.H{"result": "SUCCESS"}); return }

	var s struct{ ID uint64 }
	h.ctx.DB.Raw("SELECT id FROM stores WHERE JSON_EXTRACT(config_json,'$.douyin_shop') = ?", req.ShopID).Scan(&s)
	if s.ID == 0 { response.FailMsg(c, errcode.ErrNotFound, "未匹配门店"); return }

	result := h.autoCreateOrder(s.ID, fmt.Sprintf("抖音验券-%s", req.CouponID))
	response.OK(c, result)
}

func (h *PlatformHandler) autoCreateOrder(storeID uint64, remark string) gin.H {
	var rooms []resource.Resource
	h.ctx.DB.Where("store_id = ? AND status != ?", storeID, resource.StatusMaint).Find(&rooms)

	var picked *resource.Resource
	now := time.Now()
	st := now.UnixMilli()
	et := st + 2*3600000
	for _, r := range rooms {
		var c int64
		h.ctx.DB.Model(&order.Order{}).Where("resource_id = ? AND status IN (0,1,2,3) AND start_time < ? AND end_time > ?", r.ID, et, st).Count(&c)
		if c == 0 { picked = &r; break }
	}
	if picked == nil { return gin.H{"result": "FAIL", "reason": "无可用房间"} }

	o := order.Order{
		OrderNo:      fmt.Sprintf("PF%d%04d", time.Now().UnixMilli(), utils.ShortID()%10000),
		UserID:       0, StoreID: storeID, ResourceID: picked.ID,
		ResourceName: picked.Name, StartTime: st, EndTime: et,
		Duration: 120, UnitPrice: 0, TotalAmount: 0, Status: order.StatusPaid,
	}
	o.ID = utils.ShortID()
	nowMs := now.UnixMilli()
	o.PaidAt = &nowMs
	h.ctx.DB.Create(&o)
	h.ctx.DeviceSvc.SendSessionCmd(picked.ID, "session_start", h.ctx.Cfg.Server.PublicURL)
	return gin.H{"result": "SUCCESS", "order_id": fmt.Sprintf("%d", o.ID), "room": picked.Name}
}
