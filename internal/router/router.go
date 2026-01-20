package router

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	handlers       map[string]Handler
	defaultHandler Handler // 默认处理器，处理未注册的方法
	mu             sync.RWMutex
	logger         *logrus.Logger
}

// NewRouter 创建新的 JSON-RPC 路由器
func NewRouter(logger *logrus.Logger) *Router {
	return &Router{
		handlers:       make(map[string]Handler),
		defaultHandler: nil,
		logger:         logger,
	}
}

// NewRouterWithContext 创建新的 JSON-RPC 路由器,使用 Entry 类型 logger
func NewRouterWithContext(logger *logrus.Entry) *Router {
	return &Router{
		handlers:       make(map[string]Handler),
		defaultHandler: nil,
		logger:         logger.Logger,
	}
}

// SetDefaultHandler 设置默认处理器
func (r *Router) SetDefaultHandler(handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.defaultHandler = handler
	r.logger.Info("Default handler set")
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

// RouteWithContext 路由单个请求,使用传入的logger
func (r *Router) RouteWithContext(ctx context.Context, request *jsonrpc.Request, logger *logrus.Entry) *jsonrpc.Response {
	if request == nil {
		return jsonrpc.NewErrorResponse(nil, jsonrpc.InvalidRequestError)
	}

	logger.WithFields(logrus.Fields{
		"method": request.Method,
		"id":     request.ID,
	}).Info("Routing request")

	handler, found := r.getHandler(request.Method)
	if !found {
		if r.defaultHandler != nil {
			logger.WithField("method", request.Method).Debug("Using default handler")
			handler = r.defaultHandler
		} else {
			logger.WithField("method", request.Method).Warn("Method not found")
			return jsonrpc.NewErrorResponse(request.ID, jsonrpc.MethodNotFoundError)
		}
	}

	response, err := handler.Handle(ctx, request)
	if err != nil {
		logger.WithError(err).Error("Handler execution failed")

		if jsonErr, ok := err.(*jsonrpc.Error); ok {
			return jsonrpc.NewErrorResponse(request.ID, jsonErr)
		}

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

	response.ID = request.ID
	response.JSONRPC = jsonrpc.JSONRPCVersion

	logger.Debug("Request routed successfully")
	return response
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

	logger.Info("Routing request")

	// 查找处理器
	handler, found := r.getHandler(request.Method)
	if !found {
		// 如果没有找到特定处理器，尝试使用默认处理器
		if r.defaultHandler != nil {
			logger.WithField("method", request.Method).Debug("Using default handler")
			handler = r.defaultHandler
		} else {
			logger.WithField("method", request.Method).Warn("Method not found")
			return jsonrpc.NewErrorResponse(request.ID, jsonrpc.MethodNotFoundError)
		}
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

	r.logger.WithFields(logrus.Fields{
		"count": len(requests),
	}).Info("Routing batch requests")

	// 处理批量请求
	responses := make([]*jsonrpc.Response, len(requests))

	for i, request := range requests {
		responses[i] = r.Route(ctx, &request)
	}

	r.logger.WithFields(logrus.Fields{
		"request_count":  len(requests),
		"response_count": len(responses),
	}).Info("Batch routing completed")
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

// HandleHTTPRequestWithContext 处理HTTP请求并传递带上下文的logger
func (r *Router) HandleHTTPRequestWithContext(w http.ResponseWriter, req *http.Request, logger *logrus.Entry) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		logger.WithError(err).Error("Failed to read request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error"},"id":null}`)); err != nil {
			logger.WithError(err).Error("Failed to write error response")
		}
		return
	}

	requests, err := jsonrpc.ParseRequest(body)
	if err != nil {
		logger.WithError(err).Warn("Failed to parse JSON-RPC request")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := jsonrpc.NewErrorResponse(nil, jsonrpc.ParseError)
		data, _ := jsonrpc.MarshalResponse(resp)
		if _, err := w.Write(data); err != nil {
			logger.WithError(err).Error("Failed to write error response")
		}
		return
	}

	responses := make([]*jsonrpc.Response, 0, len(requests))
	for i := range requests {
		resp := r.RouteWithContext(context.Background(), &requests[i], logger)
		responses = append(responses, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	data, err := jsonrpc.MarshalResponses(responses)
	if err != nil {
		logger.WithError(err).Error("Failed to marshal JSON-RPC responses")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(data); err != nil {
		logger.WithError(err).Error("Failed to write response")
	}
}

// HandleHTTPRequest 处理HTTP请求（用于集成到HTTP服务器）
func (r *Router) HandleHTTPRequest(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		r.logger.WithError(err).Error("Failed to read request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error"},"id":null}`)); err != nil {
			r.logger.WithError(err).Error("Failed to write error response")
		}
		return
	}

	requests, err := jsonrpc.ParseRequest(body)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to parse JSON-RPC request")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := jsonrpc.NewErrorResponse(nil, jsonrpc.ParseError)
		data, _ := jsonrpc.MarshalResponse(resp)
		if _, err := w.Write(data); err != nil {
			r.logger.WithError(err).Error("Failed to write error response")
		}
		return
	}

	responses := make([]*jsonrpc.Response, 0, len(requests))
	for i := range requests {
		resp := r.Route(context.Background(), &requests[i])
		responses = append(responses, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	data, err := jsonrpc.MarshalResponses(responses)
	if err != nil {
		r.logger.WithError(err).Error("Failed to marshal JSON-RPC responses")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(data); err != nil {
		r.logger.WithError(err).Error("Failed to write response")
	}
}
