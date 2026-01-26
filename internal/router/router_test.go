package router

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/sirupsen/logrus"
)

// mockHandler 是 Handler 接口的 mock 实现
type mockHandler struct {
	method      string
	handleFunc  func(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error)
	shouldError bool
}

func (m *mockHandler) Method() string {
	return m.method
}

func (m *mockHandler) Handle(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock handler error")
	}
	if m.handleFunc != nil {
		return m.handleFunc(ctx, request)
	}
	return jsonrpc.NewResponse(request.ID, "mock_result")
}

func TestRouter_Register(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	handler := &mockHandler{method: "test_method"}

	err := router.Register(handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// 验证重复注册失败
	err = router.Register(handler)
	if err == nil {
		t.Error("Expected error for duplicate registration, got nil")
	}
}

func TestRouter_Register_EmptyMethod(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	handler := &mockHandler{method: ""}

	err := router.Register(handler)
	if err == nil {
		t.Error("Expected error for empty method name, got nil")
	}
}

func TestRouter_Route(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	// 注册 mock 处理器
	handler := &mockHandler{
		method: "test_method",
		handleFunc: func(ctx context.Context, req *jsonrpc.Request) (*jsonrpc.Response, error) {
			return jsonrpc.NewResponse(req.ID, "test_result")
		},
	}

	err := router.Register(handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// 创建测试请求
	request := &jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "test_method",
		ID:      "test_id",
		Params:  json.RawMessage(`["param1", "param2"]`),
	}

	// 路由请求
	response := router.Route(context.Background(), request)

	// 验证响应
	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got: %v", response.Error)
	}

	if response.ID != request.ID {
		t.Errorf("Expected ID %v, got %v", request.ID, response.ID)
	}

	// 验证结果
	var result string
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if result != "test_result" {
		t.Errorf("Expected result 'test_result', got '%s'", result)
	}
}

func TestRouter_Route_MethodNotFound(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	request := &jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "unknown_method",
		ID:      "test_id",
	}

	response := router.Route(context.Background(), request)

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error == nil {
		t.Fatal("Expected error response")
	}

	if response.Error.Code != jsonrpc.CodeMethodNotFound {
		t.Errorf("Expected error code %d, got %d", jsonrpc.CodeMethodNotFound, response.Error.Code)
	}
}

func TestRouter_Route_NilRequest(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	response := router.Route(context.Background(), nil)

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error == nil {
		t.Fatal("Expected error response")
	}

	if response.Error.Code != jsonrpc.CodeInvalidRequest {
		t.Errorf("Expected error code %d, got %d", jsonrpc.CodeInvalidRequest, response.Error.Code)
	}
}

func TestRouter_Route_HandlerError(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	// 注册会出错的处理器
	handler := &mockHandler{
		method:      "error_method",
		shouldError: true,
	}

	err := router.Register(handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	request := &jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "error_method",
		ID:      "test_id",
	}

	response := router.Route(context.Background(), request)

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error == nil {
		t.Fatal("Expected error response")
	}

	if response.Error.Code != jsonrpc.CodeInternalError {
		t.Errorf("Expected error code %d, got %d", jsonrpc.CodeInternalError, response.Error.Code)
	}
}

func TestRouter_Route_JSONRPCError(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	// 注册返回 JSON-RPC 错误的处理器
	handler := &mockHandler{
		method: "jsonrpc_error_method",
		handleFunc: func(ctx context.Context, req *jsonrpc.Request) (*jsonrpc.Response, error) {
			return nil, &jsonrpc.Error{
				Code:    jsonrpc.CodeInvalidParams,
				Message: "Invalid parameters",
			}
		},
	}

	err := router.Register(handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	request := &jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "jsonrpc_error_method",
		ID:      "test_id",
	}

	response := router.Route(context.Background(), request)

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error == nil {
		t.Fatal("Expected error response")
	}

	if response.Error.Code != jsonrpc.CodeInvalidParams {
		t.Errorf("Expected error code %d, got %d", jsonrpc.CodeInvalidParams, response.Error.Code)
	}
}

func TestRouter_RouteBatch(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	// 注册处理器
	handler := &mockHandler{
		method: "batch_method",
		handleFunc: func(ctx context.Context, req *jsonrpc.Request) (*jsonrpc.Response, error) {
			return jsonrpc.NewResponse(req.ID, fmt.Sprintf("result_for_%v", req.ID))
		},
	}

	err := router.Register(handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// 创建批量请求
	requests := []jsonrpc.Request{
		{JSONRPC: "2.0", Method: "batch_method", ID: "id1"},
		{JSONRPC: "2.0", Method: "batch_method", ID: "id2"},
		{JSONRPC: "2.0", Method: "batch_method", ID: "id3"},
	}

	responses := router.RouteBatch(context.Background(), requests)

	if len(responses) != len(requests) {
		t.Errorf("Expected %d responses, got %d", len(requests), len(responses))
	}

	for i, response := range responses {
		if response == nil {
			t.Errorf("Response %d is nil", i)
			continue
		}

		if response.Error != nil {
			t.Errorf("Response %d has unexpected error: %v", i, response.Error)
			continue
		}

		expectedID := requests[i].ID
		if response.ID != expectedID {
			t.Errorf("Response %d: expected ID %v, got %v", i, expectedID, response.ID)
		}
	}
}

func TestRouter_RouteBatch_Empty(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	responses := router.RouteBatch(context.Background(), []jsonrpc.Request{})

	if len(responses) != 1 {
		t.Errorf("Expected 1 response, got %d", len(responses))
	}

	if responses[0].Error == nil {
		t.Fatal("Expected error response for empty batch")
	}

	if responses[0].Error.Code != jsonrpc.CodeInvalidRequest {
		t.Errorf("Expected error code %d, got %d", jsonrpc.CodeInvalidRequest, responses[0].Error.Code)
	}
}

func TestRouter_GetRegisteredMethods(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	methods := []string{"method1", "method2", "method3"}

	for _, method := range methods {
		handler := &mockHandler{method: method}
		err := router.Register(handler)
		if err != nil {
			t.Fatalf("Failed to register handler %s: %v", method, err)
		}
	}

	registeredMethods := router.GetRegisteredMethods()

	if len(registeredMethods) != len(methods) {
		t.Errorf("Expected %d methods, got %d", len(methods), len(registeredMethods))
	}

	// 验证所有方法都在列表中
	methodMap := make(map[string]bool)
	for _, method := range registeredMethods {
		methodMap[method] = true
	}

	for _, expectedMethod := range methods {
		if !methodMap[expectedMethod] {
			t.Errorf("Method %s not found in registered methods", expectedMethod)
		}
	}
}

func TestRouter_HasHandler(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	handler := &mockHandler{method: "test_method"}
	err := router.Register(handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	if !router.HasHandler("test_method") {
		t.Error("Expected HasHandler to return true for registered method")
	}

	if router.HasHandler("unknown_method") {
		t.Error("Expected HasHandler to return false for unregistered method")
	}
}

func TestRouter_Unregister(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	handler := &mockHandler{method: "test_method"}
	err := router.Register(handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	if !router.HasHandler("test_method") {
		t.Error("Expected method to be registered")
	}

	router.Unregister("test_method")

	if router.HasHandler("test_method") {
		t.Error("Expected method to be unregistered")
	}
}

func TestRouter_MaxRequestSize(t *testing.T) {
	logger := logrus.New()
	router := NewRouterWithMaxSize(logger, 1024) // 1KB limit for testing

	// Register handler
	handler := &mockHandler{method: "test_method"}
	if err := router.Register(handler); err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	tests := []struct {
		name          string
		requestSize   int
		expectSuccess bool
	}{
		{
			name:          "request within limit (512 bytes)",
			requestSize:   512,
			expectSuccess: true,
		},
		{
			name:          "request exceeds limit (2KB)",
			requestSize:   2048,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			body := make([]byte, tt.requestSize)
			for i := range body {
				body[i] = ' '
			}
			jsonReq := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"test_method","params":[]}%s`, string(body))

			// Create HTTP request
			req, err := http.NewRequest("POST", "/", strings.NewReader(jsonReq))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Handle request
			router.HandleHTTPRequest(w, req)

			// Check response
			resp := w.Result()
			defer func() {
				_ = resp.Body.Close()
			}()

			if tt.expectSuccess {
				if resp.StatusCode != http.StatusOK {
					bodyBytes, _ := io.ReadAll(resp.Body)
					t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(bodyBytes))
				}
			} else {
				if resp.StatusCode != http.StatusRequestEntityTooLarge {
					bodyBytes, _ := io.ReadAll(resp.Body)
					t.Errorf("Expected status 413, got %d. Body: %s", resp.StatusCode, string(bodyBytes))
				}
			}
		})
	}
}

func TestRouter_RouteAndRouteWithContext(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	handler := &mockHandler{
		method: "test_method",
		handleFunc: func(ctx context.Context, req *jsonrpc.Request) (*jsonrpc.Response, error) {
			return jsonrpc.NewResponse(req.ID, "test_result")
		},
	}
	if err := router.Register(handler); err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	tests := []struct {
		name    string
		method  string
		wantErr bool
	}{
		{
			name:    "registered method",
			method:  "test_method",
			wantErr: false,
		},
		{
			name:    "unregistered method",
			method:  "unknown_method",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &jsonrpc.Request{
				JSONRPC: "2.0",
				Method:  tt.method,
				ID:      "test_id",
			}

			ctx := context.Background()
			loggerEntry := logger.WithField("test", "value")

			response1 := router.Route(ctx, request)
			response2 := router.RouteWithContext(ctx, request, loggerEntry)

			if response1.Error != nil && response2.Error != nil {
				if response1.Error.Code != response2.Error.Code {
					t.Errorf("Error codes mismatch: Route=%v, RouteWithContext=%v", response1.Error.Code, response2.Error.Code)
				}
				if response1.Error.Message != response2.Error.Message {
					t.Errorf("Error messages mismatch: Route=%v, RouteWithContext=%v", response1.Error.Message, response2.Error.Message)
				}
			} else if response1.Error != nil || response2.Error != nil {
				t.Errorf("One method returned error, the other didn't: Route error=%v, RouteWithContext error=%v", response1.Error, response2.Error)
			}

			if tt.wantErr {
				if response1.Error == nil {
					t.Error("Route: expected error, got nil")
				}
				if response2.Error == nil {
					t.Error("RouteWithContext: expected error, got nil")
				}
			} else {
				if response1.Error != nil {
					t.Errorf("Route: unexpected error: %v", response1.Error)
				}
				if response2.Error != nil {
					t.Errorf("RouteWithContext: unexpected error: %v", response2.Error)
				}
			}

			if response1.ID != response2.ID {
				t.Errorf("Response IDs mismatch: Route=%v, RouteWithContext=%v", response1.ID, response2.ID)
			}
		})
	}
}
