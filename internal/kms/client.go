package kms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
)

// Client 表示 MPC-KMS 客户端
type Client struct {
	config     *config.KMSConfig
	httpClient HTTPClientInterface
}

// NewClient 创建新的 MPC-KMS 客户端
func NewClient(cfg *config.KMSConfig) *Client {
	return &Client{
		config:     cfg,
		httpClient: NewHTTPClient(cfg),
	}
}

// NewClientWithHTTPClient 创建新的 MPC-KMS 客户端，使用指定的 HTTP 客户端
func NewClientWithHTTPClient(cfg *config.KMSConfig, httpClient HTTPClientInterface) *Client {
	return &Client{
		config:     cfg,
		httpClient: httpClient,
	}
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
	resp, err := c.httpClient.Do(req)
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
		// 需要审批，自动轮询任务结果
		taskResp, err := UnmarshalTaskResponse(respBody)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal task response: %w", err)
		}

		// 轮询任务完成(最多等待5分钟)
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		result, err := c.WaitForTaskCompletion(ctx, taskResp.TaskID, 5*time.Second)
		if err != nil {
			return nil, fmt.Errorf("task polling failed: %w", err)
		}

		// 返回签名结果
		var signResp SignResponse
		if err := json.Unmarshal([]byte(result.Response), &signResp); err != nil {
			return nil, fmt.Errorf("failed to parse signature from task: %w", err)
		}
		return []byte(signResp.Signature), nil

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
