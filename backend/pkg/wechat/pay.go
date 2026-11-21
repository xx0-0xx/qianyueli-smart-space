package wechat

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// PayClient 微信支付 API V3 封装
type PayClient struct {
	MchID    string
	APIv3Key string
	CertPath string
	KeyPath  string
}

func NewPayClient(mchID, apiV3Key, certPath, keyPath string) *PayClient {
	return &PayClient{
		MchID:    mchID,
		APIv3Key: apiV3Key,
		CertPath: certPath,
		KeyPath:  keyPath,
	}
}

// JSAPIOrderReq JSAPI 下单请求
type JSAPIOrderReq struct {
	AppID       string `json:"appid"`
	MchID       string `json:"mchid"`
	Description string `json:"description"`
	OutTradeNo  string `json:"out_trade_no"`
	NotifyURL   string `json:"notify_url"`
	Amount      struct {
		Total    int    `json:"total"`
		Currency string `json:"currency"`
	} `json:"amount"`
	Payer struct {
		OpenID string `json:"openid"`
	} `json:"payer"`
}

// CreateJSAPIOrder 创建 JSAPI 支付订单 (占位实现)
// 生产环境需:
//  1. 构造签名 Authorization: WECHATPAY2-SHA256-RSA2048 ...
//  2. POST https://api.mch.weixin.qq.com/v3/pay/transactions/jsapi
//  3. 解析返回的 prepay_id
func (c *PayClient) CreateJSAPIOrder(req JSAPIOrderReq) (string, error) {
	// TODO: 实现微信支付 API V3 调用
	// 1. 构建 HTTP client (带证书)
	// 2. 签名 (随机数 + 时间戳 + body)
	// 3. 发送请求
	// 4. 返回 prepay_id

	_ = req
	prepayID := fmt.Sprintf("wx_prepay_%s_%d", req.OutTradeNo, time.Now().Unix())
	return prepayID, nil
}

// VerifyCallbackSign 验证支付回调签名
func (c *PayClient) VerifyCallbackSign(body []byte, signature, timestamp, nonce string) bool {
	message := fmt.Sprintf("%s\n%s\n%s\n", timestamp, nonce, string(body))
	mac := hmac.New(sha256.New, []byte(c.APIv3Key))
	mac.Write([]byte(message))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}
