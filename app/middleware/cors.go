package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AllowedOrigins 允许跨域的域名列表
var AllowedOrigins = []string{
	"https://www.mingdao.com",
	"https://mingdao.com",
	"https://bjx.cloud",
}

// CORS 返回 CORS 中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查是否是允许的域名
		allowed := false
		for _, o := range AllowedOrigins {
			if strings.EqualFold(origin, o) || strings.HasSuffix(origin, ".mingdao.com") {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
			c.Header("Access-Control-Max-Age", "86400")
		}

		// 处理预检请求
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
