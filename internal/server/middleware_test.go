package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test-secret"
	whitelist := []string{"/", "/health", "/ready"}

	tests := []struct {
		name           string
		enabled        bool
		path           string
		authHeader     string
		apiKeyHeader   string
		expectedStatus int
	}{
		{
			name:           "disabled auth - passes without credentials",
			enabled:        false,
			path:           "/",
			authHeader:     "",
			apiKeyHeader:   "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "whitelisted path - passes without credentials",
			enabled:        true,
			path:           "/health",
			authHeader:     "",
			apiKeyHeader:   "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "whitelisted root path - passes without credentials",
			enabled:        true,
			path:           "/",
			authHeader:     "",
			apiKeyHeader:   "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "whitelisted ready path - passes without credentials",
			enabled:        true,
			path:           "/ready",
			authHeader:     "",
			apiKeyHeader:   "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid Bearer token - passes",
			enabled:        true,
			path:           "/eth_accounts",
			authHeader:     "Bearer " + secret,
			apiKeyHeader:   "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid API Key - passes",
			enabled:        true,
			path:           "/eth_accounts",
			authHeader:     "",
			apiKeyHeader:   secret,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid Bearer token - fails",
			enabled:        true,
			path:           "/eth_accounts",
			authHeader:     "Bearer wrong-secret",
			apiKeyHeader:   "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid API Key - fails",
			enabled:        true,
			path:           "/eth_accounts",
			authHeader:     "",
			apiKeyHeader:   "wrong-secret",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing auth - fails",
			enabled:        true,
			path:           "/eth_accounts",
			authHeader:     "",
			apiKeyHeader:   "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "malformed Bearer token - fails",
			enabled:        true,
			path:           "/eth_accounts",
			authHeader:     "Bearer",
			apiKeyHeader:   "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "wrong scheme in Authorization - fails",
			enabled:        true,
			path:           "/eth_accounts",
			authHeader:     "Basic " + secret,
			apiKeyHeader:   "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware(tt.enabled, secret, whitelist))

			router.Any("/*path", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			})

			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.apiKeyHeader != "" {
				req.Header.Set("X-API-Key", tt.apiKeyHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
				t.Errorf("response body: %s", w.Body.String())
			}

			if tt.expectedStatus == http.StatusUnauthorized {
				body := w.Body.String()
				// Verify generic error message (no sensitive information leakage)
				if !strings.Contains(body, "authentication failed") {
					t.Errorf("expected 'authentication failed' in response body, got: %s", body)
				}
				// Verify no sensitive information is leaked
				if strings.Contains(body, "missing") || strings.Contains(body, "invalid") || strings.Contains(body, "token") {
					t.Errorf("response body should not leak sensitive info: %s", body)
				}
			}
		})
	}
}

func TestAuthMiddleware_Precedence(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test-secret"
	whitelist := []string{"/health"}

	tests := []struct {
		name           string
		path           string
		authHeader     string
		apiKeyHeader   string
		expectedStatus int
	}{
		{
			name:           "Bearer token takes precedence over API Key when both valid",
			path:           "/eth_accounts",
			authHeader:     "Bearer " + secret,
			apiKeyHeader:   secret,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "whitelist bypasses auth even with invalid credentials",
			path:           "/health",
			authHeader:     "Bearer invalid",
			apiKeyHeader:   "invalid",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware(true, secret, whitelist))

			router.Any("/*path", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			})

			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.apiKeyHeader != "" {
				req.Header.Set("X-API-Key", tt.apiKeyHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestAuthMiddleware_PathMatching(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test-secret"
	whitelist := []string{"/health", "/metrics"}

	tests := []struct {
		name           string
		path           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "exact whitelist match",
			path:           "/health",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "whitelist prefix match",
			path:           "/metrics/detailed",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "non-whitelisted path requires auth",
			path:           "/api/v1/data",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware(true, secret, whitelist))

			router.Any("/*path", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			})

			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestAuthMiddleware_EmptyWhitelist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := "test-secret"
	whitelist := []string{}

	router := gin.New()
	router.Use(AuthMiddleware(true, secret, whitelist))

	router.Any("/*path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d with empty whitelist, got %d", http.StatusUnauthorized, w.Code)
	}
}
