package jsonrpc

import (
	"encoding/json"
	"testing"
)

func TestParseRequest_Single(t *testing.T) {
	data := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`

	requests, err := ParseRequest([]byte(data))
	if err != nil {
		t.Fatalf("ParseRequest failed: %v", err)
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	req := requests[0]
	if req.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc=2.0, got %s", req.JSONRPC)
	}
	if req.Method != "eth_blockNumber" {
		t.Errorf("Expected method=eth_blockNumber, got %s", req.Method)
	}
	if req.ID != float64(1) { // JSON 数字默认解析为 float64
		t.Errorf("Expected id=1, got %v", req.ID)
	}
}

func TestParseRequest_Batch(t *testing.T) {
	data := `[
		{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1},
		{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x...", "latest"],"id":2}
	]`

	requests, err := ParseRequest([]byte(data))
	if err != nil {
		t.Fatalf("ParseRequest failed: %v", err)
	}

	if len(requests) != 2 {
		t.Fatalf("Expected 2 requests, got %d", len(requests))
	}
}

func TestParseRequest_InvalidJSON(t *testing.T) {
	data := `invalid json`

	_, err := ParseRequest([]byte(data))
	if err == nil {
		t.Fatal("Expected error for invalid JSON")
	}
}

func TestParseRequest_InvalidVersion(t *testing.T) {
	data := `{"jsonrpc":"1.0","method":"test","id":1}`

	_, err := ParseRequest([]byte(data))
	if err == nil {
		t.Fatal("Expected error for invalid version")
	}
}

func TestParseRequest_EmptyMethod(t *testing.T) {
	data := `{"jsonrpc":"2.0","method":"","id":1}`

	_, err := ParseRequest([]byte(data))
	if err == nil {
		t.Fatal("Expected error for empty method")
	}
}

func TestNewResponse(t *testing.T) {
	result := map[string]interface{}{
		"blockNumber": "0x1234",
	}

	resp, err := NewResponse(1, result)
	if err != nil {
		t.Fatalf("NewResponse failed: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc=2.0, got %s", resp.JSONRPC)
	}
	if resp.Error != nil {
		t.Errorf("Expected no error, got %v", resp.Error)
	}
	if resp.ID != 1 {
		t.Errorf("Expected id=1, got %v", resp.ID)
	}

	// 验证结果可以正确解析
	var decodedResult map[string]interface{}
	if err := json.Unmarshal(resp.Result, &decodedResult); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if decodedResult["blockNumber"] != "0x1234" {
		t.Errorf("Expected blockNumber=0x1234, got %v", decodedResult["blockNumber"])
	}
}

func TestNewErrorResponse(t *testing.T) {
	err := &Error{
		Code:    -32601,
		Message: "Method not found",
	}

	resp := NewErrorResponse(1, err)

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc=2.0, got %s", resp.JSONRPC)
	}
	if resp.Error == nil {
		t.Fatal("Expected error")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code=-32601, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Method not found" {
		t.Errorf("Expected error message='Method not found', got %s", resp.Error.Message)
	}
	if resp.ID != 1 {
		t.Errorf("Expected id=1, got %v", resp.ID)
	}
}

func TestMarshalResponse(t *testing.T) {
	resp := &Response{
		JSONRPC: "2.0",
		Result:  json.RawMessage(`{"result":"success"}`),
		ID:      1,
	}

	data, err := MarshalResponse(resp)
	if err != nil {
		t.Fatalf("MarshalResponse failed: %v", err)
	}

	// 验证可以正确解析回来
	var decodedResp Response
	if err := json.Unmarshal(data, &decodedResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if decodedResp.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc=2.0, got %s", decodedResp.JSONRPC)
	}
}