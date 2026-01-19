package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mowind/web3signer-go/internal/config"
	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/sirupsen/logrus"
)

// Server 表示 HTTP 服务器
type Server struct {
	config *config.Config
	router *gin.Engine
	server *http.Server
	logger *logrus.Logger
}

// New 创建新的 HTTP 服务器
func New(cfg *config.Config) *Server {
	builder := NewBuilder(cfg)
	return builder.Build()
}

// setupRoutes 设置服务器路由
func (s *Server) setupRoutes() {
	// 健康检查端点
	s.router.GET("/health", s.healthHandler)
	s.router.GET("/ready", s.readyHandler)

	// JSON-RPC 端点
	s.router.POST("/", s.jsonRPCHandler)
}

// healthHandler 处理健康检查请求
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// readyHandler 处理就绪检查请求
func (s *Server) readyHandler(c *gin.Context) {
	// TODO: 添加就绪检查逻辑（如检查 KMS 连接）
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// jsonRPCHandler 处理 JSON-RPC 请求
func (s *Server) jsonRPCHandler(c *gin.Context) {
	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		s.logger.WithError(err).Error("Failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// 解析 JSON-RPC 请求
	requests, err := jsonrpc.ParseRequest(body)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to parse JSON-RPC request")
		// 返回 JSON-RPC 解析错误
		resp := jsonrpc.NewErrorResponse(nil, jsonrpc.ParseError)
		data, _ := jsonrpc.MarshalResponse(resp)
		c.Data(http.StatusOK, "application/json", data)
		return
	}

	// 处理请求
	responses := make([]*jsonrpc.Response, 0, len(requests))
	for _, req := range requests {
		resp := s.handleSingleRequest(req)
		responses = append(responses, resp)
	}

	// 序列化响应
	data, err := jsonrpc.MarshalResponses(responses)
	if err != nil {
		s.logger.WithError(err).Error("Failed to marshal JSON-RPC responses")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// handleSingleRequest 处理单个 JSON-RPC 请求
func (s *Server) handleSingleRequest(req jsonrpc.Request) *jsonrpc.Response {
	s.logger.WithFields(logrus.Fields{
		"method": req.Method,
		"id":     req.ID,
	}).Debug("Processing JSON-RPC request")

	// TODO: 根据方法名路由到不同的处理器
	// 目前返回方法未实现错误
	return jsonrpc.NewErrorResponse(req.ID, jsonrpc.MethodNotFoundError)
}

// Start 启动 HTTP 服务器
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.HTTP.Host, s.config.HTTP.Port)

	s.server = &http.Server{
		Addr:              addr,
		Handler:           s.router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	s.logger.WithFields(logrus.Fields{
		"host": s.config.HTTP.Host,
		"port": s.config.HTTP.Port,
	}).Info("Starting HTTP server")

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Fatal("HTTP server error")
		}
	}()

	return nil
}

// Stop 优雅停止 HTTP 服务器
func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		s.logger.Info("Shutting down HTTP server")
		return s.server.Shutdown(ctx)
	}
	return nil
}

// getLogLevel 将字符串日志级别转换为 logrus.Level
func getLogLevel(level string) logrus.Level {
	switch level {
	case config.LogLevelDebug:
		return logrus.DebugLevel
	case config.LogLevelInfo:
		return logrus.InfoLevel
	case config.LogLevelWarn:
		return logrus.WarnLevel
	case config.LogLevelError:
		return logrus.ErrorLevel
	case config.LogLevelFatal:
		return logrus.FatalLevel
	default:
		return logrus.InfoLevel
	}
}
