package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"smart-space-management/backend/internal/model/user"
	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/wechat"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	wechatPhoneAPIBase        = "https://api.weixin.qq.com"
	wechatPhoneHTTPTimeout    = 10 * time.Second
	wechatPhoneMaxRespBytes   = 1 << 20
	wechatPhoneCodeMaxLength  = 512
	wechatTokenErrCodeInvalid = 40001
	wechatTokenErrCodeNoAuth  = 40014
	wechatTokenErrCodeExpired = 42001
	wechatCodeErrCodeInvalid  = 40029
	wechatCodeErrCodeUsed     = 40163
)

var (
	wechatPhoneHTTPClient = &http.Client{Timeout: wechatPhoneHTTPTimeout}
	chineseMobileRe       = regexp.MustCompile(`^1[3-9]\d{9}$`)
)

type wechatPhoneAPIError struct {
	Code int
}

func (e *wechatPhoneAPIError) Error() string {
	return "wechat phone api error"
}

func wechatPhoneErrCode(err error) (int, bool) {
	var apiErr *wechatPhoneAPIError
	if errors.As(err, &apiErr) {
		return apiErr.Code, true
	}
	return 0, false
}

func parseWechatPhone(raw []byte) (string, error) {
	var result struct {
		ErrCode   int `json:"errcode"`
		PhoneInfo struct {
			PhoneNumber string `json:"phoneNumber"`
		} `json:"phone_info"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("parse wechat phone response: %w", err)
	}
	if result.ErrCode != 0 {
		return "", &wechatPhoneAPIError{Code: result.ErrCode}
	}
	phone := strings.TrimSpace(result.PhoneInfo.PhoneNumber)
	if phone == "" {
		return "", fmt.Errorf("wechat phone number is empty")
	}
	return phone, nil
}

func isValidChineseMobile(phone string) bool {
	return chineseMobileRe.MatchString(phone)
}

func normalizeWechatPhoneCode(raw string) (string, bool) {
	code := strings.TrimSpace(raw)
	if code == "" || len(code) > wechatPhoneCodeMaxLength {
		return "", false
	}
	return code, true
}

func fetchWechatPhone(ctx context.Context, accessToken, code string) (string, error) {
	payload, err := json.Marshal(map[string]string{"code": code})
	if err != nil {
		return "", fmt.Errorf("build wechat phone request failed")
	}

	endpoint := wechatPhoneAPIBase + "/wxa/business/getuserphonenumber?access_token=" + url.QueryEscape(accessToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build wechat phone request failed")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := wechatPhoneHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("wechat phone request failed")
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, wechatPhoneMaxRespBytes))
	if err != nil {
		return "", fmt.Errorf("wechat phone response read failed")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("wechat phone service unavailable")
	}
	return parseWechatPhone(raw)
}

// BindWechatPhone 通过微信手机号快速验证 code 绑定当前用户手机号。
// POST /api/v1/user/wechat-phone
func (h *UserHandler) BindWechatPhone(c *gin.Context) {
	userID := GetUserID(c)
	if userID == 0 {
		response.FailCode(c, errcode.ErrUnauthorized)
		return
	}

	var current user.User
	if err := h.ctx.DB.First(&current, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailCode(c, errcode.ErrUserNotFound)
			return
		}
		log.Printf("[WECHAT] bind phone failed user=%d reason=load_user", userID)
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if current.Status != user.StatusNormal {
		response.FailCode(c, errcode.ErrUnauthorized)
		return
	}

	var req struct {
		Code string `json:"code" binding:"required,max=512"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailMsg(c, errcode.ErrBadRequest, "缺少手机号授权 code")
		return
	}
	code, ok := normalizeWechatPhoneCode(req.Code)
	if !ok {
		response.FailMsg(c, errcode.ErrBadRequest, "缺少手机号授权 code")
		return
	}

	appID := strings.TrimSpace(h.ctx.Cfg.Wechat.Miniprogram.AppID)
	appSecret := strings.TrimSpace(h.ctx.Cfg.Wechat.Miniprogram.AppSecret)
	if appID == "" || appSecret == "" {
		log.Printf("[WECHAT] bind phone failed user=%d reason=not_configured", userID)
		response.FailMsg(c, errcode.ErrWechatLoginFail, "微信手机号功能暂不可用，请稍后重试")
		return
	}

	mp := wechat.NewMiniProgram(appID, appSecret).WithRedis(h.ctx.Redis)
	token, err := mp.GetAccessToken()
	if err != nil {
		log.Printf("[WECHAT] bind phone failed user=%d reason=access_token", userID)
		response.FailMsg(c, errcode.ErrWechatLoginFail, "微信手机号服务暂不可用，请稍后重试")
		return
	}

	phone, err := fetchWechatPhone(c.Request.Context(), token, code)
	if apiCode, isAPIError := wechatPhoneErrCode(err); isAPIError &&
		(apiCode == wechatTokenErrCodeInvalid || apiCode == wechatTokenErrCodeNoAuth || apiCode == wechatTokenErrCodeExpired) {
		wechat.InvalidateAccessToken(mp)
		token, err = mp.GetAccessToken()
		if err != nil {
			log.Printf("[WECHAT] bind phone failed user=%d reason=access_token_retry", userID)
			response.FailMsg(c, errcode.ErrWechatLoginFail, "微信手机号服务暂不可用，请稍后重试")
			return
		}
		phone, err = fetchWechatPhone(c.Request.Context(), token, code)
	}
	if err != nil {
		if apiCode, isAPIError := wechatPhoneErrCode(err); isAPIError &&
			(apiCode == wechatCodeErrCodeInvalid || apiCode == wechatCodeErrCodeUsed) {
			log.Printf("[WECHAT] bind phone failed user=%d reason=invalid_code", userID)
			response.FailMsg(c, errcode.ErrBadRequest, "手机号授权码无效或已过期，请重新授权")
			return
		}
		log.Printf("[WECHAT] bind phone failed user=%d reason=wechat_phone", userID)
		response.FailMsg(c, errcode.ErrWechatLoginFail, "微信手机号服务暂不可用，请稍后重试")
		return
	}
	if !isValidChineseMobile(phone) {
		log.Printf("[WECHAT] bind phone failed user=%d reason=invalid_format", userID)
		response.FailMsg(c, errcode.ErrWechatLoginFail, "手机号获取失败，请稍后重试")
		return
	}

	if current.Phone == phone {
		response.OK(c, gin.H{"phone": phone})
		return
	}

	result := h.ctx.DB.Model(&user.User{}).Where("id = ?", userID).Update("phone", phone)
	if result.Error != nil {
		log.Printf("[WECHAT] bind phone failed user=%d reason=save", userID)
		response.FailCode(c, errcode.ErrInternal)
		return
	}
	if result.RowsAffected != 1 {
		log.Printf("[WECHAT] bind phone failed user=%d reason=rows_affected", userID)
		response.FailCode(c, errcode.ErrUserNotFound)
		return
	}

	response.OK(c, gin.H{"phone": phone})
}
