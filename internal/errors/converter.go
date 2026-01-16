package errors

import (
	"github.com/mowind/web3signer-go/internal/downstream"
	"github.com/mowind/web3signer-go/internal/jsonrpc"
)

// Converter 错误转换器
type Converter struct{}

// NewConverter 创建新的错误转换器
func NewConverter() *Converter {
	return &Converter{}
}

// FromJSONRPC 从 JSON-RPC 错误转换
func (c *Converter) FromJSONRPC(jsonErr *jsonrpc.Error) *AppError {
	if jsonErr == nil {
		return nil
	}

	// 根据错误码映射错误类型
	var errorType ErrorType
	switch jsonErr.Code {
	case jsonrpc.CodeParseError:
		errorType = ErrorTypeJSONRPC
	case jsonrpc.CodeInvalidRequest:
		errorType = ErrorTypeJSONRPC
	case jsonrpc.CodeMethodNotFound:
		errorType = ErrorTypeMethodNotFound
	case jsonrpc.CodeInvalidParams:
		errorType = ErrorTypeInvalidParams
	case jsonrpc.CodeInternalError:
		errorType = ErrorTypeInternal
	default:
		if jsonrpc.IsServerError(jsonErr.Code) {
			errorType = ErrorTypeInternal
		} else {
			errorType = ErrorTypeJSONRPC
		}
	}

	return &AppError{
		Type:    errorType,
		Code:    jsonErr.Code,
		Message: jsonErr.Message,
		Context: map[string]interface{}{
			"original_data": jsonErr.Data,
		},
	}
}

// ToJSONRPC 转换为 JSON-RPC 错误
func (c *Converter) ToJSONRPC(appErr *AppError) *jsonrpc.Error {
	if appErr == nil {
		return nil
	}
	return appErr.ToJSONRPCError()
}

// ToJSONRPCWithCode 转换为 JSON-RPC 错误并指定错误码
func (c *Converter) ToJSONRPCWithCode(appErr *AppError, code int) *jsonrpc.Error {
	if appErr == nil {
		return nil
	}

	// 构建错误数据
	errorData := map[string]interface{}{
		"type":    string(appErr.Type),
		"code":    appErr.Code,
		"details": appErr.Details,
	}

	// 添加上下文信息
	for k, v := range appErr.Context {
		errorData[k] = v
	}

	return &jsonrpc.Error{
		Code:    code,
		Message: appErr.Message,
		Data:    errorData,
	}
}

// FromDownstream 从下游服务错误转换
func (c *Converter) FromDownstream(downstreamErr error) *AppError {
	if downstreamErr == nil {
		return nil
	}

	// 检查是否是下游服务错误类型
	if err, ok := downstreamErr.(*downstream.Error); ok {
		var errorType ErrorType
		var appErr *AppError

		switch err.Code {
		case downstream.ErrorCodeConnectionFailed:
			errorType = ErrorTypeConnection
			appErr = Wrap(err, errorType, jsonrpc.CodeServerErrorStart, "Connection to downstream service failed")
		case downstream.ErrorCodeRequestFailed:
			errorType = ErrorTypeForward
			appErr = Wrap(err, errorType, jsonrpc.CodeServerErrorStart+1, "Request forwarding failed")
		case downstream.ErrorCodeInvalidResponse:
			errorType = ErrorTypeDownstream
			appErr = Wrap(err, errorType, jsonrpc.CodeServerErrorStart+2, "Invalid response from downstream service")
		case downstream.ErrorCodeTimeout:
			errorType = ErrorTypeTimeout
			appErr = Wrap(err, errorType, jsonrpc.CodeServerErrorStart+3, "Downstream service timeout")
		case downstream.ErrorCodeIDMismatch:
			errorType = ErrorTypeDownstream
			appErr = Wrap(err, errorType, jsonrpc.CodeServerErrorStart+4, "Response ID mismatch from downstream service")
		case downstream.ErrorCodeBatchSizeMismatch:
			errorType = ErrorTypeDownstream
			appErr = Wrap(err, errorType, jsonrpc.CodeServerErrorStart+5, "Batch response size mismatch from downstream service")
		default:
			errorType = ErrorTypeDownstream
			appErr = Wrap(err, errorType, jsonrpc.CodeServerErrorStart+10, "Downstream service error")
		}

		return appErr
	}

	// 普通错误，包装为下游服务错误
	return Wrap(downstreamErr, ErrorTypeDownstream, jsonrpc.CodeServerErrorStart+10, "Downstream service error")
}

// ToDownstream 转换为下游服务错误（如果适用）
func (c *Converter) ToDownstream(appErr *AppError) error {
	if appErr == nil {
		return nil
	}

	// 只有下游服务相关的错误才转换
	switch appErr.Type {
	case ErrorTypeConnection, ErrorTypeTimeout, ErrorTypeNetwork, ErrorTypeDownstream, ErrorTypeForward:
		var downstreamCode downstream.ErrorCode
		switch appErr.Type {
		case ErrorTypeConnection:
			downstreamCode = downstream.ErrorCodeConnectionFailed
		case ErrorTypeTimeout:
			downstreamCode = downstream.ErrorCodeTimeout
		case ErrorTypeNetwork:
			downstreamCode = downstream.ErrorCodeConnectionFailed
		default:
			downstreamCode = downstream.ErrorCodeRequestFailed
		}

		return downstream.NewError(downstreamCode, appErr.Message, appErr.OriginalErr)
	}

	// 其他错误不转换，返回原始错误
	return appErr
}

// Common conversion helpers 常用转换辅助函数

// ConvertError 通用的错误转换函数
func ConvertError(err error) *AppError {
	if err == nil {
		return nil
	}

	// 如果已经是 AppError，直接返回
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	// 如果是 JSON-RPC 错误
	if jsonErr, ok := err.(*jsonrpc.Error); ok {
		return NewConverter().FromJSONRPC(jsonErr)
	}

	// 如果是下游服务错误
	if downstreamErr, ok := err.(*downstream.Error); ok {
		return NewConverter().FromDownstream(downstreamErr)
	}

	// 普通错误，包装为内部错误
	return Wrap(err, ErrorTypeInternal, jsonrpc.CodeInternalError, "Internal error")
}

// MustConvertError 转换错误，如果为 nil 则 panic
func MustConvertError(err error) *AppError {
	if err == nil {
		panic("MustConvertError: err cannot be nil")
	}
	return ConvertError(err)
}

// ConvertToJSONRPC 快速转换为 JSON-RPC 错误
func ConvertToJSONRPC(err error) *jsonrpc.Error {
	if err == nil {
		return nil
	}

	appErr := ConvertError(err)
	return appErr.ToJSONRPCError()
}

// IsErrorType 检查错误是否属于指定类型
func IsErrorType(err error, errorType ErrorType) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == errorType
	}
	return false
}

// IsRetryable 检查错误是否可重试
func IsRetryable(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		switch appErr.Type {
		case ErrorTypeConnection, ErrorTypeTimeout, ErrorTypeNetwork, ErrorTypeKMSUnavailable:
			return true
		}
	}
	return false
}

// IsClientError 检查是否是客户端错误（4xx 类）
func IsClientError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		switch appErr.Type {
		case ErrorTypeValidation, ErrorTypeInvalidParams, ErrorTypeMethodNotFound, ErrorTypeAddressMismatch:
			return true
		}
	}
	return false
}

// IsServerError 检查是否是服务器错误（5xx 类）
func IsServerError(err error) bool {
	return !IsClientError(err)
}
