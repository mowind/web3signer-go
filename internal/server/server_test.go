package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mowind/web3signer-go/internal/config"
	"github.com/mowind/web3signer-go/internal/router"
)

func TestBuilder_setGinMode(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		expected string
	}{
		{
			name:     "debug mode",
			logLevel: config.LogLevelDebug,
			expected: gin.DebugMode,
		},
		{
			name:     "info mode",
			logLevel: config.LogLevelInfo,
			expected: gin.ReleaseMode,
		},
		{
			name:     "error mode",
			logLevel: config.LogLevelError,
			expected: gin.ReleaseMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			cfg := &config.Config{
				Log: config.LogConfig{Level: tt.logLevel},
			}
			builder := NewBuilder(cfg)
			builder.setGinMode()

			mode := gin.Mode()
			if mode != tt.expected {
				t.Errorf("Expected gin mode %s, got %s", tt.expected, mode)
			}
		})
	}
}

func TestBuilder_createLogger(t *testing.T) {
	cfg := &config.Config{
		Log: config.LogConfig{Level: config.LogLevelDebug},
	}
	builder := NewBuilder(cfg)
	logger := builder.createLogger()

	if logger == nil {
		t.Fatal("Expected logger but got nil")
	}
}

func TestBuilder_createGinRouter(t *testing.T) {
	cfg := &config.Config{
		Log: config.LogConfig{Level: config.LogLevelDebug},
	}
	builder := NewBuilder(cfg)

	router := builder.createGinRouter(nil, nil)

	if router == nil {
		t.Fatal("Expected router but got nil")
	}
}

func TestBuilder_healthHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Log: config.LogConfig{Level: config.LogLevelDebug},
	}
	builder := NewBuilder(cfg)

	router := gin.New()
	router.GET("/health", builder.healthHandler(builder.createLogger()))

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	response := make(map[string]interface{})
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestBuilder_readyHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Log: config.LogConfig{Level: config.LogLevelDebug},
	}
	builder := NewBuilder(cfg)

	router := gin.New()
	router.GET("/ready", builder.readyHandler(builder.createLogger()))

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	response := make(map[string]interface{})
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("Expected status 'ready', got %v", response["status"])
	}
}

func TestBuilder_createGinRouter_healthHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Log: config.LogConfig{Level: config.LogLevelDebug},
	}
	builder := NewBuilder(cfg)

	router := builder.createGinRouter(nil, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	response := make(map[string]interface{})
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestBuilder_createGinRouter_readyHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Log: config.LogConfig{Level: config.LogLevelDebug},
	}
	builder := NewBuilder(cfg)

	router := builder.createGinRouter(nil, nil)

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	response := make(map[string]interface{})
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("Expected status 'ready', got %v", response["status"])
	}
}

func TestBuilder_createGinRouter_handleJSONRPCRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Log: config.LogConfig{Level: config.LogLevelDebug},
	}

	builder := NewBuilder(cfg)
	routerFactory := router.NewRouterFactory(builder.createLogger())
	jsonRPCRouter := routerFactory.CreateSimpleRouter()

	router := builder.createGinRouter(jsonRPCRouter, nil)

	t.Run("valid JSON-RPC request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{"jsonrpc":"2.0","method":"test","id":1}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("invalid JSON-RPC request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(`{invalid json}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("CORS preflight request", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}

		corsHeader := w.Header().Get("Access-Control-Allow-Origin")
		if corsHeader != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin *, got %s", corsHeader)
		}
	})
}

func TestBuilder_createGinRouter_Build(t *testing.T) {
	mockDownstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      uint64(1),
			"result":  "0x1",
		}); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}))
	defer mockDownstream.Close()

	cfg := &config.Config{
		KMS: config.KMSConfig{
			Endpoint:    "http://localhost:8080",
			AccessKeyID: "ak",
			SecretKey:   "sk",
			KeyID:       "key123",
			Address:     "0x1234567890123456789012345678901234567890",
		},
		Downstream: config.DownstreamConfig{
			HTTPHost: mockDownstream.URL,
			HTTPPort: 0,
			HTTPPath: "/",
		},
		Log: config.LogConfig{Level: config.LogLevelError},
	}

	builder := NewBuilder(cfg)
	server := builder.Build()

	if server == nil {
		t.Fatal("Expected server but got nil")
	}

	if server.router == nil {
		t.Error("Expected router to be initialized")
	}

	if server.logger == nil {
		t.Error("Expected logger to be initialized")
	}

	if server.jsonRPCRouter == nil {
		t.Error("Expected jsonRPCRouter to be initialized")
	}

	if server.kmsAddress == "" {
		t.Error("Expected kmsAddress to be set")
	}
}
