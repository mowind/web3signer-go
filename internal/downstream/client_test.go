package downstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/sirupsen/logrus"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.DownstreamConfig
		shouldError bool
	}{
		{
			name: "valid HTTP config",
			config: &config.DownstreamConfig{
				HTTPHost: "http://localhost",
				HTTPPort: 8545,
				HTTPPath: "/api",
			},
			shouldError: false,
		},
		{
			name: "valid HTTPS config",
			config: &config.DownstreamConfig{
				HTTPHost: "https://api.example.com",
				HTTPPort: 443,
				HTTPPath: "/jsonrpc",
			},
			shouldError: false,
		},
		{
			name: "config with port in host",
			config: &config.DownstreamConfig{
				HTTPHost: "http://localhost:8545",
				HTTPPort: 0, // 端口已经在host中
				HTTPPath: "/",
			},
			shouldError: false,
		},
		{
			name: "config without port",
			config: &config.DownstreamConfig{
				HTTPHost: "https://api.example.com",
				HTTPPort: 0, // 不使用端口
				HTTPPath: "/api/v1",
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 复制配置并验证
			configCopy := *tt.config
			if err := configCopy.Validate(); err != nil {
				t.Fatalf("Config validation failed: %v", err)
			}

			client := NewClient(&configCopy, logrus.New())
			if client == nil {
				t.Fatal("NewClient returned nil")
			}
			if client.config != &configCopy {
				t.Error("Config not set correctly")
			}
			if client.httpClient == nil {
				t.Error("HTTP client not initialized")
			}
		})
	}
}

func TestClient_ForwardRequest(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求头
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got: %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept: application/json, got: %s", r.Header.Get("Accept"))
		}

		// 读取请求体
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// 解析请求
		var req jsonrpc.Request
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("Failed to unmarshal request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// 验证请求
		if req.JSONRPC != "2.0" {
			t.Errorf("Expected JSONRPC: 2.0, got: %s", req.JSONRPC)
		}
		if req.Method != "eth_blockNumber" {
			t.Errorf("Expected method: eth_blockNumber, got: %s", req.Method)
		}

		// 返回响应
		resp := jsonrpc.Response{
			JSONRPC: "2.0",
			Result:  json.RawMessage(`"0x1234"`),
			ID:      req.ID,
		}

		respData, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respData)
	}))
	defer server.Close()

	// 创建客户端配置
	cfg := &config.DownstreamConfig{
		HTTPHost: server.URL,
		HTTPPort: 0, // 测试服务器URL已经包含端口
		HTTPPath: "/",
	}

	client := newValidatedClient(t, cfg)

	// 创建测试请求
	req := &jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "eth_blockNumber",
		ID:      123,
	}

	// 执行转发
	ctx := context.Background()
	resp, err := client.ForwardRequest(ctx, req)
	if err != nil {
		t.Fatalf("ForwardRequest failed: %v", err)
	}

	// 验证响应
	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC: 2.0, got: %s", resp.JSONRPC)
	}
	if string(resp.Result) != `"0x1234"` {
		t.Errorf("Expected result: \"0x1234\", got: %s", string(resp.Result))
	}
	// ID应该相等
	if !compareIDs(resp.ID, req.ID) {
		t.Errorf("Expected ID: %v, got: %v", req.ID, resp.ID)
	}
}

func TestClient_ForwardBatchRequest(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 读取请求体
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// 解析批量请求
		var requests []jsonrpc.Request
		if err := json.Unmarshal(body, &requests); err != nil {
			t.Errorf("Failed to unmarshal batch request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// 验证请求数量
		if len(requests) != 2 {
			t.Errorf("Expected 2 requests, got: %d", len(requests))
		}

		// 创建批量响应
		responses := []jsonrpc.Response{
			{
				JSONRPC: "2.0",
				Result:  json.RawMessage(`"0x1234"`),
				ID:      requests[0].ID,
			},
			{
				JSONRPC: "2.0",
				Result:  json.RawMessage(`"0x5678"`),
				ID:      requests[1].ID,
			},
		}

		respData, _ := json.Marshal(responses)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respData)
	}))
	defer server.Close()

	// 创建客户端配置
	cfg := &config.DownstreamConfig{
		HTTPHost: server.URL,
		HTTPPort: 0,
		HTTPPath: "/",
	}

	client := newValidatedClient(t, cfg)

	// 创建批量请求
	requests := []jsonrpc.Request{
		{
			JSONRPC: "2.0",
			Method:  "eth_blockNumber",
			ID:      1,
		},
		{
			JSONRPC: "2.0",
			Method:  "eth_chainId",
			ID:      2,
		},
	}

	// 执行转发
	ctx := context.Background()
	responses, err := client.ForwardBatchRequest(ctx, requests)
	if err != nil {
		t.Fatalf("ForwardBatchRequest failed: %v", err)
	}

	// 验证响应
	if len(responses) != 2 {
		t.Fatalf("Expected 2 responses, got: %d", len(responses))
	}
	if string(responses[0].Result) != `"0x1234"` {
		t.Errorf("Expected first result: \"0x1234\", got: %s", string(responses[0].Result))
	}
	if string(responses[1].Result) != `"0x5678"` {
		t.Errorf("Expected second result: \"0x5678\", got: %s", string(responses[1].Result))
	}
}

func TestClient_ForwardRequest_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		expectedError  string
		checkErrorType func(error) bool
	}{
		{
			name: "connection refused",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// 立即关闭连接，模拟连接失败
				hj, ok := w.(http.Hijacker)
				if !ok {
					http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
					return
				}
				conn, _, _ := hj.Hijack()
				_ = conn.Close()
			},
			expectedError:  "failed to connect to downstream service",
			checkErrorType: IsConnectionError,
		},
		{
			name: "non-200 status code",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("Internal Server Error"))
			},
			expectedError:  "request to downstream service failed",
			checkErrorType: func(err error) bool { return !IsConnectionError(err) },
		},
		{
			name: "invalid JSON response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("invalid json"))
			},
			expectedError:  "invalid response from downstream service",
			checkErrorType: IsInvalidResponseError,
		},
		{
			name: "timeout",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// 模拟超时
				time.Sleep(2 * time.Second)
				w.WriteHeader(http.StatusOK)
			},
			expectedError:  "context deadline exceeded",
			checkErrorType: IsConnectionError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试服务器
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			// 创建客户端配置
			cfg := &config.DownstreamConfig{
				HTTPHost: server.URL,
				HTTPPort: 0,
				HTTPPath: "/",
			}

			client := newValidatedClient(t, cfg)

			// 对于超时测试，使用短超时
			ctx := context.Background()
			if tt.name == "timeout" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
			}

			// 创建测试请求
			req := &jsonrpc.Request{
				JSONRPC: "2.0",
				Method:  "eth_blockNumber",
				ID:      1,
			}

			// 执行转发，期望错误
			_, err := client.ForwardRequest(ctx, req)
			if err == nil {
				t.Error("Expected error but got none")
				return
			}

			// 验证错误类型
			if tt.checkErrorType != nil && !tt.checkErrorType(err) {
				t.Errorf("Error type check failed for error: %v", err)
			}

			// 验证错误消息包含预期内容
			if tt.expectedError != "" && !contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error to contain '%s', got: %v", tt.expectedError, err)
			}
		})
	}
}

func TestClient_TestConnection(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回成功响应
		resp := jsonrpc.Response{
			JSONRPC: "2.0",
			Result:  json.RawMessage(`"test"`),
			ID:      1,
		}
		respData, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respData)
	}))
	defer server.Close()

	// 创建客户端配置
	cfg := &config.DownstreamConfig{
		HTTPHost: server.URL,
		HTTPPort: 0,
		HTTPPath: "/",
	}

	client := newValidatedClient(t, cfg)

	// 测试连接
	ctx := context.Background()
	err := client.TestConnection(ctx)
	if err != nil {
		t.Errorf("TestConnection failed: %v", err)
	}
}

func TestClient_GetEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.DownstreamConfig
		expected string
	}{
		{
			name: "HTTP with port",
			config: &config.DownstreamConfig{
				HTTPHost: "http://localhost",
				HTTPPort: 8545,
				HTTPPath: "/api",
			},
			expected: "http://localhost:8545/api",
		},
		{
			name: "HTTPS without port",
			config: &config.DownstreamConfig{
				HTTPHost: "https://api.example.com",
				HTTPPort: 0,
				HTTPPath: "/jsonrpc",
			},
			expected: "https://api.example.com/jsonrpc",
		},
		{
			name: "HTTP with port in host",
			config: &config.DownstreamConfig{
				HTTPHost: "http://localhost:8080",
				HTTPPort: 8545, // 应该被忽略，因为host中已有端口
				HTTPPath: "/",
			},
			expected: "http://localhost:8080/",
		},
		{
			name: "path without leading slash",
			config: &config.DownstreamConfig{
				HTTPHost: "http://localhost",
				HTTPPort: 8545,
				HTTPPath: "api/v1",
			},
			expected: "http://localhost:8545/api/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 复制配置以避免修改原始配置
			configCopy := *tt.config
			// 验证配置（这会修改HTTPPath）
			if err := configCopy.Validate(); err != nil {
				t.Fatalf("Config validation failed: %v", err)
			}
			client := NewClient(&configCopy, logrus.New())
			endpoint := client.GetEndpoint()
			if endpoint != tt.expected {
				t.Errorf("Expected endpoint: %s, got: %s", tt.expected, endpoint)
			}
		})
	}
}

func TestCompareIDs(t *testing.T) {
	tests := []struct {
		name     string
		id1      interface{}
		id2      interface{}
		expected bool
	}{
		{
			name:     "both nil",
			id1:      nil,
			id2:      nil,
			expected: true,
		},
		{
			name:     "one nil",
			id1:      nil,
			id2:      1,
			expected: false,
		},
		{
			name:     "same integer",
			id1:      123,
			id2:      123,
			expected: true,
		},
		{
			name:     "different integers",
			id1:      123,
			id2:      456,
			expected: false,
		},
		{
			name:     "same string",
			id1:      "test",
			id2:      "test",
			expected: true,
		},
		{
			name:     "different strings",
			id1:      "test1",
			id2:      "test2",
			expected: false,
		},
		{
			name:     "integer and string",
			id1:      123,
			id2:      "123",
			expected: true, // 转换为字符串后相等
		},
		{
			name:     "float and integer",
			id1:      123.0,
			id2:      123,
			expected: true, // 转换为字符串后相等
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareIDs(tt.id1, tt.id2)
			if result != tt.expected {
				t.Errorf("compareIDs(%v, %v) = %v, want %v", tt.id1, tt.id2, result, tt.expected)
			}
		})
	}
}

func TestSimpleForwarder(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 返回成功响应
		resp := jsonrpc.Response{
			JSONRPC: "2.0",
			Result:  json.RawMessage(`"0x1234"`),
			ID:      1,
		}
		respData, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respData)
	}))
	defer server.Close()

	// 创建客户端配置
	cfg := &config.DownstreamConfig{
		HTTPHost: server.URL,
		HTTPPort: 0,
		HTTPPath: "/",
	}

	client := newValidatedClient(t, cfg)
	forwarder := NewSimpleForwarder(client)

	// 测试转发
	ctx := context.Background()
	resp, err := forwarder.Forward(ctx, "eth_blockNumber", nil)
	if err != nil {
		t.Fatalf("Forward failed: %v", err)
	}

	// 验证响应
	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC: 2.0, got: %s", resp.JSONRPC)
	}
	if string(resp.Result) != `"0x1234"` {
		t.Errorf("Expected result: \"0x1234\", got: %s", string(resp.Result))
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

// newValidatedClient 创建并验证配置的客户端
func newValidatedClient(t *testing.T, cfg *config.DownstreamConfig) *Client {
	t.Helper()
	// 复制配置以避免修改原始配置
	configCopy := *cfg
	// 验证配置
	if err := configCopy.Validate(); err != nil {
		t.Fatalf("Config validation failed: %v", err)
	}
	return NewClient(&configCopy, logrus.New())
}

// BenchmarkCompareIDs_ToString benchmarks toString function
func BenchmarkCompareIDs_ToString(b *testing.B) {
	id := 12345
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := toString(id)
		_ = s == "12345"
	}
}

// BenchmarkCompareIDs_Sprintf benchmarks fmt.Sprintf function
func BenchmarkCompareIDs_Sprintf(b *testing.B) {
	id := 12345
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := fmt.Sprintf("%v", id)
		_ = s == "12345"
	}
}
