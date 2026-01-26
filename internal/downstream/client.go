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
	"github.com/sirupsen/logrus"
)

// Client is an HTTP client for forwarding JSON-RPC requests to Ethereum nodes.
//
// This client provides transparent proxy functionality with connection pooling
// and proper error handling for both single and batch requests.
type Client struct {
	config     *config.DownstreamConfig
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewClient creates a new downstream service client.
//
// The client is configured with a 30-second timeout and
// connection pooling (100 max idle connections per host).
//
// Parameters:
//   - cfg: Downstream service configuration (host, port, path)
//
// Returns:
//   - *Client: A new downstream client instance
func NewClient(cfg *config.DownstreamConfig) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: createTransport(),
		},
		logger: logrus.StandardLogger(),
	}
}

// createTransport 创建HTTP传输配置，用于优化连接池性能
func createTransport() *http.Transport {
	return &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		DisableCompression:    false,
		DisableKeepAlives:     false,
	}
}

// ForwardRequest forwards a single JSON-RPC request to downstream service.
//
// This method validates response ID matching and logs warnings on mismatch.
//
// Parameters:
//   - ctx: Context for request (supports cancellation and timeout)
//   - req: The JSON-RPC request to forward
//
// Returns:
//   - *jsonrpc.Response: The response from downstream service
//   - error: An error if forwarding fails
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
	defer func() {
		_ = resp.Body.Close()
	}()

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
			// 记录ID不匹配警告
			c.logger.WithFields(logrus.Fields{
				"request_id":  req.ID,
				"response_id": jsonResp.ID,
			}).Warn("JSON-RPC ID mismatch in response")
		}
	} else if req.ID != nil {
		// 如果请求有ID但响应没有，设置响应ID
		jsonResp.ID = req.ID
	}

	return &jsonResp, nil
}

// ForwardBatchRequest forwards a batch of JSON-RPC requests.
//
// This method preserves response order and validates:
//   - Response count matches request count
//   - Response IDs match request IDs (with warnings on mismatch)
//
// Parameters:
//   - ctx: Context for request (supports cancellation and timeout)
//   - requests: The JSON-RPC requests to forward
//
// Returns:
//   - []jsonrpc.Response: Ordered responses matching request order
//   - error: An error if forwarding fails
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
	defer func() {
		_ = resp.Body.Close()
	}()

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
				// 记录ID不匹配警告
				c.logger.WithFields(logrus.Fields{
					"index":       i,
					"request_id":  requests[i].ID,
					"response_id": jsonResponses[i].ID,
				}).Warn("JSON-RPC ID mismatch in batch response")
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

	// 快速路径:相同类型直接比较
	switch v1 := id1.(type) {
	case string:
		if v2, ok := id2.(string); ok {
			return v1 == v2
		}
	case int:
		if v2, ok := id2.(int); ok {
			return v1 == v2
		}
	case float64:
		if v2, ok := id2.(float64); ok {
			return v1 == v2
		}
	}

	// 降级到字符串比较(兼容不同类型)
	return fmt.Sprintf("%v", id1) == fmt.Sprintf("%v", id2)
}

// TestConnection tests connectivity to downstream Ethereum node.
//
// This method sends a web3_clientVersion request to verify
// the node is reachable and responsive.
//
// Parameters:
//   - ctx: Context for request (supports cancellation and timeout)
//
// Returns:
//   - error: An error if connection test fails
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

// GetEndpoint returns the full downstream service URL.
//
// Returns:
//   - string: The complete URL including host, port, and path
func (c *Client) GetEndpoint() string {
	return c.config.BuildURL()
}

// Close closes the downstream client.
//
// HTTP connections are managed automatically by the HTTP client,
// so this is currently a no-op.
//
// Returns:
//   - error: Always returns nil
func (c *Client) Close() error {
	// 目前不需要特殊清理，HTTP客户端会自动管理连接
	return nil
}

// GetTransport returns HTTP transport used by client.
// This is primarily for testing purposes to verify connection pool configuration.
func (c *Client) GetTransport() *http.Transport {
	if c.httpClient == nil || c.httpClient.Transport == nil {
		return nil
	}
	return c.httpClient.Transport.(*http.Transport)
}
