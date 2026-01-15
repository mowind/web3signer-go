package server

import (
	"github.com/gin-gonic/gin"
	"github.com/mowind/web3signer-go/internal/config"
	"github.com/sirupsen/logrus"
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

	router := b.createRouter()
	logger := b.createLogger()

	s := &Server{
		config: b.cfg,
		router: router,
		logger: logger,
	}

	s.setupRoutes()
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

// createRouter 创建 gin 路由器
func (b *Builder) createRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	if b.cfg.Log.Level == config.LogLevelDebug {
		router.Use(gin.Logger())
	}

	return router
}

// createLogger 创建日志器
func (b *Builder) createLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(getLogLevel(b.cfg.Log.Level))
	logger.SetFormatter(&logrus.JSONFormatter{})
	return logger
}