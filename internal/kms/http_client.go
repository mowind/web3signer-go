package kms

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
	"github.com/sirupsen/logrus"
)

// HTTPClient 处理 MPC-KMS HTTP 请求签名和执行
type HTTPClient struct {
	kmsConfig  *config.KMSConfig
	logConfig  *config.LogConfig
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewHTTPClient 创建新的 HTTP 客户端
func NewHTTPClient(kmsCfg *config.KMSConfig, logCfg *config.LogConfig) *HTTPClient {
	return &HTTPClient{
		kmsConfig: kmsCfg,
		logConfig: logCfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: newLogger(logCfg.Level, logCfg.Format),
	}
}

// SignRequest 对 HTTP 请求进行签名（根据 MPC-KMS 文档规范）
func (c *HTTPClient) SignRequest(req *http.Request, body []byte) error {
	// 1. 生成 GMT 格式的时间戳
	date := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")

	// 2. 计算 Content-SHA256
	contentSHA256 := CalculateContentSHA256(body)

	// 3. 获取 Content-Type
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	// 4. 构建签名字符串（根据文档规范）
	signingString := BuildSigningString(req.Method, contentSHA256, contentType, date)

	// 5. 计算 HMAC-SHA256 签名
	signature := CalculateHMACSHA256(signingString, c.kmsConfig.SecretKey)

	// 6. 构建 Authorization 头（根据文档规范）
	authHeader := BuildAuthorizationHeader(c.kmsConfig.AccessKeyID, signature)

	// 7. 设置请求头
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Date", date)
	req.Header.Set("Content-Type", contentType)

	return nil
}

// Do 执行已签名的 HTTP 请求
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	// 记录请求开始（debug 级别）
	c.logger.WithFields(logrus.Fields{
		"method": req.Method,
		"url":    req.URL.String(),
	}).Debug("Executing HTTP request")

	// 读取请求体以便计算哈希
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			c.logger.WithFields(logrus.Fields{
				"method": req.Method,
				"url":    req.URL.String(),
				"error":  err.Error(),
			}).Error("Failed to read request body")
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body = io.NopCloser(strings.NewReader(string(body)))
	}

	// 签名请求
	if err := c.SignRequest(req, body); err != nil {
		c.logger.WithFields(logrus.Fields{
			"method": req.Method,
			"url":    req.URL.String(),
			"error":  err.Error(),
		}).Error("Failed to sign request")
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// 执行请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"method": req.Method,
			"url":    req.URL.String(),
			"error":  err.Error(),
		}).Error("HTTP request failed")
		return nil, err
	}

	// 记录响应状态（debug 级别）
	c.logger.WithFields(logrus.Fields{
		"method":      req.Method,
		"url":         req.URL.String(),
		"status_code": resp.StatusCode,
	}).Debug("HTTP request completed")

	return resp, nil
}

// HTTPClientInterface 定义 HTTP 客户端接口
type HTTPClientInterface interface {
	// SignRequest 对 HTTP 请求进行签名
	SignRequest(req *http.Request, body []byte) error

	// Do 执行已签名的 HTTP 请求
	Do(req *http.Request) (*http.Response, error)
}

// VerifyInterfaceImplementation 验证接口实现
var _ HTTPClientInterface = (*HTTPClient)(nil)
