// Package configcenter 配置中心运行时 — Redis 缓存 + 变更监听
package configcenter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

const (
	configTTL       = 10 * time.Minute
	channelUpdate   = "ssm:config:update"
)

// Cache 配置缓存
type Cache struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewCache(db *gorm.DB, rdb *redis.Client) *Cache {
	return &Cache{db: db, redis: rdb}
}

// GetPriceRule 获取计费规则 (优先 Redis)
func (c *Cache) GetPriceRule(ctx context.Context, storeID, resourceTypeID uint64) (map[string]interface{}, error) {
	key := fmt.Sprintf("ssm:config:price_rule:%d:%d", storeID, resourceTypeID)

	// 查 Redis
	val, err := c.redis.Get(ctx, key).Result()
	if err == nil {
		var rule map[string]interface{}
		json.Unmarshal([]byte(val), &rule)
		return rule, nil
	}

	// 回退 DB
	var result map[string]interface{}
	c.db.Table("price_rules").
		Where("store_id = ? AND resource_type_id = ? AND status = 1", storeID, resourceTypeID).
		Order("is_default DESC").Limit(1).Take(&result)

	if len(result) == 0 {
		return nil, fmt.Errorf("no price rule for store=%d type=%d", storeID, resourceTypeID)
	}

	// 写回 Redis
	raw, _ := json.Marshal(result)
	c.redis.Set(ctx, key, raw, configTTL)

	return result, nil
}

// GetBookingRule 获取预约规则
func (c *Cache) GetBookingRule(ctx context.Context, storeID, resourceTypeID uint64) (map[string]interface{}, error) {
	key := fmt.Sprintf("ssm:config:booking_rule:%d:%d", storeID, resourceTypeID)

	val, err := c.redis.Get(ctx, key).Result()
	if err == nil {
		var rule map[string]interface{}
		json.Unmarshal([]byte(val), &rule)
		return rule, nil
	}

	var result map[string]interface{}
	c.db.Table("booking_rules").
		Where("store_id = ? AND resource_type_id = ?", storeID, resourceTypeID).
		Limit(1).Take(&result)

	if len(result) == 0 {
		return nil, fmt.Errorf("no booking rule for store=%d type=%d", storeID, resourceTypeID)
	}

	raw, _ := json.Marshal(result)
	c.redis.Set(ctx, key, raw, configTTL)

	return result, nil
}

// Invalidate 使缓存失效 (配置变更时调用)
func (c *Cache) Invalidate(ctx context.Context, configType string, storeID uint64) {
	pattern := fmt.Sprintf("ssm:config:%s:%d:*", configType, storeID)
	keys, err := c.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return
	}
	if len(keys) > 0 {
		c.redis.Del(ctx, keys...)
	}
}

// Listener 配置变更监听器 — 订阅 Redis 频道
type Listener struct {
	redis   *redis.Client
	cache   *Cache
	onEvent func(configType string, storeID uint64)
}

func NewListener(rdb *redis.Client, cache *Cache, onEvent func(string, uint64)) *Listener {
	return &Listener{redis: rdb, cache: cache, onEvent: onEvent}
}

// Listen 开始监听配置变更
func (l *Listener) Listen(ctx context.Context) {
	pubsub := l.redis.Subscribe(ctx, channelUpdate)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case msg := <-ch:
			if msg == nil {
				continue
			}
			var event struct {
				ConfigType string `json:"config_type"`
				StoreID    uint64 `json:"store_id"`
			}
			if json.Unmarshal([]byte(msg.Payload), &event) == nil {
				l.cache.Invalidate(ctx, event.ConfigType, event.StoreID)
				if l.onEvent != nil {
					l.onEvent(event.ConfigType, event.StoreID)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// PublishConfigUpdate 发布配置变更事件
func PublishConfigUpdate(rdb *redis.Client, configType string, storeID uint64) error {
	event := map[string]interface{}{
		"config_type": configType,
		"store_id":    storeID,
		"ts":          time.Now().UnixMilli(),
	}
	raw, _ := json.Marshal(event)
	return rdb.Publish(context.Background(), channelUpdate, raw).Err()
}
