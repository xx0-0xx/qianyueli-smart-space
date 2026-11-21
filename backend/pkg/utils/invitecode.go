package utils

import (
	"crypto/rand"
	"math/big"
)

// inviteAlphabet 邀请码字符表 — 去掉易混淆字符 0/O/1/I/L
const inviteAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// GenInviteCode 生成 10 位邀请码（约 31^10 种组合，降低在线穷举风险）
func GenInviteCode() string {
	b := make([]byte, 10)
	max := big.NewInt(int64(len(inviteAlphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			// 极端情况下退化为非安全随机（邀请码非密钥，可接受）
			b[i] = inviteAlphabet[0]
			continue
		}
		b[i] = inviteAlphabet[n.Int64()]
	}
	return string(b)
}
