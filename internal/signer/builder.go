package signer

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/umbracle/ethgo"
)

// TransactionBuilder 交易构建器
type TransactionBuilder struct{}

// NewTransactionBuilder 创建新的交易构建器
func NewTransactionBuilder() *TransactionBuilder {
	return &TransactionBuilder{}
}

// TransactionParams 表示 JSON-RPC 交易参数
type TransactionParams struct {
	From                 string           `json:"from"`
	To                   string           `json:"to,omitempty"`
	Gas                  string           `json:"gas"`
	GasPrice             string           `json:"gasPrice,omitempty"`
	MaxFeePerGas         string           `json:"maxFeePerGas,omitempty"`
	MaxPriorityFeePerGas string           `json:"maxPriorityFeePerGas,omitempty"`
	Value                string           `json:"value,omitempty"`
	Data                 string           `json:"data,omitempty"`
	Nonce                string           `json:"nonce,omitempty"`
	ChainID              string           `json:"chainId,omitempty"`
	AccessList           []AccessListItem `json:"accessList,omitempty"`
}

// AccessListItem 表示 EIP-2930 访问列表项
type AccessListItem struct {
	Address     string   `json:"address"`
	StorageKeys []string `json:"storageKeys"`
}

// BuildTransaction 从 JSON-RPC 参数构建交易
func (tb *TransactionBuilder) BuildTransaction(params TransactionParams) (*ethgo.Transaction, error) {
	// 解析基本字段
	gas, err := parseUint64(params.Gas)
	if err != nil {
		return nil, fmt.Errorf("invalid gas: %v", err)
	}

	value := new(big.Int)
	if params.Value != "" {
		if _, ok := value.SetString(params.Value, 0); !ok {
			return nil, fmt.Errorf("invalid value: %s", params.Value)
		}
	}

	var nonce uint64
	if params.Nonce != "" {
		nonce, err = parseUint64(params.Nonce)
		if err != nil {
			return nil, fmt.Errorf("invalid nonce: %v", err)
		}
	}

	// 解析数据字段
	var data []byte
	if params.Data != "" {
		data, err = parseHex(params.Data)
		if err != nil {
			return nil, fmt.Errorf("invalid data: %v", err)
		}
	}

	// 根据参数确定交易类型
	switch {
	case params.MaxFeePerGas != "" || params.MaxPriorityFeePerGas != "":
		// EIP-1559 交易
		return tb.buildEIP1559Transaction(params, gas, value, nonce, data)
	case len(params.AccessList) > 0:
		// EIP-2930 交易
		return tb.buildEIP2930Transaction(params, gas, value, nonce, data)
	default:
		// Legacy 交易
		return tb.buildLegacyTransaction(params, gas, value, nonce, data)
	}
}

// buildLegacyTransaction 构建 Legacy 交易
func (tb *TransactionBuilder) buildLegacyTransaction(params TransactionParams, gas uint64, value *big.Int, nonce uint64, data []byte) (*ethgo.Transaction, error) {
	gasPrice := uint64(0)
	if params.GasPrice != "" {
		price, err := parseUint64(params.GasPrice)
		if err != nil {
			return nil, fmt.Errorf("invalid gasPrice: %v", err)
		}
		gasPrice = price
	}

	to := ethgo.HexToAddress("")
	if params.To != "" {
		to = ethgo.HexToAddress(params.To)
	}

	return &ethgo.Transaction{
		From:     ethgo.Address{}, // 将由签名器填充
		To:       &to,
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      gas,
		Value:    value,
		Input:    data,
	}, nil
}

// buildEIP2930Transaction 构建 EIP-2930 交易
func (tb *TransactionBuilder) buildEIP2930Transaction(params TransactionParams, gas uint64, value *big.Int, nonce uint64, data []byte) (*ethgo.Transaction, error) {
	gasPrice := uint64(0)
	if params.GasPrice != "" {
		price, err := parseUint64(params.GasPrice)
		if err != nil {
			return nil, fmt.Errorf("invalid gasPrice: %v", err)
		}
		gasPrice = price
	}

	chainID := new(big.Int)
	if params.ChainID != "" {
		if _, ok := chainID.SetString(params.ChainID, 0); !ok {
			return nil, fmt.Errorf("invalid chainId: %s", params.ChainID)
		}
	}

	to := ethgo.HexToAddress("")
	if params.To != "" {
		to = ethgo.HexToAddress(params.To)
	}

	accessList := tb.convertAccessList(params.AccessList)

	return &ethgo.Transaction{
		From:       ethgo.Address{}, // 将由签名器填充
		To:         &to,
		Nonce:      nonce,
		GasPrice:   gasPrice,
		Gas:        gas,
		Value:      value,
		Input:      data,
		ChainID:    chainID,
		AccessList: accessList,
	}, nil
}

// buildEIP1559Transaction 构建 EIP-1559 交易
func (tb *TransactionBuilder) buildEIP1559Transaction(params TransactionParams, gas uint64, value *big.Int, nonce uint64, data []byte) (*ethgo.Transaction, error) {
	maxFeePerGas := new(big.Int)
	if params.MaxFeePerGas != "" {
		if _, ok := maxFeePerGas.SetString(params.MaxFeePerGas, 0); !ok {
			return nil, fmt.Errorf("invalid maxFeePerGas: %s", params.MaxFeePerGas)
		}
	}

	maxPriorityFeePerGas := new(big.Int)
	if params.MaxPriorityFeePerGas != "" {
		if _, ok := maxPriorityFeePerGas.SetString(params.MaxPriorityFeePerGas, 0); !ok {
			return nil, fmt.Errorf("invalid maxPriorityFeePerGas: %s", params.MaxPriorityFeePerGas)
		}
	}

	chainID := new(big.Int)
	if params.ChainID != "" {
		if _, ok := chainID.SetString(params.ChainID, 0); !ok {
			return nil, fmt.Errorf("invalid chainId: %s", params.ChainID)
		}
	}

	to := ethgo.HexToAddress("")
	if params.To != "" {
		to = ethgo.HexToAddress(params.To)
	}

	accessList := tb.convertAccessList(params.AccessList)

	return &ethgo.Transaction{
		From:                 ethgo.Address{}, // 将由签名器填充
		To:                   &to,
		Nonce:                nonce,
		Gas:                  gas,
		Value:                value,
		Input:                data,
		ChainID:              chainID,
		MaxFeePerGas:         maxFeePerGas,
		MaxPriorityFeePerGas: maxPriorityFeePerGas,
		AccessList:           accessList,
	}, nil
}

// convertAccessList 转换访问列表格式
func (tb *TransactionBuilder) convertAccessList(accessList []AccessListItem) ethgo.AccessList {
	if len(accessList) == 0 {
		return nil
	}

	result := make(ethgo.AccessList, 0, len(accessList))
	for _, item := range accessList {
		addr := ethgo.HexToAddress(item.Address)

		storageKeys := make([]ethgo.Hash, 0, len(item.StorageKeys))
		for _, key := range item.StorageKeys {
			hash := ethgo.HexToHash(key)
			storageKeys = append(storageKeys, hash)
		}

		result = append(result, ethgo.AccessEntry{
			Address: addr,
			Storage: storageKeys,
		})
	}

	return result
}

// ParseTransactionParams 从 JSON-RPC 参数解析交易参数
func ParseTransactionParams(params json.RawMessage) (*TransactionParams, error) {
	var txParams TransactionParams

	// 参数可能是数组格式 [params] 或直接是对象
	var rawParams json.RawMessage
	if err := json.Unmarshal(params, &rawParams); err == nil {
		// 是数组格式，取第一个元素
		var paramsArray []json.RawMessage
		if err := json.Unmarshal(params, &paramsArray); err == nil && len(paramsArray) > 0 {
			rawParams = paramsArray[0]
		} else {
			rawParams = params
		}
	} else {
		rawParams = params
	}

	if err := json.Unmarshal(rawParams, &txParams); err != nil {
		return nil, fmt.Errorf("failed to parse transaction params: %v", err)
	}

	return &txParams, nil
}

// ParseSignParams 从 JSON-RPC 参数解析签名参数
func ParseSignParams(params json.RawMessage) (address string, data []byte, err error) {
	var paramsArray []interface{}
	if err := json.Unmarshal(params, &paramsArray); err != nil {
		return "", nil, fmt.Errorf("failed to parse sign params: %v", err)
	}

	if len(paramsArray) < 2 {
		return "", nil, fmt.Errorf("insufficient parameters for eth_sign")
	}

	// 第一个参数是地址
	address, ok := paramsArray[0].(string)
	if !ok {
		return "", nil, fmt.Errorf("invalid address parameter")
	}

	// 第二个参数是要签名的数据
	dataStr, ok := paramsArray[1].(string)
	if !ok {
		return "", nil, fmt.Errorf("invalid data parameter")
	}

	data, err = parseHex(dataStr)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse data: %v", err)
	}

	if len(data) != 32 {
		return "", nil, fmt.Errorf("invalid data length: expected 32 bytes, got %d", len(data))
	}

	return address, data, nil
}

// 辅助函数

func parseUint64(s string) (uint64, error) {
	if s == "" {
		return 0, nil
	}

	i := new(big.Int)
	if _, ok := i.SetString(s, 0); !ok {
		return 0, fmt.Errorf("invalid number: %s", s)
	}

	if !i.IsUint64() {
		return 0, fmt.Errorf("number too large: %s", s)
	}

	return i.Uint64(), nil
}

func parseHex(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}

	// 移除 0x 前缀
	if len(s) >= 2 && s[0:2] == "0x" {
		s = s[2:]
	}

	// 如果长度为奇数，在前面补 0
	if len(s)%2 != 0 {
		s = "0" + s
	}

	return hex.DecodeString(s)
}
