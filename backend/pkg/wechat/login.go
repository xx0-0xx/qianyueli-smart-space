package wechat

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// MiniProgram 微信小程序 API 封装
type MiniProgram struct {
	AppID     string
	AppSecret string
	Redis     *redis.Client
}

func NewMiniProgram(appID, appSecret string) *MiniProgram {
	return &MiniProgram{AppID: appID, AppSecret: appSecret}
}

// SessionResult code2session 返回结果
type SessionResult struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	SessionKey string `json:"session_key"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// DecryptPhone 解密手机号 (微信手机号快速验证)
func (mp *MiniProgram) DecryptPhone(encryptedData, sessionKey, iv string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("base64 decode encryptedData: %w", err)
	}
	ivBytes, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return "", fmt.Errorf("base64 decode iv: %w", err)
	}
	keyBytes, err := base64.StdEncoding.DecodeString(sessionKey)
	if err != nil {
		return "", fmt.Errorf("base64 decode sessionKey: %w", err)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	mode := cipher.NewCBCDecrypter(block, ivBytes)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// PKCS7 unpad
	plaintext = pkcs7Unpad(plaintext)

	var result struct {
		PhoneNumber string `json:"phoneNumber"`
	}
	if err := json.Unmarshal(plaintext, &result); err != nil {
		return "", err
	}
	return result.PhoneNumber, nil
}

func pkcs7Unpad(data []byte) []byte {
	length := len(data)
	if length == 0 {
		return data
	}
	padding := int(data[length-1])
	if padding > length {
		return data
	}
	return data[:length-padding]
}
