package downstream

import "fmt"

// Error 表示下游服务错误
type Error struct {
	Code    ErrorCode
	Message string
	Err     error
}

// ErrorCode 错误码
type ErrorCode int

const (
	// ErrorCodeConnectionFailed 连接失败
	ErrorCodeConnectionFailed ErrorCode = iota + 1
	// ErrorCodeRequestFailed 请求失败
	ErrorCodeRequestFailed
	// ErrorCodeInvalidResponse 无效响应
	ErrorCodeInvalidResponse
	// ErrorCodeTimeout 超时
	ErrorCodeTimeout
	// ErrorCodeIDMismatch ID不匹配
	ErrorCodeIDMismatch
	// ErrorCodeBatchSizeMismatch 批量大小不匹配
	ErrorCodeBatchSizeMismatch
)

// Error 实现error接口
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("downstream error [%d]: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("downstream error [%d]: %s", e.Code, e.Message)
}

// Unwrap 返回包装的错误
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError 创建新的下游服务错误
func NewError(code ErrorCode, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsConnectionError 检查是否是连接错误
func IsConnectionError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrorCodeConnectionFailed
	}
	return false
}

// IsTimeoutError 检查是否是超时错误
func IsTimeoutError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrorCodeTimeout
	}
	return false
}

// IsInvalidResponseError 检查是否是无效响应错误
func IsInvalidResponseError(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == ErrorCodeInvalidResponse
	}
	return false
}

// WrapError 包装错误为下游服务错误
func WrapError(err error, code ErrorCode, message string) error {
	if err == nil {
		return nil
	}
	return NewError(code, message, err)
}

// ConnectionError 创建连接错误
func ConnectionError(err error) error {
	return WrapError(err, ErrorCodeConnectionFailed, "failed to connect to downstream service")
}

// RequestError 创建请求错误
func RequestError(err error) error {
	return WrapError(err, ErrorCodeRequestFailed, "request to downstream service failed")
}

// InvalidResponseError 创建无效响应错误
func InvalidResponseError(err error) error {
	return WrapError(err, ErrorCodeInvalidResponse, "invalid response from downstream service")
}

// TimeoutError 创建超时错误
func TimeoutError(err error) error {
	return WrapError(err, ErrorCodeTimeout, "request to downstream service timed out")
}

// IDMismatchError 创建ID不匹配错误
func IDMismatchError(expected, actual interface{}) error {
	return NewError(ErrorCodeIDMismatch,
		fmt.Sprintf("response ID mismatch: expected %v, got %v", expected, actual), nil)
}

// BatchSizeMismatchError 创建批量大小不匹配错误
func BatchSizeMismatchError(expected, actual int) error {
	return NewError(ErrorCodeBatchSizeMismatch,
		fmt.Sprintf("batch response size mismatch: expected %d, got %d", expected, actual), nil)
}