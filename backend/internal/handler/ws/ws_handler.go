package ws

import (
	"log"
	"net/http"

	"smart-space-management/backend/internal/ws"
	"smart-space-management/backend/pkg/utils"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // 小程序 + 管理后台都可能连接
	},
}

// WSHandler WebSocket 处理器
type WSHandler struct {
	hub    *ws.Hub
	jwtMgr *utils.JWTManager
	redis  *redis.Client
}

func NewWSHandler(hub *ws.Hub, jwtMgr *utils.JWTManager, rdb *redis.Client) *WSHandler {
	return &WSHandler{hub: hub, jwtMgr: jwtMgr, redis: rdb}
}

// Serve 处理 WebSocket 升级 —— /ws?token=xxx&store_id=123
func (h *WSHandler) Serve(w http.ResponseWriter, r *http.Request) {
	// 鉴权
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	claims, err := h.jwtMgr.ParseToken(token)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] upgrade error: %v", err)
		return
	}

	h.hub.NewClient(conn, claims.UserID, claims.StoreID)
}
