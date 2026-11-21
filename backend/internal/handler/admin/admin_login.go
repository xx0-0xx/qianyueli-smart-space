package admin

import (
	"strings"

	"smart-space-management/backend/internal/handler"
	"smart-space-management/backend/internal/model/user"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// AdminLoginReq 管理员登录请求
type AdminLoginReq struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required"`
	CaptchaID     string `json:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer"`
}

// AdminLogin 管理员账号密码登录（兼容旧桌面端，失败锁定但不要求验证码）
// POST /api/v1/admin/login
func (h *AdminHandler) AdminLogin(c *gin.Context) {
	h.adminLogin(c, false)
}

// AdminLoginWeb 管理员账号密码登录（新版 Web 端，失败后要求验证码）
// POST /api/v1/admin/login/web
func (h *AdminHandler) AdminLoginWeb(c *gin.Context) {
	h.adminLogin(c, true)
}

func (h *AdminHandler) adminLogin(c *gin.Context, secure bool) {
	var req AdminLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	ctx := c.Request.Context()
	ip := c.ClientIP()
	subject := strings.ToLower(strings.TrimSpace(req.Username))
	guard := handler.NewLoginGuard(h.ctx.Redis, h.ctx.Cfg.LoginSecurity)

	if guard.IsLocked(ctx, handler.LoginScopeAdmin, ip, subject) {
		response.FailMsg(c, errcode.ErrLoginLocked, "登录尝试过于频繁，请稍后再试")
		return
	}

	if secure && guard.NeedsCaptcha(ctx, handler.LoginScopeAdmin, ip, subject) {
		if !guard.VerifyCaptcha(ctx, req.CaptchaID, req.CaptchaAnswer) {
			challenge := guard.NewCaptcha(ctx)
			response.FailData(c, errcode.ErrCaptchaRequired, "请完成验证码后重试", challenge)
			return
		}
	}

	// 查找管理员/商家用户
	var u user.User
	if err := h.ctx.DB.Where("nickname = ? AND role IN ?",
		strings.TrimSpace(req.Username), []string{"admin", "super_admin", "merchant"}).First(&u).Error; err != nil {
		h.adminLoginFailed(c, guard, ip, subject)
		return
	}

	// bcrypt 密码校验
	if u.PasswordHash == "" {
		h.adminLoginFailed(c, guard, ip, subject)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		h.adminLoginFailed(c, guard, ip, subject)
		return
	}

	guard.Clear(ctx, handler.LoginScopeAdmin, ip, subject)

	// 生成 JWT
	claims := utils.JWTClaims{
		UserID:  u.ID,
		StoreID: u.StoreID,
		Role:    string(u.Role),
		Client:  "admin",
	}
	accessToken, _ := h.ctx.JWT.GenerateAccessToken(claims)

	response.OK(c, gin.H{
		"access_token": accessToken,
		"user": gin.H{
			"id":       u.ID,
			"nickname": u.Nickname,
			"role":     u.Role,
			"store_id": u.StoreID,
		},
	})
}

// adminLoginFailed 记录失败并按需返回验证码挑战。
func (h *AdminHandler) adminLoginFailed(
	c *gin.Context,
	guard *handler.LoginGuard,
	ip, subject string,
) {
	ctx := c.Request.Context()
	guard.RecordFailure(ctx, handler.LoginScopeAdmin, ip, subject)
	if guard.NeedsCaptcha(ctx, handler.LoginScopeAdmin, ip, subject) {
		challenge := guard.NewCaptcha(ctx)
		response.FailData(c, errcode.ErrCaptchaRequired, "请完成验证码后重试", challenge)
		return
	}
	response.FailMsg(c, errcode.ErrUnauthorized, "账号或密码错误")
}
