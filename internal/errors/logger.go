package errors

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger 扩展的日志器接口
type Logger interface {
	// 标准日志方法
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Panic(args ...interface{})

	// 带字段的日志方法
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})

	// 结构化日志方法
	Debugw(msg string, keysAndValues ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})
	Panicw(msg string, keysAndValues ...interface{})

	// 上下文相关方法
	WithContext(ctx context.Context) Logger
	WithField(key string, value interface{}) Logger
	WithFields(fields Fields) Logger
	WithError(err error) Logger
	WithRequestID(requestID string) Logger
	WithOperation(operation string) Logger

	// 请求日志方法
	LogRequest(method, path string, params interface{})
	LogResponse(method, path string, duration time.Duration, err error)
	LogOperation(operation string, startTime time.Time, err error)

	// 错误日志方法
	LogError(err error, context ...interface{})
	LogAppError(appErr *AppError, context ...interface{})

	// 配置方法
	SetLevel(level string) error
	SetFormatter(format string) error
	SetOutput(output string) error

	// 获取底层 logrus 实例
	GetUnderlying() *logrus.Logger
}

// Fields 日志字段
type Fields map[string]interface{}

// StructuredLogger 结构化日志器
type StructuredLogger struct {
	logger    *logrus.Logger
	requestID string
	operation string
	fields    Fields
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level        string `json:"level" yaml:"level"`
	Format       string `json:"format" yaml:"format"`
	Output       string `json:"output" yaml:"output"`
	EnableCaller bool   `json:"enable_caller" yaml:"enable_caller"`
	EnableTrace  bool   `json:"enable_trace" yaml:"enable_trace"`
}

// DefaultLoggerConfig 默认日志配置
func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Level:        "info",
		Format:       "json",
		Output:       "stdout",
		EnableCaller: true,
		EnableTrace:  false,
	}
}

// NewLogger 创建新的结构化日志器
func NewLogger(config *LoggerConfig) (Logger, error) {
	if config == nil {
		config = DefaultLoggerConfig()
	}

	logger := logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", config.Level, err)
	}
	logger.SetLevel(level)

	// 设置格式化器
	formatter, err := createFormatter(config.Format, config.EnableCaller, config.EnableTrace)
	if err != nil {
		return nil, fmt.Errorf("failed to create formatter: %w", err)
	}
	logger.SetFormatter(formatter)

	// 设置输出
	output, err := createOutput(config.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to create output: %w", err)
	}
	logger.SetOutput(output)

	return &StructuredLogger{
		logger: logger,
		fields: make(Fields),
	}, nil
}

// createFormatter 创建格式化器
func createFormatter(format string, enableCaller, enableTrace bool) (logrus.Formatter, error) {
	switch strings.ToLower(format) {
	case "json":
		return &logrus.JSONFormatter{
			TimestampFormat:  time.RFC3339Nano,
			DisableTimestamp: false,
			CallerPrettyfier: func(f *runtime.Frame) (string, string) {
				if !enableCaller {
					return "", ""
				}
				// 简化调用者信息
				filename := f.File
				if idx := strings.LastIndex(filename, "/"); idx >= 0 {
					filename = filename[idx+1:]
				}
				return fmt.Sprintf("%s:%d", filename, f.Line), f.Function
			},
		}, nil
	case "text":
		return &logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339Nano,
			DisableColors:   false,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported log format: %s", format)
	}
}

// createOutput 创建输出
func createOutput(output string) (*os.File, error) {
	switch strings.ToLower(output) {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		// 尝试作为文件路径
		// #nosec G304 - 日志文件路径来自配置，不是用户输入
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", output, err)
		}
		return file, nil
	}
}

// Standard logging methods 标准日志方法
func (l *StructuredLogger) Debug(args ...interface{}) {
	l.logger.Debug(args...)
}

func (l *StructuredLogger) Info(args ...interface{}) {
	l.logger.Info(args...)
}

func (l *StructuredLogger) Warn(args ...interface{}) {
	l.logger.Warn(args...)
}

func (l *StructuredLogger) Error(args ...interface{}) {
	l.logger.Error(args...)
}

func (l *StructuredLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

func (l *StructuredLogger) Panic(args ...interface{}) {
	l.logger.Panic(args...)
}

// Formatted logging methods 格式化日志方法
func (l *StructuredLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *StructuredLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *StructuredLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *StructuredLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *StructuredLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}

func (l *StructuredLogger) Panicf(format string, args ...interface{}) {
	l.logger.Panicf(format, args...)
}

// Structured logging methods 结构化日志方法
func (l *StructuredLogger) logWithFields(level logrus.Level, msg string, keysAndValues ...interface{}) {
	fields := make(logrus.Fields)

	// 添加当前字段
	for k, v := range l.fields {
		fields[k] = v
	}

	// 添加请求ID和操作信息
	if l.requestID != "" {
		fields["request_id"] = l.requestID
	}
	if l.operation != "" {
		fields["operation"] = l.operation
	}

	// 解析键值对
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		fields[key] = keysAndValues[i+1]
	}

	// 记录日志
	l.logger.WithFields(fields).Log(level, msg)
}

func (l *StructuredLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.logWithFields(logrus.DebugLevel, msg, keysAndValues...)
}

func (l *StructuredLogger) Infow(msg string, keysAndValues ...interface{}) {
	l.logWithFields(logrus.InfoLevel, msg, keysAndValues...)
}

func (l *StructuredLogger) Warnw(msg string, keysAndValues ...interface{}) {
	l.logWithFields(logrus.WarnLevel, msg, keysAndValues...)
}

func (l *StructuredLogger) Errorw(msg string, keysAndValues ...interface{}) {
	l.logWithFields(logrus.ErrorLevel, msg, keysAndValues...)
}

func (l *StructuredLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.logWithFields(logrus.FatalLevel, msg, keysAndValues...)
}

func (l *StructuredLogger) Panicw(msg string, keysAndValues ...interface{}) {
	l.logWithFields(logrus.PanicLevel, msg, keysAndValues...)
}

// Context-related methods 上下文相关方法
func (l *StructuredLogger) WithContext(ctx context.Context) Logger {
	newLogger := l.clone()

	// 从上下文中获取请求ID
	if requestID := GetRequestID(ctx); requestID != "" {
		newLogger.requestID = requestID
	}

	return newLogger
}

func (l *StructuredLogger) WithField(key string, value interface{}) Logger {
	newLogger := l.clone()
	newLogger.fields[key] = value
	return newLogger
}

func (l *StructuredLogger) WithFields(fields Fields) Logger {
	newLogger := l.clone()
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

func (l *StructuredLogger) WithError(err error) Logger {
	newLogger := l.clone()
	if appErr, ok := err.(*AppError); ok {
		newLogger.fields["error_type"] = string(appErr.Type)
		newLogger.fields["error_code"] = appErr.Code
		if appErr.Details != "" {
			newLogger.fields["error_details"] = appErr.Details
		}
	} else {
		newLogger.fields["error"] = err.Error()
	}
	return newLogger
}

func (l *StructuredLogger) WithRequestID(requestID string) Logger {
	newLogger := l.clone()
	newLogger.requestID = requestID
	return newLogger
}

func (l *StructuredLogger) WithOperation(operation string) Logger {
	newLogger := l.clone()
	newLogger.operation = operation
	return newLogger
}

// Request logging methods 请求日志方法
func (l *StructuredLogger) LogRequest(method, path string, params interface{}) {
	l.Infow("HTTP request received",
		"method", method,
		"path", path,
		"params", params,
		"timestamp", time.Now().UnixNano(),
	)
}

func (l *StructuredLogger) LogResponse(method, path string, duration time.Duration, err error) {
	fields := []interface{}{
		"method", method,
		"path", path,
		"duration_ms", duration.Milliseconds(),
		"timestamp", time.Now().UnixNano(),
	}

	if err != nil {
		fields = append(fields, "error", err.Error())
		l.Errorw("HTTP request completed with error", fields...)
	} else {
		l.Infow("HTTP request completed successfully", fields...)
	}
}

func (l *StructuredLogger) LogOperation(operation string, startTime time.Time, err error) {
	duration := time.Since(startTime)

	if err != nil {
		l.Errorw("Operation failed",
			"operation", operation,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)
	} else {
		l.Infow("Operation completed successfully",
			"operation", operation,
			"duration_ms", duration.Milliseconds(),
		)
	}
}

// Error logging methods 错误日志方法
func (l *StructuredLogger) LogError(err error, context ...interface{}) {
	if appErr, ok := err.(*AppError); ok {
		l.LogAppError(appErr, context...)
		return
	}

	fields := []interface{}{"error", err.Error()}
	for i := 0; i < len(context)-1; i += 2 {
		if key, ok := context[i].(string); ok && i+1 < len(context) {
			fields = append(fields, key, context[i+1])
		}
	}

	l.Errorw("Application error occurred", fields...)
}

func (l *StructuredLogger) LogAppError(appErr *AppError, context ...interface{}) {
	fields := []interface{}{
		"error_type", string(appErr.Type),
		"error_code", appErr.Code,
		"error_message", appErr.Message,
	}

	if appErr.Details != "" {
		fields = append(fields, "error_details", appErr.Details)
	}

	if appErr.OriginalErr != nil {
		fields = append(fields, "original_error", appErr.OriginalErr.Error())
	}

	// 添加上下文信息
	for k, v := range appErr.Context {
		fields = append(fields, fmt.Sprintf("context_%s", k), v)
	}

	// 添加额外的上下文参数
	for i := 0; i < len(context)-1; i += 2 {
		if key, ok := context[i].(string); ok && i+1 < len(context) {
			fields = append(fields, key, context[i+1])
		}
	}

	l.Errorw("Application error with context", fields...)
}

// Configuration methods 配置方法
func (l *StructuredLogger) SetLevel(level string) error {
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid log level %s: %w", level, err)
	}
	l.logger.SetLevel(logLevel)
	return nil
}

func (l *StructuredLogger) SetFormatter(format string) error {
	formatter, err := createFormatter(format, true, false)
	if err != nil {
		return fmt.Errorf("failed to create formatter: %w", err)
	}
	l.logger.SetFormatter(formatter)
	return nil
}

func (l *StructuredLogger) SetOutput(output string) error {
	outputWriter, err := createOutput(output)
	if err != nil {
		return fmt.Errorf("failed to create output: %w", err)
	}
	l.logger.SetOutput(outputWriter)
	return nil
}

// GetUnderlying 获取底层 logrus 实例
func (l *StructuredLogger) GetUnderlying() *logrus.Logger {
	return l.logger
}

// clone 克隆日志器
func (l *StructuredLogger) clone() *StructuredLogger {
	newFields := make(Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}

	return &StructuredLogger{
		logger:    l.logger,
		requestID: l.requestID,
		operation: l.operation,
		fields:    newFields,
	}
}
