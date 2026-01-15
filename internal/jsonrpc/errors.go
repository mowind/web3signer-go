package jsonrpc

import "fmt"

// 标准 JSON-RPC 错误码
const (
	// 解析错误
	CodeParseError = -32700

	// 无效请求
	CodeInvalidRequest = -32600

	// 方法不存在
	CodeMethodNotFound = -32601

	// 无效参数
	CodeInvalidParams = -32602

	// 内部错误
	CodeInternalError = -32603

	// 服务器错误（-32000 到 -32099 为服务器保留错误码）
	CodeServerErrorStart = -32000
	CodeServerErrorEnd   = -32099
)

// 标准错误
var (
	// ParseError 表示解析错误
	ParseError = &Error{
		Code:    CodeParseError,
		Message: "Parse error",
	}

	// InvalidRequestError 表示无效请求错误
	InvalidRequestError = &Error{
		Code:    CodeInvalidRequest,
		Message: "Invalid request",
	}

	// MethodNotFoundError 表示方法不存在错误
	MethodNotFoundError = &Error{
		Code:    CodeMethodNotFound,
		Message: "Method not found",
	}

	// InvalidParamsError 表示无效参数错误
	InvalidParamsError = &Error{
		Code:    CodeInvalidParams,
		Message: "Invalid params",
	}

	// InternalError 表示内部错误
	InternalError = &Error{
		Code:    CodeInternalError,
		Message: "Internal error",
	}
)

// NewServerError 创建服务器错误
func NewServerError(code int, message string, data interface{}) *Error {
	if code < CodeServerErrorStart || code > CodeServerErrorEnd {
		// 如果超出范围，使用内部错误
		return &Error{
			Code:    CodeInternalError,
			Message: message,
			Data:    data,
		}
	}

	return &Error{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewCustomError 创建自定义错误
func NewCustomError(code int, message string, data interface{}) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// Errorf 创建带格式的错误
func Errorf(code int, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// IsServerError 检查错误码是否为服务器错误
func IsServerError(code int) bool {
	return code >= CodeServerErrorStart && code <= CodeServerErrorEnd
}

// Error 实现 error 接口
func (e *Error) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("JSON-RPC error %d: %s (data: %v)", e.Code, e.Message, e.Data)
	}
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}