package signer

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/mowind/web3signer-go/internal/kms"
	"github.com/umbracle/ethgo"
)

// MPCKMSSigner 实现 ethgo.Key 接口，使用 MPC-KMS 进行签名
type MPCKMSSigner struct {
	client  kms.ClientInterface
	keyID   string
	address ethgo.Address
}

// NewMPCKMSSigner 创建新的 MPC-KMS 签名器
func NewMPCKMSSigner(client kms.ClientInterface, keyID string, address ethgo.Address) *MPCKMSSigner {
	return &MPCKMSSigner{
		client:  client,
		keyID:   keyID,
		address: address,
	}
}

// Address 返回签名器的地址（实现 ethgo.Key 接口）
func (s *MPCKMSSigner) Address() ethgo.Address {
	return s.address
}

// Sign 对哈希进行签名（实现 ethgo.Key 接口）
func (s *MPCKMSSigner) Sign(hash []byte) ([]byte, error) {
	// 将哈希转换为十六进制字符串
	hashHex := hex.EncodeToString(hash)

	// 调用 MPC-KMS 进行签名
	signatureHex, err := s.client.Sign(context.Background(), s.keyID, []byte(hashHex))
	if err != nil {
		return nil, fmt.Errorf("failed to sign with MPC-KMS: %v", err)
	}

	// 将十六进制签名转换回字节
	signature, err := hex.DecodeString(string(signatureHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %v", err)
	}

	return signature, nil
}

// SignTransaction 对交易进行签名
func (s *MPCKMSSigner) SignTransaction(tx *ethgo.Transaction) (*ethgo.Transaction, error) {
	// 创建交易的副本以避免修改原始交易
	txCopy := tx.Copy()

	// 设置 From 字段为签名器的地址
	txCopy.From = s.address

	// 计算交易哈希
	hash, err := txCopy.GetHash()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction hash: %v", err)
	}

	// 使用 MPC-KMS 对交易哈希进行签名
	signature, err := s.Sign(hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	// 解析签名 R, S, V 值
	if len(signature) != 65 {
		return nil, fmt.Errorf("invalid signature length: expected 65, got %d", len(signature))
	}

	// 设置交易签名
	txCopy.R = signature[0:32]
	txCopy.S = signature[32:64]
	txCopy.V = signature[64:65]

	// 根据交易类型调整 V 值
	if err := s.adjustVValue(txCopy); err != nil {
		return nil, fmt.Errorf("failed to adjust V value: %v", err)
	}

	return txCopy, nil
}

// adjustVValue 根据交易类型调整 V 值
func (s *MPCKMSSigner) adjustVValue(tx *ethgo.Transaction) error {
	if len(tx.V) != 1 {
		return fmt.Errorf("invalid V value length: expected 1, got %d", len(tx.V))
	}

	v := tx.V[0]

	// 根据交易类型调整 V 值
	switch tx.Type {
	case ethgo.TransactionLegacy:
		// Legacy 交易：V = 27 + recID 或 28 + recID
		if v < 27 {
			tx.V[0] = v + 27
		}

	case ethgo.TransactionAccessList:
		// EIP-2930 交易：V = recID
		// 不需要调整

	case ethgo.TransactionDynamicFee:
		// EIP-1559 交易：V = recID
		// 不需要调整

	default:
		return fmt.Errorf("unsupported transaction type: %v", tx.Type)
	}

	return nil
}

// SignTransactionWithSummary 对交易进行签名，并包含交易摘要信息
func (s *MPCKMSSigner) SignTransactionWithSummary(tx *ethgo.Transaction, summary *kms.SignSummary) (*ethgo.Transaction, error) {
	// 创建交易的副本
	txCopy := tx.Copy()

	// 设置 From 字段
	txCopy.From = s.address

	// 计算交易哈希
	hash, err := txCopy.GetHash()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction hash: %v", err)
	}

	// 将哈希转换为十六进制
	hashHex := hex.EncodeToString(hash[:])

	// 使用 MPC-KMS 进行签名，包含摘要信息
	signatureHex, err := s.client.SignWithOptions(
		context.Background(),
		s.keyID,
		[]byte(hashHex),
		kms.DataEncodingHex,
		summary,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction with summary: %v", err)
	}

	// 解析签名
	signature, err := hex.DecodeString(string(signatureHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %v", err)
	}

	if len(signature) != 65 {
		return nil, fmt.Errorf("invalid signature length: expected 65, got %d", len(signature))
	}

	// 设置交易签名
	txCopy.R = signature[0:32]
	txCopy.S = signature[32:64]
	txCopy.V = signature[64:65]

	// 调整 V 值
	if err := s.adjustVValue(txCopy); err != nil {
		return nil, fmt.Errorf("failed to adjust V value: %v", err)
	}

	return txCopy, nil
}

// CreateTransferSummary 从交易创建转账摘要
func (s *MPCKMSSigner) CreateTransferSummary(tx *ethgo.Transaction, token string, remark string) *kms.SignSummary {
	from := s.address.String()

	var to string
	if tx.To != nil {
		to = tx.To.String()
	} else {
		to = "" // 合约创建
	}

	amount := "0"
	if tx.Value != nil {
		amount = tx.Value.String()
	}

	if token == "" {
		token = "ETH"
	}

	return kms.NewTransferSummary(from, to, amount, token, remark)
}

// VerifyInterface 验证接口实现
var _ ethgo.Key = (*MPCKMSSigner)(nil)
