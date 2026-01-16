package router

import (
	"context"
	"fmt"
	"sync"

	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/sirupsen/logrus"
)

// Handler 定义 JSON-RPC 方法处理器接口
type Handler interface {
	// Handle 处理 JSON-RPC 请求
	Handle(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error)

	// Method 返回处理器支持的方法名
	Method() string
}

// Router JSON-RPC 请求路由器
type Router struct {
	handlers map[string]Handler
	mu       sync.RWMutex
	logger   *logrus.Logger
}

// NewRouter 创建新的 JSON-RPC 路由器
func NewRouter(logger *logrus.Logger) *Router {
	return &Router{
		handlers: make(map[string]Handler),
		logger:   logger,
	}
}

// Register 注册处理器
func (r *Router) Register(handler Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	method := handler.Method()
	if method == "" {
		return fmt.Errorf("handler method name cannot be empty")
	}

	if _, exists := r.handlers[method]; exists {
		return fmt.Errorf("handler for method %s already registered", method)
	}

	r.handlers[method] = handler
	r.logger.WithField("method", method).Info("Registered JSON-RPC handler")
	return nil
}

// Unregister 注销处理器
func (r *Router) Unregister(method string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.handlers, method)
	r.logger.WithField("method", method).Info("Unregistered JSON-RPC handler")
}

// Route 路由单个请求
func (r *Router) Route(ctx context.Context, request *jsonrpc.Request) *jsonrpc.Response {
	if request == nil {
		return jsonrpc.NewErrorResponse(nil, jsonrpc.InvalidRequestError)
	}

	logger := r.logger.WithFields(logrus.Fields{
		"method": request.Method,
		"id":     request.ID,
	})

	logger.Debug("Routing JSON-RPC request")

	// 查找处理器
	handler, found := r.getHandler(request.Method)
	if !found {
		logger.WithField("method", request.Method).Warn("Method not found")
		return jsonrpc.NewErrorResponse(request.ID, jsonrpc.MethodNotFoundError)
	}

	// 调用处理器
	response, err := handler.Handle(ctx, request)
	if err != nil {
		logger.WithError(err).Error("Handler execution failed")

		// 检查是否是 JSON-RPC 错误
		if jsonErr, ok := err.(*jsonrpc.Error); ok {
			return jsonrpc.NewErrorResponse(request.ID, jsonErr)
		}

		// 创建内部错误响应
		return jsonrpc.NewErrorResponse(request.ID, jsonrpc.NewServerError(
			jsonrpc.CodeInternalError,
			"Internal server error",
			err.Error(),
		))
	}

	if response == nil {
		logger.Error("Handler returned nil response")
		return jsonrpc.NewErrorResponse(request.ID, jsonrpc.InternalError)
	}

	// 设置响应 ID
	response.ID = request.ID
	response.JSONRPC = jsonrpc.JSONRPCVersion

	logger.Debug("Request routed successfully")
	return response
}

// RouteBatch 路由批量请求
func (r *Router) RouteBatch(ctx context.Context, requests []jsonrpc.Request) []*jsonrpc.Response {
	if len(requests) == 0 {
		return []*jsonrpc.Response{
			jsonrpc.NewErrorResponse(nil, jsonrpc.InvalidRequestError),
		}
	}

	r.logger.WithField("count", len(requests)).Debug("Routing batch JSON-RPC requests")

	// 处理批量请求
	responses := make([]*jsonrpc.Response, len(requests))

	for i, request := range requests {
		responses[i] = r.Route(ctx, &request)
	}

	r.logger.WithField("count", len(responses)).Debug("Batch routing completed")
	return responses
}

// getHandler 获取处理器（线程安全）
func (r *Router) getHandler(method string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, found := r.handlers[method]
	return handler, found
}

// GetRegisteredMethods 获取已注册的方法列表
func (r *Router) GetRegisteredMethods() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	methods := make([]string, 0, len(r.handlers))
	for method := range r.handlers {
		methods = append(methods, method)
	}

	return methods
}

// HasHandler 检查是否有指定方法的处理器
func (r *Router) HasHandler(method string) bool {
	_, found := r.getHandler(method)
	return found
}
