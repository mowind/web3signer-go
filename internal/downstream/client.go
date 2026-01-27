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
	"github.com/mowind/web3signer-go/internal/utils"
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
//   - logger: Logger instance for logging ID mismatch warnings
//
// Returns:
//   - *Client: A new downstream client instance
func NewClient(cfg *config.DownstreamConfig, logger *logrus.Logger) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: utils.CreateTransport(100, 90*time.Second),
		},
		logger: logger,
	}
}

// performHTTPRequest handles the common HTTP request execution logic.
// It builds the request, executes it, and returns the response body reader.
// The caller is responsible for closing the reader (which closes the response body).
func (c *Client) performHTTPRequest(ctx context.Context, reqData []byte) (io.ReadCloser, error) {
	// Build URL
	url := c.config.BuildURL()

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqData))
	if err != nil {
		return nil, WrapError(err, ErrorCodeRequestFailed, "failed to create HTTP request")
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, ConnectionError(err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		defer func() {
			_ = resp.Body.Close()
		}()
		// Read small amount of body for error message
		limitReader := io.LimitReader(resp.Body, 1024)
		respBody, _ := io.ReadAll(limitReader)
		return nil, RequestError(fmt.Errorf("downstream service returned status %d: %s",
			resp.StatusCode, string(respBody)))
	}

	return resp.Body, nil
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
	// Serialize request
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, WrapError(err, ErrorCodeInvalidResponse, "failed to marshal request")
	}

	// Execute HTTP request
	bodyReader, err := c.performHTTPRequest(ctx, reqData)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = bodyReader.Close()
	}()

	// Parse JSON-RPC response using stream decoder
	var jsonResp jsonrpc.Response
	decoder := json.NewDecoder(bodyReader)
	// Disallow unknown fields to ensure strict parsing
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&jsonResp); err != nil {
		return nil, InvalidResponseError(err)
	}

	// Validate response ID
	if req.ID != nil && jsonResp.ID != nil {
		if !compareIDs(req.ID, jsonResp.ID) {
			c.logger.WithFields(logrus.Fields{
				"request_id":  req.ID,
				"response_id": jsonResp.ID,
			}).Warn("JSON-RPC ID mismatch in response")
		}
	} else if req.ID != nil {
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
	// Serialize batch request
	reqData, err := json.Marshal(requests)
	if err != nil {
		return nil, WrapError(err, ErrorCodeInvalidResponse, "failed to marshal batch request")
	}

	// Execute HTTP request
	bodyReader, err := c.performHTTPRequest(ctx, reqData)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = bodyReader.Close()
	}()

	// Read all body to handle potentially mixed response types (array vs object)
	// We need the full body here because we might need to try multiple parsing strategies
	// For standard successful batch responses, this is still a slight overhead but safer
	respBody, err := io.ReadAll(bodyReader)
	if err != nil {
		return nil, WrapError(err, ErrorCodeInvalidResponse, "failed to read response body")
	}

	// Parse batch response
	var jsonResponses []jsonrpc.Response
	if err := json.Unmarshal(respBody, &jsonResponses); err != nil {
		// Fallback: try parsing as single response
		var singleResp jsonrpc.Response
		if err := json.Unmarshal(respBody, &singleResp); err != nil {
			return nil, InvalidResponseError(err)
		}
		jsonResponses = []jsonrpc.Response{singleResp}
	}

	// Validate response count
	if len(jsonResponses) != len(requests) {
		return nil, BatchSizeMismatchError(len(requests), len(jsonResponses))
	}

	// Validate response IDs
	for i := range jsonResponses {
		if requests[i].ID != nil && jsonResponses[i].ID != nil {
			if !compareIDs(requests[i].ID, jsonResponses[i].ID) {
				c.logger.WithFields(logrus.Fields{
					"index":       i,
					"request_id":  requests[i].ID,
					"response_id": jsonResponses[i].ID,
				}).Warn("JSON-RPC ID mismatch in batch response")
			}
		} else if requests[i].ID != nil {
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
