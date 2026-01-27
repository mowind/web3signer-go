package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
	"github.com/mowind/web3signer-go/internal/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd 表示基础命令
var rootCmd = &cobra.Command{
	Use:   "web3signer",
	Short: "web3signer-go is a Go implementation of web3signer with MPC-KMS signing support",
	Long: `web3signer-go is inspired by Consensys/web3signer, but only supports MPC-KMS signing.

It provides an HTTP JSON-RPC interface that:
1. Signs transactions using MPC-KMS
2. Forwards other JSON-RPC methods to a downstream service
3. Supports eth_sign, eth_signTransaction, and eth_sendTransaction methods`,
	Version: Version,
	Run:     run,
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// 全局标志
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.web3signer.yaml)")

	// 注册所有标志
	if err := registerFlags(rootCmd); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to register flags: %v\n", err)
		os.Exit(1)
	}
}

// initConfig 初始化配置
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName(".web3signer")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("WEB3SIGNER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// run 是主命令的执行函数
func run(cmd *cobra.Command, args []string) {
	// 加载配置
	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// 打印配置摘要
	fmt.Printf("Starting web3signer-go with configuration: %s\n", cfg.String())

	// 创建并启动服务器
	server := server.New(&cfg)
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
		os.Exit(1)
	}

	// 等待中断信号
	waitForInterrupt(server)
}

// waitForInterrupt 等待中断信号并优雅关闭服务器
func waitForInterrupt(server *server.Server) {
	// 创建信号通道
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	sig := <-sigChan
	fmt.Printf("\nReceived signal: %v. Shutting down...\n", sig)

	// 创建关闭上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭服务器
	if err := server.Stop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
		// 不调用os.Exit，让defer正常执行
		return
	}

	fmt.Println("Server shutdown complete")
}
