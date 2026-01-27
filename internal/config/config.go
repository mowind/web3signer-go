package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/mowind/web3signer-go/internal/utils"
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

	// 认证配置
	Auth AuthConfig `mapstructure:"auth"`
}

// HTTPConfig 定义 HTTP 服务器配置
type HTTPConfig struct {
	Host             string   `mapstructure:"host"`
	Port             int      `mapstructure:"port"`
	TLSCertFile      string   `mapstructure:"tls-cert-file"`
	TLSKeyFile       string   `mapstructure:"tls-key-file"`
	TLSAutoRedirect  bool     `mapstructure:"tls-auto-redirect"`
	MaxRequestSizeMB int64    `mapstructure:"max-request-size-mb"` // 最大请求体大小（MB），用于防止DoS攻击
	AllowedOrigins   []string `mapstructure:"allowed-origins"`     // CORS 允许的源列表，支持 "*" 允许所有源
}

// Validate 验证 HTTP 配置
func (c *HTTPConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("http-host is required")
	}
	if c.Port <= 0 || c.Port > MaxPort {
		return fmt.Errorf("http-port must be between 1 and %d", MaxPort)
	}
	if c.TLSCertFile != "" && c.TLSKeyFile == "" {
		return fmt.Errorf("tls-key-file is required when tls-cert-file is set")
	}
	if c.TLSKeyFile != "" && c.TLSCertFile == "" {
		return fmt.Errorf("tls-cert-file is required when tls-key-file is set")
	}
	if c.TLSCertFile != "" {
		if _, err := os.Stat(c.TLSCertFile); os.IsNotExist(err) {
			return fmt.Errorf("tls-cert-file does not exist: %s", c.TLSCertFile)
		}
	}
	if c.TLSKeyFile != "" {
		if _, err := os.Stat(c.TLSKeyFile); os.IsNotExist(err) {
			return fmt.Errorf("tls-key-file does not exist: %s", c.TLSKeyFile)
		}
	}
	if c.MaxRequestSizeMB <= 0 {
		c.MaxRequestSizeMB = 10
	}

	// 设置安全的默认CORS允许源
	if len(c.AllowedOrigins) == 0 {
		c.AllowedOrigins = []string{"http://localhost:*", "http://127.0.0.1:*"}
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
	// 验证地址格式
	if !utils.IsValidEthAddress(c.Address) {
		return fmt.Errorf("kms-address has invalid Ethereum address format: '%s'", c.Address)
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
	if c.HTTPPort > 0 {
		u, err := url.Parse(baseURL)
		if err == nil && u.Port() == "" {
			baseURL = strings.TrimSuffix(baseURL, "/")
			baseURL = fmt.Sprintf("%s:%d", baseURL, c.HTTPPort)
		}
	}
	return baseURL + c.HTTPPath
}

func hasPort(urlStr string) bool {
	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return u.Port() != ""
}

// LogConfig 定义日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`  // 日志级别
	Format string `mapstructure:"format"` // 日志格式 (json/text)
}

// Validate 验证日志配置
func (c *LogConfig) Validate() error {
	// 验证级别
	if !validLogLevels[strings.ToLower(c.Level)] {
		return fmt.Errorf("log-level must be one of: debug, info, warn, error, fatal, got: %s", c.Level)
	}

	// 验证格式
	if c.Format == "" {
		c.Format = DefaultLogFormat // 默认 text
	}
	if !validLogFormats[strings.ToLower(c.Format)] {
		return fmt.Errorf("log-format must be one of: json, text, got: %s", c.Format)
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

// AuthConfig 定义认证配置
type AuthConfig struct {
	Enabled   bool     `mapstructure:"enabled"`   // 是否启用认证
	Secret    string   `mapstructure:"secret"`    // 认证密钥（用于 JWT 或 API Key）
	Whitelist []string `mapstructure:"whitelist"` // 白名单路径（不需要认证的路径）
}

// Validate 验证认证配置
func (c *AuthConfig) Validate() error {
	if c.Enabled {
		if c.Secret == "" {
			return fmt.Errorf("auth-secret is required when auth is enabled")
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
			"Log: {Level: %s, Format: %s}, "+
			"Auth: {Enabled: %v, Secret: [REDACTED], Whitelist: %v}",
		c.HTTP.Host, c.HTTP.Port,
		c.KMS.Endpoint, c.KMS.KeyID,
		c.Downstream.HTTPHost, c.Downstream.HTTPPort, c.Downstream.HTTPPath,
		c.Log.Level, c.Log.Format,
		c.Auth.Enabled, c.Auth.Whitelist,
	)
}
