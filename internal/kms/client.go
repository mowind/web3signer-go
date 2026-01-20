package kms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
	"github.com/sirupsen/logrus"
)

// newLogger 创建新的日志记录器
func newLogger(level, format string) *logrus.Logger {
	logger := logrus.New()

	// 设置日志级别
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// 设置日志格式
	switch strings.ToLower(format) {
	case config.LogFormatJSON:
		logger.SetFormatter(&logrus.JSONFormatter{})
	case config.LogFormatText:
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	default:
		// 默认使用 text 格式
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return logger
}

// Client 表示 MPC-KMS 客户端
type Client struct {
	kmsConfig  *config.KMSConfig
	logConfig  *config.LogConfig
	httpClient HTTPClientInterface
	logger     *logrus.Logger
}

// NewClient 创建新的 MPC-KMS 客户端
func NewClient(kmsCfg *config.KMSConfig, logCfg *config.LogConfig) *Client {
	return &Client{
		kmsConfig:  kmsCfg,
		logConfig:  logCfg,
		httpClient: NewHTTPClient(kmsCfg, logCfg),
		logger:     newLogger(logCfg.Level, logCfg.Format),
	}
}

// NewClientWithHTTPClient 创建新的 MPC-KMS 客户端，使用指定的 HTTP 客户端
func NewClientWithHTTPClient(kmsCfg *config.KMSConfig, logCfg *config.LogConfig, httpClient HTTPClientInterface) *Client {
	return &Client{
		kmsConfig:  kmsCfg,
		logConfig:  logCfg,
		httpClient: httpClient,
		logger:     newLogger(logCfg.Level, logCfg.Format),
	}
}

// NewClientWithLogger 创建新的 MPC-KMS 客户端，使用指定的日志记录器（测试用）
func NewClientWithLogger(kmsCfg *config.KMSConfig, logCfg *config.LogConfig, httpClient HTTPClientInterface, logger *logrus.Logger) *Client {
	return &Client{
		kmsConfig:  kmsCfg,
		logConfig:  logCfg,
		httpClient: httpClient,
		logger:     logger,
	}
}

// Sign 调用 MPC-KMS 签名端点
func (c *Client) Sign(ctx context.Context, keyID string, message []byte) ([]byte, error) {
	return c.SignWithOptions(ctx, keyID, message, DataEncodingHex, nil, "")
}

// SignWithOptions 调用 MPC-KMS 签名端点，支持更多选项
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

	// 构建请求URL
	url := fmt.Sprintf("%s/api/v1/keys/%s/sign", c.kmsConfig.Endpoint, keyID)

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
		// 需要审批，自动轮询任务结果
		taskResp, err := UnmarshalTaskResponse(respBody)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal task response: %w", err)
		}

		c.logger.WithFields(logrus.Fields{
			"key_id":  keyID,
			"task_id": taskResp.TaskID,
			"status":  "pending_approval",
		}).Info("Sign request requires approval, starting task polling")

		// 轮询任务完成(最多等待5分钟)
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		result, err := c.WaitForTaskCompletion(ctx, taskResp.TaskID, 5*time.Second)
		if err != nil {
			c.logger.WithFields(logrus.Fields{
				"task_id": taskResp.TaskID,
				"error":   err.Error(),
			}).Error("Task polling failed")
			return nil, fmt.Errorf("task polling failed: %w", err)
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

// GetTaskResult 获取任务结果
func (c *Client) GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s/api/v1/tasks/%s", c.kmsConfig.Endpoint, taskID)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 执行请求
	resp, err := c.httpClient.Do(req)
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

	// 记录任务结果响应（安全，不包含敏感信息）
	c.logger.WithFields(logrus.Fields{
		"task_id":       taskID,
		"status_code":   resp.StatusCode,
		"response_body": string(respBody),
	}).Debug("Task result response")

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
	startTime := time.Now()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	attempt := 0

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			attempt++

			result, err := c.GetTaskResult(ctx, taskID)
			if err != nil {
				return nil, err
			}

			// 记录每次轮询状态（debug 级别避免刷屏）
			c.logger.WithFields(logrus.Fields{
				"task_id": taskID,
				"status":  result.Status,
				"attempt": attempt,
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
						"total_attempts": attempt,
						"duration_ms":    duration,
					}).Info("Task completed successfully")
					return result, nil
				}
				c.logger.WithFields(logrus.Fields{
					"task_id":        taskID,
					"status":         "done",
					"total_attempts": attempt,
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
}
