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
	Endpoint      string `mapstructure:"endpoint"`
	AccessKeyID   string `mapstructure:"access-key-id"`
	SecretKey     string `mapstructure:"secret-key"`
	KeyID         string `mapstructure:"key-id"`
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
	return nil
}

// DownstreamConfig 定义下游服务配置
type DownstreamConfig struct {
	HTTPHost string `mapstructure:"http-host"`
	HTTPPort int    `mapstructure:"http-port"`
	HTTPPath string `mapstructure:"http-path"`
}

// Validate 验证下游服务配置
func (c *DownstreamConfig) Validate() error {
	if c.HTTPHost == "" {
		return fmt.Errorf("downstream-http-host is required")
	}
	if c.HTTPPort <= 0 || c.HTTPPort > MaxPort {
		return fmt.Errorf("downstream-http-port must be between 1 and %d", MaxPort)
	}
	if c.HTTPPath == "" {
		return fmt.Errorf("downstream-http-path is required")
	}
	return nil
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