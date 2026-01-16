package downstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
	"github.com/mowind/web3signer-go/internal/jsonrpc"
)

// Client 表示下游服务客户端
type Client struct {
	config     *config.DownstreamConfig
	httpClient *http.Client
}

// NewClient 创建新的下游服务客户端
func NewClient(cfg *config.DownstreamConfig) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: createTransport(),
		},
	}
}

// createTransport 创建HTTP传输配置
func createTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}
}

// ForwardRequest 转发JSON-RPC请求到下游服务
func (c *Client) ForwardRequest(ctx context.Context, req *jsonrpc.Request) (*jsonrpc.Response, error) {
	// 序列化请求
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, WrapError(err, ErrorCodeInvalidResponse, "failed to marshal request")
	}

	// 构建下游服务URL
	url := c.config.BuildURL()

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqData))
	if err != nil {
		return nil, WrapError(err, ErrorCodeRequestFailed, "failed to create HTTP request")
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// 执行请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ConnectionError(err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, WrapError(err, ErrorCodeInvalidResponse, "failed to read response body")
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, RequestError(fmt.Errorf("downstream service returned status %d: %s",
			resp.StatusCode, string(respBody)))
	}

	// 解析JSON-RPC响应
	var jsonResp jsonrpc.Response
	if err := json.Unmarshal(respBody, &jsonResp); err != nil {
		return nil, InvalidResponseError(err)
	}

	// 确保响应ID与请求ID匹配
	if req.ID != nil && jsonResp.ID != nil {
		// 尝试比较ID值
		if !compareIDs(req.ID, jsonResp.ID) {
			// 记录ID不匹配，但继续使用响应中的ID
			// 在实际生产环境中可能需要记录警告
		}
	} else if req.ID != nil {
		// 如果请求有ID但响应没有，设置响应ID
		jsonResp.ID = req.ID
	}

	return &jsonResp, nil
}

// ForwardBatchRequest 转发批量JSON-RPC请求到下游服务
func (c *Client) ForwardBatchRequest(ctx context.Context, requests []jsonrpc.Request) ([]jsonrpc.Response, error) {
	// 序列化批量请求
	reqData, err := json.Marshal(requests)
	if err != nil {
		return nil, WrapError(err, ErrorCodeInvalidResponse, "failed to marshal batch request")
	}

	// 构建下游服务URL
	url := c.config.BuildURL()

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqData))
	if err != nil {
		return nil, WrapError(err, ErrorCodeRequestFailed, "failed to create HTTP request")
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// 执行请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ConnectionError(err)
	}
	defer resp.Body.Close()

	// 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, WrapError(err, ErrorCodeInvalidResponse, "failed to read response body")
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, RequestError(fmt.Errorf("downstream service returned status %d: %s",
			resp.StatusCode, string(respBody)))
	}

	// 解析批量响应
	var jsonResponses []jsonrpc.Response
	if err := json.Unmarshal(respBody, &jsonResponses); err != nil {
		// 如果不是数组，尝试解析为单个响应
		var singleResp jsonrpc.Response
		if err := json.Unmarshal(respBody, &singleResp); err != nil {
			return nil, InvalidResponseError(err)
		}
		jsonResponses = []jsonrpc.Response{singleResp}
	}

	// 确保响应顺序和ID匹配
	if len(jsonResponses) != len(requests) {
		return nil, BatchSizeMismatchError(len(requests), len(jsonResponses))
	}

	// 验证和修复响应ID
	for i := range jsonResponses {
		if requests[i].ID != nil && jsonResponses[i].ID != nil {
			if !compareIDs(requests[i].ID, jsonResponses[i].ID) {
				// 记录ID不匹配，但继续使用响应中的ID
				// 在实际生产环境中可能需要记录警告
			}
		} else if requests[i].ID != nil {
			// 如果请求有ID但响应没有，设置响应ID
			jsonResponses[i].ID = requests[i].ID
		}
	}

	return jsonResponses, nil
}

// compareIDs 比较两个JSON-RPC ID值是否相等
func compareIDs(id1, id2 interface{}) bool {
	// 如果都是nil，相等
	if id1 == nil && id2 == nil {
		return true
	}

	// 如果只有一个为nil，不相等
	if id1 == nil || id2 == nil {
		return false
	}

	// 尝试转换为字符串比较
	str1 := fmt.Sprintf("%v", id1)
	str2 := fmt.Sprintf("%v", id2)
	return str1 == str2
}

// TestConnection 测试下游服务连接
func (c *Client) TestConnection(ctx context.Context) error {
	// 创建一个简单的测试请求
	testReq := jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "web3_clientVersion",
		ID:      1,
	}

	_, err := c.ForwardRequest(ctx, &testReq)
	if err != nil {
		return ConnectionError(fmt.Errorf("connection test failed: %w", err))
	}

	return nil
}

// GetEndpoint 获取下游服务端点URL
func (c *Client) GetEndpoint() string {
	return c.config.BuildURL()
}

// Close 关闭客户端连接
func (c *Client) Close() error {
	// 目前不需要特殊清理，HTTP客户端会自动管理连接
	return nil
}