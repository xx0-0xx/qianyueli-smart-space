package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"smart-space-management/backend/internal/model/user"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	ctx *Context
}

// 昵称校验：去首尾空白、去控制字符、限长 1-20、敏感词
var nickBanned = []string{"admin", "管理员", "客服", "官方", "系统", "微信", "腾讯"}

// sanitizeNickname 清洗昵称，返回清洗后昵称与错误
func sanitizeNickname(raw string) (string, error) {
	// 1. 去首尾空白
	s := strings.TrimSpace(raw)

	// 2. 去控制字符（含换行、制表、不可见字符）
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			continue
		}
		b.WriteRune(r)
	}
	s = b.String()

	// 3. 长度 1-20（按字符计）
	if utf8.RuneCountInString(s) < 1 || utf8.RuneCountInString(s) > 20 {
		return "", fmt.Errorf("昵称长度需为1-20个字符")
	}

	// 4. 敏感词（防冒充官方/客服）
	lower := strings.ToLower(s)
	for _, w := range nickBanned {
		if strings.Contains(lower, strings.ToLower(w)) {
			return "", fmt.Errorf("昵称包含不允许的词汇")
		}
	}

	return s, nil
}

func NewUserHandler(ctx *Context) *UserHandler {
	return &UserHandler{ctx: ctx}
}

// WechatLoginReq 微信登录请求
type WechatLoginReq struct {
	Code     string `json:"code" binding:"required"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	StoreID  uint64 `json:"store_id"` // 可选：商家登录时需要绑定门店
}

// MockLogin 模拟登录（测试用，接受任意code）
// POST /api/v1/login
func (h *UserHandler) MockLogin(c *gin.Context) {
	// 安全：模拟登录需显式开启 server.enable_mock（与 mode 解耦，避免误开 debug 重新暴露提权入口）
	if !h.ctx.Cfg.Server.EnableMock {
		c.JSON(404, gin.H{"code": 404, "message": "not found"})
		return
	}
	var req struct {
		Code     string `json:"code"`
		NickName string `json:"nick_name"`
		Role     string `json:"role"`
		StoreID  uint64 `json:"store_id"`
	}
	if c.ShouldBindJSON(&req) != nil || req.Code == "" {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	openID := "mock_" + req.Code
	var u user.User
	role := user.UserRole(req.Role)
	if role == "" {
		role = user.RoleUser
	}
	if h.ctx.DB.Where("open_id = ?", openID).First(&u).Error != nil {
		u = user.User{OpenID: openID, Nickname: req.NickName, Role: role, StoreID: req.StoreID}
		u.ID = utils.ShortID()
		h.ctx.DB.Create(&u)
	} else {
		// 仅更新昵称，不覆盖角色和门店
		if req.NickName != "" {
			u.Nickname = req.NickName
			h.ctx.DB.Save(&u)
		}
	}
	claims := utils.JWTClaims{UserID: u.ID, Role: string(u.Role), StoreID: u.StoreID, Client: "user"}
	token, _ := h.ctx.JWT.GenerateAccessToken(claims)
	response.OK(c, gin.H{"token": token, "user_id": u.ID, "role": u.Role})
}

// WechatLogin 微信 code2session 登录
// POST /api/v1/login/wechat
func (h *UserHandler) WechatLogin(c *gin.Context) {
	var req WechatLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	// 开发模式：没有真实微信 AppID 时用 code 作为 openid
	openID := req.Code
	unionID := ""
	if h.ctx.Cfg.Wechat.Miniprogram.AppID != "" {
		var err error
		openID, unionID, err = h.code2session(req.Code)
		if err != nil {
			response.FailMsg(c, errcode.ErrWechatLoginFail, err.Error())
			return
		}
	}

	// 查找或创建用户
	var u user.User
	result := h.ctx.DB.Where("open_id = ?", openID).First(&u)
	if result.Error != nil {
		// 新用户注册
		u = user.User{
			OpenID:   openID,
			UnionID:  unionID,
			Nickname: req.Nickname,
			Avatar:   req.Avatar,
			Role:     user.RoleUser,
			Status:   user.StatusNormal,
		}
		u.ID = utils.ShortID()
		if err := h.ctx.DB.Create(&u).Error; err != nil {
			response.FailCode(c, errcode.ErrInternal)
			return
		}
	} else {
		// 更新昵称头像（昵称需清洗校验）
		updates := map[string]interface{}{}
		if req.Nickname != "" {
			if nick, err := sanitizeNickname(req.Nickname); err == nil {
				updates["nickname"] = nick
			}
		}
		if req.Avatar != "" && !strings.HasPrefix(req.Avatar, "http") {
			updates["avatar"] = req.Avatar
		}
		if len(updates) > 0 {
			h.ctx.DB.Model(&u).Updates(updates)
		}
	}

	// 生成 JWT
	claims := utils.JWTClaims{
		UserID:  u.ID,
		StoreID: u.StoreID,
		Role:    string(u.Role),
		Client:  "user",
	}
	accessToken, _ := h.ctx.JWT.GenerateAccessToken(claims)
	refreshToken, _ := h.ctx.JWT.GenerateRefreshToken(claims)

	response.OK(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    h.ctx.Cfg.JWT.AccessTokenTTL.Seconds(),
		"user": gin.H{
			"id":       u.ID,
			"nickname": u.Nickname,
			"avatar":   u.Avatar,
			"role":     u.Role,
			"store_id": u.StoreID,
		},
	})
}

// RefreshToken 用刷新令牌换取新的访问令牌
// POST /api/v1/refresh
func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}

	claims, err := h.ctx.JWT.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		response.FailCode(c, errcode.ErrTokenInvalid)
		return
	}

	// 校验用户仍存在且状态正常
	var u user.User
	if err := h.ctx.DB.First(&u, claims.UserID).Error; err != nil || u.Status != user.StatusNormal {
		response.FailCode(c, errcode.ErrUnauthorized)
		return
	}

	accessToken, err := h.ctx.JWT.GenerateAccessToken(utils.JWTClaims{
		UserID:  u.ID,
		StoreID: u.StoreID,
		Role:    string(u.Role),
		Client:  claims.Client,
	})
	if err != nil {
		response.FailCode(c, errcode.ErrInternal)
		return
	}

	response.OK(c, gin.H{
		"access_token": accessToken,
		"expires_in":   h.ctx.Cfg.JWT.AccessTokenTTL.Seconds(),
	})
}

// GetUserInfo 获取当前用户信息
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	userID := GetUserID(c)
	var u user.User
	if err := h.ctx.DB.First(&u, userID).Error; err != nil {
		response.FailCode(c, errcode.ErrUserNotFound)
		return
	}
	response.OK(c, gin.H{
		"id":       u.ID,
		"nickname": u.Nickname,
		"avatar":   u.Avatar,
		"phone":    u.Phone,
		"role":     u.Role,
		"store_id": u.StoreID,
	})
}

// UpdateUserInfo 更新用户信息
func (h *UserHandler) UpdateUserInfo(c *gin.Context) {
	userID := GetUserID(c)
	var req struct {
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailCode(c, errcode.ErrBadRequest)
		return
	}
	updates := map[string]interface{}{}
	if req.Nickname != "" {
		nick, err := sanitizeNickname(req.Nickname)
		if err != nil {
			response.FailMsg(c, errcode.ErrBadRequest, err.Error())
			return
		}
		updates["nickname"] = nick
	}
	// 头像：仅接受本站上传路径（防止外部 URL 追踪/SSRF）
	if req.Avatar != "" {
		req.Avatar = strings.TrimSpace(req.Avatar)
		if strings.HasPrefix(req.Avatar, "http://") || strings.HasPrefix(req.Avatar, "https://") {
			response.FailMsg(c, errcode.ErrBadRequest, "头像地址不合法")
			return
		}
		updates["avatar"] = req.Avatar
	}
	if len(updates) == 0 {
		response.OKMsg(c, "nothing to update")
		return
	}
	h.ctx.DB.Model(&user.User{}).Where("id = ?", userID).Updates(updates)
	response.OKMsg(c, "updated")
}

// code2session 调用微信 code2session 接口
func (h *UserHandler) code2session(code string) (openID, unionID string, err error) {
	appID := h.ctx.Cfg.Wechat.Miniprogram.AppID
	secret := h.ctx.Cfg.Wechat.Miniprogram.AppSecret

	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		appID, secret, code)

	resp, err := http.Get(url)
	if err != nil {
		return "", "", fmt.Errorf("http err: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		SessionKey string `json:"session_key"`
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("parse err: %w", err)
	}
	if result.ErrCode != 0 {
		return "", "", fmt.Errorf("wechat api err: %d %s", result.ErrCode, result.ErrMsg)
	}

	return result.OpenID, result.UnionID, nil
}
