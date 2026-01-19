package server

import (
	"math/big"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mowind/web3signer-go/internal/config"
	"github.com/mowind/web3signer-go/internal/downstream"
	"github.com/mowind/web3signer-go/internal/kms"
	"github.com/mowind/web3signer-go/internal/router"
	"github.com/mowind/web3signer-go/internal/signer"
	"github.com/sirupsen/logrus"
	"github.com/umbracle/ethgo"
	ethgojsonrpc "github.com/umbracle/ethgo/jsonrpc"
)

// Builder 服务器构建器
type Builder struct {
	cfg *config.Config
}

// NewBuilder 创建新的服务器构建器
func NewBuilder(cfg *config.Config) *Builder {
	return &Builder{cfg: cfg}
}

// Build 构建服务器
func (b *Builder) Build() *Server {
	b.setGinMode()

	logger := b.createLogger()

	downstreamClient := downstream.NewClient(&b.cfg.Downstream)

	downstreamEndpoint := b.cfg.Downstream.BuildURL()
	rpcClient, err := ethgojsonrpc.NewClient(downstreamEndpoint)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create downstream RPC client")
	}

	var chainIDHex string
	err = rpcClient.Call("eth_chainId", &chainIDHex)
	if err != nil {
		logger.WithError(err).Fatal("Failed to get chainId from downstream")
	}

	chainID := new(big.Int)
	if len(chainIDHex) >= 2 && chainIDHex[0:2] == "0x" {
		chainID.SetString(chainIDHex[2:], 16)
	} else {
		chainID.SetString(chainIDHex, 0)
	}

	logger.WithField("chainId", chainID).Info("Retrieved chainId from downstream")

	kmsClient := kms.NewClient(&b.cfg.KMS)
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
	router.Use(gin.Recovery())
	router.Use(b.corsMiddleware())

	if b.cfg.Log.Level == config.LogLevelDebug {
		router.Use(gin.Logger())
	}

	// JSON-RPC端点，路由到jsonRPCRouter
	router.POST("/", b.handleJSONRPCRequest(jsonRPCRouter))
	router.OPTIONS("/", b.handleJSONRPCRequest(jsonRPCRouter))

	// 健康检查端点
	router.GET("/health", b.healthHandler(logger))

	// 就绪检查端点
	router.GET("/ready", b.readyHandler(logger))

	return router
}

// createLogger 创建日志器
func (b *Builder) createLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(getLogLevel(b.cfg.Log.Level))
	logger.SetFormatter(&logrus.JSONFormatter{})
	return logger
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
		jsonRPCRouter.HandleHTTPRequest(c.Writer, c.Request)
	}
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
