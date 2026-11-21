package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestElysClient(t *testing.T, handler http.HandlerFunc) *ElysClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewElysClient(srv.URL, "test-key")
}

func TestElysPrepare(t *testing.T) {
	c := newTestElysClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/hexiao/v2/ddzh-tuangou-receipt-prepare" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("auth = %s", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"code":    10000,
			"message": "success",
			"data": map[string]any{
				"payAmount":       7990,
				"dealPrice":       79.9,
				"actualPayAmount": 6990,
				"ticketName":      "畅玩5小时",
				"dealId":          "deal-1",
				"ticketInfo":      "ticket-info",
				"ticketData":      "",
			},
		})
	})

	res, err := c.Prepare(context.Background(), 123, 2, "0106803637807")
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if res.PayAmount != 7990 || res.TicketName != "畅玩5小时" || res.TicketInfo != "ticket-info" {
		t.Fatalf("result = %+v", res)
	}
}

func TestElysConsumeSuccess(t *testing.T) {
	c := newTestElysClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":           true,
			"code":              10000,
			"message":           "履约成功",
			"wxdgOrderNo":       "wxdg-1",
			"consumeCredential": []string{"cred-1"},
			"data": []map[string]any{{
				"orderId":         "order-1",
				"receiptCode":     "receipt-1",
				"dealId":          "deal-1",
				"dealTitle":       "畅玩5小时",
				"payAmount":       7990,
				"actualPayAmount": 6990,
			}},
		})
	})

	res, err := c.Consume(context.Background(), 123, 2, "0106803637807", "ticket-info", 1)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if res.WxdgOrderNo != "wxdg-1" || len(res.Data) != 1 || res.Data[0].PayAmount != 7990 {
		t.Fatalf("result = %+v", res)
	}
}

func TestElysConsumeDuplicate(t *testing.T) {
	c := newTestElysClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success":   false,
			"code":      10500,
			"message":   "优惠券已核销",
			"errorCode": "ALREADY_CONSUMED",
		})
	})

	_, err := c.Consume(context.Background(), 123, 2, "code", "ticket-info", 1)
	if err == nil {
		t.Fatal("expected error")
	}
	e, ok := err.(*ElysError)
	if !ok || e.Code != 10500 || e.ErrorCode != "ALREADY_CONSUMED" {
		t.Fatalf("err = %#v", err)
	}
}
