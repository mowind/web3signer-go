package config

const (
	// MaxPort 最大端口号
	MaxPort = 65535

	// LogLevelDebug 调试日志级别
	LogLevelDebug = "debug"
	// LogLevelInfo 信息日志级别
	LogLevelInfo = "info"
	// LogLevelWarn 警告日志级别
	LogLevelWarn = "warn"
	// LogLevelError 错误日志级别
	LogLevelError = "error"
	// LogLevelFatal 致命日志级别
	LogLevelFatal = "fatal"

	// LogFormatJSON JSON 日志格式
	LogFormatJSON = "json"
	// LogFormatText 文本日志格式
	LogFormatText = "text"

	// DefaultHTTPHost 默认 HTTP 主机
	DefaultHTTPHost = "localhost"
	// DefaultHTTPPort 默认 HTTP 端口
	DefaultHTTPPort = 9000
	// DefaultMaxRequestSizeMB 默认最大请求大小（MB）
	DefaultMaxRequestSizeMB int64 = 10

	// DefaultDownstreamHost 默认下游服务主机（完整URL）
	DefaultDownstreamHost = "http://localhost"
	// DefaultDownstreamPort 默认下游服务端口
	DefaultDownstreamPort = 8545
	// DefaultDownstreamPath 默认下游服务路径
	DefaultDownstreamPath = "/"

	// DefaultLogLevel 默认日志级别
	DefaultLogLevel = LogLevelInfo
	// DefaultLogFormat 默认日志格式
	DefaultLogFormat = LogFormatText
)

// Validator 验证器接口
type Validator interface {
	Validate() error
}

// 有效的日志级别
var validLogLevels = map[string]bool{
	LogLevelDebug: true,
	LogLevelInfo:  true,
	LogLevelWarn:  true,
	LogLevelError: true,
	LogLevelFatal: true,
}

// 有效的日志格式
var validLogFormats = map[string]bool{
	LogFormatJSON: true,
	LogFormatText: true,
}
