package kms

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
	"github.com/mowind/web3signer-go/internal/utils"
	"github.com/sirupsen/logrus"
)

// HTTPClient is an HTTP client with MPC-KMS HMAC-SHA256 authentication.
//
// It automatically signs all requests according to MPC-KMS authentication specification,
// including timestamp generation, content hashing, and HMAC-SHA256 signature calculation.
type HTTPClient struct {
	kmsConfig  *config.KMSConfig
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewHTTPClient creates a new MPC-KMS HTTP client.
//
// The client is configured with a 30-second timeout and optimized connection pooling:
//   - MaxIdleConns: 100 (maximum idle connections across all hosts)
//   - MaxIdleConnsPerHost: 100 (maximum idle connections per host)
//   - IdleConnTimeout: 90s (timeout for idle connections)
//   - ResponseHeaderTimeout: 10s (timeout for receiving response headers)
//
// Parameters:
//   - kmsCfg: KMS configuration including endpoint and credentials
//   - logger: Logger for request/response logging
//
// Returns:
//   - *HTTPClient: A new HTTP client instance
func NewHTTPClient(kmsCfg *config.KMSConfig, logger *logrus.Logger) *HTTPClient {
	return &HTTPClient{
		kmsConfig: kmsCfg,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: utils.CreateTransport(100, 90*time.Second),
		},
		logger: logger,
	}
}

// SignRequest signs an HTTP request according to MPC-KMS specification.
//
// This method performs HMAC-SHA256 authentication:
//  1. Generate GMT timestamp
//  2. Calculate Content-SHA256 (base64 encoded)
//  3. Build signing string: VERB\nContent-SHA256\nContent-Type\nDate
//  4. Calculate HMAC-SHA256 signature
//  5. Set Authorization header: "MPC-KMS AK:Signature"
//
// Parameters:
//   - req: The HTTP request to sign (will be modified in place)
//   - body: The request body bytes for Content-SHA256 calculation
//
// Returns:
//   - error: An error if signing fails
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

// Do executes an HTTP request with automatic signing.
//
// This method:
//  1. Reads the request body (if present)
//  2. Signs the request using SignRequest
//  3. Executes the request
//  4. Logs the result at debug level
//
// Parameters:
//   - req: The HTTP request to execute (will be signed and sent)
//
// Returns:
//   - *http.Response: The HTTP response
//   - error: An error if request execution fails
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

// HTTPClientInterface defines the HTTP client interface for MPC-KMS requests.
//
// This interface allows for mocking and testing of HTTP operations.
type HTTPClientInterface interface {
	// SignRequest signs an HTTP request according to MPC-KMS specification.
	//
	// Parameters:
	//   - req: The HTTP request to sign
	//   - body: The request body bytes
	//
	// Returns:
	//   - error: An error if signing fails
	SignRequest(req *http.Request, body []byte) error

	// Do executes an HTTP request with automatic signing.
	//
	// Parameters:
	//   - req: The HTTP request to execute
	//
	// Returns:
	//   - *http.Response: The HTTP response
	//   - error: An error if execution fails
	Do(req *http.Request) (*http.Response, error)
}

// VerifyInterfaceImplementation 验证接口实现
var _ HTTPClientInterface = (*HTTPClient)(nil)

// GetTransport returns the HTTP transport used by the client.
// This is primarily for testing purposes to verify connection pool configuration.
func (c *HTTPClient) GetTransport() *http.Transport {
	if c.httpClient == nil || c.httpClient.Transport == nil {
		return nil
	}
	return c.httpClient.Transport.(*http.Transport)
}
