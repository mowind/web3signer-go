package router

import (
	"context"

	"github.com/mowind/web3signer-go/internal/downstream"
	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/mowind/web3signer-go/internal/signer"
	"github.com/sirupsen/logrus"
)

// RouterFactory 路由器工厂，简化路由器的创建和配置
type RouterFactory struct {
	logger *logrus.Logger
}

// NewRouterFactory 创建路由器工厂
func NewRouterFactory(logger *logrus.Logger) *RouterFactory {
	return &RouterFactory{
		logger: logger,
	}
}

// CreateRouter 创建完整配置的路由器
func (f *RouterFactory) CreateRouter(mpcSigner *signer.MPCKMSSigner, downstreamClient downstream.ClientInterface) *Router {
	router := NewRouter(f.logger)

	// 注册签名处理器
	signHandler := NewSignHandler(mpcSigner, downstreamClient, f.logger)

	// 注意：SignHandler 处理多个方法，所以我们需要为每个方法注册同一个处理器
	// 在实际实现中，我们可能需要一个更智能的路由机制
	if err := router.Register(&MethodHandler{
		handler: signHandler,
		method:  "eth_accounts",
	}); err != nil {
		f.logger.WithError(err).Error("Failed to register eth_accounts handler")
	}

	if err := router.Register(&MethodHandler{
		handler: signHandler,
		method:  "eth_sign",
	}); err != nil {
		f.logger.WithError(err).Error("Failed to register eth_sign handler")
	}

	if err := router.Register(&MethodHandler{
		handler: signHandler,
		method:  "eth_signTransaction",
	}); err != nil {
		f.logger.WithError(err).Error("Failed to register eth_signTransaction handler")
	}

	if err := router.Register(&MethodHandler{
		handler: signHandler,
		method:  "eth_sendTransaction",
	}); err != nil {
		f.logger.WithError(err).Error("Failed to register eth_sendTransaction handler")
	}

	// 注册转发处理器（处理所有其他方法）
	forwardHandler := NewForwardHandler(downstreamClient, f.logger)
	router.SetDefaultHandler(&MethodHandler{
		handler: forwardHandler,
		method:  "forward_handler", // 这个会处理所有非签名方法
	})

	return router
}

// MethodHandler 包装处理器，使其符合 Handler 接口
type MethodHandler struct {
	handler Handler
	method  string
}

// Method 返回方法名
func (m *MethodHandler) Method() string {
	return m.method
}

// Handle 处理请求
func (m *MethodHandler) Handle(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error) {
	return m.handler.Handle(ctx, request)
}

// CreateSimpleRouter 创建简化的路由器（用于测试）
func (f *RouterFactory) CreateSimpleRouter() *Router {
	return NewRouter(f.logger)
}
