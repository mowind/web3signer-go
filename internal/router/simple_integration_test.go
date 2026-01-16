package router

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/sirupsen/logrus"
)

// Simple mock handler for testing individual components
type simpleMockHandler struct {
	method string
	result interface{}
}

func (h *simpleMockHandler) Method() string {
	return h.method
}

func (h *simpleMockHandler) Handle(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error) {
	return jsonrpc.NewResponse(request.ID, h.result)
}

func TestSimpleIntegration_RouterAndHandlers(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建路由器
	router := NewRouter(logger)

	// 注册一些简单的处理器
	handlers := []Handler{
		&simpleMockHandler{method: "eth_sign", result: "mock_signature"},
		&simpleMockHandler{method: "eth_signTransaction", result: "mock_signed_tx"},
		&simpleMockHandler{method: "eth_accounts", result: []string{}},
		&simpleMockHandler{method: "eth_getBalance", result: "0x123456"},
	}

	for _, handler := range handlers {
		err := router.Register(handler)
		if err != nil {
			t.Fatalf("Failed to register handler %s: %v", handler.Method(), err)
		}
	}

	// 测试用例
	testCases := []struct {
		name           string
		method         string
		params         json.RawMessage
		expectError    bool
		expectedResult interface{}
	}{
		{
			name:           "eth_sign",
			method:         "eth_sign",
			params:         json.RawMessage(`["0x123...", "0xabc..."]`),
			expectError:    false,
			expectedResult: "mock_signature",
		},
		{
			name:           "eth_signTransaction",
			method:         "eth_signTransaction",
			params:         json.RawMessage(`[{"from": "0x123...", "to": "0x456..."}]`),
			expectError:    false,
			expectedResult: "mock_signed_tx",
		},
		{
			name:           "eth_accounts",
			method:         "eth_accounts",
			params:         json.RawMessage(`[]`),
			expectError:    false,
			expectedResult: []interface{}{},
		},
		{
			name:           "eth_getBalance",
			method:         "eth_getBalance",
			params:         json.RawMessage(`["0x123...", "latest"]`),
			expectError:    false,
			expectedResult: "0x123456",
		},
		{
			name:        "unknown_method",
			method:      "eth_unknownMethod",
			params:      json.RawMessage(`["param"]`),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := &jsonrpc.Request{
				JSONRPC: "2.0",
				Method:  tc.method,
				ID:      "test_id",
				Params:  tc.params,
			}

			response := router.Route(context.Background(), request)

			if response == nil {
				t.Fatal("Expected response, got nil")
			}

			if tc.expectError {
				if response.Error == nil {
					t.Error("Expected error response")
				}
				return
			}

			if response.Error != nil {
				t.Errorf("Unexpected error: %v", response.Error)
				return
			}

			// 验证结果
			var result interface{}
			if err := json.Unmarshal(response.Result, &result); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}

			// 比较结果 - 使用 JSON 序列化进行深度比较
			expectedJSON, _ := json.Marshal(tc.expectedResult)
			resultJSON, _ := json.Marshal(result)

			if string(expectedJSON) != string(resultJSON) {
				t.Errorf("Expected result %v, got %v", tc.expectedResult, result)
			}
		})
	}
}

func TestSimpleIntegration_BatchRequests(t *testing.T) {
	logger := logrus.New()

	// 创建路由器
	router := NewRouter(logger)

	// 注册处理器
	handler := &simpleMockHandler{method: "test_method", result: "batch_result"}
	err := router.Register(handler)
	if err != nil {
		t.Fatalf("Failed to register handler: %v", err)
	}

	// 创建批量请求
	requests := []jsonrpc.Request{
		{JSONRPC: "2.0", Method: "test_method", ID: "id1"},
		{JSONRPC: "2.0", Method: "test_method", ID: "id2"},
		{JSONRPC: "2.0", Method: "test_method", ID: "id3"},
	}

	// 路由批量请求
	responses := router.RouteBatch(context.Background(), requests)

	// 验证响应
	if len(responses) != len(requests) {
		t.Errorf("Expected %d responses, got %d", len(requests), len(responses))
	}

	// 验证每个响应
	for i, response := range responses {
		if response == nil {
			t.Errorf("Response %d is nil", i)
			continue
		}

		if response.Error != nil {
			t.Errorf("Response %d has unexpected error: %v", i, response.Error)
			continue
		}

		if response.ID != requests[i].ID {
			t.Errorf("Response %d: expected ID %v, got %v", i, requests[i].ID, response.ID)
		}

		var result string
		if err := json.Unmarshal(response.Result, &result); err != nil {
			t.Errorf("Failed to unmarshal result for response %d: %v", i, err)
			continue
		}

		if result != "batch_result" {
			t.Errorf("Response %d: expected 'batch_result', got '%s'", i, result)
		}
	}
}
