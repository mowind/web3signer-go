package errors

import (
	"context"

	"github.com/google/uuid"
)

// context keys
type contextKey string

const (
	// RequestIDKey 请求ID的context key
	RequestIDKey contextKey = "request_id"
	// OperationKey 操作名称的context key
	OperationKey contextKey = "operation"
)

// NewContextWithRequestID 创建带有请求ID的context
func NewContextWithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		requestID = GenerateRequestID()
	}
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// NewContextWithOperation 创建带有操作名称的context
func NewContextWithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, OperationKey, operation)
}

// GetRequestID 从context获取请求ID
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetOperation 从context获取操作名称
func GetOperation(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if operation, ok := ctx.Value(OperationKey).(string); ok {
		return operation
	}
	return ""
}

// GenerateRequestID 生成新的请求ID
func GenerateRequestID() string {
	return uuid.New().String()
}
