package errors

import (
	"fmt"

	"github.com/mowind/web3signer-go/internal/jsonrpc"
)

// ErrorType 错误类型
type ErrorType string

const (
	// 系统级错误
	ErrorTypeInternal   ErrorType = "INTERNAL_ERROR"
	ErrorTypeConfig     ErrorType = "CONFIG_ERROR"
	ErrorTypeValidation ErrorType = "VALIDATION_ERROR"

	// 网络/连接错误
	ErrorTypeConnection ErrorType = "CONNECTION_ERROR"
	ErrorTypeTimeout    ErrorType = "TIMEOUT_ERROR"
	ErrorTypeNetwork    ErrorType = "NETWORK_ERROR"

	// KMS 相关错误
	ErrorTypeKMSSign        ErrorType = "KMS_SIGN_ERROR"
	ErrorTypeKMSAuth        ErrorType = "KMS_AUTH_ERROR"
	ErrorTypeKMSUnavailable ErrorType = "KMS_UNAVAILABLE"

	// 签名相关错误
	ErrorTypeSign             ErrorType = "SIGN_ERROR"
	ErrorTypeInvalidSignature ErrorType = "INVALID_SIGNATURE"
	ErrorTypeAddressMismatch  ErrorType = "ADDRESS_MISMATCH"

	// 交易相关错误
	ErrorTypeInvalidTransaction ErrorType = "INVALID_TRANSACTION"
	ErrorTypeTransactionBuild   ErrorType = "TRANSACTION_BUILD_ERROR"

	// JSON-RPC 相关错误
	ErrorTypeJSONRPC        ErrorType = "JSONRPC_ERROR"
	ErrorTypeMethodNotFound ErrorType = "METHOD_NOT_FOUND"
	ErrorTypeInvalidParams  ErrorType = "INVALID_PARAMS"

	// 下游服务错误
	ErrorTypeDownstream ErrorType = "DOWNSTREAM_ERROR"
	ErrorTypeForward    ErrorType = "FORWARD_ERROR"
)

// AppError 应用统一的错误类型
type AppError struct {
	Type        ErrorType              `json:"type"`
	Code        int                    `json:"code"`
	Message     string                 `json:"message"`
	Details     string                 `json:"details,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	OriginalErr error                  `json:"-"`
}

// New 创建新的应用错误
func New(errorType ErrorType, code int, message string) *AppError {
	return &AppError{
		Type:    errorType,
		Code:    code,
		Message: message,
		Context: make(map[string]interface{}),
	}
}

// Newf 创建带格式的应用错误
func Newf(errorType ErrorType, code int, format string, args ...interface{}) *AppError {
	return New(errorType, code, fmt.Sprintf(format, args...))
}

// Wrap 包装现有错误
func Wrap(err error, errorType ErrorType, code int, message string) *AppError {
	if err == nil {
		return nil
	}

	appErr := New(errorType, code, message)
	appErr.OriginalErr = err
	appErr.Details = err.Error()
	return appErr
}

// Wrapf 包装现有错误并带格式
func Wrapf(err error, errorType ErrorType, code int, format string, args ...interface{}) *AppError {
	if err == nil {
		return nil
	}
	return Wrap(err, errorType, code, fmt.Sprintf(format, args...))
}

// WithContext 添加上下文信息
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithDetails 添加详细信息
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.OriginalErr != nil && e.Details != "" {
		return fmt.Sprintf("%s [%s:%d]: %s (details: %s)", e.Message, e.Type, e.Code, e.OriginalErr.Error(), e.Details)
	}
	if e.OriginalErr != nil {
		return fmt.Sprintf("%s [%s:%d]: %s", e.Message, e.Type, e.Code, e.OriginalErr.Error())
	}
	if e.Details != "" {
		return fmt.Sprintf("%s [%s:%d] (details: %s)", e.Message, e.Type, e.Code, e.Details)
	}
	return fmt.Sprintf("%s [%s:%d]", e.Message, e.Type, e.Code)
}

// Unwrap 返回原始错误
func (e *AppError) Unwrap() error {
	return e.OriginalErr
}

// Is 检查错误类型
func (e *AppError) Is(target error) bool {
	if targetErr, ok := target.(*AppError); ok {
		return e.Type == targetErr.Type
	}
	return false
}

// ToJSONRPCError 转换为 JSON-RPC 错误
func (e *AppError) ToJSONRPCError() *jsonrpc.Error {
	// 根据错误类型映射到合适的 JSON-RPC 错误码
	var jsonrpcCode int
	switch e.Type {
	case ErrorTypeInvalidParams, ErrorTypeValidation:
		jsonrpcCode = jsonrpc.CodeInvalidParams
	case ErrorTypeMethodNotFound:
		jsonrpcCode = jsonrpc.CodeMethodNotFound
	case ErrorTypeInternal, ErrorTypeConfig:
		jsonrpcCode = jsonrpc.CodeInternalError
	case ErrorTypeConnection, ErrorTypeTimeout, ErrorTypeNetwork:
		jsonrpcCode = jsonrpc.CodeServerErrorStart
	default:
		jsonrpcCode = jsonrpc.CodeInternalError
	}

	// 构建错误数据
	errorData := map[string]interface{}{
		"type":    string(e.Type),
		"code":    e.Code,
		"details": e.Details,
	}

	// 添加上下文信息
	for k, v := range e.Context {
		errorData[k] = v
	}

	return &jsonrpc.Error{
		Code:    jsonrpcCode,
		Message: e.Message,
		Data:    errorData,
	}
}

// Common errors 常用错误
var (
	// 内部错误
	ErrInternal   = New(ErrorTypeInternal, jsonrpc.CodeInternalError, "Internal server error")
	ErrConfig     = New(ErrorTypeConfig, jsonrpc.CodeInternalError, "Configuration error")
	ErrValidation = New(ErrorTypeValidation, jsonrpc.CodeInvalidParams, "Validation failed")

	// 连接错误
	ErrConnection = New(ErrorTypeConnection, jsonrpc.CodeServerErrorStart, "Connection failed")
	ErrTimeout    = New(ErrorTypeTimeout, jsonrpc.CodeServerErrorStart+1, "Request timeout")
	ErrNetwork    = New(ErrorTypeNetwork, jsonrpc.CodeServerErrorStart+2, "Network error")

	// KMS 错误
	ErrKMSSign        = New(ErrorTypeKMSSign, jsonrpc.CodeServerErrorStart+10, "KMS signing failed")
	ErrKMSAuth        = New(ErrorTypeKMSAuth, jsonrpc.CodeServerErrorStart+11, "KMS authentication failed")
	ErrKMSUnavailable = New(ErrorTypeKMSUnavailable, jsonrpc.CodeServerErrorStart+12, "KMS service unavailable")

	// 签名错误
	ErrSign             = New(ErrorTypeSign, jsonrpc.CodeServerErrorStart+20, "Signing failed")
	ErrInvalidSignature = New(ErrorTypeInvalidSignature, jsonrpc.CodeServerErrorStart+21, "Invalid signature")
	ErrAddressMismatch  = New(ErrorTypeAddressMismatch, jsonrpc.CodeInvalidParams, "Address mismatch")

	// 交易错误
	ErrInvalidTransaction = New(ErrorTypeInvalidTransaction, jsonrpc.CodeInvalidParams, "Invalid transaction")
	ErrTransactionBuild   = New(ErrorTypeTransactionBuild, jsonrpc.CodeInternalError, "Transaction building failed")

	// JSON-RPC 错误
	ErrMethodNotFound = New(ErrorTypeMethodNotFound, jsonrpc.CodeMethodNotFound, "Method not found")
	ErrInvalidParams  = New(ErrorTypeInvalidParams, jsonrpc.CodeInvalidParams, "Invalid parameters")

	// 下游服务错误
	ErrDownstream = New(ErrorTypeDownstream, jsonrpc.CodeServerErrorStart+30, "Downstream service error")
	ErrForward    = New(ErrorTypeForward, jsonrpc.CodeServerErrorStart+31, "Request forwarding failed")
)
