package router

import (
	"context"
	"fmt"

	"github.com/mowind/web3signer-go/internal/downstream"
	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/sirupsen/logrus"
)

// ForwardHandler 处理转发到下游服务的 JSON-RPC 方法
//
// # ForwardHandler 处理转发到下游服务的 JSON-RPC 方法
//
// ForwardHandler 处理转发到下游服务的 JSON-RPC 方法
//
//lint:ignore SA1019 // downstream.ClientInterface is used for backward compatibility
//lint:ignore SA1019 // downstream.ClientInterface is used for backward compatibility
//lint:ignore SA1019 // downstream.ClientInterface is used for backward compatibility
type ForwardHandler struct {
	*BaseHandler
	client downstream.ClientInterface
}

// NewForwardHandler 创建转发处理器
func NewForwardHandler(client downstream.ClientInterface, logger *logrus.Logger) *ForwardHandler { //nolint:staticcheck // SA1019: backward compatibility
	return &ForwardHandler{
		BaseHandler: NewBaseHandler("forward", logger),
		client:      client,
	}
}

// Method 返回处理器支持的方法名
func (h *ForwardHandler) Method() string {
	return "forward_handler" // 这个处理器处理所有不支持的方法
}

// Handle 处理 JSON-RPC 请求
func (h *ForwardHandler) Handle(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error) {
	h.LogRequest(request)

	// 特殊处理 eth_accounts - 返回空数组
	if request.Method == "eth_accounts" {
		return h.handleEthAccounts(ctx, request)
	}

	// 转发到下游服务
	response, err := h.forwardToDownstream(ctx, request)
	if err != nil {
		h.logger.WithError(err).Error("Failed to forward request to downstream")
		return h.CreateErrorResponse(request.ID, jsonrpc.CodeInternalError,
			"Failed to forward request", err.Error()), nil
	}

	h.LogResponse(request, response, nil)
	return response, nil
}

// handleEthAccounts 处理 eth_accounts 方法
func (h *ForwardHandler) handleEthAccounts(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error) {
	h.logger.Info("Returning empty accounts array")

	// 返回空数组（web3signer 不管理账户）
	emptyAccounts := []string{}
	return h.CreateSuccessResponse(request.ID, emptyAccounts)
}

// forwardToDownstream 转发请求到下游服务
func (h *ForwardHandler) forwardToDownstream(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error) {
	logger := h.logger.WithFields(logrus.Fields{
		"method": request.Method,
		"id":     request.ID,
	})

	logger.Info("Forwarding to downstream")

	// 使用下游客户端转发请求
	response, err := h.client.ForwardRequest(ctx, request)
	if err != nil {
		logger.WithError(err).Error("Downstream service error")
		return nil, fmt.Errorf("downstream service error: %v", err)
	}

	logger.Info("Request forwarded successfully")
	return response, nil
}

// IsForwardMethod 判断方法是否应该被转发
func IsForwardMethod(method string) bool {
	// 签名方法由 SignHandler 处理
	if IsSignMethod(method) {
		return false
	}

	// 以下方法特殊处理
	switch method {
	case "eth_accounts":
		return true // 返回空数组
	case "eth_getBalance",
		"eth_getBlockByNumber",
		"eth_getBlockByHash",
		"eth_getTransactionByHash",
		"eth_getTransactionReceipt",
		"eth_blockNumber",
		"eth_chainId",
		"net_version",
		"web3_clientVersion":
		return true // 常见查询方法，转发到下游
	default:
		// 其他所有未知方法都转发到下游
		return true
	}
}
