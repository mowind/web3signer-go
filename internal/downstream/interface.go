package downstream

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mowind/web3signer-go/internal/jsonrpc"
)

// ClientInterface 定义下游服务客户端接口
type ClientInterface interface {
	// ForwardRequest 转发单个JSON-RPC请求到下游服务
	ForwardRequest(ctx context.Context, req *jsonrpc.Request) (*jsonrpc.Response, error)

	// ForwardBatchRequest 转发批量JSON-RPC请求到下游服务
	ForwardBatchRequest(ctx context.Context, requests []jsonrpc.Request) ([]jsonrpc.Response, error)

	// TestConnection 测试下游服务连接
	TestConnection(ctx context.Context) error

	// GetEndpoint 获取下游服务端点URL
	GetEndpoint() string

	// Close 关闭客户端连接
	Close() error
}

// Forwarder 定义转发器接口
type Forwarder interface {
	// Forward 转发请求到下游服务
	Forward(ctx context.Context, method string, params interface{}) (*jsonrpc.Response, error)

	// ForwardBatch 转发批量请求到下游服务
	ForwardBatch(ctx context.Context, requests []jsonrpc.Request) ([]jsonrpc.Response, error)
}

// SimpleForwarder 简单的转发器实现
type SimpleForwarder struct {
	client ClientInterface
}

// NewSimpleForwarder 创建新的简单转发器
func NewSimpleForwarder(client ClientInterface) *SimpleForwarder {
	return &SimpleForwarder{
		client: client,
	}
}

// Forward 转发请求到下游服务
func (f *SimpleForwarder) Forward(ctx context.Context, method string, params interface{}) (*jsonrpc.Response, error) {
	// 序列化参数
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	// 创建请求
	req := &jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
		ID:      1, // 使用固定ID，转发时会保持
	}

	return f.client.ForwardRequest(ctx, req)
}

// ForwardBatch 转发批量请求到下游服务
func (f *SimpleForwarder) ForwardBatch(ctx context.Context, requests []jsonrpc.Request) ([]jsonrpc.Response, error) {
	return f.client.ForwardBatchRequest(ctx, requests)
}

// VerifyInterfaceImplementation 验证接口实现
var (
	_ ClientInterface = (*Client)(nil)
	_ Forwarder       = (*SimpleForwarder)(nil)
)
