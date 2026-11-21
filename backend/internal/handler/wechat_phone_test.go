package handler

import (
	"errors"
	"strings"
	"testing"
)

func TestParseWechatPhone(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{
			name: "success",
			raw:  `{"errcode":0,"errmsg":"ok","phone_info":{"phoneNumber":"13800138000"}}`,
			want: "13800138000",
		},
		{
			name: "success with whitespace",
			raw:  `{"errcode":0,"errmsg":"ok","phone_info":{"phoneNumber":" 13900139000 "}}`,
			want: "13900139000",
		},
		{
			name:    "wechat error code",
			raw:     `{"errcode":40029,"errmsg":"invalid code"}`,
			wantErr: true,
		},
		{
			name:    "empty phone number",
			raw:     `{"errcode":0,"errmsg":"ok","phone_info":{"phoneNumber":""}}`,
			wantErr: true,
		},
		{
			name:    "missing phone info",
			raw:     `{"errcode":0,"errmsg":"ok"}`,
			wantErr: true,
		},
		{
			name:    "invalid json",
			raw:     `not-json`,
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			phone, err := parseWechatPhone([]byte(tc.raw))
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseWechatPhone() err=%v wantErr=%v", err, tc.wantErr)
			}
			if phone != tc.want {
				t.Fatalf("parseWechatPhone()=%q want %q", phone, tc.want)
			}
		})
	}
}

func TestNormalizeWechatPhoneCode(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		want   string
		wantOK bool
	}{
		{"valid", "code-123", "code-123", true},
		{"trimmed", "  code-123  ", "code-123", true},
		{"empty", "", "", false},
		{"blank", "   ", "", false},
		{"too long", strings.Repeat("a", 513), "", false},
		{"max length", strings.Repeat("a", 512), strings.Repeat("a", 512), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, ok := normalizeWechatPhoneCode(tc.raw)
			if ok != tc.wantOK || code != tc.want {
				t.Fatalf("normalizeWechatPhoneCode(%q)=(%q,%v) want (%q,%v)", tc.raw, code, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestWechatPhoneErrCode(t *testing.T) {
	if _, ok := wechatPhoneErrCode(nil); ok {
		t.Fatal("nil error should not carry a wechat code")
	}
	if _, ok := wechatPhoneErrCode(errors.New("plain error")); ok {
		t.Fatal("plain error should not match")
	}
	code, ok := wechatPhoneErrCode(&wechatPhoneAPIError{Code: 40001})
	if !ok || code != 40001 {
		t.Fatalf("wechatPhoneErrCode()=(%d,%v) want (40001,true)", code, ok)
	}

	_, err := parseWechatPhone([]byte(`{"errcode":42001,"errmsg":"token expired"}`))
	code, ok = wechatPhoneErrCode(err)
	if !ok || code != 42001 {
		t.Fatalf("parse error code=(%d,%v) want (42001,true)", code, ok)
	}
	if strings.Contains(err.Error(), "42001") || strings.Contains(err.Error(), "token expired") {
		t.Fatalf("parse error must not expose upstream details: %q", err)
	}
}

func TestIsValidChineseMobile(t *testing.T) {
	cases := []struct {
		phone string
		want  bool
	}{
		{"13800138000", true},
		{"19912345678", true},
		{"16600000000", true},
		{"", false},
		{" 13800138000 ", false},
		{"12800138000", false},
		{"1380013800", false},
		{"138001380000", false},
		{"1380013800a", false},
		{"+8613800138000", false},
	}
	for _, tc := range cases {
		t.Run(tc.phone, func(t *testing.T) {
			if got := isValidChineseMobile(tc.phone); got != tc.want {
				t.Fatalf("isValidChineseMobile(%q)=%v want %v", tc.phone, got, tc.want)
			}
		})
	}
}
