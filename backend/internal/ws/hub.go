// Package ws WebSocket 连接管理 — 设备状态实时推送核心
package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

// Redis 频道
const (
	ChannelDeviceState = "ssm:device:state" // 设备状态变更
	ChannelOrderEvent  = "ssm:order:event"  // 订单事件 (预约开始/结束)
)

// Message 推送的消息体
type Message struct {
	Type    string      `json:"type"`              // device_state / order_event / config_update
	Payload interface{} `json:"payload"`
	Time    int64       `json:"time"`
}

// Client 单个 WS 连接
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   uint64
	storeID  uint64 // 关注的门店
	channels map[string]bool
	mu       sync.Mutex
}

// Hub 维护所有活跃的 WS 连接
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	redis      *redis.Client
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewHub(rdb *redis.Client) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		redis:      rdb,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Run 启动 Hub 主循环 — 订阅 Redis 频道并广播
func (h *Hub) Run() {
	// 订阅 Redis 频道
	pubsub := h.redis.Subscribe(h.ctx, ChannelDeviceState, ChannelOrderEvent)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WS] client connected user=%d store=%d total=%d",
				client.userID, client.storeID, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("[WS] client disconnected user=%d total=%d",
				client.userID, len(h.clients))

		case msg := <-ch:
			if msg == nil {
				continue
			}
			h.broadcast(msg)

		case <-h.ctx.Done():
			return
		}
	}
}

// Stop 停止 Hub
func (h *Hub) Stop() {
	h.cancel()
	h.mu.Lock()
	for client := range h.clients {
		client.conn.Close()
		delete(h.clients, client)
	}
	h.mu.Unlock()
}

// broadcast 将 Redis 消息广播给匹配的客户端
func (h *Hub) broadcast(msg *redis.Message) {
	var envelope Message
	if err := json.Unmarshal([]byte(msg.Payload), &envelope); err != nil {
		log.Printf("[WS] bad redis message: %v", err)
		return
	}

	data, _ := json.Marshal(envelope)

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		// 按频道做简单路由 — 也可按 storeID 过滤
		if _, ok := client.channels[msg.Channel]; ok || len(client.channels) == 0 {
			select {
			case client.send <- data:
			default:
				// 客户端消费太慢，跳过
				go func(c *Client) {
					h.unregister <- c
				}(client)
			}
		}
	}
}

// PublishDeviceState 向 Redis 发布设备状态变更
func PublishDeviceState(rdb *redis.Client, deviceID uint64, storeID uint64, state interface{}) error {
	msg := Message{
		Type:    "device_state",
		Payload: state,
		Time:    time.Now().UnixMilli(),
	}
	raw, _ := json.Marshal(msg)
	return rdb.Publish(context.Background(), ChannelDeviceState, raw).Err()
}

// PublishOrderEvent 向 Redis 发布订单事件
func PublishOrderEvent(rdb *redis.Client, orderID uint64, storeID uint64, event string) error {
	msg := Message{
		Type: "order_event",
		Payload: map[string]interface{}{
			"order_id": orderID,
			"store_id": storeID,
			"event":    event,
		},
		Time: time.Now().UnixMilli(),
	}
	raw, _ := json.Marshal(msg)
	return rdb.Publish(context.Background(), ChannelOrderEvent, raw).Err()
}

// writePump 向客户端写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second) // ping interval
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump 读取客户端消息 (心跳)
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(4096)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// NewClient 创建客户端并启动读写协程
func (h *Hub) NewClient(conn *websocket.Conn, userID, storeID uint64) *Client {
	client := &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		storeID:  storeID,
		channels: map[string]bool{ChannelDeviceState: true, ChannelOrderEvent: true},
	}
	h.register <- client

	go client.writePump()
	go client.readPump()

	return client
}

// ClientCount 当前连接数
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
