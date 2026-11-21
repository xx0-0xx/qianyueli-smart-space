package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims 标准 JWT 载荷
type JWTClaims struct {
	UserID  uint64 `json:"uid"`
	StoreID uint64 `json:"sid,omitempty"`    // 商家绑定的门店 ID
	Role    string `json:"role"`             // user | merchant | admin | super_admin
	Client  string `json:"client,omitempty"` // user | merchant | admin：签发该 token 的客户端
	jwt.RegisteredClaims
}

// JWTManager JWT 令牌管理
type JWTManager struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewJWTManager(secret string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		secret:          []byte(secret),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

// GenerateAccessToken 生成访问令牌
func (m *JWTManager) GenerateAccessToken(claims JWTClaims) (string, error) {
	now := time.Now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
		Issuer:    "smart-space",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// GenerateRefreshToken 生成刷新令牌
func (m *JWTManager) GenerateRefreshToken(claims JWTClaims) (string, error) {
	now := time.Now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTokenTTL)),
		Issuer:    "smart-space-refresh",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseRefreshToken 解析并校验刷新令牌（issuer 必须为 smart-space-refresh）
func (m *JWTManager) ParseRefreshToken(tokenStr string) (*JWTClaims, error) {
	claims, err := m.ParseToken(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.Issuer != "smart-space-refresh" {
		return nil, fmt.Errorf("not a refresh token")
	}
	return claims, nil
}

// ParseToken 解析并验证令牌
func (m *JWTManager) ParseToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
