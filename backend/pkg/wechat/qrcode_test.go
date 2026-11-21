package wechat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func fakeTokenClient(calls *int, token string, expiresIn int64) *http.Client {
	return &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			*calls = *calls + 1
			body := fmt.Sprintf(`{"access_token":%q,"expires_in":%d}`, token, expiresIn)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	}
}

func TestGetAccessTokenErrorHidesCredentials(t *testing.T) {
	oldClient := tokenClient
	defer func() {
		tokenClient = oldClient
		InvalidateAccessToken()
	}()
	InvalidateAccessToken()
	tokenClient = &http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("dial failure")
		}),
	}

	appID := "test-app-id-value"
	appSecret := "test-app-secret-value"
	_, err := NewMiniProgram(appID, appSecret).GetAccessToken()
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, forbidden := range []string{appID, appSecret, "api.weixin.qq.com"} {
		if strings.Contains(msg, forbidden) {
			t.Fatalf("error must not contain %q: %q", forbidden, msg)
		}
	}
}

func TestGetAccessTokenCachesToken(t *testing.T) {
	oldClient := tokenClient
	defer func() {
		tokenClient = oldClient
		InvalidateAccessToken()
	}()
	InvalidateAccessToken()

	calls := 0
	tokenClient = fakeTokenClient(&calls, "token-123", 7200)

	mp := NewMiniProgram("appid", "secret")
	first, err := mp.GetAccessToken()
	if err != nil || first != "token-123" {
		t.Fatalf("GetAccessToken()=%q err=%v", first, err)
	}
	second, err := mp.GetAccessToken()
	if err != nil || second != "token-123" {
		t.Fatalf("cached GetAccessToken()=%q err=%v", second, err)
	}
	if calls != 1 {
		t.Fatalf("expected one upstream call, got %d", calls)
	}

	tokenMu.Lock()
	remaining := time.Until(tokenExpire)
	tokenMu.Unlock()
	if remaining > 6900*time.Second || remaining < 6800*time.Second {
		t.Fatalf("unexpected token cache ttl: %v", remaining)
	}
}

func TestGetAccessTokenDoesNotCacheShortLivedToken(t *testing.T) {
	oldClient := tokenClient
	defer func() {
		tokenClient = oldClient
		InvalidateAccessToken()
	}()
	InvalidateAccessToken()

	calls := 0
	tokenClient = fakeTokenClient(&calls, "short-token", 200)

	mp := NewMiniProgram("appid", "secret")
	if _, err := mp.GetAccessToken(); err != nil {
		t.Fatalf("first GetAccessToken() err=%v", err)
	}
	if _, err := mp.GetAccessToken(); err != nil {
		t.Fatalf("second GetAccessToken() err=%v", err)
	}
	if calls != 2 {
		t.Fatalf("short-lived token should not be cached, calls=%d", calls)
	}

	tokenMu.Lock()
	defer tokenMu.Unlock()
	if tokenCache != "" {
		t.Fatalf("short-lived token must not be cached, got %q", tokenCache)
	}
}

func TestAccessTokenCacheTTL(t *testing.T) {
	cases := []struct {
		name      string
		expiresIn int64
		want      time.Duration
	}{
		{"expires_in with margin", 7200, 6900 * time.Second},
		{"minimum cacheable", 301, time.Second},
		{"expires_in at margin", 300, 0},
		{"too short expires_in", 200, 0},
		{"missing expires_in", 0, 7000 * time.Second},
		{"negative expires_in", -1, 7000 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := accessTokenCacheTTL(tc.expiresIn); got != tc.want {
				t.Fatalf("accessTokenCacheTTL(%d)=%v want %v", tc.expiresIn, got, tc.want)
			}
		})
	}
}

func TestInvalidateAccessToken(t *testing.T) {
	tokenMu.Lock()
	tokenCache = "cached-token"
	tokenExpire = time.Now().Add(time.Hour)
	tokenMu.Unlock()

	InvalidateAccessToken()

	tokenMu.Lock()
	defer tokenMu.Unlock()
	if tokenCache != "" || !tokenExpire.IsZero() {
		t.Fatalf("cache not cleared: token=%q expire=%v", tokenCache, tokenExpire)
	}
}

func newTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{
		Addr:        server.Addr(),
		MaxRetries:  -1,
		DialTimeout: 100 * time.Millisecond,
	})
	return client, server
}

func TestGetAccessTokenUsesRedisCache(t *testing.T) {
	oldClient := tokenClient
	defer func() {
		tokenClient = oldClient
		InvalidateAccessToken()
	}()
	InvalidateAccessToken()

	rdb, server := newTestRedis(t)
	defer server.Close()
	defer rdb.Close()

	calls := 0
	tokenClient = fakeTokenClient(&calls, "redis-token", 7200)
	mp := NewMiniProgram("redis-appid", "secret").WithRedis(rdb)

	if token, err := mp.GetAccessToken(); err != nil || token != "redis-token" {
		t.Fatalf("first GetAccessToken()=%q err=%v", token, err)
	}

	cacheKey := accessTokenCacheKey("redis-appid")
	ctx := context.Background()
	if token, err := rdb.Get(ctx, cacheKey).Result(); err != nil || token != "redis-token" {
		t.Fatalf("redis cache token=%q err=%v", token, err)
	}
	if ttl := server.TTL(cacheKey); ttl <= 0 || ttl > 6900*time.Second {
		t.Fatalf("unexpected redis ttl: %v", ttl)
	}

	InvalidateAccessToken()
	if token, err := mp.GetAccessToken(); err != nil || token != "redis-token" {
		t.Fatalf("redis cached GetAccessToken()=%q err=%v", token, err)
	}
	if calls != 1 {
		t.Fatalf("expected Redis cache hit without upstream call, calls=%d", calls)
	}

	InvalidateAccessToken(mp)
	if token, err := mp.GetAccessToken(); err != nil || token != "redis-token" {
		t.Fatalf("post-invalidate GetAccessToken()=%q err=%v", token, err)
	}
	if calls != 2 {
		t.Fatalf("expected upstream refetch after invalidation, calls=%d", calls)
	}
}

func TestGetAccessTokenFallsBackWhenRedisUnavailable(t *testing.T) {
	oldClient := tokenClient
	defer func() {
		tokenClient = oldClient
		InvalidateAccessToken()
	}()
	InvalidateAccessToken()

	rdb, server := newTestRedis(t)
	defer rdb.Close()
	server.Close()

	calls := 0
	tokenClient = fakeTokenClient(&calls, "fallback-token", 7200)
	mp := NewMiniProgram("fallback-appid", "secret").WithRedis(rdb)

	first, err := mp.GetAccessToken()
	if err != nil || first != "fallback-token" {
		t.Fatalf("first GetAccessToken()=%q err=%v", first, err)
	}
	second, err := mp.GetAccessToken()
	if err != nil || second != "fallback-token" {
		t.Fatalf("second GetAccessToken()=%q err=%v", second, err)
	}
	if calls != 1 {
		t.Fatalf("expected memory fallback without second upstream call, calls=%d", calls)
	}
}
