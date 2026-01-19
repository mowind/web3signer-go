package router

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/mowind/web3signer-go/internal/kms"
	"github.com/mowind/web3signer-go/internal/signer"
	"github.com/sirupsen/logrus"
	"github.com/umbracle/ethgo"
)

// mockKMSClient 用于测试的 mock KMS 客户端
type testKMSClient struct{}

func (c *testKMSClient) Sign(ctx context.Context, keyID string, message []byte) ([]byte, error) {
	// 返回一个模拟的十六进制编码的 65 字节签名
	signature := make([]byte, 65)
	for i := 0; i < 65; i++ {
		signature[i] = byte(i + 1) // 避免 0 值
	}
	// 返回十六进制编码的签名
	hexSignature := hex.EncodeToString(signature)
	return []byte(hexSignature), nil
}

func (c *testKMSClient) SignWithOptions(ctx context.Context, keyID string, message []byte, encoding kms.DataEncoding, summary *kms.SignSummary, callbackURL string) ([]byte, error) {
	return c.Sign(ctx, keyID, message)
}

func (c *testKMSClient) GetTaskResult(ctx context.Context, taskID string) (*kms.TaskResult, error) {
	return &kms.TaskResult{Status: kms.TaskStatusDone}, nil
}

func (c *testKMSClient) WaitForTaskCompletion(ctx context.Context, taskID string, interval time.Duration) (*kms.TaskResult, error) {
	return &kms.TaskResult{Status: kms.TaskStatusDone}, nil
}

func (c *testKMSClient) Do(req *http.Request) (*http.Response, error) {
	return nil, nil
}

// mockDownstreamClient 用于测试的 mock 下游客户端
type testDownstreamClient struct{}

func (c *testDownstreamClient) ForwardRequest(ctx context.Context, req *jsonrpc.Request) (*jsonrpc.Response, error) {
	// 模拟下游服务响应
	return jsonrpc.NewResponse(req.ID, "downstream_result")
}

func (c *testDownstreamClient) ForwardBatchRequest(ctx context.Context, requests []jsonrpc.Request) ([]jsonrpc.Response, error) {
	responses := make([]jsonrpc.Response, len(requests))
	for i, req := range requests {
		resp, _ := jsonrpc.NewResponse(req.ID, "batch_result")
		responses[i] = *resp
	}
	return responses, nil
}

func (c *testDownstreamClient) TestConnection(ctx context.Context) error {
	return nil
}

func (c *testDownstreamClient) GetEndpoint() string {
	return "http://test-downstream:8545"
}

func (c *testDownstreamClient) Close() error {
	return nil
}

func TestIntegration_CompleteFlow(t *testing.T) {
	// 设置 logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建 MPC-KMS 签名器
	testAddress := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	mpcSigner := signer.NewMPCKMSSigner(&testKMSClient{}, "test-key-id", testAddress)

	// 创建下游客户端
	downstreamClient := &testDownstreamClient{}

	// 创建路由器工厂
	factory := NewRouterFactory(logger)
	router := factory.CreateRouter(mpcSigner, downstreamClient)

	// 测试用例
	testCases := []struct {
		name           string
		method         string
		params         json.RawMessage
		expectError    bool
		expectedResult interface{}
	}{
		{
			name:           "eth_sign - success",
			method:         "eth_sign",
			params:         json.RawMessage(`["0x1234567890123456789012345678901234567890", "0x000000000000000000000000000000000000000000000000000000000000dead"]`),
			expectError:    false,
			expectedResult: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40", // 模拟签名
		},
		{
			name:        "eth_sign - wrong address",
			method:      "eth_sign",
			params:      json.RawMessage(`["0xwrongaddress", "0x000000000000000000000000000000000000000000000000000000000000dead"]`),
			expectError: true,
		},
		{
			name:   "eth_signTransaction - success",
			method: "eth_signTransaction",
			params: json.RawMessage(`[{
				"from": "0x1234567890123456789012345678901234567890",
				"to": "0x0987654321098765432109876543210987654321",
				"gas": "21000",
				"gasPrice": "20000000000",
				"value": "1000000000000000000",
				"nonce": "5"
			}]`),
			expectError: false,
		},
		{
			name:           "eth_accounts - returns KMS address",
			method:         "eth_accounts",
			params:         json.RawMessage(`[]`),
			expectError:    false,
			expectedResult: []string{"0x1234567890123456789012345678901234567890"},
		},
		{
			name:           "eth_getBalance - forwarded to downstream",
			method:         "eth_getBalance",
			params:         json.RawMessage(`["0x1234567890123456789012345678901234567890", "latest"]`),
			expectError:    false,
			expectedResult: "downstream_result",
		},
		{
			name:           "unknown_method - forwarded to downstream",
			method:         "eth_unknownMethod",
			params:         json.RawMessage(`["param1", "param2"]`),
			expectError:    false,
			expectedResult: "downstream_result",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建请求
			request := &jsonrpc.Request{
				JSONRPC: "2.0",
				Method:  tc.method,
				ID:      "test_id",
				Params:  tc.params,
			}

			// 路由请求
			response := router.Route(context.Background(), request)

			// 验证响应
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
			if tc.expectedResult != nil {
				var result interface{}
				if err := json.Unmarshal(response.Result, &result); err != nil {
					t.Fatalf("Failed to unmarshal result: %v", err)
				}

				// 简单的结果验证
				switch tc.method {
				case "eth_accounts":
					accounts, ok := result.([]interface{})
					if !ok {
						t.Fatalf("Expected array for eth_accounts, got %T", result)
					}
					if len(accounts) != 1 {
						t.Errorf("Expected 1 address for eth_accounts, got %d", len(accounts))
					}
					if accounts[0] != "0x1234567890123456789012345678901234567890" {
						t.Errorf("Expected address 0x1234567890123456789012345678901234567890, got %v", accounts[0])
					}
				case "eth_getBalance", "eth_unknownMethod":
					if result != tc.expectedResult {
						t.Errorf("Expected result %v, got %v", tc.expectedResult, result)
					}
				}
			}
		})
	}
}

func TestIntegration_BatchRequests(t *testing.T) {
	// 设置 logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建 MPC-KMS 签名器
	testAddress := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	mpcSigner := signer.NewMPCKMSSigner(&testKMSClient{}, "test-key-id", testAddress)

	// 创建下游客户端
	downstreamClient := &testDownstreamClient{}

	// 创建路由器
	factory := NewRouterFactory(logger)
	router := factory.CreateRouter(mpcSigner, downstreamClient)

	// 创建批量请求
	requests := []jsonrpc.Request{
		{
			JSONRPC: "2.0",
			Method:  "eth_sign",
			ID:      "id1",
			Params:  json.RawMessage(`["0x1234567890123456789012345678901234567890", "0x000000000000000000000000000000000000000000000000000000000000dead"]`),
		},
		{
			JSONRPC: "2.0",
			Method:  "eth_accounts",
			ID:      "id2",
			Params:  json.RawMessage(`[]`),
		},
		{
			JSONRPC: "2.0",
			Method:  "eth_getBalance",
			ID:      "id3",
			Params:  json.RawMessage(`["0x1234567890123456789012345678901234567890", "latest"]`),
		},
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

		// 验证 ID 匹配
		if response.ID != requests[i].ID {
			t.Errorf("Response %d: expected ID %v, got %v", i, requests[i].ID, response.ID)
		}

		// 验证有结果
		if len(response.Result) == 0 {
			t.Errorf("Response %d has empty result", i)
		}
	}
}

func TestIntegration_HandlerRegistration(t *testing.T) {
	logger := logrus.New()
	router := NewRouter(logger)

	// 注册多个处理器
	handlers := []Handler{
		&mockHandler{method: "eth_sign"},
		&mockHandler{method: "eth_signTransaction"},
		&mockHandler{method: "eth_sendTransaction"},
		&mockHandler{method: "eth_accounts"},
		&mockHandler{method: "eth_getBalance"},
	}

	for _, handler := range handlers {
		err := router.Register(handler)
		if err != nil {
			t.Fatalf("Failed to register handler %s: %v", handler.Method(), err)
		}
	}

	// 验证注册的方法
	registeredMethods := router.GetRegisteredMethods()
	if len(registeredMethods) != len(handlers) {
		t.Errorf("Expected %d registered methods, got %d", len(handlers), len(registeredMethods))
	}

	// 验证每个方法都存在
	for _, handler := range handlers {
		if !router.HasHandler(handler.Method()) {
			t.Errorf("Handler %s not found in router", handler.Method())
		}
	}

	// 测试注销
	router.Unregister("eth_sign")
	if router.HasHandler("eth_sign") {
		t.Error("Handler eth_sign should be unregistered")
	}
}
