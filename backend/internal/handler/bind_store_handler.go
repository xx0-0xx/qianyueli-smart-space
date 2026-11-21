package handler

import (
	"fmt"
	"strconv"
	"time"

	"smart-space-management/backend/internal/model/store"
	"smart-space-management/backend/internal/model/user"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// BindStoreReq 商家绑定门店请求
type BindStoreReq struct {
	BindCode string `json:"bind_code" binding:"required"` // 邀请码或门店 ID
}

// BindStore 商家通过邀请码绑定门店
// POST /api/v1/merchant/bind-store
func (h *UserHandler) BindStore(c *gin.Context) {
	var req BindStoreReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	userID := GetUserID(c)
	if userID == 0 {
		response.FailCode(c, errcode.ErrUnauthorized)
		return
	}

	// 查找用户
	var u user.User
	if h.ctx.DB.First(&u, userID).Error != nil {
		response.FailCode(c, errcode.ErrUserNotFound)
		return
	}

	// 已绑定门店则直接返回，无需重复操作
	if u.Role == user.RoleMerchant && u.StoreID > 0 {
		response.OK(c, gin.H{
			"store_id": fmt.Sprintf("%d", u.StoreID),
			"role":     user.RoleMerchant,
		})
		return
	}

	var st store.Store
	if err := h.ctx.DB.Where("invite_code = ?", normalizeInviteCode(req.BindCode)).First(&st).Error; err != nil {
		response.FailMsg(c, errcode.ErrBadRequest, "邀请码无效，请检查")
		return
	}
	h.bindUserToStore(c, &u, st.ID)
}

// bindUserToStore 把用户升级为商家并绑定门店，返回新 token
func (h *UserHandler) bindUserToStore(c *gin.Context, u *user.User, storeID uint64) {
	u.Role = user.RoleMerchant
	u.StoreID = storeID
	h.ctx.DB.Save(u)

	claims := utils.JWTClaims{UserID: u.ID, StoreID: storeID, Role: string(user.RoleMerchant), Client: "user"}
	newToken, _ := h.ctx.JWT.GenerateAccessToken(claims)
	response.OK(c, gin.H{
		"store_id":     fmt.Sprintf("%d", storeID),
		"role":         user.RoleMerchant,
		"access_token": newToken,
	})
}

// ListMerchantStores 商家名下的所有门店
// GET /api/v1/merchant/stores
func (h *UserHandler) ListMerchantStores(c *gin.Context) {
	userID := GetUserID(c)
	if userID == 0 {
		response.FailCode(c, errcode.ErrUnauthorized)
		return
	}

	var links []user.MerchantStore
	h.ctx.DB.Where("merchant_id = ?", userID).Find(&links)

	// 兼容：老数据只有 user.store_id，没有关联表记录
	if len(links) == 0 {
		var u user.User
		if h.ctx.DB.First(&u, userID).Error == nil && u.StoreID > 0 {
			ms := user.MerchantStore{MerchantID: userID, StoreID: u.StoreID}
			ms.ID = utils.ShortID()
			links = append(links, ms)
			h.ctx.DB.Create(&ms)
		}
	}

	var storeIDs []uint64
	for _, l := range links {
		storeIDs = append(storeIDs, l.StoreID)
	}
	var stores []store.Store
	if len(storeIDs) > 0 {
		h.ctx.DB.Where("id IN ?", storeIDs).Find(&stores)
	}

	response.OK(c, stores)
}

// BindMerchantStore 商家绑定新门店（通过邀请码）
// POST /api/v1/merchant/stores/bind
func (h *UserHandler) BindMerchantStore(c *gin.Context) {
	var req BindStoreReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	userID := GetUserID(c)
	if userID == 0 {
		response.FailCode(c, errcode.ErrUnauthorized)
		return
	}

	var st store.Store
	if err := h.ctx.DB.Where("invite_code = ?", normalizeInviteCode(req.BindCode)).First(&st).Error; err != nil {
		response.FailMsg(c, errcode.ErrBadRequest, "邀请码无效")
		return
	}

	// 已绑定则跳过
	var count int64
	h.ctx.DB.Model(&user.MerchantStore{}).
		Where("merchant_id = ? AND store_id = ?", userID, st.ID).Count(&count)
	if count == 0 {
		ms := user.MerchantStore{MerchantID: userID, StoreID: st.ID}
		ms.ID = utils.ShortID()
		if err := h.ctx.DB.Create(&ms).Error; err != nil {
			response.FailMsg(c, errcode.ErrInternal, "绑定失败: "+err.Error())
			return
		}
	}

	response.OK(c, gin.H{
		"store_id":   st.ID,
		"store_name": st.Name,
	})
}

// InviteLoginReq 邀请码登录请求
type InviteLoginReq struct {
	InviteCode    string `json:"invite_code" binding:"required"`
	CaptchaID     string `json:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer"`
}

// InviteLogin 商家通过邀请码直接登录（小程序端，无验证码）。
// POST /api/v1/merchant/invite-login
func (h *UserHandler) InviteLogin(c *gin.Context) {
	h.inviteLogin(c, false)
}

// InviteLoginWeb 商家通过邀请码登录（Web/桌面端，失败后要求验证码）。
// POST /api/v1/merchant/invite-login/web
func (h *UserHandler) InviteLoginWeb(c *gin.Context) {
	h.inviteLogin(c, true)
}

func (h *UserHandler) inviteLogin(c *gin.Context, secure bool) {
	var req InviteLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	code := normalizeInviteCode(req.InviteCode)
	guard := h.loginGuard()
	ctx := c.Request.Context()
	ip := c.ClientIP()

	if guard.IsLocked(ctx, LoginScopeInvite, ip, code) {
		response.FailMsg(c, errcode.ErrLoginLocked, "登录尝试过于频繁，请稍后再试")
		return
	}

	if secure && guard.NeedsCaptcha(ctx, LoginScopeInvite, ip, code) {
		if !guard.VerifyCaptcha(ctx, req.CaptchaID, req.CaptchaAnswer) {
			challenge := guard.NewCaptcha(ctx)
			response.FailData(c, errcode.ErrCaptchaRequired, "请完成验证码后重试", challenge)
			return
		}
	}

	// 查找邀请码对应的门店（不包含已删除门店）。
	var st store.Store
	if err := h.ctx.DB.Where("invite_code = ?", code).First(&st).Error; err != nil {
		guard.RecordFailure(ctx, LoginScopeInvite, ip, code)
		if secure && guard.NeedsCaptcha(ctx, LoginScopeInvite, ip, code) {
			challenge := guard.NewCaptcha(ctx)
			response.FailData(c, errcode.ErrCaptchaRequired, "请完成验证码后重试", challenge)
			return
		}
		response.FailMsg(c, errcode.ErrBadRequest, "邀请码无效")
		return
	}

	guard.Clear(ctx, LoginScopeInvite, ip, code)

	// 查找该门店已有的商家用户，没有则创建一个。
	var u user.User
	err := h.ctx.DB.Where("store_id = ? AND role = ?", st.ID, "merchant").First(&u).Error
	if err != nil {
		// 创建新的商家用户
		u = user.User{
			Nickname: st.Name,
			OpenID:   "web_merchant_" + strconv.FormatUint(st.ID, 10),
			Role:     user.RoleMerchant,
			StoreID:  st.ID,
			Status:   user.StatusNormal,
		}
		u.ID = utils.ShortID()
		u.CreatedAt = time.Now()
		u.UpdatedAt = time.Now()
		h.ctx.DB.Create(&u)
	}

	// 生成 JWT
	claims := utils.JWTClaims{
		UserID:  u.ID,
		StoreID: st.ID,
		Role:    string(user.RoleMerchant),
		Client:  "merchant",
	}
	token, _ := h.ctx.JWT.GenerateAccessToken(claims)

	response.OK(c, gin.H{
		"access_token": token,
		"user": gin.H{
			"id":       u.ID,
			"nickname": u.Nickname,
			"role":     user.RoleMerchant,
			"store_id": st.ID,
		},
		"store": gin.H{
			"id":   st.ID,
			"name": st.Name,
		},
	})
}
