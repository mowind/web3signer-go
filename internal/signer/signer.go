package signer

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/mowind/web3signer-go/internal/kms"
	"github.com/sirupsen/logrus"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/fastrlp"
)

// MPCKMSSigner 实现 ethgo.Key 接口，使用 MPC-KMS 进行签名
type MPCKMSSigner struct {
	client  kms.ClientInterface
	keyID   string
	address ethgo.Address
	chainID *big.Int
}

// NewMPCKMSSigner 创建新的 MPC-KMS 签名器
func NewMPCKMSSigner(client kms.ClientInterface, keyID string, address ethgo.Address, chainID *big.Int) *MPCKMSSigner {
	return &MPCKMSSigner{
		client:  client,
		keyID:   keyID,
		address: address,
		chainID: chainID,
	}
}

// Address 返回签名器的地址（实现 ethgo.Key 接口）
func (s *MPCKMSSigner) Address() ethgo.Address {
	return s.address
}

// Sign 对哈希进行签名（实现 ethgo.Key 接口）
func (s *MPCKMSSigner) Sign(hash []byte) ([]byte, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("invalid hash length: expected 32 bytes, got %d", len(hash))
	}

	hashHex := hex.EncodeToString(hash)

	signatureHex, err := s.client.Sign(context.Background(), s.keyID, []byte(hashHex))
	if err != nil {
		return nil, fmt.Errorf("failed to sign with MPC-KMS: %v", err)
	}

	signature, err := hex.DecodeString(string(signatureHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %v", err)
	}

	return signature, nil
}

// SignTransaction 对交易进行签名
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

	// 记录原始交易信息
	originalFields := logrus.Fields{
		"original_nonce":    tx.Nonce,
		"original_gas":      tx.Gas,
		"original_gasPrice": tx.GasPrice,
		"original_type":     tx.Type,
	}

	if tx.ChainID != nil {
		originalFields["original_chainID"] = tx.ChainID.String()
	}
	if tx.MaxFeePerGas != nil {
		originalFields["original_maxFeePerGas"] = tx.MaxFeePerGas.String()
	}
	if tx.MaxPriorityFeePerGas != nil {
		originalFields["original_maxPriorityFeePerGas"] = tx.MaxPriorityFeePerGas.String()
	}

	logrus.WithFields(originalFields).Info("Original transaction before signing")

	hash := s.signHash(signedTx)
	logrus.WithField("hash", hex.EncodeToString(hash)).Info("Transaction hash for signing")

	signature, err := s.Sign(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %v", err)
	}

	if len(signature) != 65 {
		return nil, fmt.Errorf("invalid signature length: expected 65, got %d", len(signature))
	}

	logrus.WithFields(logrus.Fields{
		"signature_length": len(signature),
		"signature_v":      signature[64],
	}).Info("Received signature from KMS")

	rTrimmed := s.trimBytesZeros(signature[0:32])
	sTrimmed := s.trimBytesZeros(signature[32:64])
	signedTx.R = rTrimmed
	signedTx.S = sTrimmed

	vv := uint64(signature[64])
	chainID := uint64(0)
	if s.chainID != nil {
		chainID = s.chainID.Uint64()
	}

	logrus.WithFields(logrus.Fields{
		"signature_v_raw": vv,
		"chainID":         chainID,
		"tx_type":         signedTx.Type,
	}).Info("Signature parameters")

	if signedTx.Type == ethgo.TransactionLegacy {
		vv = vv + 35 + chainID*2
		logrus.WithField("final_v", vv).Info("Calculated final V for legacy transaction")
	}

	vBytes := new(big.Int).SetUint64(vv).Bytes()
	signedTx.V = vBytes

	// 准备日志字段
	fields := logrus.Fields{
		"tx_type":         signedTx.Type,
		"nonce":           signedTx.Nonce,
		"gas":             signedTx.Gas,
		"gasPrice":        signedTx.GasPrice,
		"signature_R":     hex.EncodeToString(signedTx.R),
		"signature_S":     hex.EncodeToString(signedTx.S),
		"signature_V":     hex.EncodeToString(signedTx.V),
		"signature_V_len": len(signedTx.V),
		"from":            signedTx.From.String(),
	}

	// 处理指针字段
	if signedTx.To != nil {
		fields["to"] = signedTx.To.String()
	} else {
		fields["to"] = nil
	}

	if signedTx.Value != nil {
		fields["value"] = signedTx.Value.String()
	} else {
		fields["value"] = "0"
	}

	if signedTx.ChainID != nil {
		fields["chainID"] = signedTx.ChainID.String()
	} else {
		fields["chainID"] = nil
	}

	if signedTx.MaxFeePerGas != nil {
		fields["maxFeePerGas"] = signedTx.MaxFeePerGas.String()
	} else {
		fields["maxFeePerGas"] = nil
	}

	if signedTx.MaxPriorityFeePerGas != nil {
		fields["maxPriorityFee"] = signedTx.MaxPriorityFeePerGas.String()
	} else {
		fields["maxPriorityFee"] = nil
	}

	logrus.WithFields(fields).Info("Signed transaction before RLP marshaling")

	return signedTx, nil
}

// signHash 计算交易的签名哈希
func (s *MPCKMSSigner) signHash(tx *ethgo.Transaction) []byte {
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
			panic(err)
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

	return ethgo.Keccak256(dst)
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

// SignTransactionWithSummary 对交易进行签名，并包含交易摘要信息
func (s *MPCKMSSigner) SignTransactionWithSummary(tx *ethgo.Transaction, summary *kms.SignSummary) (*ethgo.Transaction, error) {
	txCopy := tx.Copy()
	txCopy.From = s.address

	hash := s.signHash(txCopy)
	hashHex := hex.EncodeToString(hash)

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

	signature, err := hex.DecodeString(string(signatureHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %v", err)
	}

	if len(signature) != 65 {
		return nil, fmt.Errorf("invalid signature length: expected 65, got %d", len(signature))
	}

	txCopy.R = s.trimBytesZeros(signature[0:32])
	txCopy.S = s.trimBytesZeros(signature[32:64])

	vv := uint64(signature[64])
	chainID := uint64(0)
	if s.chainID != nil {
		chainID = s.chainID.Uint64()
	}

	if txCopy.Type == ethgo.TransactionLegacy {
		vv = vv + 35 + chainID*2
	}

	txCopy.V = new(big.Int).SetUint64(vv).Bytes()

	return txCopy, nil
}

// CreateTransferSummary 从交易创建转账摘要
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
