package kms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
	"github.com/sirupsen/logrus"
)

// Client is an MPC-KMS client for signing operations.
//
// It wraps an HTTP client with HMAC-SHA256 authentication and provides
// methods for signing data with support for asynchronous approval workflows.
type Client struct {
	kmsConfig  *config.KMSConfig
	httpClient HTTPClientInterface
	logger     *logrus.Logger

	// URL caching to avoid repeated string concatenation
	signURL         string
	taskURLTemplate string
	urlMu           sync.RWMutex
}

// NewClient creates a new MPC-KMS client with default HTTP client.
//
// Parameters:
//   - kmsCfg: KMS configuration including endpoint, credentials, and key ID
//   - logger: Logger for request/response logging
//
// Returns:
//   - *Client: A new MPC-KMS client instance
func NewClient(kmsCfg *config.KMSConfig, logger *logrus.Logger) *Client {
	return &Client{
		kmsConfig:  kmsCfg,
		httpClient: NewHTTPClient(kmsCfg, logger),
		logger:     logger,
	}
}

// NewClientWithHTTPClient creates a new MPC-KMS client with custom HTTP client.
//
// Use this method for testing or when custom HTTP client configuration is needed.
//
// Parameters:
//   - kmsCfg: KMS configuration including endpoint, credentials, and key ID
//   - logger: Logger for request/response logging
//   - httpClient: Custom HTTP client implementing HTTPClientInterface
//
// Returns:
//   - *Client: A new MPC-KMS client instance
func NewClientWithHTTPClient(kmsCfg *config.KMSConfig, logger *logrus.Logger, httpClient HTTPClientInterface) *Client {
	return &Client{
		kmsConfig:  kmsCfg,
		httpClient: httpClient,
		logger:     logger,
	}
}

// NewClientWithLogger creates a new MPC-KMS client with custom HTTP client and logger.
//
// This method is deprecated; use NewClientWithHTTPClient instead.
//
// Parameters:
//   - kmsCfg: KMS configuration including endpoint, credentials, and key ID
//   - logger: Logger for request/response logging
//   - httpClient: Custom HTTP client implementing HTTPClientInterface
//
// Returns:
//   - *Client: A new MPC-KMS client instance
func NewClientWithLogger(kmsCfg *config.KMSConfig, logger *logrus.Logger, httpClient HTTPClientInterface) *Client {
	return &Client{
		kmsConfig:  kmsCfg,
		httpClient: httpClient,
		logger:     logger,
	}
}

// resetURLCache resets the cached URLs. Used for testing when the endpoint changes.
func (c *Client) resetURLCache() {
	c.urlMu.Lock()
	defer c.urlMu.Unlock()
	c.signURL = ""
	c.taskURLTemplate = ""
}

// getSignURL returns the pre-computed sign endpoint URL with lazy initialization.
// Thread-safe via sync.RWMutex.
func (c *Client) getSignURL(keyID string) string {
	c.urlMu.RLock()
	if c.signURL != "" {
		defer c.urlMu.RUnlock()
		return fmt.Sprintf("%s%s/sign", c.signURL, keyID)
	}
	c.urlMu.RUnlock()

	c.urlMu.Lock()
	defer c.urlMu.Unlock()

	// Double check
	if c.signURL == "" {
		c.signURL = fmt.Sprintf("%s/api/v1/keys/", c.kmsConfig.Endpoint)
	}
	return fmt.Sprintf("%s%s/sign", c.signURL, keyID)
}

// getTaskURL returns the pre-computed task endpoint URL with lazy initialization.
// Thread-safe via sync.RWMutex.
func (c *Client) getTaskURL(taskID string) string {
	c.urlMu.RLock()
	if c.taskURLTemplate != "" {
		defer c.urlMu.RUnlock()
		return fmt.Sprintf("%s%s", c.taskURLTemplate, taskID)
	}
	c.urlMu.RUnlock()

	c.urlMu.Lock()
	defer c.urlMu.Unlock()

	// Double check
	if c.taskURLTemplate == "" {
		c.taskURLTemplate = fmt.Sprintf("%s/api/v1/tasks/", c.kmsConfig.Endpoint)
	}
	return fmt.Sprintf("%s%s", c.taskURLTemplate, taskID)
}

// Sign signs the given message using the specified key ID.
//
// This is a convenience method that calls SignWithOptions with default encoding (HEX).
//
// Parameters:
//   - ctx: Context for the request (supports cancellation and timeout)
//   - keyID: The KMS key identifier to use for signing
//   - message: The message bytes to be signed
//
// Returns:
//   - []byte: The signature bytes
//   - error: An error if the signing operation fails
func (c *Client) Sign(ctx context.Context, keyID string, message []byte) ([]byte, error) {
	return c.SignWithOptions(ctx, keyID, message, DataEncodingHex, nil, "")
}

// SignWithOptions signs the given message with extended options.
//
// This method supports:
//   - Data encoding format (PLAIN, BASE64, HEX)
//   - Transaction summary for approval workflow
//   - Callback URL for asynchronous notifications
//
// If the request requires approval (returns HTTP 201), it automatically polls
// for task completion with a 5-minute timeout.
//
// Parameters:
//   - ctx: Context for the request (supports cancellation and timeout)
//   - keyID: The KMS key identifier to use for signing
//   - message: The message bytes to be signed
//   - encoding: Data encoding format (DataEncodingPlain, DataEncodingBase64, DataEncodingHex)
//   - summary: Optional transaction summary for approval workflow
//   - callbackURL: Optional URL for asynchronous approval notifications
//
// Returns:
//   - []byte: The signature bytes
//   - error: An error if the signing operation fails
func (c *Client) SignWithOptions(ctx context.Context, keyID string, message []byte, encoding DataEncoding, summary *SignSummary, callbackURL string) ([]byte, error) {
	startTime := time.Now()

	// 记录请求开始
	c.logger.WithFields(logrus.Fields{
		"key_id":       keyID,
		"encoding":     encoding,
		"endpoint":     c.kmsConfig.Endpoint,
		"has_summary":  summary != nil,
		"has_callback": callbackURL != "",
	}).Info("Starting sign request")

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

	// 记录请求体（用于调试）
	// 注意：请求体包含待签名数据和交易摘要，不是敏感数据（如私钥）
	// 这些数据对开发调试签名流程非常有价值
	c.logger.WithFields(logrus.Fields{
		"key_id":       keyID,
		"request_body": string(reqBody),
	}).Debug("Sign request body")

	url := c.getSignURL(keyID)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"key_id": keyID,
			"url":    url,
			"error":  err.Error(),
		}).Error("Failed to execute sign request")
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

	// 统一响应日志格式 - 使用 has_signature 布尔值
	c.logger.WithFields(logrus.Fields{
		"key_id":        keyID,
		"endpoint":      c.kmsConfig.Endpoint,
		"status_code":   resp.StatusCode,
		"has_signature": resp.StatusCode == http.StatusOK,
	}).Debug("Sign response received")

	// Debug 级别记录完整响应体（用于调试签名流程）
	if c.logger.IsLevelEnabled(logrus.DebugLevel) {
		c.logger.WithFields(logrus.Fields{
			"key_id":        keyID,
			"response_body": string(respBody),
		}).Debug("Sign response body")
	}

	// 检查HTTP状态码
	switch resp.StatusCode {
	case http.StatusOK:
		duration := time.Since(startTime).Milliseconds()
		// 直接返回签名结果
		signResp, err := UnmarshalSignResponse(respBody)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal sign response: %w", err)
		}

		c.logger.WithFields(logrus.Fields{
			"key_id":      keyID,
			"duration_ms": duration,
			"status":      "completed",
		}).Info("Sign request completed successfully")

		return []byte(signResp.Signature), nil

	case http.StatusCreated:
		taskResp, err := UnmarshalTaskResponse(respBody)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal task response: %w", err)
		}

		c.logger.WithFields(logrus.Fields{
			"key_id":  keyID,
			"task_id": taskResp.TaskID,
			"status":  "pending_approval",
		}).Info("Sign request requires approval, starting task polling")

		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		result, err := c.WaitForTaskCompletion(ctx, taskResp.TaskID, 5*time.Second)
		if err != nil {
			c.logger.WithFields(logrus.Fields{
				"task_id": taskResp.TaskID,
				"error":   err.Error(),
			}).Error("Task polling failed")
			return nil, fmt.Errorf("task polling failed for task %s: %w", taskResp.TaskID, err)
		}

		// 返回签名结果
		var signResp SignResponse
		if err := json.Unmarshal([]byte(result.Response), &signResp); err != nil {
			return nil, fmt.Errorf("failed to parse signature from task: %w", err)
		}

		duration := time.Since(startTime).Milliseconds()
		c.logger.WithFields(logrus.Fields{
			"key_id":      keyID,
			"task_id":     taskResp.TaskID,
			"duration_ms": duration,
			"status":      "approved_and_completed",
		}).Info("Sign request completed after approval")

		return []byte(signResp.Signature), nil

	default:
		// 处理错误响应
		errResp, _ := UnmarshalErrorResponse(respBody)
		if errResp != nil {
			c.logger.WithFields(logrus.Fields{
				"key_id":      keyID,
				"status_code": resp.StatusCode,
				"error_code":  errResp.Code,
				"message":     errResp.Message,
			}).Error("MPC-KMS returned error response")
			return nil, fmt.Errorf("MPC-KMS error (code: %d): %s", errResp.Code, errResp.Message)
		}
		c.logger.WithFields(logrus.Fields{
			"key_id":      keyID,
			"status_code": resp.StatusCode,
		}).Error("MPC-KMS request failed with unexpected status")
		return nil, fmt.Errorf("MPC-KMS request failed with status: %d", resp.StatusCode)
	}
}

// GetTaskResult retrieves the result of an asynchronous signing task.
//
// This method is used to check the status of a task that requires approval.
//
// Parameters:
//   - ctx: Context for the request (supports cancellation and timeout)
//   - taskID: The task ID to query
//
// Returns:
//   - *TaskResult: The task result with status and response data
//   - error: An error if the task retrieval fails
func (c *Client) GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error) {
	url := c.getTaskURL(taskID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for task %s: %w", taskID, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task request for task %s: %w", taskID, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for task %s: %w", taskID, err)
	}

	c.logger.WithFields(logrus.Fields{
		"task_id":       taskID,
		"status_code":   resp.StatusCode,
		"response_body": string(respBody),
	}).Debug("Task result response")

	if resp.StatusCode != http.StatusOK {
		errResp, _ := UnmarshalErrorResponse(respBody)
		if errResp != nil {
			return nil, fmt.Errorf("MPC-KMS error for task %s (code: %d): %s", taskID, errResp.Code, errResp.Message)
		}
		return nil, fmt.Errorf("MPC-KMS request failed for task %s with status: %d", taskID, resp.StatusCode)
	}

	// 解析任务结果
	taskResult, err := UnmarshalTaskResult(respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal task result: %w", err)
	}

	return taskResult, nil
}

// WaitForTaskCompletion waits for an asynchronous signing task to complete.
//
// This method polls the task status at the specified interval until:
//   - Task completes (TaskStatusDone)
//   - Task fails (TaskStatusFailed)
//   - Task is rejected (TaskStatusRejected)
//   - Max attempts reached (5 minutes total)
//   - Context is cancelled or times out
//
// Parameters:
//   - ctx: Context for the request (supports cancellation and timeout)
//   - taskID: The task ID to monitor
//   - interval: The polling interval between status checks
//
// Returns:
//   - *TaskResult: The task result when complete
//   - error: An error if task fails, is rejected, or context is cancelled
func (c *Client) WaitForTaskCompletion(ctx context.Context, taskID string, interval time.Duration) (*TaskResult, error) {
	startTime := time.Now()
	maxAttempts := int(5 * time.Minute / interval)

	for attempt := 0; attempt < maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
			result, err := c.GetTaskResult(ctx, taskID)
			if err != nil {
				return nil, err
			}

			// 记录每次轮询状态（debug 级别避免刷屏）
			c.logger.WithFields(logrus.Fields{
				"task_id": taskID,
				"status":  result.Status,
				"attempt": attempt + 1,
			}).Debug("Task status check")

			switch result.Status {
			case TaskStatusDone:
				// 任务完成，解析签名结果
				duration := time.Since(startTime).Milliseconds()
				if result.Response != "" {
					var signResp SignResponse
					if err := json.Unmarshal([]byte(result.Response), &signResp); err != nil {
						return nil, fmt.Errorf("failed to parse signature from task result: %w", err)
					}
					// 返回包含签名结果的任务结果
					c.logger.WithFields(logrus.Fields{
						"task_id":        taskID,
						"status":         "done",
						"total_attempts": attempt + 1,
						"duration_ms":    duration,
					}).Info("Task completed successfully")
					return result, nil
				}
				c.logger.WithFields(logrus.Fields{
					"task_id":        taskID,
					"status":         "done",
					"total_attempts": attempt + 1,
					"duration_ms":    duration,
				}).Info("Task completed (no response data)")
				return result, nil
			case TaskStatusFailed:
				c.logger.WithFields(logrus.Fields{
					"task_id": taskID,
					"status":  "failed",
					"message": result.Message,
				}).Error("Task failed")
				return nil, fmt.Errorf("task failed: %s", result.Message)
			case TaskStatusRejected:
				c.logger.WithFields(logrus.Fields{
					"task_id": taskID,
					"status":  "rejected",
					"message": result.Message,
				}).Error("Task rejected")
				return nil, fmt.Errorf("task rejected: %s", result.Message)
			case TaskStatusPendingApproval, TaskStatusApproved:
				// 继续等待
				continue
			default:
				c.logger.WithFields(logrus.Fields{
					"task_id": taskID,
					"status":  result.Status,
				}).Error("Unknown task status")
				return nil, fmt.Errorf("unknown task status: %s", result.Status)
			}
		}
	}

	// 达到最大尝试次数
	return nil, fmt.Errorf("task polling timeout after %d attempts", maxAttempts)
}
