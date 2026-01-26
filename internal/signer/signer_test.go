package signer

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/mowind/web3signer-go/internal/kms"
	"github.com/umbracle/ethgo"
)

// mockKMSClient 是 MPC-KMS 客户端的 mock 实现
type mockKMSClient struct {
	signFunc              func(ctx context.Context, keyID string, message []byte) ([]byte, error)
	signWithOptionsFunc   func(ctx context.Context, keyID string, message []byte, encoding kms.DataEncoding, summary *kms.SignSummary, callbackURL string) ([]byte, error)
	getTaskResultFunc     func(ctx context.Context, taskID string) (*kms.TaskResult, error)
	waitForTaskCompletion func(ctx context.Context, taskID string, interval time.Duration) (*kms.TaskResult, error)
	doFunc                func(req *http.Request) (*http.Response, error)
}

func (m *mockKMSClient) Sign(ctx context.Context, keyID string, message []byte) ([]byte, error) {
	if m.signFunc != nil {
		return m.signFunc(ctx, keyID, message)
	}
	// 返回一个有效的十六进制编码的签名（65字节）
	signature := make([]byte, 65)
	for i := 0; i < 65; i++ {
		signature[i] = byte(i + 1)
	}
	return []byte(hex.EncodeToString(signature)), nil
}

func (m *mockKMSClient) SignWithOptions(ctx context.Context, keyID string, message []byte, encoding kms.DataEncoding, summary *kms.SignSummary, callbackURL string) ([]byte, error) {
	if m.signWithOptionsFunc != nil {
		return m.signWithOptionsFunc(ctx, keyID, message, encoding, summary, callbackURL)
	}
	return []byte("mock_signature_with_options"), nil
}

func (m *mockKMSClient) GetTaskResult(ctx context.Context, taskID string) (*kms.TaskResult, error) {
	if m.getTaskResultFunc != nil {
		return m.getTaskResultFunc(ctx, taskID)
	}
	return &kms.TaskResult{Status: kms.TaskStatusDone}, nil
}

func (m *mockKMSClient) WaitForTaskCompletion(ctx context.Context, taskID string, interval time.Duration) (*kms.TaskResult, error) {
	if m.waitForTaskCompletion != nil {
		return m.waitForTaskCompletion(ctx, taskID, interval)
	}
	return &kms.TaskResult{Status: kms.TaskStatusDone}, nil
}

func (m *mockKMSClient) Do(req *http.Request) (*http.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return nil, nil
}

func TestMPCKMSSigner_Address(t *testing.T) {
	expectedAddress := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	client := &mockKMSClient{}
	signer := NewMPCKMSSigner(client, "test-key-id", expectedAddress, big.NewInt(1))

	address := signer.Address()
	if address != expectedAddress {
		t.Errorf("Expected address %s, got %s", expectedAddress.String(), address.String())
	}
}

func TestMPCKMSSigner_Sign(t *testing.T) {
	expectedHash := make([]byte, 32)
	for i := 0; i < 32; i++ {
		expectedHash[i] = byte(i)
	}

	client := &mockKMSClient{
		signFunc: func(ctx context.Context, keyID string, message []byte) ([]byte, error) {
			// 验证输入参数
			if keyID != "test-key-id" {
				t.Errorf("Expected keyID %s, got %s", "test-key-id", keyID)
			}

			if !bytes.Equal(message, expectedHash) {
				t.Errorf("Expected message %x, got %x", expectedHash, message)
			}

			// 返回一个有效的十六进制编码的签名（65字节）
			signature := make([]byte, 65)
			for i := 0; i < 65; i++ {
				signature[i] = byte(i + 50)
			}
			return []byte(hex.EncodeToString(signature)), nil
		},
	}

	address := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	signer := NewMPCKMSSigner(client, "test-key-id", address, big.NewInt(1))

	signature, err := signer.Sign(expectedHash)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	// 验证返回的签名长度是否正确
	if len(signature) != 65 {
		t.Errorf("Expected signature length 65, got %d", len(signature))
	}
}

func TestMPCKMSSigner_SignTransaction(t *testing.T) {
	// 创建一个 Legacy 交易
	toAddr := ethgo.HexToAddress("0x0987654321098765432109876543210987654321")
	tx := &ethgo.Transaction{
		To:       &toAddr,
		Nonce:    5,
		GasPrice: 20000000000,
		Gas:      21000,
		Value:    big.NewInt(1000000000000000000), // 1 ETH
		Input:    []byte{},
	}

	client := &mockKMSClient{
		signFunc: func(ctx context.Context, keyID string, message []byte) ([]byte, error) {
			// 返回一个模拟的 65 字节签名
			signature := make([]byte, 65)
			for i := 0; i < 65; i++ {
				signature[i] = byte(i + 1)
			}
			return []byte(hex.EncodeToString(signature)), nil
		},
	}

	address := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	signer := NewMPCKMSSigner(client, "test-key-id", address, big.NewInt(1))

	signedTx, err := signer.SignTransaction(tx)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// 验证交易被正确签名
	if signedTx.From != address {
		t.Errorf("Expected From address %s, got %s", address.String(), signedTx.From.String())
	}

	if len(signedTx.R) != 32 {
		t.Errorf("Expected R length 32, got %d", len(signedTx.R))
	}

	if len(signedTx.S) != 32 {
		t.Errorf("Expected S length 32, got %d", len(signedTx.S))
	}

	if len(signedTx.V) != 1 {
		t.Errorf("Expected V length 1, got %d", len(signedTx.V))
	}
}

func TestMPCKMSSigner_SignTransactionWithSummary(t *testing.T) {
	toAddr := ethgo.HexToAddress("0x0987654321098765432109876543210987654321")
	tx := &ethgo.Transaction{
		To:       &toAddr,
		Nonce:    5,
		GasPrice: 20000000000,
		Gas:      21000,
		Value:    big.NewInt(1000000000000000000),
		Input:    []byte{},
	}

	expectedSummary := &kms.SignSummary{
		Type:   "TRANSFER",
		From:   "0x1234567890123456789012345678901234567890",
		To:     "0x0987654321098765432109876543210987654321",
		Amount: "1000000000000000000",
		Token:  "ETH",
		Remark: "test remark",
	}

	client := &mockKMSClient{
		signWithOptionsFunc: func(ctx context.Context, keyID string, message []byte, encoding kms.DataEncoding, summary *kms.SignSummary, callbackURL string) ([]byte, error) {
			// 验证摘要信息
			if summary.Type != expectedSummary.Type {
				t.Errorf("Expected summary type %s, got %s", expectedSummary.Type, summary.Type)
			}
			if summary.From != expectedSummary.From {
				t.Errorf("Expected from %s, got %s", expectedSummary.From, summary.From)
			}
			if summary.To != expectedSummary.To {
				t.Errorf("Expected to %s, got %s", expectedSummary.To, summary.To)
			}
			if summary.Amount != expectedSummary.Amount {
				t.Errorf("Expected amount %s, got %s", expectedSummary.Amount, summary.Amount)
			}
			if summary.Token != expectedSummary.Token {
				t.Errorf("Expected token %s, got %s", expectedSummary.Token, summary.Token)
			}

			// 返回模拟签名
			signature := make([]byte, 65)
			for i := 0; i < 65; i++ {
				signature[i] = byte(i + 100)
			}
			return []byte(hex.EncodeToString(signature)), nil
		},
	}

	address := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	signer := NewMPCKMSSigner(client, "test-key-id", address, big.NewInt(1))

	signedTx, err := signer.SignTransactionWithSummary(tx, expectedSummary)
	if err != nil {
		t.Fatalf("Failed to sign transaction with summary: %v", err)
	}

	if signedTx.From != address {
		t.Errorf("Expected From address %s, got %s", address.String(), signedTx.From.String())
	}
}

func TestMPCKMSSigner_CreateTransferSummary(t *testing.T) {
	toAddr := ethgo.HexToAddress("0x0987654321098765432109876543210987654321")
	tx := &ethgo.Transaction{
		To:    &toAddr,
		Value: big.NewInt(500000000000000000), // 0.5 ETH
	}

	address := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	client := &mockKMSClient{}
	signer := NewMPCKMSSigner(client, "test-key-id", address, big.NewInt(1))

	summary := signer.CreateTransferSummary(tx, "ETH", "test transfer")

	if summary.Type != "TRANSFER" {
		t.Errorf("Expected type TRANSFER, got %s", summary.Type)
	}

	if summary.From != address.String() {
		t.Errorf("Expected from %s, got %s", address.String(), summary.From)
	}

	expectedTo := "0x0987654321098765432109876543210987654321"
	if summary.To != expectedTo {
		t.Errorf("Expected to %s, got %s", expectedTo, summary.To)
	}

	if summary.Amount != "500000000000000000" {
		t.Errorf("Expected amount %s, got %s", "500000000000000000", summary.Amount)
	}

	if summary.Token != "ETH" {
		t.Errorf("Expected token ETH, got %s", summary.Token)
	}

	if summary.Remark != "test transfer" {
		t.Errorf("Expected remark 'test transfer', got %s", summary.Remark)
	}
}

func TestMPCKMSSigner_CreateTransferSummary_ContractCreation(t *testing.T) {
	// 测试合约创建交易（To 为 nil）
	tx := &ethgo.Transaction{
		To:    nil, // 合约创建
		Value: big.NewInt(0),
	}

	address := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	client := &mockKMSClient{}
	signer := NewMPCKMSSigner(client, "test-key-id", address, big.NewInt(1))

	summary := signer.CreateTransferSummary(tx, "", "")

	if summary.To != "" {
		t.Errorf("Expected empty to address for contract creation, got %s", summary.To)
	}

	if summary.Amount != "0" {
		t.Errorf("Expected amount 0, got %s", summary.Amount)
	}

	if summary.Token != "ETH" {
		t.Errorf("Expected token ETH, got %s", summary.Token)
	}
}

func TestMPCKMSSigner_Sign_InvalidSignatureLength(t *testing.T) {
	client := &mockKMSClient{
		signFunc: func(ctx context.Context, keyID string, message []byte) ([]byte, error) {
			// 返回长度不为 65 的签名
			return []byte("too_short"), nil
		},
	}

	address := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	signer := NewMPCKMSSigner(client, "test-key-id", address, big.NewInt(1))

	_, err := signer.Sign([]byte("test_hash"))
	if err == nil {
		t.Error("Expected error for invalid signature length, got none")
	}
}

func TestMPCKMSSigner_SignTransaction_InvalidSignatureLength(t *testing.T) {
	toAddr := ethgo.HexToAddress("0x0987654321098765432109876543210987654321")
	tx := &ethgo.Transaction{
		To:       &toAddr,
		Nonce:    5,
		GasPrice: 20000000000,
		Gas:      21000,
		Value:    big.NewInt(1000000000000000000),
		Input:    []byte{},
	}

	client := &mockKMSClient{
		signFunc: func(ctx context.Context, keyID string, message []byte) ([]byte, error) {
			// 返回长度不为 65 的签名
			return []byte("too_short"), nil
		},
	}

	address := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	signer := NewMPCKMSSigner(client, "test-key-id", address, big.NewInt(1))

	_, err := signer.SignTransaction(tx)
	if err == nil {
		t.Error("Expected error for invalid signature length, got none")
	}
}

func TestMPCKMSSigner_Sign_KMSError(t *testing.T) {
	client := &mockKMSClient{
		signFunc: func(ctx context.Context, keyID string, message []byte) ([]byte, error) {
			return nil, fmt.Errorf("KMS error")
		},
	}

	address := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	signer := NewMPCKMSSigner(client, "test-key-id", address, big.NewInt(1))

	_, err := signer.Sign(make([]byte, 32))
	if err == nil {
		t.Error("Expected error when KMS fails, got none")
	}

	if err.Error() != "failed to sign with MPC-KMS: KMS error" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestMPCKMSSigner_SignTransaction_AccessListTransaction(t *testing.T) {
	toAddr := ethgo.HexToAddress("0x0987654321098765432109876543210987654321")
	tx := &ethgo.Transaction{
		Type:  ethgo.TransactionAccessList,
		To:    &toAddr,
		Nonce: 5,
		Gas:   21000,
		Value: big.NewInt(1000000000000000000),
		Input: []byte{},
		AccessList: ethgo.AccessList{
			{
				Address: ethgo.Address{},
				Storage: []ethgo.Hash{},
			},
		},
		ChainID: big.NewInt(1),
	}

	client := &mockKMSClient{
		signFunc: func(ctx context.Context, keyID string, message []byte) ([]byte, error) {
			signature := make([]byte, 65)
			for i := 0; i < 65; i++ {
				signature[i] = byte(i + 1)
			}
			return []byte(hex.EncodeToString(signature)), nil
		},
	}

	address := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	signer := NewMPCKMSSigner(client, "test-key-id", address, big.NewInt(1))

	signedTx, err := signer.SignTransaction(tx)
	if err != nil {
		t.Fatalf("Failed to sign transaction with AccessList: %v", err)
	}

	if len(signedTx.R) != 32 {
		t.Errorf("Expected R length 32, got %d", len(signedTx.R))
	}

	if len(signedTx.S) != 32 {
		t.Errorf("Expected S length 32, got %d", len(signedTx.S))
	}
}

func TestMPCKMSSigner_SignTransaction_DynamicFeeTransaction(t *testing.T) {
	toAddr := ethgo.HexToAddress("0x0987654321098765432109876543210987654321")
	tx := &ethgo.Transaction{
		Type:                 ethgo.TransactionDynamicFee,
		To:                   &toAddr,
		Nonce:                5,
		Gas:                  21000,
		MaxFeePerGas:         big.NewInt(30000000000),
		MaxPriorityFeePerGas: big.NewInt(2000000000),
		Value:                big.NewInt(1000000000000000000),
		Input:                []byte{},
		ChainID:              big.NewInt(1),
	}

	client := &mockKMSClient{
		signFunc: func(ctx context.Context, keyID string, message []byte) ([]byte, error) {
			signature := make([]byte, 65)
			for i := 0; i < 65; i++ {
				signature[i] = byte(i + 1)
			}
			return []byte(hex.EncodeToString(signature)), nil
		},
	}

	address := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	signer := NewMPCKMSSigner(client, "test-key-id", address, big.NewInt(1))

	signedTx, err := signer.SignTransaction(tx)
	if err != nil {
		t.Fatalf("Failed to sign dynamic fee transaction: %v", err)
	}

	if len(signedTx.R) != 32 {
		t.Errorf("Expected R length 32, got %d", len(signedTx.R))
	}

	if len(signedTx.S) != 32 {
		t.Errorf("Expected S length 32, got %d", len(signedTx.S))
	}
}
