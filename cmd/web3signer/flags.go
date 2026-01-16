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
		default:
			return fmt.Errorf("unsupported flag type: %T for flag %s", v, flag.Name)
		}

		// 绑定到 viper
		if err := viper.BindPFlag(flag.BindTo, cmd.Flags().Lookup(flag.Name)); err != nil {
			return fmt.Errorf("failed to bind flag %s: %w", flag.Name, err)
		}

		// 标记必需标志
		if flag.Required {
			cmd.MarkFlagRequired(flag.Name)
		}
	}

	return nil
}
