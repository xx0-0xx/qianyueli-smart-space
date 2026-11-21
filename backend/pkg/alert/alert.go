// Package alert 主动告警通知（企业微信/钉钉机器人）
package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"smart-space-management/backend/config"
)

// Notifier 告警通知器
type Notifier struct {
	cfg    config.AlertConfig
	client *http.Client
}

func New(cfg config.AlertConfig) *Notifier {
	return &Notifier{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Send 发送全局告警（未配置则静默返回 nil）
func (n *Notifier) Send(text string) error {
	if !n.cfg.Enabled || n.cfg.Webhook == "" {
		return nil
	}
	return n.SendTo(n.cfg.Webhook, text)
}

// SendTo 发送到指定 webhook（企业微信/钉钉群机器人，两者文本格式一致）
// 用于「按门店」推送：每个门店一个群机器人地址
func (n *Notifier) SendTo(webhook, text string) error {
	if webhook == "" {
		return nil
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": text},
	})

	resp, err := n.client.Post(webhook, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("alert send: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("alert webhook status: %d", resp.StatusCode)
	}
	return nil
}

// Enabled 是否启用全局告警
func (n *Notifier) Enabled() bool {
	return n.cfg.Enabled
}
