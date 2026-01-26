package router

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/mowind/web3signer-go/internal/signer"
	"github.com/sirupsen/logrus"
	"github.com/umbracle/ethgo"
)

// Test_validateRequest_Success 测试验证请求成功
func Test_validateRequest_Success(t *testing.T) {
	handler := createSimpleTestHandler(t)
	testAddress := "0x1234567890123456789012345678901234567890"

	request := &jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "eth_sendTransaction",
		ID:      "test_id",
		Params: json.RawMessage(`{
			"from": "` + testAddress + `",
			"to": "0x0987654321098765432109876543210987654321",
			"gas": "0x5208",
			"gasPrice": "0x4a817c800",
			"value": "0xde0b6b3a7640000"
		}`),
	}

	tx, err := handler.validateRequest(request)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if tx == nil {
		t.Fatal("Expected transaction, got nil")
	}

	if tx.From.String() != testAddress {
		t.Errorf("Expected from address %s, got %s", testAddress, tx.From.String())
	}
}

// Test_validateRequest_WrongAddress 测试地址不匹配
func Test_validateRequest_WrongAddress(t *testing.T) {
	handler := createSimpleTestHandler(t)

	request := &jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "eth_sendTransaction",
		ID:      "test_id",
		Params: json.RawMessage(`{
			"from": "0x0000000000000000000000000000000000000000",
			"to": "0x0987654321098765432109876543210987654321",
			"gas": "0x5208"
		}`),
	}

	_, err := handler.validateRequest(request)
	if err == nil {
		t.Error("Expected error for wrong address, got nil")
	}

	if err != nil && err.Error() != "from address mismatch" {
		t.Errorf("Expected 'from address mismatch' error, got %v", err)
	}
}

// Test_validateRequest_InvalidParams 测试无效参数
func Test_validateRequest_InvalidParams(t *testing.T) {
	handler := createSimpleTestHandler(t)

	request := &jsonrpc.Request{
		JSONRPC: "2.0",
		Method:  "eth_sendTransaction",
		ID:      "test_id",
		Params:  json.RawMessage(`{invalid json}`),
	}

	_, err := handler.validateRequest(request)
	if err == nil {
		t.Error("Expected error for invalid params, got nil")
	}
}

// createSimpleTestHandler 创建测试用的 SignHandler（仅用于 validateRequest 测试）
func createSimpleTestHandler(t *testing.T) *SignHandler {
	testAddress := "0x1234567890123456789012345678901234567890"
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mpcSigner := signer.NewMPCKMSSigner(&testKMSClient{}, "test-key-id", ethgo.HexToAddress(testAddress), big.NewInt(1))
	mockForward := newMockDownstreamClient()

	return &SignHandler{
		BaseHandler:   NewBaseHandler("sign", logger),
		signer:        mpcSigner,
		client:        mockForward,
		downstreamRPC: nil,
	}
}
