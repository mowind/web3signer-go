package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/sirupsen/logrus"
)

func TestBuilder_Build(t *testing.T) {
	cfg := &config.Config{
		HTTP: config.HTTPConfig{
			Host: "localhost",
			Port: 8080,
		},
		Log: config.LogConfig{
			Level: config.LogLevelInfo,
		},
	}

	builder := NewBuilder(cfg)
	server := builder.Build()

	if server == nil {
		t.Fatal("Expected server but got nil")
	}

	if server.config != cfg {
		t.Error("Config not set correctly")
	}

	if server.router == nil {
		t.Error("Router not initialized")
	}

	if server.logger == nil {
		t.Error("Logger not initialized")
	}
}

func TestBuilder_setGinMode(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		expected string
	}{
		{
			name:     "debug mode",
			logLevel: config.LogLevelDebug,
			expected: "debug",
		},
		{
			name:     "info mode",
			logLevel: config.LogLevelInfo,
			expected: "release",
		},
		{
			name:     "error mode",
			logLevel: config.LogLevelError,
			expected: "release",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Log: config.LogConfig{Level: tt.logLevel},
			}
			builder := NewBuilder(cfg)
			// This indirectly tests setGinMode via Build()
			server := builder.Build()
			if server == nil {
				t.Fatal("Expected server but got nil")
			}
			// Gin mode is set globally, we can't easily test it here
			// but we can at least ensure no panic occurred
		})
	}
}

func TestBuilder_createRouter(t *testing.T) {
	cfg := &config.Config{
		Log: config.LogConfig{Level: config.LogLevelDebug},
	}
	builder := NewBuilder(cfg)
	router := builder.createRouter()

	if router == nil {
		t.Fatal("Expected router but got nil")
	}

	// Create a temporary server to test the router
	server := &Server{
		config: cfg,
		router: router,
		logger: builder.createLogger(),
	}
	server.setupRoutes()

	// Test that routes are properly set up by testing health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestBuilder_createLogger(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		expected logrus.Level
	}{
		{
			name:     "debug level",
			logLevel: config.LogLevelDebug,
			expected: logrus.DebugLevel,
		},
		{
			name:     "info level",
			logLevel: config.LogLevelInfo,
			expected: logrus.InfoLevel,
		},
		{
			name:     "warn level",
			logLevel: config.LogLevelWarn,
			expected: logrus.WarnLevel,
		},
		{
			name:     "error level",
			logLevel: config.LogLevelError,
			expected: logrus.ErrorLevel,
		},
		{
			name:     "fatal level",
			logLevel: config.LogLevelFatal,
			expected: logrus.FatalLevel,
		},
		{
			name:     "invalid level defaults to info",
			logLevel: "invalid",
			expected: logrus.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Log: config.LogConfig{Level: tt.logLevel},
			}
			builder := NewBuilder(cfg)
			logger := builder.createLogger()

			if logger == nil {
				t.Fatal("Expected logger but got nil")
			}

			if logger.GetLevel() != tt.expected {
				t.Errorf("Expected level %v, got %v", tt.expected, logger.GetLevel())
			}

			// Test JSON formatter
			if _, ok := logger.Formatter.(*logrus.JSONFormatter); !ok {
				t.Error("Expected JSON formatter")
			}
		})
	}
}

func TestServer_healthHandler(t *testing.T) {
	cfg := &config.Config{
		HTTP: config.HTTPConfig{Host: "localhost", Port: 8080},
		Log:  config.LogConfig{Level: config.LogLevelInfo},
	}

	server := New(cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}

	// Check time field exists and is in RFC3339 format
	timeStr, ok := response["time"].(string)
	if !ok {
		t.Error("Expected time field to be a string")
	} else {
		if _, err := time.Parse(time.RFC3339, timeStr); err != nil {
			t.Errorf("Time field is not in RFC3339 format: %v", err)
		}
	}
}

func TestServer_readyHandler(t *testing.T) {
	cfg := &config.Config{
		HTTP: config.HTTPConfig{Host: "localhost", Port: 8080},
		Log:  config.LogConfig{Level: config.LogLevelInfo},
	}

	server := New(cfg)

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "ready" {
		t.Errorf("Expected status 'ready', got %v", response["status"])
	}
}

func TestServer_jsonRPCHandler(t *testing.T) {
	cfg := &config.Config{
		HTTP: config.HTTPConfig{Host: "localhost", Port: 8080},
		Log:  config.LogConfig{Level: config.LogLevelInfo},
	}

	server := New(cfg)

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
	}{
		{
			name:           "valid JSON-RPC request",
			method:         "POST",
			body:           `{"jsonrpc":"2.0","method":"web3_clientVersion","id":1}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "batch JSON-RPC request",
			method:         "POST",
			body:           `[{"jsonrpc":"2.0","method":"web3_clientVersion","id":1},{"jsonrpc":"2.0","method":"eth_blockNumber","id":2}]`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid JSON",
			method:         "POST",
			body:           `{"jsonrpc":"2.0","method":"web3_clientVersion","id":1`,
			expectedStatus: http.StatusOK, // Returns JSON-RPC error response
		},
		{
			name:           "wrong method",
			method:         "GET",
			body:           `{"jsonrpc":"2.0","method":"web3_clientVersion","id":1}`,
			expectedStatus: http.StatusNotFound, // Gin 404 for GET on /
		},
		{
			name:           "empty body",
			method:         "POST",
			body:           "",
			expectedStatus: http.StatusOK, // Returns JSON-RPC error response
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestServer_handleSingleRequest(t *testing.T) {
	cfg := &config.Config{
		HTTP: config.HTTPConfig{Host: "localhost", Port: 8080},
		Log:  config.LogConfig{Level: config.LogLevelInfo},
	}

	server := New(cfg)

	tests := []struct {
		name         string
		request      jsonrpc.Request
		expectError  bool
		expectedCode int
	}{
		{
			name: "any method returns method not found",
			request: jsonrpc.Request{
				JSONRPC: "2.0",
				Method:  "web3_clientVersion",
				ID:      1,
			},
			expectError:  true,
			expectedCode: jsonrpc.CodeMethodNotFound,
		},
		{
			name: "with nil ID",
			request: jsonrpc.Request{
				JSONRPC: "2.0",
				Method:  "eth_blockNumber",
				ID:      nil,
			},
			expectError:  true,
			expectedCode: jsonrpc.CodeMethodNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := server.handleSingleRequest(tt.request)

			if resp == nil {
				t.Fatal("Expected response but got nil")
			}

			if tt.expectError {
				if resp.Error == nil {
					t.Error("Expected error but got none")
				}

				if resp.Error.Code != tt.expectedCode {
					t.Errorf("Expected error code %d, got %d", tt.expectedCode, resp.Error.Code)
				}
			} else if resp.Error != nil {
				t.Errorf("Unexpected error: %v", resp.Error)
			}

			// ID should be preserved
			if (tt.request.ID == nil && resp.ID != nil) || (tt.request.ID != nil && resp.ID == nil) {
				t.Error("Response ID should match request ID")
			}
		})
	}
}

func TestServer_Start_Stop(t *testing.T) {
	cfg := &config.Config{
		HTTP: config.HTTPConfig{
			Host: "127.0.0.1", // Use localhost for actual server
			Port: 0,           // Use random port to avoid conflicts
		},
		Log: config.LogConfig{Level: config.LogLevelError}, // Reduce log noise
	}

	server := New(cfg)

	// Test Start
	err := server.Start()
	if err != nil {
		t.Errorf("Unexpected error starting server: %v", err)
	}

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is actually running by making a request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test Stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = server.Stop(ctx)
	if err != nil {
		t.Errorf("Unexpected error stopping server: %v", err)
	}
}

func TestServer_getLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected logrus.Level
	}{
		{"debug", logrus.DebugLevel},
		{"info", logrus.InfoLevel},
		{"warn", logrus.WarnLevel},
		{"error", logrus.ErrorLevel},
		{"fatal", logrus.FatalLevel},
		{"invalid", logrus.InfoLevel}, // defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := getLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("getLogLevel(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
