package server

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware authenticates requests using JWT Bearer tokens or X-API-Key headers.
func AuthMiddleware(enabled bool, secret string, whitelist []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		// 安全的前缀匹配：路径必须精确匹配白名单项，或者是其子路径（以 '/' 分隔）
		// 例如：/api 匹配 /api 和 /api/sub，但不匹配 /api-private 或 /apiadmin
		for _, whitelistedPath := range whitelist {
			// 根路径 "/" 只精确匹配自身
			if whitelistedPath == "/" {
				if path == "/" {
					c.Next()
					return
				}
				continue
			}

			// 安全前缀匹配非根路径
			if len(path) >= len(whitelistedPath) &&
				path[:len(whitelistedPath)] == whitelistedPath &&
				(len(path) == len(whitelistedPath) || path[len(whitelistedPath)] == '/') {
				// 路径在白名单中，跳过认证
				c.Next()
				return
			}
		}

		// Check Authorization header (Bearer token or JWT)
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			// Use constant-time comparison to prevent timing attacks
			if len(parts) == 2 && parts[0] == "Bearer" && subtle.ConstantTimeCompare([]byte(parts[1]), []byte(secret)) == 1 {
				c.Next()
				return
			}
			// Return generic error message to avoid information leakage
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authentication failed",
				"code":  http.StatusUnauthorized,
			})
			return
		}

		// Check X-API-Key header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			// Use constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(apiKey), []byte(secret)) == 1 {
				c.Next()
				return
			}
			// Return generic error message to avoid information leakage
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authentication failed",
				"code":  http.StatusUnauthorized,
			})
			return
		}

		// No valid authentication provided
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "authentication failed",
			"code":  http.StatusUnauthorized,
		})
	}
}
