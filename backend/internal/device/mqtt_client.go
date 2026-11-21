// Package device MQTT 客户端 — 设备通信核心
package device

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"smart-space-management/backend/config"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Client MQTT 客户端封装
type Client struct {
	client mqtt.Client
	topics []string
	mu     sync.Mutex
	// 指令回调
	onStatus      func(deviceSN string, payload map[string]interface{})
	onEvent       func(deviceSN string, eventType string, payload map[string]interface{})
	onRaw         func(deviceSN string, text string) // 非 JSON 原始文本（成品控制器 ASCII 上报）
	onClientState func(clientID string, online bool)
}

var globalClient *Client

func SetGlobalClient(c *Client) { globalClient = c }
func GetGlobalClient() *Client  { return globalClient }

// NewClient 创建 MQTT 客户端
func NewClient(cfg config.MQTTConfig) (*Client, error) {
	c := &Client{topics: []string{}}
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.Broker)
	opts.SetClientID(fmt.Sprintf("%s-%d", cfg.ClientIDPrefx, time.Now().UnixNano()%10000))
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetKeepAlive(cfg.Keepalive)
	opts.SetPingTimeout(cfg.PingTimeout)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(30 * time.Second)
	opts.SetCleanSession(true)

	// 连接回调
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Println("[MQTT] connected to broker")
		// 重连后需要重新订阅；首次连接时 topics 为空，由 Subscribe 完成。
		c.resubscribe()
	})
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("[MQTT] connection lost: %v", err)
	})

	// TLS
	if cfg.TLS {
		tlsConfig, err := loadTLSConfig(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("mqtt tls: %w", err)
		}
		opts.SetTLSConfig(tlsConfig)
	}

	c.client = mqtt.NewClient(opts)
	// 重试直到连上（EMQX 可能还在启动）
	for i := 0; i < 30; i++ {
		token := c.client.Connect()
		if token.WaitTimeout(10*time.Second) && token.Error() == nil {
			return c, nil
		}
		log.Printf("[MQTT] 连接失败，第%d次重试...", i+1)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("mqtt connect failed after 30 retries")
}

// Subscribe 订阅设备上报主题；可选传入 broker 客户端上下线回调。
func (c *Client) Subscribe(topics []string, onStatus func(string, map[string]interface{}), onEvent func(string, string, map[string]interface{}), onRaw func(string, string), onClientState ...func(clientID string, online bool)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.onStatus = onStatus
	c.onEvent = onEvent
	c.onRaw = onRaw
	if len(onClientState) > 0 {
		c.onClientState = onClientState[0]
	}
	c.topics = topics

	return c.subscribeLocked(false)
}

// resubscribe 在 MQTT 重连后恢复订阅；CleanSession 会清空服务端订阅。
func (c *Client) resubscribe() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil || len(c.topics) == 0 {
		return
	}
	if err := c.subscribeLocked(true); err != nil {
		log.Printf("[MQTT] re-subscribe failed: %v", err)
	}
}

func (c *Client) subscribeLocked(reconnect bool) error {
	allTopics := append([]string(nil), c.topics...)
	if c.onClientState != nil {
		allTopics = append(allTopics,
			"$SYS/brokers/+/clients/+/connected",
			"$SYS/brokers/+/clients/+/disconnected",
		)
	}

	var firstErr error
	for _, topic := range allTopics {
		token := c.client.Subscribe(topic, 1, c.messageHandler)
		if token.Wait() && token.Error() != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("subscribe %s: %w", topic, token.Error())
			}
			log.Printf("[MQTT] subscribe failed: %s: %v", topic, token.Error())
			continue
		}
		if reconnect {
			log.Printf("[MQTT] re-subscribed: %s", topic)
		} else {
			log.Printf("[MQTT] subscribed: %s", topic)
		}
	}
	return firstErr
}

// PublishCommand 向设备下发控制指令（自制固件，JSON 格式）
func (c *Client) PublishCommand(deviceSN string, cmd string, params map[string]interface{}) error {
	topic := fmt.Sprintf("ssm/device/%s/cmd", deviceSN)
	payload := map[string]interface{}{
		"cmd":    cmd,
		"params": params,
		"ts":     time.Now().UnixMilli(),
	}
	raw, _ := json.Marshal(payload)

	token := c.client.Publish(topic, 1, false, raw)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// PublishRaw 向设备下发纯文本指令（成品控制器，ASCII 格式，如 "a1" / "mp3=00001"）
func (c *Client) PublishRaw(deviceSN string, text string) error {
	topic := fmt.Sprintf("ssm/device/%s/cmd", deviceSN)
	token := c.client.Publish(topic, 1, false, []byte(text))
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// PublishOTA 下发 OTA 升级指令
func (c *Client) PublishOTA(deviceSN string, firmwareURL string, version string, md5 string) error {
	topic := fmt.Sprintf("ssm/device/%s/ota", deviceSN)
	payload := map[string]interface{}{
		"cmd":     "upgrade",
		"url":     firmwareURL,
		"version": version,
		"md5":     md5,
		"ts":      time.Now().UnixMilli(),
	}
	raw, _ := json.Marshal(payload)

	token := c.client.Publish(topic, 1, false, raw)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// Close 断开连接
func (c *Client) Close() {
	c.client.Disconnect(500)
	log.Println("[MQTT] disconnected")
}

// messageHandler MQTT 消息处理
func (c *Client) messageHandler(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	deviceSN := extractDeviceSN(topic)

	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
		// 非 JSON → 成品控制器 ASCII 上报（如 "n1" / "s!-60,n1,n2,n3,n4" / "rest_ok"）
		if c.onRaw != nil && deviceSN != "" {
			c.onRaw(deviceSN, strings.TrimSpace(string(msg.Payload())))
		} else {
			log.Printf("[MQTT] bad payload on %s: %v", topic, err)
		}
		return
	}

	if clientID, online, ok := parseClientStateTopic(topic); ok {
		if c.onClientState != nil {
			c.onClientState(clientID, online)
		}
		return
	}

	// 判断消息类型
	switch {
	case isStatusTopic(topic):
		if c.onStatus != nil {
			c.onStatus(deviceSN, payload)
		}
	case contains(topic, "/event"):
		eventType, _ := payload["event"].(string)
		if c.onEvent != nil {
			c.onEvent(deviceSN, eventType, payload)
		}
	default:
		log.Printf("[MQTT] unknown topic: %s", topic)
	}
}

// loadTLSConfig 加载 TLS 配置
func loadTLSConfig(caCertPath string) (*tls.Config, error) {
	certpool := x509.NewCertPool()
	pemCerts, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}
	certpool.AppendCertsFromPEM(pemCerts)
	return &tls.Config{RootCAs: certpool}, nil
}

func isStatusTopic(topic string) bool {
	return contains(topic, "/status") || contains(topic, "/report")
}

func parseClientStateTopic(topic string) (clientID string, online bool, ok bool) {
	parts := splitTopic(topic)
	if len(parts) < 6 || parts[0] != "$SYS" || parts[1] != "brokers" || parts[3] != "clients" {
		return "", false, false
	}
	switch parts[5] {
	case "connected":
		return parts[4], true, true
	case "disconnected":
		return parts[4], false, true
	default:
		return "", false, false
	}
}

func DeviceSNFromMQTTClientID(clientID string) string {
	if strings.HasPrefix(clientID, "ESP_") {
		return strings.TrimPrefix(clientID, "ESP_")
	}
	for _, prefix := range []string{"r4_", "s3_"} {
		if strings.HasPrefix(clientID, prefix) {
			remaining := strings.TrimPrefix(clientID, prefix)
			if i := strings.LastIndex(remaining, "_"); i > 0 {
				return remaining[:i]
			}
			return remaining
		}
	}
	return ""
}

func extractDeviceSN(topic string) string {
	// ssm/device/SN001/status → SN001
	parts := splitTopic(topic)
	for i, p := range parts {
		if p == "device" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func splitTopic(t string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(t); i++ {
		if t[i] == '/' {
			parts = append(parts, t[start:i])
			start = i + 1
		}
	}
	parts = append(parts, t[start:])
	return parts
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
