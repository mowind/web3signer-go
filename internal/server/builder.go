package server

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mowind/web3signer-go/internal/config"
	"github.com/mowind/web3signer-go/internal/downstream"
	"github.com/mowind/web3signer-go/internal/kms"
	"github.com/mowind/web3signer-go/internal/router"
	"github.com/mowind/web3signer-go/internal/signer"
	"github.com/sirupsen/logrus"
	ginlogrus "github.com/toorop/gin-logrus"
	"github.com/umbracle/ethgo"
	ethgojsonrpc "github.com/umbracle/ethgo/jsonrpc"
)

// Builder 服务器构建器
type Builder struct {
	cfg    *config.Config
	logger *logrus.Logger
}

// NewBuilder 创建新的服务器构建器
func NewBuilder(cfg *config.Config) *Builder {
	return &Builder{cfg: cfg}
}

// Build 构建服务器
func (b *Builder) Build() *Server {
	b.setGinMode()

	logger := b.createLogger()
	b.logger = logger

	downstreamClient := downstream.NewClient(&b.cfg.Downstream)

	downstreamEndpoint := b.cfg.Downstream.BuildURL()
	rpcClient, err := ethgojsonrpc.NewClient(downstreamEndpoint)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create downstream RPC client")
	}

	chainID, err := rpcClient.Eth().ChainID()
	if err != nil {
		logger.WithError(err).Fatal("Failed to get chainId from downstream")
	}

	logger.WithField("chainId", chainID).Info("Retrieved chainId from downstream")

	kmsClient := kms.NewClient(&b.cfg.KMS, &b.cfg.Log)
	kmsAddress := ethgo.HexToAddress(b.cfg.KMS.Address)
	mpcSigner := signer.NewMPCKMSSigner(kmsClient, b.cfg.KMS.KeyID, kmsAddress, chainID)

	routerFactory := router.NewRouterFactory(logger)
	jsonRPCRouter := routerFactory.CreateRouter(mpcSigner, downstreamClient)

	router := b.createGinRouter(jsonRPCRouter, logger)

	s := &Server{
		config:        b.cfg,
		router:        router,
		logger:        logger,
		jsonRPCRouter: jsonRPCRouter,
		kmsAddress:    b.cfg.KMS.Address,
	}

	return s
}

// setGinMode 设置 gin 模式
func (b *Builder) setGinMode() {
	if b.cfg.Log.Level == config.LogLevelDebug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
}

// createGinRouter 创建 gin 路由器
func (b *Builder) createGinRouter(jsonRPCRouter *router.Router, logger *logrus.Logger) *gin.Engine {
	router := gin.New()

	// 添加请求 ID 中间件
	router.Use(b.requestIDMiddleware())
	router.Use(ginlogrus.Logger(logger))
	router.Use(gin.Recovery())
	router.Use(b.corsMiddleware())

	// JSON-RPC端点，路由到jsonRPCRouter
	router.POST("/", b.handleJSONRPCRequest(jsonRPCRouter))
	router.OPTIONS("/", b.handleJSONRPCRequest(jsonRPCRouter))

	// 健康检查端点
	router.GET("/health", b.healthHandler(logger))

	// 就绪检查端点
	router.GET("/ready", b.readyHandler(logger))

	return router
}

// requestIDMiddleware 生成并传递请求 ID
func (b *Builder) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成或获取请求 ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// 保存到上下文
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// generateRequestID 生成唯一请求 ID
func generateRequestID() string {
	return fmt.Sprintf("%s-%d", randomString(8), time.Now().UnixNano())
}

// randomString 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// createLogger 创建日志器
func (b *Builder) createLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(getLogLevel(b.cfg.Log.Level))

	// 根据配置设置格式（替换硬编码的 JSONFormatter）
	logger.SetFormatter(b.createLogFormatter())

	return logger
}

// createLogFormatter 创建日志格式化器
func (b *Builder) createLogFormatter() logrus.Formatter {
	switch strings.ToLower(b.cfg.Log.Format) {
	case config.LogFormatJSON:
		return &logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05Z07:00",
		}
	case config.LogFormatText:
		return &logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
			ForceColors:     b.isTerminal(),
		}
	default:
		// 默认使用 text
		return &logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
		}
	}
}

// isTerminal 检查输出是否为终端
func (b *Builder) isTerminal() bool {
	// 简化判断：如果格式是 text 且级别是 debug，启用颜色
	return b.cfg.Log.Format == config.LogFormatText &&
		b.cfg.Log.Level == config.LogLevelDebug
}

// healthHandler 处理健康检查请求
func (b *Builder) healthHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"time":   time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
}

// readyHandler 处理就绪检查请求
func (b *Builder) readyHandler(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ready",
			"time":   time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
}

// handleJSONRPCRequest 处理JSON-RPC请求
func (b *Builder) handleJSONRPCRequest(jsonRPCRouter *router.Router) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := b.getLoggerWithContext(c)
		jsonRPCRouter.HandleHTTPRequestWithContext(c.Writer, c.Request, logger)
	}
}

// getLoggerWithContext 获取带上下文的 logger
func (b *Builder) getLoggerWithContext(c *gin.Context) *logrus.Entry {
	logger := b.logger.WithField("component", "http_server")
	if requestID, exists := c.Get("request_id"); exists {
		logger = logger.WithField("request_id", requestID)
	}
	return logger
}

// corsMiddleware 处理CORS请求
func (b *Builder) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
