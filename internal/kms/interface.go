package kms

import (
	"context"
	"time"
)

// ClientInterface 定义 MPC-KMS 客户端接口
type ClientInterface interface {
	// Sign 对数据进行签名
	Sign(ctx context.Context, keyID string, message []byte) ([]byte, error)

	// SignWithOptions 对数据进行签名，支持更多选项
	SignWithOptions(ctx context.Context, keyID string, message []byte, encoding DataEncoding, summary *SignSummary, callbackURL string) ([]byte, error)

	// GetTaskResult 获取任务结果
	GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error)

	// WaitForTaskCompletion 等待任务完成
	WaitForTaskCompletion(ctx context.Context, taskID string, interval time.Duration) (*TaskResult, error)
}

// Signer 定义签名器接口
type Signer interface {
	// SignMessage 对消息进行签名
	SignMessage(ctx context.Context, message []byte) ([]byte, error)

	// SignTransaction 对交易进行签名
	SignTransaction(ctx context.Context, transactionData []byte) ([]byte, error)
}

// MPCKMSSigner 实现 Signer 接口，使用 MPC-KMS 进行签名
type MPCKMSSigner struct {
	client ClientInterface
	keyID  string
}

// NewMPCKMSSigner 创建新的 MPC-KMS 签名器
func NewMPCKMSSigner(client ClientInterface, keyID string) *MPCKMSSigner {
	return &MPCKMSSigner{
		client: client,
		keyID:  keyID,
	}
}

// SignMessage 对消息进行签名
func (s *MPCKMSSigner) SignMessage(ctx context.Context, message []byte) ([]byte, error) {
	return s.client.Sign(ctx, s.keyID, message)
}

// SignTransaction 对交易进行签名
func (s *MPCKMSSigner) SignTransaction(ctx context.Context, transactionData []byte) ([]byte, error) {
	// 创建交易摘要
	summary := &SignSummary{
		Type: string(SummaryTypeTransfer),
		// TODO: 从交易数据中提取 from, to, amount, token 等信息
		From:   "0x0000000000000000000000000000000000000000",
		To:     "0x0000000000000000000000000000000000000000",
		Amount: "0",
		Token:  "ETH",
	}

	return s.client.SignWithOptions(ctx, s.keyID, transactionData, DataEncodingHex, summary, "")
}

// VerifyInterfaceImplementation 验证接口实现
var (
	_ ClientInterface = (*Client)(nil)
	_ Signer          = (*MPCKMSSigner)(nil)
)
