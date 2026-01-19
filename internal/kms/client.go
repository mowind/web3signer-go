package kms

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
)

// Client 表示 MPC-KMS 客户端
type Client struct {
	config     *config.KMSConfig
	httpClient *http.Client
}

// NewClient 创建新的 MPC-KMS 客户端
func NewClient(cfg *config.KMSConfig) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SignRequest 对 HTTP 请求进行签名（根据 MPC-KMS 文档规范）
func (c *Client) SignRequest(req *http.Request, body []byte) error {
	// 1. 生成 GMT 格式的时间戳
	date := time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")

	// 2. 计算 Content-SHA256
	contentSHA256 := calculateContentSHA256(body)

	// 3. 获取 Content-Type
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	// 4. 构建签名字符串（根据文档规范）
	signingString := buildSigningString(req.Method, contentSHA256, contentType, date)

	// 5. 计算 HMAC-SHA256 签名
	signature := calculateHMACSHA256(signingString, c.config.SecretKey)

	// 6. 构建 Authorization 头（根据文档规范）
	authHeader := buildAuthorizationHeader(c.config.AccessKeyID, signature)

	// 7. 设置请求头
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Date", date)
	req.Header.Set("Content-Type", contentType)

	return nil
}

// calculateContentSHA256 计算内容的 SHA256 哈希（base64编码）
func calculateContentSHA256(data []byte) string {
	if len(data) == 0 {
		// 空内容的 SHA256
		hash := sha256.Sum256([]byte(""))
		return base64.StdEncoding.EncodeToString(hash[:])
	}
	hash := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// buildSigningString 构建签名字符串（根据文档规范）
func buildSigningString(verb, contentSHA256, contentType, date string) string {
	// 格式：VERB + "\n" + Content-SHA256 + "\n" + Content-Type + "\n" + Date
	return fmt.Sprintf("%s\n%s\n%s\n%s",
		verb,
		contentSHA256,
		contentType,
		date,
	)
}

// calculateHMACSHA256 计算 HMAC-SHA256 签名（base64编码）
func calculateHMACSHA256(message, secretKey string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// buildAuthorizationHeader 构建 Authorization 头（根据文档规范）
func buildAuthorizationHeader(accessKeyID, signature string) string {
	// 格式：MPC-KMS AK:Signature
	return fmt.Sprintf("MPC-KMS %s:%s", accessKeyID, signature)
}

// Do 执行已签名的 HTTP 请求
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// 读取请求体以便计算哈希
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body = io.NopCloser(strings.NewReader(string(body)))
	}

	// 签名请求
	if err := c.SignRequest(req, body); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// 执行请求
	return c.httpClient.Do(req)
}

// Sign 调用 MPC-KMS 签名端点
func (c *Client) Sign(ctx context.Context, keyID string, message []byte) ([]byte, error) {
	return c.SignWithOptions(ctx, keyID, message, DataEncodingHex, nil, "")
}

// SignWithOptions 调用 MPC-KMS 签名端点，支持更多选项
func (c *Client) SignWithOptions(ctx context.Context, keyID string, message []byte, encoding DataEncoding, summary *SignSummary, callbackURL string) ([]byte, error) {
	// 构建签名请求
	signReq := NewSignRequest(message, encoding)
	if summary != nil {
		signReq.WithSummary(summary)
	}
	if callbackURL != "" {
		signReq.WithCallbackURL(callbackURL)
	}

	// 序列化请求体
	reqBody, err := signReq.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sign request: %w", err)
	}

	// 构建请求URL
	url := fmt.Sprintf("%s/api/v1/keys/%s/sign", c.config.Endpoint, keyID)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute sign request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 检查HTTP状态码
	switch resp.StatusCode {
	case http.StatusOK:
		// 直接返回签名结果
		signResp, err := UnmarshalSignResponse(respBody)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal sign response: %w", err)
		}
		return []byte(signResp.Signature), nil

	case http.StatusCreated:
		// 需要审批，返回任务ID
		taskResp, err := UnmarshalTaskResponse(respBody)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal task response: %w", err)
		}
		// TODO: 实现任务轮询逻辑
		return nil, fmt.Errorf("signature requires approval, task_id: %s", taskResp.TaskID)

	default:
		// 处理错误响应
		errResp, _ := UnmarshalErrorResponse(respBody)
		if errResp != nil {
			return nil, fmt.Errorf("MPC-KMS error (code: %d): %s", errResp.Code, errResp.Message)
		}
		return nil, fmt.Errorf("MPC-KMS request failed with status: %d", resp.StatusCode)
	}
}

// GetTaskResult 获取任务结果
func (c *Client) GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s/api/v1/tasks/%s", c.config.Endpoint, taskID)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 执行请求
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		errResp, _ := UnmarshalErrorResponse(respBody)
		if errResp != nil {
			return nil, fmt.Errorf("MPC-KMS error (code: %d): %s", errResp.Code, errResp.Message)
		}
		return nil, fmt.Errorf("MPC-KMS request failed with status: %d", resp.StatusCode)
	}

	// 解析任务结果
	taskResult, err := UnmarshalTaskResult(respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal task result: %w", err)
	}

	return taskResult, nil
}

// WaitForTaskCompletion 等待任务完成
func (c *Client) WaitForTaskCompletion(ctx context.Context, taskID string, interval time.Duration) (*TaskResult, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			result, err := c.GetTaskResult(ctx, taskID)
			if err != nil {
				return nil, err
			}

			switch result.Status {
			case TaskStatusDone:
				// 任务完成，解析签名结果
				if result.Response != "" {
					var signResp SignResponse
					if err := json.Unmarshal([]byte(result.Response), &signResp); err != nil {
						return nil, fmt.Errorf("failed to parse signature from task result: %w", err)
					}
					// 返回包含签名结果的任务结果
					return result, nil
				}
				return result, nil
			case TaskStatusFailed:
				return nil, fmt.Errorf("task failed: %s", result.Message)
			case TaskStatusRejected:
				return nil, fmt.Errorf("task rejected: %s", result.Message)
			case TaskStatusPendingApproval, TaskStatusApproved:
				// 继续等待
				continue
			default:
				return nil, fmt.Errorf("unknown task status: %s", result.Status)
			}
		}
	}
}
