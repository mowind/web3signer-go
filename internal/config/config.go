package config

import (
	"fmt"
	"strings"
)

// Config 表示应用程序的完整配置
type Config struct {
	// HTTP 服务器配置
	HTTP HTTPConfig `mapstructure:"http"`

	// MPC-KMS 配置
	KMS KMSConfig `mapstructure:"kms"`

	// 下游服务配置
	Downstream DownstreamConfig `mapstructure:"downstream"`

	// 日志配置
	Log LogConfig `mapstructure:"log"`
}

// HTTPConfig 定义 HTTP 服务器配置
type HTTPConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// Validate 验证 HTTP 配置
func (c *HTTPConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("http-host is required")
	}
	if c.Port <= 0 || c.Port > MaxPort {
		return fmt.Errorf("http-port must be between 1 and %d", MaxPort)
	}
	return nil
}

// KMSConfig 定义 MPC-KMS 配置
type KMSConfig struct {
	Endpoint    string `mapstructure:"endpoint"`
	AccessKeyID string `mapstructure:"access-key-id"`
	SecretKey   string `mapstructure:"secret-key"`
	KeyID       string `mapstructure:"key-id"`
	Address     string `mapstructure:"address"` // KMS管理的以太坊地址
}

// Validate 验证 KMS 配置
func (c *KMSConfig) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("kms-endpoint is required")
	}
	if c.AccessKeyID == "" {
		return fmt.Errorf("kms-access-key-id is required")
	}
	if c.SecretKey == "" {
		return fmt.Errorf("kms-secret-key is required")
	}
	if c.KeyID == "" {
		return fmt.Errorf("kms-key-id is required")
	}
	if c.Address == "" {
		return fmt.Errorf("kms-address is required")
	}
	return nil
}

// DownstreamConfig 定义下游服务配置
type DownstreamConfig struct {
	HTTPHost string `mapstructure:"http-host"` // 完整的host，如 http://127.0.0.1 或 https://api.example.com
	HTTPPort int    `mapstructure:"http-port"` // 端口，如果host中已包含端口或不需要端口，可以为0
	HTTPPath string `mapstructure:"http-path"` // 路径，如 /api/v1/jsonrpc
}

// Validate 验证下游服务配置
func (c *DownstreamConfig) Validate() error {
	if c.HTTPHost == "" {
		return fmt.Errorf("downstream-http-host is required")
	}
	// 验证host格式
	if !strings.HasPrefix(c.HTTPHost, "http://") && !strings.HasPrefix(c.HTTPHost, "https://") {
		return fmt.Errorf("downstream-http-host must start with http:// or https://")
	}
	if c.HTTPPort < 0 || c.HTTPPort > MaxPort {
		return fmt.Errorf("downstream-http-port must be between 0 and %d", MaxPort)
	}
	if c.HTTPPath == "" {
		return fmt.Errorf("downstream-http-path is required")
	}
	// 确保路径以/开头
	if !strings.HasPrefix(c.HTTPPath, "/") {
		c.HTTPPath = "/" + c.HTTPPath
	}
	return nil
}

// BuildURL 构建完整的下游服务URL
func (c *DownstreamConfig) BuildURL() string {
	baseURL := c.HTTPHost
	// 如果指定了端口且端口大于0，添加到host中
	if c.HTTPPort > 0 {
		// 检查host是否已经包含端口
		if !hasPort(baseURL) {
			// 移除可能的尾部斜杠
			baseURL = strings.TrimSuffix(baseURL, "/")
			baseURL = fmt.Sprintf("%s:%d", baseURL, c.HTTPPort)
		}
	}
	// 添加路径
	return baseURL + c.HTTPPath
}

// hasPort 检查URL是否已经包含端口
func hasPort(url string) bool {
	// 简单的端口检查逻辑
	// 移除协议部分
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")

	// 查找最后一个冒号（可能是端口分隔符）
	lastColon := strings.LastIndex(url, ":")
	if lastColon == -1 {
		return false
	}

	// 检查冒号后的部分是否为数字
	portPart := url[lastColon+1:]
	// 检查是否包含路径分隔符
	if strings.Contains(portPart, "/") {
		return false
	}

	// 尝试解析为端口号
	for _, ch := range portPart {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// LogConfig 定义日志配置
type LogConfig struct {
	Level string `mapstructure:"level"`
}

// Validate 验证日志配置
func (c *LogConfig) Validate() error {
	if !validLogLevels[strings.ToLower(c.Level)] {
		return fmt.Errorf("log-level must be one of: debug, info, warn, error, fatal, got: %s", c.Level)
	}
	return nil
}

// Validate 验证配置是否有效
func (c *Config) Validate() error {
	// 设置默认值
	if c.Log.Level == "" {
		c.Log.Level = DefaultLogLevel
	}

	// 验证所有子配置
	validators := []Validator{&c.HTTP, &c.KMS, &c.Downstream, &c.Log}
	for _, v := range validators {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// String 返回配置的安全摘要（不包含敏感信息）
func (c *Config) String() string {
	return fmt.Sprintf(
		"HTTP: {Host: %s, Port: %d}, "+
			"KMS: {Endpoint: %s, KeyID: %s, AccessKeyID: [REDACTED], SecretKey: [REDACTED]}, "+
			"Downstream: {Host: %s, Port: %d, Path: %s}, "+
			"Log: {Level: %s}",
		c.HTTP.Host, c.HTTP.Port,
		c.KMS.Endpoint, c.KMS.KeyID,
		c.Downstream.HTTPHost, c.Downstream.HTTPPort, c.Downstream.HTTPPath,
		c.Log.Level,
	)
}
