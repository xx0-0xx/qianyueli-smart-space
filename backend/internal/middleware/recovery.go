package middleware

import (
	"log"
	"net"
	"os"
	"runtime/debug"
	"time"

	"smart-space-management/backend/pkg/errcode"
	"smart-space-management/backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// Recovery panic 恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// 检查是否为 broken pipe
				if ne, ok := r.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if se.Err.Error() == "broken pipe" {
							c.Abort()
							return
						}
					}
				}

				stack := string(debug.Stack())
				log.Printf("[PANIC] %v\n%s", r, stack)

				response.FailCode(c, errcode.ErrInternal)
			}
		}()
		c.Next()
	}
}

// Logger 请求日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		status := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		ip := c.ClientIP()

		log.Printf("[HTTP] %s | %3d | %12v | %15s | %s",
			time.Now().Format("2006-01-02 15:04:05"),
			status,
			latency,
			ip,
			method+" "+path,
		)

		// 慢查询告警
		if latency > time.Second {
			log.Printf("[SLOW] %s %s took %v", method, path, latency)
		}
	}
}

// RequestID 为每个请求注入唯一 ID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = newRequestID()
		}
		c.Set("request_id", rid)
		c.Header("X-Request-ID", rid)
		c.Next()
	}
}

func newRequestID() string {
	return time.Now().Format("20060102150405") + random4()
}

func random4() string {
	b := make([]byte, 4)
	for i := range b {
		b[i] = byte(time.Now().UnixNano()%26) + 'A'
	}
	return string(b)
}
