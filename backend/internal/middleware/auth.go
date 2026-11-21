package middleware

import (
	"net/http"
	"strings"

	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"
	"smart-space-management/backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// Auth JWT 鉴权中间件
func Auth(jwtMgr *utils.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			response.FailCode(c, errcode.ErrUnauthorized)
			return
		}

		claims, err := jwtMgr.ParseToken(token)
		if err != nil {
			response.FailCode(c, errcode.ErrTokenInvalid)
			return
		}

		// 将用户信息注入上下文
		c.Set("user_id", claims.UserID)
		c.Set("store_id", claims.StoreID)
		c.Set("user_role", claims.Role)
		c.Set("client", claims.Client)
		c.Next()
	}
}

// AdminOnly 超级管理员权限校验
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("user_role")
		if role != "admin" && role != "super_admin" {
			response.FailCode(c, errcode.ErrPermissionDenied)
			return
		}
		c.Next()
	}
}

// MerchantOnly 商家权限校验
func MerchantOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("user_role")
		if role != "merchant" && role != "admin" && role != "super_admin" {
			response.FailCode(c, errcode.ErrPermissionDenied)
			return
		}
		c.Next()
	}
}

// extractToken 从 Authorization header 提取 Bearer token。
// 仅 WebSocket 升级请求允许通过 ?token= 传递，避免 token 进入访问日志/Referer。
func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		if c.Request.URL.Path == "/ws" || strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
			auth = c.Query("token")
		}
		if auth == "" {
			return ""
		}
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	// 无 Bearer 前缀时直接返回
	return auth
}

// CORS 跨域中间件。
// allowedOrigins 为空时回退为放行所有来源（开发环境）；否则仅回显白名单内的 Origin。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll := len(allowedOrigins) == 0
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[strings.TrimRight(strings.ToLower(o), "/")] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		c.Header("Vary", "Origin")

		switch {
		case origin == "":
			// 非浏览器跨域请求（小程序、服务端），无需 CORS 头
		case allowAll:
			c.Header("Access-Control-Allow-Origin", "*")
		case allowed[strings.TrimRight(strings.ToLower(origin), "/")]:
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Requested-With,X-Store-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
