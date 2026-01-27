package kms

import (
	"math/big"

	"github.com/umbracle/ethgo"
)

// ParseTransactionSummary 从 RLP 编码的交易数据中解析交易摘要
//
// 该函数从交易数据中提取关键字段用于 KMS 审批流程：
// - From: 发送方地址（从签名中恢复，或默认为空）
// - To: 接收方地址（合约创建时为空）
// - Amount: 交易金额（wei）
// - Token: 代币符号（默认为 "ETH"）
//
// 支持的交易类型：
// - Legacy 交易（Type 0）
// - EIP-2930 AccessList 交易（Type 1）
// - EIP-1559 DynamicFee 交易（Type 2）
//
// Parameters:
//   - transactionData: RLP 编码的交易数据
//
// Returns:
//   - *SignSummary: 交易摘要，包含 from/to/amount/token 等信息
//   - error: 解析失败时返回错误
func ParseTransactionSummary(transactionData []byte) (*SignSummary, error) {
	tx := &ethgo.Transaction{}

	// 解析 RLP 编码的交易数据
	if err := tx.UnmarshalRLP(transactionData); err != nil {
		return nil, err
	}

	// 提取 From 地址
	// 注意：RLP 解析后的交易不包含 From 地址，需要在签名后从签名中恢复
	// 这里暂时留空，由调用方填充或保持为空
	from := ""

	// 提取 To 地址
	var to string
	if tx.To != nil {
		to = tx.To.String()
	}
	// 合约创建时，tx.To 为 nil，to 留空

	// 提取 Amount（Value）
	amount := "0"
	if tx.Value != nil && tx.Value.Cmp(big.NewInt(0)) != 0 {
		amount = tx.Value.String()
	}

	// 默认使用 ETH 作为代币符号
	token := "ETH"

	return NewTransferSummary(from, to, amount, token, ""), nil
}
