package utils

import (
	"strings"
	"testing"
)

func TestGenInviteCodeLengthAndAlphabet(t *testing.T) {
	allowed := make(map[rune]bool, len(inviteAlphabet))
	for _, ch := range inviteAlphabet {
		allowed[ch] = true
	}

	for i := 0; i < 200; i++ {
		code := GenInviteCode()
		if len(code) != 10 {
			t.Fatalf("邀请码长度应为 10，实际为 %d: %q", len(code), code)
		}
		for _, ch := range code {
			if !allowed[ch] {
				t.Fatalf("邀请码包含非法字符 %q: %q", ch, code)
			}
		}
		if strings.ContainsAny(code, "0O1IL") {
			t.Fatalf("邀请码包含易混淆字符: %q", code)
		}
	}
}
