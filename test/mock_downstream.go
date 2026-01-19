package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/mowind/web3signer-go/internal/jsonrpc"
)

// MockDownstreamServer 模拟下游 JSON-RPC 服务
type MockDownstreamServer struct {
	server     *httptest.Server
	mu         sync.RWMutex
	handlers   map[string]HandlerFunc
	responses  map[string]interface{}
	shouldFail bool
	delay      time.Duration
}

// HandlerFunc 处理函数类型
type HandlerFunc func(params json.RawMessage) (interface{}, error)

// NewMockDownstreamServer 创建新的 mock 下游服务器
func NewMockDownstreamServer() *MockDownstreamServer {
	mock := &MockDownstreamServer{
		handlers:  make(map[string]HandlerFunc),
		responses: make(map[string]interface{}),
		delay:     0,
	}

	// 注册默认处理器
	mock.registerDefaultHandlers()

	mock.server = httptest.NewServer(http.HandlerFunc(mock.handleRequest))
	return mock
}

// registerDefaultHandlers 注册默认的 JSON-RPC 方法处理器
func (m *MockDownstreamServer) registerDefaultHandlers() {
	// eth_getBalance
	m.RegisterHandler("eth_getBalance", func(params json.RawMessage) (interface{}, error) {
		var args []interface{}
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("invalid params")
		}
		if len(args) < 1 {
			return nil, fmt.Errorf("missing address parameter")
		}
		// 返回模拟的余额 (1 ETH)
		return "0xde0b6b3a7640000", nil // 1e18 wei
	})

	// eth_getTransactionCount
	m.RegisterHandler("eth_getTransactionCount", func(params json.RawMessage) (interface{}, error) {
		var args []interface{}
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("invalid params")
		}
		if len(args) < 1 {
			return nil, fmt.Errorf("missing address parameter")
		}
		// 返回模拟的 nonce
		return "0x5", nil
	})

	// eth_gasPrice
	m.RegisterHandler("eth_gasPrice", func(params json.RawMessage) (interface{}, error) {
		// 返回模拟的 gas price (20 Gwei)
		return "0x4a817c800", nil // 20e9 wei
	})

	// eth_estimateGas
	m.RegisterHandler("eth_estimateGas", func(params json.RawMessage) (interface{}, error) {
		// 返回模拟的 gas 估算 (21000)
		return "0x5208", nil // 21000
	})

	// eth_getBlockByNumber
	m.RegisterHandler("eth_getBlockByNumber", func(params json.RawMessage) (interface{}, error) {
		var args []interface{}
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("invalid params")
		}
		if len(args) < 2 {
			return nil, fmt.Errorf("missing parameters")
		}

		// 返回模拟的区块信息
		blockNumber := "0x123456"
		if args[0] == "latest" {
			blockNumber = "0x123456"
		}

		return map[string]interface{}{
			"number":       blockNumber,
			"hash":         "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			"parentHash":   "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			"timestamp":    fmt.Sprintf("0x%x", time.Now().Unix()),
			"gasLimit":     "0x1c9c380",
			"gasUsed":      "0x5208",
			"transactions": []string{},
		}, nil
	})

	// eth_getTransactionReceipt
	m.RegisterHandler("eth_getTransactionReceipt", func(params json.RawMessage) (interface{}, error) {
		var args []interface{}
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("invalid params")
		}
		if len(args) < 1 {
			return nil, fmt.Errorf("missing transaction hash parameter")
		}

		// 返回模拟的交易收据
		return map[string]interface{}{
			"blockHash":        "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			"blockNumber":      "0x123456",
			"transactionHash":  args[0],
			"transactionIndex": "0x0",
			"from":             "0x1234567890123456789012345678901234567890",
			"to":               "0x0987654321098765432109876543210987654321",
			"gasUsed":          "0x5208",
			"status":           "0x1", // 成功
		}, nil
	})

	// net_version
	m.RegisterHandler("net_version", func(params json.RawMessage) (interface{}, error) {
		return "1", nil // 主网
	})

	// web3_clientVersion
	m.RegisterHandler("web3_clientVersion", func(params json.RawMessage) (interface{}, error) {
		return "MockDownstream/v1.0.0", nil
	})
}

// RegisterHandler 注册自定义处理器
func (m *MockDownstreamServer) RegisterHandler(method string, handler HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[method] = handler
}

// SetResponse 设置固定响应（优先级高于处理器）
func (m *MockDownstreamServer) SetResponse(method string, response interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[method] = response
}

// SetShouldFail 设置是否应该失败
func (m *MockDownstreamServer) SetShouldFail(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
}

// SetDelay 设置响应延迟
func (m *MockDownstreamServer) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = delay
}

// URL 返回服务器 URL
func (m *MockDownstreamServer) URL() string {
	return m.server.URL
}

// Close 关闭服务器
func (m *MockDownstreamServer) Close() {
	m.server.Close()
}

// handleRequest 处理 HTTP 请求
func (m *MockDownstreamServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		m.writeJSONRPCError(w, nil, jsonrpc.CodeInvalidRequest, "Only POST method is allowed")
		return
	}

	// 模拟延迟
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	// 解析 JSON-RPC 请求
	var requests []jsonrpc.Request
	var singleRequest jsonrpc.Request

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&singleRequest); err == nil && singleRequest.JSONRPC != "" {
		// 单个请求
		requests = []jsonrpc.Request{singleRequest}
	} else {
		// 尝试解析为批量请求
		if err := json.NewDecoder(r.Body).Decode(&requests); err != nil || len(requests) == 0 {
			m.writeJSONRPCError(w, nil, jsonrpc.CodeInvalidRequest, "Invalid JSON-RPC request")
			return
		}
	}

	// 处理请求
	if len(requests) == 1 {
		// 单个请求
		response := m.handleSingleRequest(&requests[0])
		m.writeResponse(w, response)
	} else {
		// 批量请求
		responses := m.handleBatchRequest(requests)
		m.writeResponse(w, responses)
	}
}

// handleSingleRequest 处理单个请求
func (m *MockDownstreamServer) handleSingleRequest(request *jsonrpc.Request) *jsonrpc.Response {
	if m.shouldFail {
		return jsonrpc.NewErrorResponse(request.ID, jsonrpc.InternalError)
	}

	// 检查是否有固定响应
	m.mu.RLock()
	if response, exists := m.responses[request.Method]; exists {
		m.mu.RUnlock()
		resp, _ := jsonrpc.NewResponse(request.ID, response)
		return resp
	}

	// 检查是否有处理器
	handler, exists := m.handlers[request.Method]
	m.mu.RUnlock()

	if !exists {
		return jsonrpc.NewErrorResponse(request.ID, jsonrpc.MethodNotFoundError)
	}

	// 调用处理器
	result, err := handler(request.Params)
	if err != nil {
		return jsonrpc.NewErrorResponse(request.ID, jsonrpc.InternalError)
	}

	resp, _ := jsonrpc.NewResponse(request.ID, result)
	return resp
}

// handleBatchRequest 处理批量请求
func (m *MockDownstreamServer) handleBatchRequest(requests []jsonrpc.Request) []jsonrpc.Response {
	responses := make([]jsonrpc.Response, len(requests))

	for i, request := range requests {
		response := m.handleSingleRequest(&request)
		responses[i] = *response
	}

	return responses
}

// writeResponse 写入响应
func (m *MockDownstreamServer) writeResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// writeJSONRPCError 写入 JSON-RPC 错误响应
func (m *MockDownstreamServer) writeJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string) {
	err := &jsonrpc.Error{
		Code:    code,
		Message: message,
	}
	response := jsonrpc.NewErrorResponse(id, err)
	m.writeResponse(w, response)
}

// MockDownstreamClient 用于测试的 mock 下游客户端
type MockDownstreamClient struct {
	server *MockDownstreamServer
}

// NewMockDownstreamClient 创建新的 mock 下游客户端
func NewMockDownstreamClient(server *MockDownstreamServer) *MockDownstreamClient {
	return &MockDownstreamClient{
		server: server,
	}
}

// ForwardRequest 实现转发接口
func (c *MockDownstreamClient) ForwardRequest(ctx context.Context, req *jsonrpc.Request) (*jsonrpc.Response, error) {
	// 创建 HTTP 请求
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.server.URL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 执行请求
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// 解析响应
	var response jsonrpc.Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// ForwardBatchRequest 实现批量转发接口
func (c *MockDownstreamClient) ForwardBatchRequest(ctx context.Context, requests []jsonrpc.Request) ([]jsonrpc.Response, error) {
	// 创建批量请求
	body, err := json.Marshal(requests)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.server.URL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 执行请求
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// 解析响应
	var responses []jsonrpc.Response
	if err := json.NewDecoder(resp.Body).Decode(&responses); err != nil {
		return nil, err
	}

	return responses, nil
}

// TestConnection 测试连接
func (c *MockDownstreamClient) TestConnection(ctx context.Context) error {
	// 尝试调用 web3_clientVersion
	req := &jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "web3_clientVersion",
		ID:      1,
		Params:  json.RawMessage(`[]`),
	}

	_, err := c.ForwardRequest(ctx, req)
	return err
}

// GetEndpoint 获取端点
func (c *MockDownstreamClient) GetEndpoint() string {
	return c.server.URL()
}

// Close 关闭客户端
func (c *MockDownstreamClient) Close() error {
	return nil
}
