package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/mowind/web3signer-go/internal/router"
	"github.com/mowind/web3signer-go/internal/signer"
	"github.com/sirupsen/logrus"
	"github.com/umbracle/ethgo"
)

// TestEndToEnd_CompleteIntegration 完整的端到端集成测试
func TestEndToEnd_CompleteIntegration(t *testing.T) {
	// 设置日志
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建 mock 服务
	kmsServer := NewMockKMSServer()
	defer kmsServer.Close()

	downstreamServer := NewMockDownstreamServer()
	defer downstreamServer.Close()

	// 配置 mock 服务
	kmsServer.AddValidKey("test-key-id")
	kmsServer.SetAccessKey("test-access-key", "test-secret-key")

	// 创建客户端
	kmsClient := NewMockKMSClient(kmsServer)
	kmsClient.SetCredentials("test-access-key", "test-secret-key")

	downstreamClient := NewMockDownstreamClient(downstreamServer)

	// 创建 MPC-KMS 签名器
	testAddress := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	mpcSigner := signer.NewMPCKMSSigner(kmsClient, "test-key-id", testAddress)

	// 创建路由器工厂和路由器
	routerFactory := router.NewRouterFactory(logger)
	router := routerFactory.CreateRouter(mpcSigner, downstreamClient)

	// 创建 HTTP 处理器
	handler := createTestHandler(router, logger)

	// 启动测试服务器
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// 测试用例
	testCases := []struct {
		name           string
		method         string
		params         interface{}
		expectError    bool
		expectedResult interface{}
	}{
		{
			name:        "eth_sign - success",
			method:      "eth_sign",
			params:      []interface{}{"0x1234567890123456789012345678901234567890", "0xdeadbeef"},
			expectError: false,
		},
		{
			name:        "eth_sign - wrong address",
			method:      "eth_sign",
			params:      []interface{}{"0xwrongaddress", "0xdeadbeef"},
			expectError: true,
		},
		{
			name:   "eth_signTransaction - success",
			method: "eth_signTransaction",
			params: []interface{}{map[string]interface{}{
				"from":     "0x1234567890123456789012345678901234567890",
				"to":       "0x0987654321098765432109876543210987654321",
				"gas":      "21000",
				"gasPrice": "20000000000",
				"value":    "1000000000000000000",
				"nonce":    "5",
			}},
			expectError: false,
		},
		{
			name:           "eth_accounts - returns empty array",
			method:         "eth_accounts",
			params:         []interface{}{},
			expectError:    false,
			expectedResult: []interface{}{},
		},
		{
			name:           "eth_getBalance - forwarded to downstream",
			method:         "eth_getBalance",
			params:         []interface{}{"0x1234567890123456789012345678901234567890", "latest"},
			expectError:    false,
			expectedResult: "0xde0b6b3a7640000", // 1 ETH
		},
		{
			name:           "eth_getTransactionCount - forwarded to downstream",
			method:         "eth_getTransactionCount",
			params:         []interface{}{"0x1234567890123456789012345678901234567890", "latest"},
			expectError:    false,
			expectedResult: "0x5",
		},
		{
			name:           "eth_gasPrice - forwarded to downstream",
			method:         "eth_gasPrice",
			params:         []interface{}{},
			expectError:    false,
			expectedResult: "0x4a817c800", // 20 Gwei
		},
		{
			name:           "net_version - forwarded to downstream",
			method:         "net_version",
			params:         []interface{}{},
			expectError:    false,
			expectedResult: "1",
		},
		{
			name:        "unknown_method - forwarded to downstream",
			method:      "eth_unknownMethod",
			params:      []interface{}{"param1", "param2"},
			expectError: true, // 应该返回 Method not found 错误
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建 JSON-RPC 请求
			request := map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  tc.method,
				"params":  tc.params,
				"id":      1,
			}

			// 发送请求
			resp, err := sendJSONRPCRequest(ts.URL, request)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}

			// 验证响应
			if tc.expectError {
				respMap, ok := resp.(map[string]interface{})
				if !ok || respMap["error"] == nil {
					t.Error("Expected error response")
				}
				return
			}

			respMap, ok := resp.(map[string]interface{})
			if !ok {
				t.Fatal("Expected map response")
			}

			if respMap["error"] != nil {
				t.Errorf("Unexpected error: %v", respMap["error"])
				return
			}

			// 验证结果
			if tc.expectedResult != nil {
				result := respMap["result"]
				if !compareResults(result, tc.expectedResult) {
					t.Errorf("Expected result %v, got %v", tc.expectedResult, result)
				}
			}
		})
	}
}

// TestEndToEnd_BatchRequests 批量请求测试
func TestEndToEnd_BatchRequests(t *testing.T) {
	// 设置日志
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建 mock 服务
	kmsServer := NewMockKMSServer()
	defer kmsServer.Close()

	downstreamServer := NewMockDownstreamServer()
	defer downstreamServer.Close()

	// 配置 mock 服务
	kmsServer.AddValidKey("test-key-id")
	kmsServer.SetAccessKey("test-access-key", "test-secret-key")

	// 创建客户端
	kmsClient := NewMockKMSClient(kmsServer)
	kmsClient.SetCredentials("test-access-key", "test-secret-key")

	downstreamClient := NewMockDownstreamClient(downstreamServer)

	// 创建 MPC-KMS 签名器
	testAddress := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	mpcSigner := signer.NewMPCKMSSigner(kmsClient, "test-key-id", testAddress)

	// 创建路由器工厂和路由器
	routerFactory := router.NewRouterFactory(logger)
	router := routerFactory.CreateRouter(mpcSigner, downstreamClient)

	// 创建 HTTP 处理器
	handler := createTestHandler(router, logger)

	// 启动测试服务器
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// 创建批量请求
	requests := []map[string]interface{}{
		{
			"jsonrpc": "2.0",
			"method":  "eth_sign",
			"params":  []interface{}{"0x1234567890123456789012345678901234567890", "0xdeadbeef"},
			"id":      1,
		},
		{
			"jsonrpc": "2.0",
			"method":  "eth_accounts",
			"params":  []interface{}{},
			"id":      2,
		},
		{
			"jsonrpc": "2.0",
			"method":  "eth_getBalance",
			"params":  []interface{}{"0x1234567890123456789012345678901234567890", "latest"},
			"id":      3,
		},
	}

	// 发送批量请求
	resp, err := sendJSONRPCRequest(ts.URL, requests)
	if err != nil {
		t.Fatalf("Failed to send batch request: %v", err)
	}

	// 验证响应是数组
	var responses []interface{}
	respJSON, _ := json.Marshal(resp)
	if err := json.Unmarshal(respJSON, &responses); err != nil {
		// 可能是单个响应
		var singleResp interface{}
		if err := json.Unmarshal(respJSON, &singleResp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		responses = []interface{}{singleResp}
	}

	if len(responses) != len(requests) {
		t.Errorf("Expected %d responses, got %d", len(requests), len(responses))
	}

	// 验证每个响应
	for i, response := range responses {
		respMap, ok := response.(map[string]interface{})
		if !ok {
			t.Errorf("Response %d is not an object", i)
			continue
		}

		// 验证 ID 匹配（处理不同类型）
		if !compareIDs(requests[i]["id"], respMap["id"]) {
			t.Errorf("Response %d: ID mismatch, expected %v, got %v", i, requests[i]["id"], respMap["id"])
		}

		// 验证没有错误
		if respMap["error"] != nil {
			t.Errorf("Response %d has unexpected error: %v", i, respMap["error"])
		}

		// 验证有结果
		if respMap["result"] == nil {
			t.Errorf("Response %d has no result", i)
		}
	}
}

// TestEndToEnd_ErrorHandling 错误处理测试
func TestEndToEnd_ErrorHandling(t *testing.T) {
	// 设置日志
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 测试 KMS 认证失败
	t.Run("KMS Authentication Failure", func(t *testing.T) {
		kmsServer := NewMockKMSServer()
		defer kmsServer.Close()

		downstreamServer := NewMockDownstreamServer()
		defer downstreamServer.Close()

		// 配置错误的认证信息
		kmsServer.AddValidKey("test-key-id")
		kmsServer.SetAccessKey("wrong-access-key", "wrong-secret-key")

		kmsClient := NewMockKMSClient(kmsServer)
		kmsClient.SetCredentials("test-access-key", "test-secret-key") // 错误的凭据

		downstreamClient := NewMockDownstreamClient(downstreamServer)

		testAddress := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
		mpcSigner := signer.NewMPCKMSSigner(kmsClient, "test-key-id", testAddress)

		routerFactory := router.NewRouterFactory(logger)
		router := routerFactory.CreateRouter(mpcSigner, downstreamClient)

		// 测试签名请求
		request := &jsonrpc.Request{
			JSONRPC: "2.0",
			Method:  "eth_sign",
			ID:      1,
			Params:  json.RawMessage(`["0x1234567890123456789012345678901234567890", "0xdeadbeef"]`),
		}

		response := router.Route(context.Background(), request)
		if response.Error == nil {
			t.Error("Expected authentication error")
		}
	})

	// 测试无效的密钥ID
	t.Run("Invalid Key ID", func(t *testing.T) {
		kmsServer := NewMockKMSServer()
		defer kmsServer.Close()

		downstreamServer := NewMockDownstreamServer()
		defer downstreamServer.Close()

		// 不添加有效的密钥
		kmsServer.SetAccessKey("test-access-key", "test-secret-key")

		kmsClient := NewMockKMSClient(kmsServer)
		kmsClient.SetCredentials("test-access-key", "test-secret-key")

		downstreamClient := NewMockDownstreamClient(downstreamServer)

		testAddress := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
		mpcSigner := signer.NewMPCKMSSigner(kmsClient, "invalid-key-id", testAddress)

		routerFactory := router.NewRouterFactory(logger)
		router := routerFactory.CreateRouter(mpcSigner, downstreamClient)

		// 测试签名请求
		request := &jsonrpc.Request{
			JSONRPC: "2.0",
			Method:  "eth_sign",
			ID:      1,
			Params:  json.RawMessage(`["0x1234567890123456789012345678901234567890", "0xdeadbeef"]`),
		}

		response := router.Route(context.Background(), request)
		if response.Error == nil {
			t.Error("Expected key not found error")
		}
	})

	// 测试下游服务失败
	t.Run("Downstream Service Failure", func(t *testing.T) {
		kmsServer := NewMockKMSServer()
		defer kmsServer.Close()

		downstreamServer := NewMockDownstreamServer()
		defer downstreamServer.Close()

		// 配置 KMS
		kmsServer.AddValidKey("test-key-id")
		kmsServer.SetAccessKey("test-access-key", "test-secret-key")

		kmsClient := NewMockKMSClient(kmsServer)
		kmsClient.SetCredentials("test-access-key", "test-secret-key")

		// 设置下游服务失败
		downstreamServer.SetShouldFail(true)
		downstreamClient := NewMockDownstreamClient(downstreamServer)

		testAddress := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
		mpcSigner := signer.NewMPCKMSSigner(kmsClient, "test-key-id", testAddress)

		routerFactory := router.NewRouterFactory(logger)
		router := routerFactory.CreateRouter(mpcSigner, downstreamClient)

		// 测试转发请求
		request := &jsonrpc.Request{
			JSONRPC: "2.0",
			Method:  "eth_getBalance",
			ID:      1,
			Params:  json.RawMessage(`["0x1234567890123456789012345678901234567890", "latest"]`),
		}

		response := router.Route(context.Background(), request)
		if response.Error == nil {
			t.Error("Expected downstream service error")
		}
	})
}

// TestEndToEnd_ConcurrentRequests 并发请求测试
func TestEndToEnd_ConcurrentRequests(t *testing.T) {
	// 设置日志
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// 创建 mock 服务
	kmsServer := NewMockKMSServer()
	defer kmsServer.Close()

	downstreamServer := NewMockDownstreamServer()
	defer downstreamServer.Close()

	// 配置 mock 服务
	kmsServer.AddValidKey("test-key-id")
	kmsServer.SetAccessKey("test-access-key", "test-secret-key")

	// 创建客户端
	kmsClient := NewMockKMSClient(kmsServer)
	kmsClient.SetCredentials("test-access-key", "test-secret-key")

	downstreamClient := NewMockDownstreamClient(downstreamServer)

	// 创建 MPC-KMS 签名器
	testAddress := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	mpcSigner := signer.NewMPCKMSSigner(kmsClient, "test-key-id", testAddress)

	// 创建路由器工厂和路由器
	routerFactory := router.NewRouterFactory(logger)
	router := routerFactory.CreateRouter(mpcSigner, downstreamClient)

	// 创建 HTTP 处理器
	handler := createTestHandler(router, logger)

	// 启动测试服务器
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// 并发发送请求
	numRequests := 10
	done := make(chan bool, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			request := map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "eth_sign",
				"params":  []interface{}{"0x1234567890123456789012345678901234567890", fmt.Sprintf("0xdeadbeef%d", id)},
				"id":      id,
			}

			resp, err := sendJSONRPCRequest(ts.URL, request)
			if err != nil {
				errors <- err
				done <- false
				return
			}

			respMap, ok := resp.(map[string]interface{})
			if !ok {
				errors <- fmt.Errorf("request %d: expected map response", id)
				done <- false
				return
			}

			if respMap["error"] != nil {
				errors <- fmt.Errorf("request %d failed: %v", id, respMap["error"])
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// 等待所有请求完成
	successCount := 0
	for i := 0; i < numRequests; i++ {
		if <-done {
			successCount++
		}
	}

	// 检查错误
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent request error: %v", err)
	}

	if successCount != numRequests {
		t.Errorf("Expected %d successful requests, got %d", numRequests, successCount)
	}
}

// createTestHandler 创建测试用的 HTTP 处理器
func createTestHandler(router *router.Router, logger *logrus.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		// 解析 JSON-RPC 请求
		requests, err := jsonrpc.ParseRequest(body)
		if err != nil {
			errResp := jsonrpc.NewErrorResponse(nil, jsonrpc.ParseError)
			respData, _ := json.Marshal(errResp)
			w.Header().Set("Content-Type", "application/json")
			w.Write(respData)
			return
		}

		// 处理请求
		responses := make([]*jsonrpc.Response, 0, len(requests))
		for _, req := range requests {
			resp := router.Route(r.Context(), &req)
			responses = append(responses, resp)
		}

		// 序列化响应
		var respData []byte
		if len(responses) == 1 {
			respData, _ = json.Marshal(responses[0])
		} else {
			respData, _ = json.Marshal(responses)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(respData)
	})
}

// Helper functions

func sendJSONRPCRequest(url string, request interface{}) (interface{}, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 根据请求类型决定返回类型
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 尝试解析为数组（批量请求）
	var arrayResult []interface{}
	if err := json.Unmarshal(respBody, &arrayResult); err == nil {
		return arrayResult, nil
	}

	// 尝试解析为对象（单个请求）
	var objectResult map[string]interface{}
	if err := json.Unmarshal(respBody, &objectResult); err == nil {
		return objectResult, nil
	}

	return nil, fmt.Errorf("failed to parse response as JSON")
}

func compareResults(actual, expected interface{}) bool {
	// 简单的结果比较
	actualJSON, _ := json.Marshal(actual)
	expectedJSON, _ := json.Marshal(expected)
	return string(actualJSON) == string(expectedJSON)
}

func compareIDs(expected, actual interface{}) bool {
	// 将两个 ID 都转换为 float64 进行比较（JSON 数字默认是 float64）
	expectedFloat := toFloat64(expected)
	actualFloat := toFloat64(actual)
	return expectedFloat == actualFloat
}

func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		// 处理字符串类型的 ID
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			return floatVal
		}
	}
	return 0
}
