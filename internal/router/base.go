package router

import (
	"encoding/json"
	"fmt"

	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/sirupsen/logrus"
)

// BaseHandler 提供处理器的基础功能
type BaseHandler struct {
	method string
	logger *logrus.Logger
}

// NewBaseHandler 创建基础处理器
func NewBaseHandler(method string, logger *logrus.Logger) *BaseHandler {
	return &BaseHandler{
		method: method,
		logger: logger,
	}
}

// Method 返回方法名
func (h *BaseHandler) Method() string {
	return h.method
}

// ValidateParams 验证参数是否为数组格式且长度正确
func (h *BaseHandler) ValidateParams(params json.RawMessage, expectedLength int) ([]interface{}, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("params is required")
	}

	var paramsArray []interface{}
	if err := json.Unmarshal(params, &paramsArray); err != nil {
		return nil, fmt.Errorf("params must be an array: %v", err)
	}

	if len(paramsArray) != expectedLength {
		return nil, fmt.Errorf("expected %d parameters, got %d", expectedLength, len(paramsArray))
	}

	return paramsArray, nil
}

// ParseParams 解析参数到指定类型
func (h *BaseHandler) ParseParams(params json.RawMessage, target interface{}) error {
	if len(params) == 0 {
		return fmt.Errorf("params is required")
	}

	// 尝试解析为数组格式，取第一个元素
	var paramsArray []json.RawMessage
	if err := json.Unmarshal(params, &paramsArray); err == nil && len(paramsArray) > 0 {
		return json.Unmarshal(paramsArray[0], target)
	}

	// 直接解析为对象
	return json.Unmarshal(params, target)
}

// CreateSuccessResponse 创建成功响应
func (h *BaseHandler) CreateSuccessResponse(id interface{}, result interface{}) (*jsonrpc.Response, error) {
	response, err := jsonrpc.NewResponse(id, result)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create success response")
		return nil, fmt.Errorf("failed to create response: %v", err)
	}
	return response, nil
}

// CreateErrorResponse 创建错误响应
func (h *BaseHandler) CreateErrorResponse(id interface{}, code int, message string, data interface{}) *jsonrpc.Response {
	err := &jsonrpc.Error{
		Code:    code,
		Message: message,
		Data:    data,
	}
	return jsonrpc.NewErrorResponse(id, err)
}

// CreateInvalidParamsResponse 创建无效参数响应
func (h *BaseHandler) CreateInvalidParamsResponse(id interface{}, message string) *jsonrpc.Response {
	return h.CreateErrorResponse(id, jsonrpc.CodeInvalidParams, message, nil)
}

// LogRequest 记录请求日志
func (h *BaseHandler) LogRequest(request *jsonrpc.Request) {
	h.logger.WithFields(logrus.Fields{
		"method": request.Method,
		"id":     request.ID,
		"params": string(request.Params),
	}).Debug("Processing JSON-RPC request")
}

// LogResponse 记录响应日志
func (h *BaseHandler) LogResponse(request *jsonrpc.Request, response *jsonrpc.Response, err error) {
	fields := logrus.Fields{
		"method": request.Method,
		"id":     request.ID,
	}

	if err != nil {
		fields["error"] = err.Error()
		h.logger.WithFields(fields).Error("Request processing failed")
	} else if response.Error != nil {
		fields["error_code"] = response.Error.Code
		fields["error_message"] = response.Error.Message
		h.logger.WithFields(fields).Warn("Request returned error")
	} else {
		h.logger.WithFields(fields).Debug("Request processed successfully")
	}
}
