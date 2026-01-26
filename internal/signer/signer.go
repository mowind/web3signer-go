package signer

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/mowind/web3signer-go/internal/kms"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/fastrlp"
)

// MPCKMSSigner implements ethgo.Key interface using MPC-KMS for signing.
//
// This signer wraps an MPC-KMS client to provide Ethereum key signing capabilities.
// It handles transaction signing with proper EIP-1559 and EIP-2930 support.
type MPCKMSSigner struct {
	client  kms.ClientInterface
	keyID   string
	address ethgo.Address
	chainID *big.Int
}

// NewMPCKMSSigner creates a new MPC-KMS signer instance.
//
// Parameters:
//   - client: The MPC-KMS client for signing operations
//   - keyID: The KMS key identifier for signing
//   - address: The Ethereum address associated with the key
//   - chainID: The chain ID for transaction signing (for EIP-1559)
//
// Returns:
//   - *MPCKMSSigner: A new signer instance
func NewMPCKMSSigner(client kms.ClientInterface, keyID string, address ethgo.Address, chainID *big.Int) *MPCKMSSigner {
	return &MPCKMSSigner{
		client:  client,
		keyID:   keyID,
		address: address,
		chainID: chainID,
	}
}

// Address returns the signer's Ethereum address.
//
// This implements the ethgo.Key interface.
//
// Returns:
//   - ethgo.Address: The address associated with this signer
func (s *MPCKMSSigner) Address() ethgo.Address {
	return s.address
}

// Sign signs a 32-byte hash using MPC-KMS.
//
// This implements the ethgo.Key interface for signing message hashes.
// The hash should be the Keccak-256 hash of data to sign.
//
// Parameters:
//   - hash: 32-byte hash to sign (typically Keccak-256)
//
// Returns:
//   - []byte: 65-byte signature (r, s, v values)
//   - error: An error if hash is invalid or signing fails
func (s *MPCKMSSigner) Sign(hash []byte) ([]byte, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("invalid hash length: expected 32 bytes, got %d", len(hash))
	}

	signatureHex, err := s.client.Sign(context.Background(), s.keyID, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign with MPC-KMS: %v", err)
	}

	signature, err := hex.DecodeString(string(signatureHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %v", err)
	}

	return signature, nil
}

// SignTransaction signs an Ethereum transaction.
//
// This method creates a copy of the transaction, computes its hash,
// signs it using MPC-KMS, and applies the signature (r, s, v values).
//
// Parameters:
//   - tx: The transaction to sign
//
// Returns:
//   - *ethgo.Transaction: A new transaction with signature applied
//   - error: An error if signing fails
func (s *MPCKMSSigner) SignTransaction(tx *ethgo.Transaction) (*ethgo.Transaction, error) {
	// 创建新的交易，手动复制所有字段
	signedTx := &ethgo.Transaction{
		From:     s.address,
		Nonce:    tx.Nonce,
		Gas:      tx.Gas,
		GasPrice: tx.GasPrice,
		Type:     tx.Type,
	}

	// 复制指针字段（如果有值）
	if tx.To != nil {
		toCopy := *tx.To
		signedTx.To = &toCopy
	} else {
		signedTx.To = nil
	}

	if tx.Value != nil {
		valueCopy := new(big.Int).Set(tx.Value)
		signedTx.Value = valueCopy
	}

	if tx.ChainID != nil {
		chainIDCopy := new(big.Int).Set(tx.ChainID)
		signedTx.ChainID = chainIDCopy
	}

	if tx.MaxFeePerGas != nil {
		maxFeeCopy := new(big.Int).Set(tx.MaxFeePerGas)
		signedTx.MaxFeePerGas = maxFeeCopy
	}

	if tx.MaxPriorityFeePerGas != nil {
		maxPriorityCopy := new(big.Int).Set(tx.MaxPriorityFeePerGas)
		signedTx.MaxPriorityFeePerGas = maxPriorityCopy
	}

	// 复制 Input 数据
	if tx.Input != nil {
		inputCopy := make([]byte, len(tx.Input))
		copy(inputCopy, tx.Input)
		signedTx.Input = inputCopy
	}

	// 复制 AccessList
	if tx.AccessList != nil {
		signedTx.AccessList = tx.AccessList
	}

	// 使用内部签名方法
	return s.signTransactionInternal(signedTx, func(hash []byte) ([]byte, error) {
		return s.Sign(hash)
	})
}

// signTransactionInternal 内部签名逻辑，处理签名应用
func (s *MPCKMSSigner) signTransactionInternal(tx *ethgo.Transaction, signFunc func([]byte) ([]byte, error)) (*ethgo.Transaction, error) {
	hash, err := s.signHash(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to compute transaction hash: %w", err)
	}

	signature, err := signFunc(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	if len(signature) != 65 {
		return nil, fmt.Errorf("invalid signature length: expected 65, got %d", len(signature))
	}

	tx.R = s.trimBytesZeros(signature[0:32])
	tx.S = s.trimBytesZeros(signature[32:64])

	// 使用 big.Int 计算 V 值，防止 chainID 增长导致的溢出
	vBigInt := new(big.Int).SetUint64(uint64(signature[64]))

	if tx.Type == ethgo.TransactionLegacy {
		// Legacy 交易: v = signature_v + 35 + chainID * 2
		vBigInt.Add(vBigInt, big.NewInt(35))
		if s.chainID != nil {
			chainIDBigInt := new(big.Int).Mul(s.chainID, big.NewInt(2))
			vBigInt.Add(vBigInt, chainIDBigInt)
		}
	}

	tx.V = vBigInt.Bytes()

	return tx, nil
}

// signHash 计算交易的签名哈希
func (s *MPCKMSSigner) signHash(tx *ethgo.Transaction) ([]byte, error) {
	a := fastrlp.DefaultArenaPool.Get()
	defer fastrlp.DefaultArenaPool.Put(a)

	v := a.NewArray()

	if tx.Type != ethgo.TransactionLegacy {
		v.Set(a.NewBigInt(s.chainID))
	}

	v.Set(a.NewUint(tx.Nonce))

	if tx.Type == ethgo.TransactionDynamicFee {
		v.Set(a.NewBigInt(tx.MaxPriorityFeePerGas))
		v.Set(a.NewBigInt(tx.MaxFeePerGas))
	} else {
		v.Set(a.NewUint(tx.GasPrice))
	}

	v.Set(a.NewUint(tx.Gas))
	if tx.To == nil {
		v.Set(a.NewNull())
	} else {
		v.Set(a.NewCopyBytes((*tx.To)[:]))
	}
	v.Set(a.NewBigInt(tx.Value))
	v.Set(a.NewCopyBytes(tx.Input))

	if tx.Type != ethgo.TransactionLegacy {
		accessList, err := tx.AccessList.MarshalRLPWith(a)
		if err != nil {
			return nil, err
		}
		v.Set(accessList)
	}

	if s.chainID != nil && s.chainID.Uint64() != 0 && tx.Type == ethgo.TransactionLegacy {
		v.Set(a.NewUint(s.chainID.Uint64()))
		v.Set(a.NewUint(0))
		v.Set(a.NewUint(0))
	}

	dst := v.MarshalTo(nil)

	if tx.Type != ethgo.TransactionLegacy {
		dst = append([]byte{byte(tx.Type)}, dst...)
	}

	return ethgo.Keccak256(dst), nil
}

// trimBytesZeros 移除字节切片的前导零
func (s *MPCKMSSigner) trimBytesZeros(b []byte) []byte {
	var i int
	for i = 0; i < len(b); i++ {
		if b[i] != 0x0 {
			break
		}
	}
	if i == len(b) {
		return []byte{0}
	}
	return b[i:]
}

// SignTransactionWithSummary signs an Ethereum transaction with approval summary.
//
// This method signs a transaction and includes a summary for KMS approval workflow.
// The summary is displayed to approvers showing transaction details.
//
// Parameters:
//   - tx: The transaction to sign
//   - summary: Transaction summary for approval display (from, to, amount, token)
//
// Returns:
//   - *ethgo.Transaction: A new transaction with signature applied
//   - error: An error if signing fails
func (s *MPCKMSSigner) SignTransactionWithSummary(tx *ethgo.Transaction, summary *kms.SignSummary) (*ethgo.Transaction, error) {
	txCopy := tx.Copy()
	txCopy.From = s.address

	// 使用内部签名方法
	return s.signTransactionInternal(txCopy, func(hash []byte) ([]byte, error) {
		signatureHex, err := s.client.SignWithOptions(
			context.Background(),
			s.keyID,
			hash,
			kms.DataEncodingHex,
			summary,
			"",
		)
		if err != nil {
			return nil, err
		}
		return hex.DecodeString(string(signatureHex))
	})
}

// CreateTransferSummary creates a transfer summary from transaction details.
//
// This method extracts relevant transaction information for approval display.
//
// Parameters:
//   - tx: The transaction to extract details from
//   - token: The token symbol (e.g., "ETH", "USDT"). Defaults to "ETH" if empty.
//   - remark: Optional transaction description/remark
//
// Returns:
//   - *kms.SignSummary: A transaction summary for KMS approval workflow
func (s *MPCKMSSigner) CreateTransferSummary(tx *ethgo.Transaction, token string, remark string) *kms.SignSummary {
	from := s.address.String()

	var to string
	if tx.To != nil {
		to = tx.To.String()
	} else {
		to = ""
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
