package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	accessTokenURL          = "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s"
	accessTokenHTTPTimeout  = 10 * time.Second
	accessTokenRedisTimeout = 2 * time.Second
	accessTokenDefaultTTL   = 7000 * time.Second
	accessTokenMargin       = 300 * time.Second
	accessTokenMaxBytes     = 1 << 20
)

// accessToken 缓存（微信 token 2 小时有效）
var (
	tokenMu     sync.Mutex
	tokenCache  string
	tokenExpire time.Time
	tokenClient = &http.Client{Timeout: accessTokenHTTPTimeout}
)

func accessTokenCacheKey(appID string) string {
	return "ssm:wechat:access_token:" + appID
}

// WithRedis 为小程序配置可选的 Redis 客户端，用于共享 access_token 缓存。
func (mp *MiniProgram) WithRedis(rdb *redis.Client) *MiniProgram {
	mp.Redis = rdb
	return mp
}

// InvalidateAccessToken 清除缓存的微信 access_token。
// 传入 MiniProgram 时同时清除其 Redis 缓存；Redis 不可用时仅清除内存缓存。
func InvalidateAccessToken(instances ...*MiniProgram) {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	tokenCache = ""
	tokenExpire = time.Time{}
	for _, mp := range instances {
		if mp == nil || mp.Redis == nil || mp.AppID == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), accessTokenRedisTimeout)
		_ = mp.Redis.Del(ctx, accessTokenCacheKey(mp.AppID)).Err()
		cancel()
	}
}

func accessTokenCacheTTL(expiresIn int64) time.Duration {
	if expiresIn <= 0 {
		return accessTokenDefaultTTL
	}
	if expiresIn <= 300 {
		return 0
	}
	return time.Duration(expiresIn)*time.Second - accessTokenMargin
}

// cachedAccessToken 返回 Redis 或内存中的缓存 token。调用方必须持有 tokenMu。
func (mp *MiniProgram) cachedAccessToken(cacheKey string) (string, bool) {
	if mp.Redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), accessTokenRedisTimeout)
		token, err := mp.Redis.Get(ctx, cacheKey).Result()
		cancel()
		if err == nil && token != "" {
			return token, true
		}
	}
	if tokenCache != "" && time.Now().Before(tokenExpire) {
		return tokenCache, true
	}
	return "", false
}

// storeAccessToken 写入 Redis 和内存缓存。调用方必须持有 tokenMu。
func (mp *MiniProgram) storeAccessToken(cacheKey, token string, ttl time.Duration) {
	tokenCache = token
	tokenExpire = time.Now().Add(ttl)

	if mp.Redis == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), accessTokenRedisTimeout)
	_ = mp.Redis.Set(ctx, cacheKey, token, ttl).Err()
	cancel()
}

// GetAccessToken 获取小程序全局 access_token（优先 Redis，回退内存缓存）
func (mp *MiniProgram) GetAccessToken() (string, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	if mp.AppID == "" || mp.AppSecret == "" {
		return "", fmt.Errorf("wechat access token not configured")
	}

	cacheKey := accessTokenCacheKey(mp.AppID)
	if token, ok := mp.cachedAccessToken(cacheKey); ok {
		return token, nil
	}

	url := fmt.Sprintf(accessTokenURL, mp.AppID, mp.AppSecret)
	resp, err := tokenClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("wechat access token request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("wechat access token service unavailable")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, accessTokenMaxBytes))
	if err != nil {
		return "", fmt.Errorf("wechat access token response read failed")
	}

	var r struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("wechat access token response invalid")
	}
	if r.ErrCode != 0 || r.AccessToken == "" {
		return "", fmt.Errorf("wechat access token unavailable")
	}

	if ttl := accessTokenCacheTTL(r.ExpiresIn); ttl > 0 {
		mp.storeAccessToken(cacheKey, r.AccessToken, ttl)
	}
	return r.AccessToken, nil
}

// GetMiniProgramCode 生成小程序码（getwxacodeunlimit）
// scene: 参数（如 "store=6913002"），page: 页面路径（如 "pages/index/index"）
// 返回图片字节
func (mp *MiniProgram) GetMiniProgramCode(scene, page string) ([]byte, error) {
	token, err := mp.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get wxacode failed")
	}

	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token=%s", token)
	payload := map[string]interface{}{
		"scene":       scene,
		"page":        page,
		"check_path":  false,
		"env_version": "release",
	}
	body, _ := json.Marshal(payload)

	resp, err := tokenClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("get wxacode failed")
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, accessTokenMaxBytes))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wxacode service unavailable")
	}
	// 微信错误时返回 JSON，成功时返回图片二进制
	if len(data) > 0 && data[0] == '{' {
		return nil, fmt.Errorf("wxacode request failed")
	}
	return data, nil
}
