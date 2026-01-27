package main

import (
	"fmt"

	"github.com/mowind/web3signer-go/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Flag 定义命令行标志
type Flag struct {
	Name         string
	DefaultValue interface{}
	Description  string
	BindTo       string // viper 键名
	Required     bool
}

// flags 定义所有命令行标志
var flags = []Flag{
	// HTTP 服务器配置
	{
		Name:         "http-host",
		DefaultValue: config.DefaultHTTPHost,
		Description:  "HTTP server host",
		BindTo:       "http.host",
	},
	{
		Name:         "http-port",
		DefaultValue: config.DefaultHTTPPort,
		Description:  "HTTP server port",
		BindTo:       "http.port",
	},
	{
		Name:         "tls-cert-file",
		DefaultValue: "",
		Description:  "Path to TLS certificate file",
		BindTo:       "http.tls-cert-file",
	},
	{
		Name:         "tls-key-file",
		DefaultValue: "",
		Description:  "Path to TLS private key file",
		BindTo:       "http.tls-key-file",
	},
	{
		Name:         "tls-auto-redirect",
		DefaultValue: false,
		Description:  "Auto redirect HTTP to HTTPS",
		BindTo:       "http.tls-auto-redirect",
	},
	{
		Name:         "http-max-request-size",
		DefaultValue: int64(10),
		Description:  "Maximum request body size in MB (prevents DoS)",
		BindTo:       "http.max-request-size-mb",
	},
	{
		Name:         "cors-allowed-origins",
		DefaultValue: []string{},
		Description:  "CORS allowed origins (comma-separated), empty means allow all",
		BindTo:       "http.allowed-origins",
	},

	// MPC-KMS 配置
	{
		Name:         "kms-endpoint",
		DefaultValue: "",
		Description:  "MPC-KMS endpoint URL (required)",
		BindTo:       "kms.endpoint",
		Required:     true,
	},
	{
		Name:         "kms-access-key-id",
		DefaultValue: "",
		Description:  "MPC-KMS access key ID (required)",
		BindTo:       "kms.access-key-id",
		Required:     true,
	},
	{
		Name:         "kms-secret-key",
		DefaultValue: "",
		Description:  "MPC-KMS secret key (required)",
		BindTo:       "kms.secret-key",
		Required:     true,
	},
	{
		Name:         "kms-key-id",
		DefaultValue: "",
		Description:  "MPC-KMS key ID (required)",
		BindTo:       "kms.key-id",
		Required:     true,
	},
	{
		Name:         "kms-address",
		DefaultValue: "",
		Description:  "Ethereum address managed by MPC-KMS (required)",
		BindTo:       "kms.address",
		Required:     true,
	},

	// 下游服务配置
	{
		Name:         "downstream-http-host",
		DefaultValue: config.DefaultDownstreamHost,
		Description:  "Downstream HTTP service host",
		BindTo:       "downstream.http-host",
	},
	{
		Name:         "downstream-http-port",
		DefaultValue: config.DefaultDownstreamPort,
		Description:  "Downstream HTTP service port",
		BindTo:       "downstream.http-port",
	},
	{
		Name:         "downstream-http-path",
		DefaultValue: config.DefaultDownstreamPath,
		Description:  "Downstream HTTP service path",
		BindTo:       "downstream.http-path",
	},

	// 日志配置
	{
		Name:         "log-level",
		DefaultValue: config.DefaultLogLevel,
		Description:  "Log level (debug, info, warn, error, fatal)",
		BindTo:       "log.level",
	},
	{
		Name:         "log-format",
		DefaultValue: config.DefaultLogFormat,
		Description:  "Log format (json or text)",
		BindTo:       "log.format",
	},
}

// registerFlags 注册所有命令行标志
func registerFlags(cmd *cobra.Command) error {
	for _, flag := range flags {
		// 根据类型添加标志
		switch v := flag.DefaultValue.(type) {
		case string:
			cmd.Flags().String(flag.Name, v, flag.Description)
		case int:
			cmd.Flags().Int(flag.Name, v, flag.Description)
		case int64:
			cmd.Flags().Int64(flag.Name, v, flag.Description)
		case bool:
			cmd.Flags().Bool(flag.Name, v, flag.Description)
		case []string:
			cmd.Flags().StringSlice(flag.Name, v, flag.Description)
		default:
			return fmt.Errorf("unsupported flag type: %T for flag %s", v, flag.Name)
		}

		// 绑定到 viper
		if err := viper.BindPFlag(flag.BindTo, cmd.Flags().Lookup(flag.Name)); err != nil {
			return fmt.Errorf("failed to bind flag %s: %w", flag.Name, err)
		}

		// 标记必需标志
		if flag.Required {
			if err := cmd.MarkFlagRequired(flag.Name); err != nil {
				return fmt.Errorf("failed to mark flag %s as required: %w", flag.Name, err)
			}
		}
	}

	return nil
}
