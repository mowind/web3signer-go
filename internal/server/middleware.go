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
		for _, whitelistedPath := range whitelist {
			if strings.HasPrefix(path, whitelistedPath) {
				if whitelistedPath == "/" && path != "/" {
					continue
				}
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
