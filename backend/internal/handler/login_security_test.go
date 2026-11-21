package handler

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"smart-space-management/backend/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func TestNormalizeInviteCode(t *testing.T) {
	cases := map[string]string{
		"  abcd2345  ": "ABCD2345",
		"abcd2345":     "ABCD2345",
		"":             "",
	}
	for input, expected := range cases {
		if got := normalizeInviteCode(input); got != expected {
			t.Fatalf("normalizeInviteCode(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestSubjectHashStable(t *testing.T) {
	a := subjectHash("ABCD2345")
	b := subjectHash("ABCD2345")
	if a != b {
		t.Fatalf("同一登录主体的哈希应稳定: %q != %q", a, b)
	}
	if len(a) != 64 {
		t.Fatalf("SHA-256 十六进制长度应为 64，实际 %d", len(a))
	}
	if subjectHash("ABCD2345") == subjectHash("ABCD2346") {
		t.Fatal("不同登录主体不应产生相同哈希")
	}
	if subjectHash(" Admin ") != subjectHash("admin") {
		t.Fatal("登录主体应忽略大小写和首尾空格")
	}
}

func TestCaptchaHelpers(t *testing.T) {
	for i := 0; i < 100; i++ {
		n := randomInt(1, 9)
		if n < 1 || n > 9 {
			t.Fatalf("randomInt 超出范围: %d", n)
		}
	}
	id := randomHex(16)
	if len(id) != 32 {
		t.Fatalf("randomHex(16) 长度应为 32，实际 %d", len(id))
	}
	if strings.Trim(id, "0123456789abcdef") != "" {
		t.Fatalf("randomHex 含非十六进制字符: %q", id)
	}
}

func newLoginGuardTest(t *testing.T) (*LoginGuard, *miniredis.Miniredis) {
	t.Helper()
	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("启动 miniredis 失败: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	cfg := config.LoginSecurityConfig{
		Window:          time.Minute,
		IPMaxFailures:   10,
		CodeMaxFailures: 4,
		LockTTL:         time.Minute,
		CaptchaAfter:    2,
		CaptchaTTL:      time.Minute,
	}
	return NewLoginGuard(client, cfg), server
}

func TestLoginFailureLockAndClear(t *testing.T) {
	guard, server := newLoginGuardTest(t)
	defer server.Close()

	ctx := context.Background()
	ip := "203.0.113.10"
	subject := "ABCD2345"

	guard.RecordFailure(ctx, LoginScopeInvite, ip, subject)
	if guard.NeedsCaptcha(ctx, LoginScopeInvite, ip, subject) {
		t.Fatal("1 次失败不应触发验证码")
	}
	guard.RecordFailure(ctx, LoginScopeInvite, ip, subject)
	if !guard.NeedsCaptcha(ctx, LoginScopeInvite, ip, subject) {
		t.Fatal("2 次失败应触发验证码")
	}

	guard.RecordFailure(ctx, LoginScopeInvite, ip, subject)
	guard.RecordFailure(ctx, LoginScopeInvite, ip, subject)
	if !guard.IsLocked(ctx, LoginScopeInvite, ip, subject) {
		t.Fatal("达到失败上限后登录主体应被锁定")
	}

	guard.Clear(ctx, LoginScopeInvite, ip, subject)
	if guard.IsLocked(ctx, LoginScopeInvite, ip, subject) {
		t.Fatal("登录成功后应清除锁定")
	}
	if guard.NeedsCaptcha(ctx, LoginScopeInvite, ip, subject) {
		t.Fatal("登录成功后应清除失败计数")
	}
}

func TestAdminScopeIsIsolated(t *testing.T) {
	guard, server := newLoginGuardTest(t)
	defer server.Close()

	ctx := context.Background()
	ip := "203.0.113.10"
	subject := "admin"

	for i := 0; i < 4; i++ {
		guard.RecordFailure(ctx, LoginScopeAdmin, ip, subject)
	}
	if !guard.IsLocked(ctx, LoginScopeAdmin, ip, subject) {
		t.Fatal("管理员登录达到失败上限后应被锁定")
	}
	if guard.IsLocked(ctx, LoginScopeInvite, ip, subject) {
		t.Fatal("管理员锁定不应影响邀请码登录")
	}
}

func TestCaptchaChallengeIsOneTime(t *testing.T) {
	guard, server := newLoginGuardTest(t)
	defer server.Close()

	ctx := context.Background()
	challenge := guard.NewCaptcha(ctx)
	if challenge == nil {
		t.Fatal("验证码生成失败")
	}
	question := challenge["question"]
	id := challenge["captcha_id"]
	if id == "" || question == "" {
		t.Fatalf("验证码内容不完整: %#v", challenge)
	}

	parts := strings.Fields(question)
	if len(parts) < 5 {
		t.Fatalf("验证码格式异常: %q", question)
	}
	a, _ := strconv.Atoi(parts[0])
	b, _ := strconv.Atoi(parts[2])
	answer := strconv.Itoa(a + b)

	if guard.VerifyCaptcha(ctx, id, "wrong") {
		t.Fatal("错误验证码不应通过")
	}
	// 错误尝试会消费验证码，需要重新生成
	challenge = guard.NewCaptcha(ctx)
	parts = strings.Fields(challenge["question"])
	a, _ = strconv.Atoi(parts[0])
	b, _ = strconv.Atoi(parts[2])
	answer = strconv.Itoa(a + b)
	if !guard.VerifyCaptcha(ctx, challenge["captcha_id"], answer) {
		t.Fatal("正确验证码应通过")
	}
	if guard.VerifyCaptcha(ctx, challenge["captcha_id"], answer) {
		t.Fatal("验证码只能使用一次")
	}
}
