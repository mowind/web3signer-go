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

// Handler defines a JSON-RPC method handler interface.
//
// Implementations of this interface can be registered with the Router
// to handle specific JSON-RPC methods.
type Handler interface {
	// Handle processes a JSON-RPC request.
	//
	// Parameters:
	//   - ctx: Context for request (supports cancellation and timeout)
	//   - request: The JSON-RPC request to handle
	//
	// Returns:
	//   - *jsonrpc.Response: The response to return to client
	//   - error: An error if handling fails
	Handle(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error)

	// Method returns the JSON-RPC method name this handler supports.
	//
	// Returns:
	//   - string: The method name (e.g., "eth_sendTransaction")
	Method() string
}

// Router routes JSON-RPC requests to appropriate handlers.
//
// This router supports:
//   - Method-based handler registration
//   - Default handler for unregistered methods
//   - Thread-safe operations
//   - Request size limiting
type Router struct {
	handlers       map[string]Handler
	defaultHandler Handler // 默认处理器，处理未注册的方法
	mu             sync.RWMutex
	logger         *logrus.Logger
	maxRequestSize int64 // 最大请求体大小（字节）
}

// NewRouter creates a new JSON-RPC router with default settings.
//
// Default max request size is 10MB.
//
// Parameters:
//   - logger: The logger to use for request logging
//
// Returns:
//   - *Router: A new router instance
func NewRouter(logger *logrus.Logger) *Router {
	return NewRouterWithMaxSize(logger, 10*1024*1024)
}

// NewRouterWithMaxSize creates a new JSON-RPC router with custom max request size.
//
// Parameters:
//   - logger: The logger to use for request logging
//   - maxRequestSize: Maximum allowed request body size in bytes
//
// Returns:
//   - *Router: A new router instance
func NewRouterWithMaxSize(logger *logrus.Logger, maxRequestSize int64) *Router {
	return &Router{
		handlers:       make(map[string]Handler),
		defaultHandler: nil,
		logger:         logger,
		maxRequestSize: maxRequestSize,
	}
}

// NewRouterWithContext creates a new JSON-RPC router with the provided logger entry.
//
// This allows the router to share context fields from the parent logger.
//
// Parameters:
//   - logger: The logrus entry to use
//
// Returns:
//   - *Router: A new router instance
func (r *Router) NewRouterWithContext(logger *logrus.Entry) *Router {
	return NewRouterWithContextAndMaxSize(logger, 10*1024*1024)
}

// NewRouterWithContextAndMaxSize creates a new router with context logger and size limit.
//
// Parameters:
//   - logger: The logrus entry to use
//   - maxRequestSize: Maximum allowed request body size in bytes
//
// Returns:
//   - *Router: A new router instance
func NewRouterWithContextAndMaxSize(logger *logrus.Entry, maxRequestSize int64) *Router {
	return &Router{
		handlers:       make(map[string]Handler),
		defaultHandler: nil,
		logger:         logger.Logger,
		maxRequestSize: maxRequestSize,
	}
}

// SetDefaultHandler sets the default handler for unregistered methods.
//
// This handler is called when a method is not registered.
//
// Parameters:
//   - handler: The handler to use for unregistered methods
func (r *Router) SetDefaultHandler(handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.defaultHandler = handler
	r.logger.Info("Default handler set")
}

// Register registers a JSON-RPC method handler.
//
// The handler's Method() return value is used as the registration key.
//
// Parameters:
//   - handler: The handler to register
//
// Returns:
//   - error: An error if handler method is empty or already registered
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

// Unregister removes a handler for the specified method.
//
// Parameters:
//   - method: The JSON-RPC method name to unregister
func (r *Router) Unregister(method string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.handlers, method)
	r.logger.WithField("method", method).Info("Unregistered JSON-RPC handler")
}

// routeRequest is a helper function that handles routing logic for a single request.
//
// It performs handler lookup, execution, and error handling.
//
// Parameters:
//   - ctx: Request context
//   - request: The JSON-RPC request
//   - logger: Context-aware logger
//
// Returns:
//   - *jsonrpc.Response: The execution result
func (r *Router) routeRequest(ctx context.Context, request *jsonrpc.Request, logger *logrus.Entry) *jsonrpc.Response {
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

// RouteWithContext routes a single request using the provided logger entry.
//
// This is useful for maintaining log context across request lifecycle.
//
// Parameters:
//   - ctx: Request context
//   - request: The JSON-RPC request
//   - logger: Context-aware logger
//
// Returns:
//   - *jsonrpc.Response: The handler response
func (r *Router) RouteWithContext(ctx context.Context, request *jsonrpc.Request, logger *logrus.Entry) *jsonrpc.Response {
	return r.routeRequest(ctx, request, logger)
}

// Route routes a single JSON-RPC request to the appropriate handler.
//
// This method handles request validation, method lookup, handler execution,
// and error response generation.
//
// Parameters:
//   - ctx: Context for request (supports cancellation and timeout)
//   - request: The JSON-RPC request to route
//
// Returns:
//   - *jsonrpc.Response: The response from the handler
func (r *Router) Route(ctx context.Context, request *jsonrpc.Request) *jsonrpc.Response {
	if request == nil {
		r.logger.Warn("Received nil JSON-RPC request")
		return jsonrpc.NewErrorResponse(nil, jsonrpc.InvalidRequestError)
	}
	logger := r.logger.WithFields(logrus.Fields{
		"method": request.Method,
		"id":     request.ID,
	})
	return r.routeRequest(ctx, request, logger)
}

// MaxBatchSize defines the maximum number of requests allowed in a batch
const MaxBatchSize = 100

// DefaultBatchWorkerCount defines the default number of workers for batch request processing
const DefaultBatchWorkerCount = 50

// RouteBatch routes a batch of JSON-RPC requests.
//
// Each request in the batch is routed independently using a worker pool.
//
// Parameters:
//   - ctx: Context for requests (supports cancellation and timeout)
//   - requests: The JSON-RPC requests to route
//
// Returns:
//   - []*jsonrpc.Response: Ordered responses matching request order
func (r *Router) RouteBatch(ctx context.Context, requests []jsonrpc.Request) []*jsonrpc.Response {
	if len(requests) == 0 {
		return []*jsonrpc.Response{
			jsonrpc.NewErrorResponse(nil, jsonrpc.InvalidRequestError),
		}
	}

	if len(requests) > MaxBatchSize {
		r.logger.WithField("count", len(requests)).Warn("Batch size exceeds limit")
		return []*jsonrpc.Response{
			jsonrpc.NewErrorResponse(nil, jsonrpc.NewServerError(
				-32602, "Invalid params", fmt.Sprintf("Batch size exceeds maximum limit of %d", MaxBatchSize)),
			),
		}
	}

	r.logger.WithFields(logrus.Fields{
		"count": len(requests),
	}).Info("Routing batch requests")

	// Create responses array
	responses := make([]*jsonrpc.Response, len(requests))

	taskCount := len(requests)
	taskCh := make(chan int, taskCount)

	workerCount := DefaultBatchWorkerCount
	if taskCount < workerCount {
		workerCount = taskCount
	}

	var wg sync.WaitGroup

	// Start task feeder goroutine (fan-in)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(taskCh)
		for i := 0; i < taskCount; i++ {
			taskCh <- i
		}
	}()

	// Start workers (fan-out)
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for idx := range taskCh {
				if ctx.Err() != nil {
					break
				}

				func() {
					defer func() {
						if p := recover(); p != nil {
							r.logger.WithField("worker_id", workerID).WithField("panic", p).Error("Worker panic recovered")
							responses[idx] = jsonrpc.NewErrorResponse(
								requests[idx].ID,
								jsonrpc.NewServerError(-32603, "Internal error", "Processing failed"),
							)
						}
					}()

					responses[idx] = r.Route(ctx, &requests[idx])
					if responses[idx] == nil {
						r.logger.WithField("worker_id", workerID).WithField("idx", idx).Warn("Route returned nil response, setting error")
						responses[idx] = jsonrpc.NewErrorResponse(
							requests[idx].ID,
							jsonrpc.NewServerError(-32603, "Internal error", "Processing failed"),
						)
					}
				}()
			}
		}(i)
	}

	wg.Wait()

	r.logger.WithFields(logrus.Fields{
		"request_count":  taskCount,
		"response_count": len(responses),
	}).Info("Batch routing completed")
	return responses
}

// getHandler retrieves a registered handler for the given method name.
//
// This method is thread-safe using a read lock.
//
// Parameters:
//   - method: The JSON-RPC method name
//
// Returns:
//   - Handler: The handler instance if found
//   - bool: True if found, false otherwise
func (r *Router) getHandler(method string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handler, found := r.handlers[method]
	return handler, found
}

// GetRegisteredMethods returns a list of all registered method names.
//
// Returns:
//   - []string: List of registered JSON-RPC method names
func (r *Router) GetRegisteredMethods() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	methods := make([]string, 0, len(r.handlers))
	for method := range r.handlers {
		methods = append(methods, method)
	}

	return methods
}

// HasHandler checks if a handler is registered for the given method.
//
// Parameters:
//   - method: The JSON-RPC method name to check
//
// Returns:
//   - bool: True if handler is registered, false otherwise
func (r *Router) HasHandler(method string) bool {
	_, found := r.getHandler(method)
	return found
}

// parseAndRoute parses the request body and routes requests to handlers.
//
// This is a helper method used by HandleHTTPRequestWithContext.
//
// Parameters:
//   - w: HTTP response writer
//   - req: HTTP request
//   - logger: Logger entry for tracing
//   - body: The request body content
func (r *Router) parseAndRoute(w http.ResponseWriter, req *http.Request, logger *logrus.Entry, body []byte) {
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

	if len(requests) > MaxBatchSize {
		logger.WithField("count", len(requests)).Warn("Batch size exceeds limit")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := jsonrpc.NewErrorResponse(nil, jsonrpc.NewServerError(
			-32602, "Invalid params", fmt.Sprintf("Batch size exceeds maximum limit of %d", MaxBatchSize)),
		)
		data, _ := jsonrpc.MarshalResponse(resp)
		if _, err := w.Write(data); err != nil {
			logger.WithError(err).Error("Failed to write error response")
		}
		return
	}

	// If we have default handler and it supports batch forwarding, use optimized batch handling
	if r.defaultHandler != nil {
		// Check if default handler is ForwardHandler by inspecting its method
		if fwdHandler, ok := r.defaultHandler.(*ForwardHandler); ok {
			r.handleBatchWithForwarding(w, req, logger, requests, fwdHandler)
			return
		}
	}

	// Fallback to sequential processing for single request or non-forward handlers
	responses := make([]*jsonrpc.Response, 0, len(requests))
	for i := range requests {
		resp := r.RouteWithContext(req.Context(), &requests[i], logger)
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

// handleBatchWithForwarding processes batch requests by separating sign and forward requests
// for optimized batch forwarding to downstream services.
//
// It routes sign requests through registered handlers and forwards other requests
// in bulk to the downstream service, preserving request order in responses.
func (r *Router) handleBatchWithForwarding(w http.ResponseWriter, req *http.Request, logger *logrus.Entry, requests []jsonrpc.Request, fwdHandler *ForwardHandler) {
	if len(requests) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := jsonrpc.NewErrorResponse(nil, jsonrpc.InvalidRequestError)
		data, _ := jsonrpc.MarshalResponse(resp)
		if _, err := w.Write(data); err != nil {
			logger.WithError(err).Error("Failed to write response")
		}
		return
	}

	// Create response array to maintain order
	responses := make([]*jsonrpc.Response, len(requests))

	// Split requests into sign requests and forward requests
	signIndices := make([]int, 0)
	forwardIndices := make([]int, 0)
	forwardRequests := make([]jsonrpc.Request, 0)

	for i, request := range requests {
		if IsSignMethod(request.Method) {
			signIndices = append(signIndices, i)
		} else {
			forwardIndices = append(forwardIndices, i)
			forwardRequests = append(forwardRequests, request)
		}
	}

	// Process sign requests sequentially
	ctx := req.Context()
	for _, idx := range signIndices {
		handler, found := r.getHandler(requests[idx].Method)
		if !found {
			responses[idx] = jsonrpc.NewErrorResponse(requests[idx].ID, jsonrpc.MethodNotFoundError)
			continue
		}

		response, err := handler.Handle(ctx, &requests[idx])
		switch {
		case err != nil:
			if jsonErr, ok := err.(*jsonrpc.Error); ok {
				responses[idx] = jsonrpc.NewErrorResponse(requests[idx].ID, jsonErr)
			} else {
				responses[idx] = jsonrpc.NewErrorResponse(requests[idx].ID, jsonrpc.NewServerError(
					jsonrpc.CodeInternalError,
					"Internal server error",
					err.Error(),
				))
			}
		case response == nil:
			responses[idx] = jsonrpc.NewErrorResponse(requests[idx].ID, jsonrpc.InternalError)
		default:
			response.ID = requests[idx].ID
			response.JSONRPC = jsonrpc.JSONRPCVersion
			responses[idx] = response
		}
	}

	// Process forward requests in batch if there are any
	if len(forwardRequests) > 0 {
		downstreamClient := fwdHandler.Client()
		if batchResponses, err := downstreamClient.ForwardBatchRequest(ctx, forwardRequests); err == nil {
			for i, idx := range forwardIndices {
				if i < len(batchResponses) {
					responses[idx] = &batchResponses[i]
				} else {
					responses[idx] = jsonrpc.NewErrorResponse(requests[idx].ID, jsonrpc.InternalError)
				}
			}
		} else {
			for _, idx := range forwardIndices {
				responses[idx] = jsonrpc.NewErrorResponse(requests[idx].ID, jsonrpc.NewServerError(
					jsonrpc.CodeInternalError,
					"Failed to forward batch request",
					err.Error(),
				))
			}
		}
	}

	// Write response
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

// HandleHTTPRequestWithContext handles HTTP requests with context-aware logging.
//
// This method parses JSON-RPC requests from HTTP and routes them.
// It supports CORS preflight (OPTIONS) requests.
//
// Parameters:
//   - w: HTTP response writer
//   - req: HTTP request
//   - logger: Logger entry with context fields for tracing
func (r *Router) HandleHTTPRequestWithContext(w http.ResponseWriter, req *http.Request, logger *logrus.Entry) {
	if req.Method == "OPTIONS" {
		maxBody := r.maxRequestSize
		limitedBody := http.MaxBytesReader(w, req.Body, maxBody)
		body, err := io.ReadAll(limitedBody)
		if err != nil {
			logger.WithError(err).Error("Failed to read request body")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			if _, err := w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32602,"message":"Request entity too large"},"id":null}`)); err != nil {
				logger.WithError(err).Error("Failed to write error response")
			}
			return
		}
		r.parseAndRoute(w, req, logger, body)
		return
	}

	maxBody := r.maxRequestSize
	limitedBody := http.MaxBytesReader(w, req.Body, maxBody)
	body, err := io.ReadAll(limitedBody)
	if err != nil {
		logger.WithError(err).WithField("max_size_bytes", maxBody).Error("Request body too large")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		if _, err := w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32602,"message":"Request entity too large"},"id":null}`)); err != nil {
			logger.WithError(err).Error("Failed to write error response")
		}
		return
	}

	r.parseAndRoute(w, req, logger, body)
}

// HandleHTTPRequest handles HTTP requests for server integration.
//
// This is a convenience method that uses the router's logger.
//
// Parameters:
//   - w: HTTP response writer
//   - req: HTTP request
func (r *Router) HandleHTTPRequest(w http.ResponseWriter, req *http.Request) {
	if req.Method == "OPTIONS" {
		maxBody := r.maxRequestSize
		limitedBody := http.MaxBytesReader(w, req.Body, maxBody)
		body, err := io.ReadAll(limitedBody)
		if err != nil {
			r.logger.WithError(err).Error("Failed to read request body")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			if _, err := w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32602,"message":"Request entity too large"},"id":null}`)); err != nil {
				r.logger.WithError(err).Error("Failed to write error response")
			}
			return
		}
		r.parseAndRouteSimple(w, req, body)
		return
	}

	maxBody := r.maxRequestSize
	limitedBody := http.MaxBytesReader(w, req.Body, maxBody)
	body, err := io.ReadAll(limitedBody)
	if err != nil {
		r.logger.WithError(err).WithField("max_size_bytes", maxBody).Error("Request body too large")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		if _, err := w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32602,"message":"Request entity too large"},"id":null}`)); err != nil {
			r.logger.WithError(err).Error("Failed to write error response")
		}
		return
	}

	r.parseAndRouteSimple(w, req, body)
}

// parseAndRouteSimple parses and routes requests using the router's default logger.
//
// This is a helper method used by HandleHTTPRequest.
//
// Parameters:
//   - w: HTTP response writer
//   - req: HTTP request
//   - body: The request body content
func (r *Router) parseAndRouteSimple(w http.ResponseWriter, req *http.Request, body []byte) {
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

	if len(requests) > MaxBatchSize {
		r.logger.WithField("count", len(requests)).Warn("Batch size exceeds limit")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := jsonrpc.NewErrorResponse(nil, jsonrpc.NewServerError(
			-32602, "Invalid params", fmt.Sprintf("Batch size exceeds maximum limit of %d", MaxBatchSize)),
		)
		data, _ := jsonrpc.MarshalResponse(resp)
		if _, err := w.Write(data); err != nil {
			r.logger.WithError(err).Error("Failed to write error response")
		}
		return
	}

	responses := make([]*jsonrpc.Response, 0, len(requests))
	for i := range requests {
		resp := r.Route(req.Context(), &requests[i])
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
