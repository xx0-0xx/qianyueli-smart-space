package middleware

import (
	"sync"
	"time"

	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// TokenBucket 简单令牌桶
type TokenBucket struct {
	rate       float64 // 每秒补充令牌数
	capacity   int     // 桶容量
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
	mu         sync.Mutex
}

func NewTokenBucket(rate float64, capacity int) *TokenBucket {
	now := time.Now()
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     float64(capacity),
		lastRefill: now,
		lastSeen:   now,
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	tb.lastSeen = now
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}
	tb.lastRefill = now

	if tb.tokens < 1 {
		return false
	}
	tb.tokens--
	return true
}

// PerClientStore 按客户端 IP 存储桶（带空闲回收，避免长期运行内存无界增长）
type PerClientStore struct {
	buckets map[string]*TokenBucket
	mu      sync.Mutex
}

const (
	bucketIdleTTL     = 10 * time.Minute
	bucketSweepPeriod = 5 * time.Minute
)

func NewPerClientStore() *PerClientStore {
	s := &PerClientStore{buckets: make(map[string]*TokenBucket)}
	go s.sweepLoop()
	return s
}

func (s *PerClientStore) Get(key string) *TokenBucket {
	s.mu.Lock()
	defer s.mu.Unlock()

	if b, ok := s.buckets[key]; ok {
		return b
	}
	// 默认 10 req/s, 容量 20
	b := NewTokenBucket(10, 20)
	s.buckets[key] = b
	return b
}

func (s *PerClientStore) sweepLoop() {
	ticker := time.NewTicker(bucketSweepPeriod)
	defer ticker.Stop()
	for range ticker.C {
		s.sweep()
	}
}

func (s *PerClientStore) sweep() {
	cutoff := time.Now().Add(-bucketIdleTTL)
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, b := range s.buckets {
		b.mu.Lock()
		idle := b.lastSeen.Before(cutoff)
		b.mu.Unlock()
		if idle {
			delete(s.buckets, k)
		}
	}
}

// RateLimit 基于令牌桶的 IP 限流中间件
func RateLimit(store *PerClientStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		bucket := store.Get(ip)
		if !bucket.Allow() {
			response.FailCode(c, errcode.ErrRateLimited)
			return
		}
		c.Next()
	}
}
