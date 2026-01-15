package jsonrpc

import (
	"encoding/json"
	"fmt"
)

// Request 表示 JSON-RPC 2.0 请求
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// Response 表示 JSON-RPC 2.0 响应
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

// Error 表示 JSON-RPC 2.0 错误
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ParseRequest 解析 JSON-RPC 请求
func ParseRequest(data []byte) ([]Request, error) {
	// 尝试解析为单个请求
	var singleReq Request
	if err := json.Unmarshal(data, &singleReq); err == nil {
		// 验证单个请求
		if err := validateRequest(&singleReq); err != nil {
			return nil, err
		}
		return []Request{singleReq}, nil
	}

	// 尝试解析为批量请求
	var batchReqs []Request
	if err := json.Unmarshal(data, &batchReqs); err != nil {
		return nil, fmt.Errorf("invalid JSON-RPC request: %v", err)
	}

	// 验证批量请求
	if len(batchReqs) == 0 {
		return nil, fmt.Errorf("empty batch request")
	}

	for i := range batchReqs {
		if err := validateRequest(&batchReqs[i]); err != nil {
			return nil, fmt.Errorf("request at index %d: %v", i, err)
		}
	}

	return batchReqs, nil
}

// validateRequest 验证单个请求
func validateRequest(req *Request) error {
	if req.JSONRPC != JSONRPCVersion {
		return fmt.Errorf("invalid jsonrpc version: %s", req.JSONRPC)
	}

	if req.Method == "" {
		return fmt.Errorf("method is required")
	}

	// ID 可以是 null、字符串或数字
	if req.ID != nil {
		switch v := req.ID.(type) {
		case string, float64, int, int64:
			// 有效类型
		default:
			return fmt.Errorf("invalid id type: %T", v)
		}
	}

	return nil
}

// NewResponse 创建成功响应
func NewResponse(id interface{}, result interface{}) (*Response, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return &Response{
		JSONRPC: JSONRPCVersion,
		Result:  resultJSON,
		ID:      id,
	}, nil
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(id interface{}, err *Error) *Response {
	return &Response{
		JSONRPC: JSONRPCVersion,
		Error:   err,
		ID:      id,
	}
}

// MarshalResponse 序列化响应
func MarshalResponse(resp *Response) ([]byte, error) {
	return json.Marshal(resp)
}

// MarshalResponses 序列化批量响应
func MarshalResponses(responses []*Response) ([]byte, error) {
	if len(responses) == 1 {
		return MarshalResponse(responses[0])
	}
	return json.Marshal(responses)
}