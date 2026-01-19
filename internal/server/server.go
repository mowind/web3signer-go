package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mowind/web3signer-go/internal/config"
	"github.com/mowind/web3signer-go/internal/router"
	"github.com/sirupsen/logrus"
)

// Server 表示 HTTP 服务器
type Server struct {
	config        *config.Config
	router        *gin.Engine
	server        *http.Server
	logger        *logrus.Logger
	jsonRPCRouter *router.Router
	kmsAddress    string
}

// New 创建新的 HTTP 服务器
func New(cfg *config.Config) *Server {
	builder := NewBuilder(cfg)
	return builder.Build()
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
