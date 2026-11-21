// Package resource 资源业务逻辑层 —— 分布式锁 + 状态机
package resource

import (
	"context"
	"fmt"
	"time"

	"smart-space-management/backend/internal/model/resource"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// Service 资源服务
type Service struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewService(db *gorm.DB, rdb *redis.Client) *Service {
	return &Service{db: db, redis: rdb}
}

// Lock 通过 Redis 分布式锁锁定资源 — 防止并发预约
// 返回锁值 (lockVal)，用于后续释放
func (s *Service) Lock(ctx context.Context, resourceID uint64, userID uint64, ttl time.Duration) (string, error) {
	lockKey := fmt.Sprintf("ssm:resource:lock:%d", resourceID)
	lockVal := fmt.Sprintf("%d:%d", userID, time.Now().UnixNano())

	ok, err := s.redis.SetNX(ctx, lockKey, lockVal, ttl).Result()
	if err != nil {
		return "", fmt.Errorf("redis setnx: %w", err)
	}
	if !ok {
		return "", fmt.Errorf("resource %d is locked", resourceID)
	}

	return lockVal, nil
}

// Unlock 释放资源锁 — Lua 脚本确保仅持有者释放
func (s *Service) Unlock(ctx context.Context, resourceID uint64, lockVal string) error {
	lockKey := fmt.Sprintf("ssm:resource:lock:%d", resourceID)
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`
	return s.redis.Eval(ctx, script, []string{lockKey}, lockVal).Err()
}

// GetStatus 获取资源当前状态 (优先 Redis，回退 DB)
func (s *Service) GetStatus(ctx context.Context, resourceID uint64) (resource.ResourceStatus, error) {
	key := fmt.Sprintf("ssm:resource:status:%d", resourceID)
	val, err := s.redis.Get(ctx, key).Int()
	if err == nil {
		return resource.ResourceStatus(val), nil
	}

	var r resource.Resource
	if err := s.db.First(&r, resourceID).Error; err != nil {
		return 0, err
	}
	// 写入缓存
	s.redis.Set(ctx, key, int(r.Status), 10*time.Second)
	return r.Status, nil
}

// SetStatus 更新资源状态并刷新缓存
func (s *Service) SetStatus(ctx context.Context, resourceID uint64, status resource.ResourceStatus) error {
	if err := s.db.Model(&resource.Resource{}).Where("id = ?", resourceID).
		Update("status", status).Error; err != nil {
		return err
	}
	key := fmt.Sprintf("ssm:resource:status:%d", resourceID)
	s.redis.Del(ctx, key)
	return nil
}

// IsAvailable 资源是否可预约
func (s *Service) IsAvailable(ctx context.Context, resourceID uint64) bool {
	status, err := s.GetStatus(ctx, resourceID)
	if err != nil {
		return false
	}
	return status == resource.StatusFree
}
