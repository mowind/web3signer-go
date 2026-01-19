package errors

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewLogger(t *testing.T) {
	config := &LoggerConfig{
		Level:        "info",
		Format:       "json",
		Output:       "stdout",
		EnableCaller: false,
		EnableTrace:  false,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}
}

func TestNewLogger_InvalidLevel(t *testing.T) {
	config := &LoggerConfig{
		Level:  "invalid_level",
		Format: "json",
		Output: "stdout",
	}

	_, err := NewLogger(config)
	if err == nil {
		t.Error("Expected error for invalid log level")
	}
}

func TestNewLogger_InvalidFormat(t *testing.T) {
	config := &LoggerConfig{
		Level:  "info",
		Format: "invalid_format",
		Output: "stdout",
	}

	_, err := NewLogger(config)
	if err == nil {
		t.Error("Expected error for invalid log format")
	}
}

func TestLogger_StructuredLogging(t *testing.T) {
	// 使用 buffer 捕获日志输出
	var buf bytes.Buffer

	logger := &StructuredLogger{
		logger: createTestLogger(&buf),
		fields: make(Fields),
	}

	// 测试结构化日志
	logger.Infow("Test message",
		"key1", "value1",
		"key2", 123,
		"key3", true,
	)

	output := buf.String()

	// 验证输出包含字段
	if !strings.Contains(output, `"key1":"value1"`) {
		t.Error("Expected output to contain key1=value1")
	}
	if !strings.Contains(output, `"key2":123`) {
		t.Error("Expected output to contain key2=123")
	}
	if !strings.Contains(output, `"key3":true`) {
		t.Error("Expected output to contain key3=true")
	}
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer

	baseLogger := &StructuredLogger{
		logger: createTestLogger(&buf),
		fields: make(Fields),
	}

	// 添加字段
	logger := baseLogger.WithFields(Fields{
		"service": "web3signer",
		"version": "1.0.0",
	})

	logger.Infow("Test message")

	output := buf.String()

	if !strings.Contains(output, `"service":"web3signer"`) {
		t.Error("Expected output to contain service field")
	}
	if !strings.Contains(output, `"version":"1.0.0"`) {
		t.Error("Expected output to contain version field")
	}
}

func TestLogger_WithError(t *testing.T) {
	var buf bytes.Buffer

	logger := &StructuredLogger{
		logger: createTestLogger(&buf),
		fields: make(Fields),
	}

	// 测试普通错误
	normalErr := fmt.Errorf("normal error")
	logger.WithError(normalErr).Errorw("Error occurred")

	output := buf.String()
	if !strings.Contains(output, `"error":"normal error"`) {
		t.Error("Expected output to contain normal error")
	}

	// 清空 buffer
	buf.Reset()

	// 测试 AppError
	appErr := New(ErrorTypeKMSAuth, 1001, "KMS authentication failed").
		WithDetails("Invalid credentials").
		WithContext("key_id", "test-key")

	logger.WithError(appErr).Errorw("App error occurred")

	output = buf.String()
	if !strings.Contains(output, `"error_type":"KMS_AUTH_ERROR"`) {
		t.Error("Expected output to contain error_type")
	}
	if !strings.Contains(output, `"error_code":1001`) {
		t.Error("Expected output to contain error_code")
	}
	if !strings.Contains(output, `"error_details":"Invalid credentials"`) {
		t.Error("Expected output to contain error_details")
	}
}

func TestLogger_LogRequest(t *testing.T) {
	var buf bytes.Buffer

	logger := &StructuredLogger{
		logger: createTestLogger(&buf),
		fields: make(Fields),
	}

	logger.LogRequest("POST", "/api/v1/sign", map[string]interface{}{
		"key_id": "test-key",
		"data":   "0x1234",
	})

	output := buf.String()

	if !strings.Contains(output, `"msg":"HTTP request received"`) {
		t.Error("Expected log message")
	}
	if !strings.Contains(output, `"method":"POST"`) {
		t.Error("Expected method field")
	}
	if !strings.Contains(output, `"path":"/api/v1/sign"`) {
		t.Error("Expected path field")
	}
}

func TestLogger_LogResponse(t *testing.T) {
	var buf bytes.Buffer

	logger := &StructuredLogger{
		logger: createTestLogger(&buf),
		fields: make(Fields),
	}

	// 测试成功响应
	logger.LogResponse("POST", "/api/v1/sign", 150*time.Millisecond, nil)

	output := buf.String()
	if !strings.Contains(output, `"msg":"HTTP request completed successfully"`) {
		t.Error("Expected success message")
	}
	if !strings.Contains(output, `"duration_ms":150`) {
		t.Error("Expected duration field")
	}

	// 清空 buffer
	buf.Reset()

	// 测试错误响应
	testErr := fmt.Errorf("test error")
	logger.LogResponse("POST", "/api/v1/sign", 200*time.Millisecond, testErr)

	output = buf.String()
	if !strings.Contains(output, `"msg":"HTTP request completed with error"`) {
		t.Error("Expected error message")
	}
	if !strings.Contains(output, `"error":"test error"`) {
		t.Error("Expected error field")
	}
}

func TestLogger_LogOperation(t *testing.T) {
	var buf bytes.Buffer

	logger := &StructuredLogger{
		logger: createTestLogger(&buf),
		fields: make(Fields),
	}

	// 测试成功操作
	startTime := time.Now()
	time.Sleep(10 * time.Millisecond) // 模拟操作耗时
	logger.LogOperation("KMS_SIGN", startTime, nil)

	output := buf.String()
	if !strings.Contains(output, `"msg":"Operation completed successfully"`) {
		t.Error("Expected success message")
	}
	if !strings.Contains(output, `"operation":"KMS_SIGN"`) {
		t.Error("Expected operation field")
	}

	// 清空 buffer
	buf.Reset()

	// 测试失败操作
	startTime = time.Now()
	time.Sleep(5 * time.Millisecond)
	testErr := fmt.Errorf("operation failed")
	logger.LogOperation("KMS_SIGN", startTime, testErr)

	output = buf.String()
	if !strings.Contains(output, `"msg":"Operation failed"`) {
		t.Error("Expected failure message")
	}
	if !strings.Contains(output, `"error":"operation failed"`) {
		t.Error("Expected error field")
	}
}

func TestLogger_LogAppError(t *testing.T) {
	var buf bytes.Buffer

	logger := &StructuredLogger{
		logger: createTestLogger(&buf),
		fields: make(Fields),
	}

	appErr := New(ErrorTypeKMSAuth, 1001, "KMS authentication failed").
		WithDetails("Invalid API key").
		WithContext("key_id", "test-key").
		WithContext("endpoint", "https://kms.example.com")

	logger.LogAppError(appErr, "additional_context", "test value")

	output := buf.String()

	// 验证错误基本信息
	if !strings.Contains(output, `"error_type":"KMS_AUTH_ERROR"`) {
		t.Error("Expected error_type field")
	}
	if !strings.Contains(output, `"error_code":1001`) {
		t.Error("Expected error_code field")
	}
	if !strings.Contains(output, `"error_message":"KMS authentication failed"`) {
		t.Error("Expected error_message field")
	}
	if !strings.Contains(output, `"error_details":"Invalid API key"`) {
		t.Error("Expected error_details field")
	}

	// 验证上下文信息
	if !strings.Contains(output, `"context_key_id":"test-key"`) {
		t.Error("Expected context_key_id field")
	}
	if !strings.Contains(output, `"context_endpoint":"https://kms.example.com"`) {
		t.Error("Expected context_endpoint field")
	}

	// 验证额外上下文
	if !strings.Contains(output, `"additional_context":"test value"`) {
		t.Error("Expected additional_context field")
	}
}

func TestLogger_WithContext(t *testing.T) {
	var buf bytes.Buffer

	baseLogger := &StructuredLogger{
		logger: createTestLogger(&buf),
		fields: make(Fields),
	}

	// 创建带请求ID的context
	ctx := NewContextWithRequestID(context.Background(), "test-request-id")
	logger := baseLogger.WithContext(ctx)

	logger.Infow("Test message with context")

	output := buf.String()
	if !strings.Contains(output, `"request_id":"test-request-id"`) {
		t.Error("Expected request_id field from context")
	}
}

func TestLogger_Configuration(t *testing.T) {
	var buf bytes.Buffer

	logger := &StructuredLogger{
		logger: createTestLogger(&buf),
		fields: make(Fields),
	}

	// 测试设置级别
	err := logger.SetLevel("debug")
	if err != nil {
		t.Errorf("Failed to set level: %v", err)
	}

	// 测试设置格式化器
	err = logger.SetFormatter("text")
	if err != nil {
		t.Errorf("Failed to set formatter: %v", err)
	}

	// 测试设置输出
	err = logger.SetOutput("stdout")
	if err != nil {
		t.Errorf("Failed to set output: %v", err)
	}
}

// Helper function to create test logger
func createTestLogger(output *bytes.Buffer) *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(output)
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})
	return logger
}

func TestContext_RequestID(t *testing.T) {
	// 测试生成请求ID
	requestID := GenerateRequestID()
	if requestID == "" {
		t.Error("Expected non-empty request ID")
	}

	// 测试格式（应该是UUID格式）
	if len(requestID) != 36 { // UUID格式长度
		t.Errorf("Expected request ID length 36, got %d", len(requestID))
	}

	// 测试唯一性
	requestID2 := GenerateRequestID()
	if requestID == requestID2 {
		t.Error("Expected different request IDs")
	}

	// 测试context操作
	ctx := NewContextWithRequestID(context.Background(), requestID)
	retrievedID := GetRequestID(ctx)

	if retrievedID != requestID {
		t.Errorf("Expected request ID %s, got %s", requestID, retrievedID)
	}

	// 测试空context
	nilCtx := NewContextWithRequestID(context.Background(), "")
	nilID := GetRequestID(nilCtx)
	if nilID == "" {
		t.Error("Expected generated request ID for empty input")
	}
}

func TestContext_Operation(t *testing.T) {
	ctx := NewContextWithOperation(context.Background(), "KMS_SIGN")
	operation := GetOperation(ctx)

	if operation != "KMS_SIGN" {
		t.Errorf("Expected operation 'KMS_SIGN', got '%s'", operation)
	}

	// 测试空context
	emptyCtx := context.Background()
	emptyOperation := GetOperation(emptyCtx)
	if emptyOperation != "" {
		t.Errorf("Expected empty operation, got '%s'", emptyOperation)
	}

	// 测试nil context
	nilOperation := GetOperation(context.Background())
	if nilOperation != "" {
		t.Errorf("Expected empty operation for nil context, got '%s'", nilOperation)
	}
}

func TestDefaultLoggerConfig(t *testing.T) {
	config := DefaultLoggerConfig()

	if config.Level != "info" {
		t.Errorf("Expected default level 'info', got '%s'", config.Level)
	}
	if config.Format != "json" {
		t.Errorf("Expected default format 'json', got '%s'", config.Format)
	}
	if config.Output != "stdout" {
		t.Errorf("Expected default output 'stdout', got '%s'", config.Output)
	}
	if !config.EnableCaller {
		t.Error("Expected EnableCaller to be true")
	}
	if config.EnableTrace {
		t.Error("Expected EnableTrace to be false")
	}
}
