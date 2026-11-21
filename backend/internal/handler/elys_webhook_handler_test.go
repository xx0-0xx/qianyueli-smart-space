package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"testing"
	"time"
)

func signElys(apiKey string, raw []byte) (string, string) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write([]byte(ts + "."))
	mac.Write(raw)
	return ts, "v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyElysSignature(t *testing.T) {
	raw := []byte(`{"eventId":"evt_1","event":"merchant.authorization.succeeded"}`)
	ts, sig := signElys("key-123", raw)
	if !verifyElysSignature("key-123", ts, sig, raw) {
		t.Fatal("valid signature rejected")
	}
	if verifyElysSignature("key-123", ts, sig, []byte(`{"x":1}`)) {
		t.Fatal("tampered body accepted")
	}
	old := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	if verifyElysSignature("key-123", old, sig, raw) {
		t.Fatal("old timestamp accepted")
	}
}

func TestValidElysRequestID(t *testing.T) {
	if !validElysRequestID("webhook-event-123") {
		t.Fatal("valid request id rejected")
	}
	if validElysRequestID(" bad id") {
		t.Fatal("invalid request id accepted")
	}
}
