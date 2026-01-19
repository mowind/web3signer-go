package errors

import (
	"fmt"
	"testing"

	"github.com/mowind/web3signer-go/internal/downstream"
	"github.com/mowind/web3signer-go/internal/jsonrpc"
)

func TestNew(t *testing.T) {
	appErr := New(ErrorTypeInternal, jsonrpc.CodeInternalError, "Test error")

	if appErr.Type != ErrorTypeInternal {
		t.Errorf("Expected type %s, got %s", ErrorTypeInternal, appErr.Type)
	}

	if appErr.Code != jsonrpc.CodeInternalError {
		t.Errorf("Expected code %d, got %d", jsonrpc.CodeInternalError, appErr.Code)
	}

	if appErr.Message != "Test error" {
		t.Errorf("Expected message 'Test error', got '%s'", appErr.Message)
	}

	if appErr.Context == nil {
		t.Error("Expected Context to be initialized")
	}
}

func TestNewf(t *testing.T) {
	appErr := Newf(ErrorTypeValidation, jsonrpc.CodeInvalidParams, "Invalid param: %s", "test_param")

	if appErr.Message != "Invalid param: test_param" {
		t.Errorf("Expected message 'Invalid param: test_param', got '%s'", appErr.Message)
	}
}

func TestWrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	appErr := Wrap(originalErr, ErrorTypeKMSAuth, jsonrpc.CodeServerErrorStart, "KMS authentication failed")

	if appErr.OriginalErr != originalErr {
		t.Error("Expected OriginalErr to be set")
	}

	if appErr.Details != "original error" {
		t.Errorf("Expected details 'original error', got '%s'", appErr.Details)
	}

	if appErr.Type != ErrorTypeKMSAuth {
		t.Errorf("Expected type %s, got %s", ErrorTypeKMSAuth, appErr.Type)
	}
}

func TestWrap_NilError(t *testing.T) {
	appErr := Wrap(nil, ErrorTypeInternal, jsonrpc.CodeInternalError, "Test")
	if appErr != nil {
		t.Error("Expected nil for nil error input")
	}
}

func TestWithContext(t *testing.T) {
	appErr := New(ErrorTypeInternal, jsonrpc.CodeInternalError, "Test error")
	_ = appErr.WithContext("key1", "value1").WithContext("key2", 123)

	if appErr.Context["key1"] != "value1" {
		t.Errorf("Expected context key1='value1', got '%v'", appErr.Context["key1"])
	}

	if appErr.Context["key2"] != 123 {
		t.Errorf("Expected context key2=123, got '%v'", appErr.Context["key2"])
	}
}

func TestWithDetails(t *testing.T) {
	appErr := New(ErrorTypeInternal, jsonrpc.CodeInternalError, "Test error")
	_ = appErr.WithDetails("Additional details")

	if appErr.Details != "Additional details" {
		t.Errorf("Expected details 'Additional details', got '%s'", appErr.Details)
	}
}

func TestAppError_Error(t *testing.T) {
	testCases := []struct {
		name     string
		appErr   *AppError
		expected string
	}{
		{
			name:     "Simple error",
			appErr:   New(ErrorTypeInternal, jsonrpc.CodeInternalError, "Internal error"),
			expected: "Internal error [INTERNAL_ERROR:-32603]",
		},
		{
			name:     "Error with original error",
			appErr:   Wrap(fmt.Errorf("database connection failed"), ErrorTypeConnection, jsonrpc.CodeServerErrorStart, "Connection failed"),
			expected: "Connection failed [CONNECTION_ERROR:-32000]: database connection failed (details: database connection failed)",
		},
		{
			name:     "Error with details",
			appErr:   Wrap(fmt.Errorf("timeout"), ErrorTypeTimeout, jsonrpc.CodeServerErrorStart+1, "Request timeout").WithDetails("Connection timeout after 30s"),
			expected: "Request timeout [TIMEOUT_ERROR:-31999]: timeout (details: Connection timeout after 30s)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.appErr.Error()
			if result != tc.expected {
				t.Errorf("Expected error '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	originalErr := fmt.Errorf("original error")
	appErr := Wrap(originalErr, ErrorTypeInternal, jsonrpc.CodeInternalError, "Wrapped error")

	unwrapped := appErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Expected unwrapped error to be original error")
	}
}

func TestAppError_Is(t *testing.T) {
	appErr1 := New(ErrorTypeInternal, jsonrpc.CodeInternalError, "Error 1")
	appErr2 := New(ErrorTypeKMSAuth, jsonrpc.CodeServerErrorStart, "Error 2")
	appErr3 := New(ErrorTypeInternal, jsonrpc.CodeInternalError, "Error 3")

	if !appErr1.Is(appErr3) {
		t.Error("Expected appErr1.Is(appErr3) to be true (same type)")
	}

	if appErr1.Is(appErr2) {
		t.Error("Expected appErr1.Is(appErr2) to be false (different types)")
	}

	// 测试非 AppError
	if appErr1.Is(fmt.Errorf("some error")) {
		t.Error("Expected Is() to return false for non-AppError")
	}
}

func TestAppError_ToJSONRPCError(t *testing.T) {
	testCases := []struct {
		name         string
		appErr       *AppError
		expectedCode int
	}{
		{
			name:         "Validation error",
			appErr:       New(ErrorTypeValidation, 1001, "Validation failed"),
			expectedCode: jsonrpc.CodeInvalidParams,
		},
		{
			name:         "Method not found",
			appErr:       New(ErrorTypeMethodNotFound, 1002, "Method not found"),
			expectedCode: jsonrpc.CodeMethodNotFound,
		},
		{
			name:         "Internal error",
			appErr:       New(ErrorTypeInternal, 1003, "Internal error"),
			expectedCode: jsonrpc.CodeInternalError,
		},
		{
			name:         "Connection error",
			appErr:       New(ErrorTypeConnection, 1004, "Connection failed"),
			expectedCode: jsonrpc.CodeServerErrorStart,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonErr := tc.appErr.ToJSONRPCError()
			if jsonErr.Code != tc.expectedCode {
				t.Errorf("Expected JSON-RPC code %d, got %d", tc.expectedCode, jsonErr.Code)
			}
			if jsonErr.Message != tc.appErr.Message {
				t.Errorf("Expected message '%s', got '%s'", tc.appErr.Message, jsonErr.Message)
			}
		})
	}
}

func TestCommonErrors(t *testing.T) {
	// 验证预定义的错误
	testCases := []struct {
		err     *AppError
		type_   ErrorType
		code    int
		message string
	}{
		{ErrInternal, ErrorTypeInternal, jsonrpc.CodeInternalError, "Internal server error"},
		{ErrValidation, ErrorTypeValidation, jsonrpc.CodeInvalidParams, "Validation failed"},
		{ErrConnection, ErrorTypeConnection, jsonrpc.CodeServerErrorStart, "Connection failed"},
		{ErrKMSSign, ErrorTypeKMSSign, jsonrpc.CodeServerErrorStart + 10, "KMS signing failed"},
		{ErrSign, ErrorTypeSign, jsonrpc.CodeServerErrorStart + 20, "Signing failed"},
		{ErrMethodNotFound, ErrorTypeMethodNotFound, jsonrpc.CodeMethodNotFound, "Method not found"},
	}

	for _, tc := range testCases {
		if tc.err.Type != tc.type_ {
			t.Errorf("Expected type %s, got %s", tc.type_, tc.err.Type)
		}
		if tc.err.Code != tc.code {
			t.Errorf("Expected code %d, got %d", tc.code, tc.err.Code)
		}
		if tc.err.Message != tc.message {
			t.Errorf("Expected message '%s', got '%s'", tc.message, tc.err.Message)
		}
	}
}

func TestConverter_FromJSONRPC(t *testing.T) {
	converter := NewConverter()

	testCases := []struct {
		name         string
		jsonErr      *jsonrpc.Error
		expectedType ErrorType
	}{
		{
			name:         "Parse error",
			jsonErr:      jsonrpc.ParseError,
			expectedType: ErrorTypeJSONRPC,
		},
		{
			name:         "Method not found",
			jsonErr:      jsonrpc.MethodNotFoundError,
			expectedType: ErrorTypeMethodNotFound,
		},
		{
			name:         "Invalid params",
			jsonErr:      jsonrpc.InvalidParamsError,
			expectedType: ErrorTypeInvalidParams,
		},
		{
			name: "Custom server error",
			jsonErr: &jsonrpc.Error{
				Code:    jsonrpc.CodeServerErrorStart - 5, // -32005, which is in server error range
				Message: "Custom server error",
			},
			expectedType: ErrorTypeInternal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appErr := converter.FromJSONRPC(tc.jsonErr)
			if appErr.Type != tc.expectedType {
				t.Errorf("Expected type %s, got %s", tc.expectedType, appErr.Type)
			}
			if appErr.Code != tc.jsonErr.Code {
				t.Errorf("Expected code %d, got %d", tc.jsonErr.Code, appErr.Code)
			}
		})
	}
}

func TestConverter_ToJSONRPC(t *testing.T) {
	converter := NewConverter()

	appErr := New(ErrorTypeKMSAuth, 1001, "KMS authentication failed").
		WithContext("key_id", "test-key").
		WithDetails("Invalid credentials")

	jsonErr := converter.ToJSONRPC(appErr)

	if jsonErr.Code != jsonrpc.CodeInternalError {
		t.Errorf("Expected JSON-RPC code %d, got %d", jsonrpc.CodeInternalError, jsonErr.Code)
	}

	if jsonErr.Message != appErr.Message {
		t.Errorf("Expected message '%s', got '%s'", appErr.Message, jsonErr.Message)
	}

	// 验证错误数据包含上下文信息
	errData, ok := jsonErr.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected error data to be a map")
	}

	if errData["type"] != string(ErrorTypeKMSAuth) {
		t.Errorf("Expected error data type %s, got %v", ErrorTypeKMSAuth, errData["type"])
	}

	if errData["key_id"] != "test-key" {
		t.Errorf("Expected error data key_id='test-key', got %v", errData["key_id"])
	}
}

func TestConverter_FromDownstream(t *testing.T) {
	converter := NewConverter()

	testCases := []struct {
		name          string
		downstreamErr error
		expectedType  ErrorType
		expectedCode  int
	}{
		{
			name:          "Connection failed",
			downstreamErr: downstream.ConnectionError(fmt.Errorf("connection refused")),
			expectedType:  ErrorTypeConnection,
			expectedCode:  jsonrpc.CodeServerErrorStart,
		},
		{
			name:          "Request failed",
			downstreamErr: downstream.RequestError(fmt.Errorf("HTTP 500")),
			expectedType:  ErrorTypeForward,
			expectedCode:  jsonrpc.CodeServerErrorStart + 1,
		},
		{
			name:          "Timeout error",
			downstreamErr: downstream.TimeoutError(fmt.Errorf("timeout after 30s")),
			expectedType:  ErrorTypeTimeout,
			expectedCode:  jsonrpc.CodeServerErrorStart + 3,
		},
		{
			name:          "Invalid response",
			downstreamErr: downstream.InvalidResponseError(fmt.Errorf("invalid JSON")),
			expectedType:  ErrorTypeDownstream,
			expectedCode:  jsonrpc.CodeServerErrorStart + 2,
		},
		{
			name:          "ID mismatch",
			downstreamErr: downstream.IDMismatchError("expected", "actual"),
			expectedType:  ErrorTypeDownstream,
			expectedCode:  jsonrpc.CodeServerErrorStart + 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			appErr := converter.FromDownstream(tc.downstreamErr)
			if appErr.Type != tc.expectedType {
				t.Errorf("Expected type %s, got %s", tc.expectedType, appErr.Type)
			}
			if appErr.Code != tc.expectedCode {
				t.Errorf("Expected code %d, got %d", tc.expectedCode, appErr.Code)
			}
		})
	}
}

func TestConvertError(t *testing.T) {
	testCases := []struct {
		name         string
		inputErr     error
		expectedType ErrorType
	}{
		{
			name:         "AppError",
			inputErr:     New(ErrorTypeInternal, 1001, "Internal error"),
			expectedType: ErrorTypeInternal,
		},
		{
			name:         "JSON-RPC error",
			inputErr:     jsonrpc.InvalidParamsError,
			expectedType: ErrorTypeInvalidParams,
		},
		{
			name:         "Downstream error",
			inputErr:     downstream.ConnectionError(fmt.Errorf("connection failed")),
			expectedType: ErrorTypeConnection,
		},
		{
			name:         "Plain error",
			inputErr:     fmt.Errorf("some error"),
			expectedType: ErrorTypeInternal,
		},
		{
			name:         "Nil error",
			inputErr:     nil,
			expectedType: ErrorType(""), // nil input should return nil
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ConvertError(tc.inputErr)
			if tc.inputErr == nil {
				if result != nil {
					t.Error("Expected nil for nil input")
				}
				return
			}
			if result.Type != tc.expectedType {
				t.Errorf("Expected type %s, got %s", tc.expectedType, result.Type)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Connection error - retryable",
			err:      New(ErrorTypeConnection, jsonrpc.CodeServerErrorStart, "Connection failed"),
			expected: true,
		},
		{
			name:     "Timeout error - retryable",
			err:      New(ErrorTypeTimeout, jsonrpc.CodeServerErrorStart+1, "Timeout"),
			expected: true,
		},
		{
			name:     "KMS unavailable - retryable",
			err:      New(ErrorTypeKMSUnavailable, jsonrpc.CodeServerErrorStart+12, "KMS unavailable"),
			expected: true,
		},
		{
			name:     "Validation error - not retryable",
			err:      New(ErrorTypeValidation, jsonrpc.CodeInvalidParams, "Validation failed"),
			expected: false,
		},
		{
			name:     "Internal error - not retryable",
			err:      New(ErrorTypeInternal, jsonrpc.CodeInternalError, "Internal error"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsRetryable(tc.err)
			if result != tc.expected {
				t.Errorf("Expected IsRetryable=%v for error type %s", tc.expected, tc.err.(*AppError).Type)
			}
		})
	}
}

func TestIsClientError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Validation error - client error",
			err:      New(ErrorTypeValidation, jsonrpc.CodeInvalidParams, "Validation failed"),
			expected: true,
		},
		{
			name:     "Invalid params - client error",
			err:      New(ErrorTypeInvalidParams, jsonrpc.CodeInvalidParams, "Invalid params"),
			expected: true,
		},
		{
			name:     "Method not found - client error",
			err:      New(ErrorTypeMethodNotFound, jsonrpc.CodeMethodNotFound, "Method not found"),
			expected: true,
		},
		{
			name:     "Address mismatch - client error",
			err:      New(ErrorTypeAddressMismatch, jsonrpc.CodeInvalidParams, "Address mismatch"),
			expected: true,
		},
		{
			name:     "Internal error - server error",
			err:      New(ErrorTypeInternal, jsonrpc.CodeInternalError, "Internal error"),
			expected: false,
		},
		{
			name:     "Connection error - server error",
			err:      New(ErrorTypeConnection, jsonrpc.CodeServerErrorStart, "Connection failed"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsClientError(tc.err)
			if result != tc.expected {
				t.Errorf("Expected IsClientError=%v for error type %s", tc.expected, tc.err.(*AppError).Type)
			}
		})
	}
}

func TestMustConvertError(t *testing.T) {
	// 测试正常转换
	err := fmt.Errorf("test error")
	result := MustConvertError(err)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.Type != ErrorTypeInternal {
		t.Errorf("Expected type %s, got %s", ErrorTypeInternal, result.Type)
	}

	// 测试 nil 输入应该 panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil input")
		}
	}()
	_ = MustConvertError(nil)
}
