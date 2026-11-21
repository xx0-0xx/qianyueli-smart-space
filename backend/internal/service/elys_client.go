package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	ElysPlatformMeituan = 1
	ElysPlatformDouyin  = 2
)

type ElysClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewElysClient(baseURL, apiKey string) *ElysClient {
	if baseURL == "" {
		baseURL = "https://newopen.elys.cn"
	}
	return &ElysClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 20 * time.Second},
	}
}

type ElysError struct {
	HTTPStatus int
	Code       int
	Success    bool
	ErrorCode  string
	Message    string
	TraceID    string
	Raw        string
}

func (e *ElysError) Error() string {
	return fmt.Sprintf("elys error status=%d code=%d errorCode=%s message=%s",
		e.HTTPStatus, e.Code, e.ErrorCode, e.Message)
}

type elysEnvelope struct {
	Success           bool            `json:"success"`
	Code              int             `json:"code"`
	Message           string          `json:"message"`
	TraceID           any             `json:"traceId"`
	Data              json.RawMessage `json:"data"`
	ErrorCode         string          `json:"errorCode"`
	WxdgOrderNo       string          `json:"wxdgOrderNo"`
	ConsumeCredential []string        `json:"consumeCredential"`
	PlatformResult    json.RawMessage `json:"platformResult"`
}

type ElysPrepareResult struct {
	PayAmount       int64   `json:"payAmount"`
	DealPrice       float64 `json:"dealPrice"`
	ActualPayAmount *int64  `json:"actualPayAmount"`
	TicketName      string  `json:"ticketName"`
	DealID          string  `json:"dealId"`
	TicketInfo      string  `json:"ticketInfo"`
	TicketData      string  `json:"ticketData"`
}

type ElysConsumeItem struct {
	OrderID         string  `json:"orderId"`
	ReceiptCode     string  `json:"receiptCode"`
	DealID          string  `json:"dealId"`
	DealTitle       string  `json:"dealTitle"`
	DealPrice       float64 `json:"dealPrice"`
	PayAmount       int64   `json:"payAmount"`
	ActualPayAmount *int64  `json:"actualPayAmount"`
}

type ElysConsumeResult struct {
	WxdgOrderNo       string
	ConsumeCredential []string
	Data              []ElysConsumeItem
	PlatformResult    json.RawMessage
	TraceID           string
}

func (c *ElysClient) AuthURL(ctx context.Context, shopID uint64, platform int) (string, error) {
	env, err := c.do(ctx, "/api/hexiao/v2/get/auth/url", map[string]any{
		"shopId":   shopID,
		"platform": platform,
	})
	if err != nil {
		return "", err
	}
	var url string
	if err := json.Unmarshal(env.Data, &url); err != nil {
		return "", err
	}
	return url, nil
}

func (c *ElysClient) Prepare(ctx context.Context, shopID uint64, platform int, code string) (*ElysPrepareResult, error) {
	env, err := c.do(ctx, "/api/hexiao/v2/ddzh-tuangou-receipt-prepare", map[string]any{
		"shopId":   shopID,
		"platform": platform,
		"code":     code,
	})
	if err != nil {
		return nil, err
	}
	var res ElysPrepareResult
	if err := json.Unmarshal(env.Data, &res); err != nil {
		return nil, err
	}
	if res.PayAmount <= 0 || res.TicketInfo == "" {
		return nil, fmt.Errorf("elys prepare returned empty amount or ticketInfo")
	}
	return &res, nil
}

func (c *ElysClient) Consume(ctx context.Context, shopID uint64, platform int, code, ticketInfo string, num int) (*ElysConsumeResult, error) {
	env, err := c.do(ctx, "/api/hexiao/v2/ddzh-tuangou-receipt-consume", map[string]any{
		"shopId":     shopID,
		"platform":   platform,
		"code":       code,
		"ticketInfo": ticketInfo,
		"num":        num,
	})
	if err != nil {
		return nil, err
	}
	var items []ElysConsumeItem
	if err := json.Unmarshal(env.Data, &items); err != nil {
		return nil, err
	}
	return &ElysConsumeResult{
		WxdgOrderNo:       env.WxdgOrderNo,
		ConsumeCredential: env.ConsumeCredential,
		Data:              items,
		PlatformResult:    env.PlatformResult,
		TraceID:           traceString(env.TraceID),
	}, nil
}

func (c *ElysClient) do(ctx context.Context, path string, body any) (*elysEnvelope, error) {
	rawBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}

	var env elysEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("elys invalid json: status=%d body=%s", resp.StatusCode, string(raw))
	}
	if !env.Success {
		return nil, &ElysError{
			HTTPStatus: resp.StatusCode,
			Code:       env.Code,
			Success:    env.Success,
			ErrorCode:  env.ErrorCode,
			Message:    env.Message,
			TraceID:    traceString(env.TraceID),
			Raw:        string(raw),
		}
	}
	return &env, nil
}

func traceString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}
