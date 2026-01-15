# ethgo 库研究文档

## 概述

ethgo 是一个轻量级的 Go 语言 SDK，用于与以太坊兼容的区块链交互。它提供了完整的以太坊交易处理、签名、序列化和 JSON-RPC 集成功能。

**核心特性：**
- 轻量级设计，依赖少
- 支持所有以太坊交易类型（Legacy, EIP-2930, EIP-1559）
- 完整的签名和密钥管理
- 内置 JSON-RPC 客户端
- 提供 CLI 工具和 Go 库两种形式

## 1. 接口规范

### 核心数据结构

#### Transaction 结构体
```go
type Transaction struct {
    Type                 TransactionType  // 交易类型：0=Legacy, 1=EIP-2930, 2=EIP-1559
    Hash                 Hash             // 交易哈希
    From                 Address          // 发送方地址
    To                   *Address         // 接收方地址（nil 表示合约创建）
    Input                []byte           // 交易数据
    GasPrice             uint64           // Gas 价格（Legacy 交易）
    Gas                  uint64           // Gas 限制
    Value                *big.Int         // 转账金额
    Nonce                uint64           // 随机数
    V, R, S              []byte           // 签名分量
    BlockHash            Hash             // 区块哈希
    BlockNumber          uint64           // 区块号
    TxnIndex             uint64           // 交易索引
    ChainID              *big.Int         // 链 ID
    AccessList           AccessList       // EIP-2930 访问列表
    MaxPriorityFeePerGas *big.Int         // EIP-1559 优先费
    MaxFeePerGas         *big.Int         // EIP-1559 最大费
}
```

#### 交易类型常量
```go
const (
    TransactionLegacy    TransactionType = 0  // 传统交易
    TransactionAccessList TransactionType = 1 // EIP-2930 访问列表交易
    TransactionDynamicFee TransactionType = 2 // EIP-1559 动态费用交易
)
```

#### Key 接口（签名）
```go
type Key interface {
    Address() Address                    // 获取地址
    Sign(hash []byte) ([]byte, error)    // 签名哈希
}
```

#### Signer 接口（交易签名器）
```go
type Signer interface {
    RecoverSender(tx *Transaction) (Address, error)  // 从签名恢复发送方
    SignTx(tx *Transaction, key Key) error           // 签名交易
}
```

## 2. 基础使用

### 安装
```bash
go get github.com/umbracle/ethgo
```

### 初始化 JSON-RPC 客户端
```go
import (
    "github.com/umbracle/ethgo/jsonrpc"
)

func main() {
    // 连接到以太坊节点
    client, err := jsonrpc.NewClient("https://mainnet.infura.io/v3/YOUR-PROJECT-ID")
    if err != nil {
        panic(err)
    }

    // 获取以太坊客户端接口
    eth := client.Eth()
}
```

### 构建基本交易
```go
import (
    "github.com/umbracle/ethgo"
    "math/big"
)

func createLegacyTransaction() *ethgo.Transaction {
    to := ethgo.HexToAddress("0x742d35Cc6634C0532925a3b844Bc9e90F1A6B5A3")

    return &ethgo.Transaction{
        Type:     ethgo.TransactionLegacy,
        To:       &to,
        Value:    big.NewInt(1000000000000000000), // 1 ETH
        Gas:      21000,
        GasPrice: 20000000000, // 20 Gwei
        Nonce:    1,
        ChainID:  big.NewInt(1), // 主网
    }
}
```

## 3. 进阶技巧

### 构建 EIP-1559 动态费用交易
```go
func createEIP1559Transaction() *ethgo.Transaction {
    to := ethgo.HexToAddress("0x742d35Cc6634C0532925a3b844Bc9e90F1A6B5A3")

    return &ethgo.Transaction{
        Type:                 ethgo.TransactionDynamicFee,
        To:                   &to,
        Value:                big.NewInt(1000000000000000000),
        Gas:                  21000,
        MaxPriorityFeePerGas: big.NewInt(2000000000),  // 2 Gwei
        MaxFeePerGas:         big.NewInt(30000000000), // 30 Gwei
        Nonce:                1,
        ChainID:              big.NewInt(1),
    }
}
```

### 序列化和反序列化交易
```go
import (
    "encoding/hex"
    "github.com/umbracle/ethgo"
)

// 序列化交易为 RLP
func serializeTransaction(tx *ethgo.Transaction) ([]byte, error) {
    var buf []byte
    err := tx.MarshalRLPTo(&buf)
    return buf, err
}

// 反序列化 RLP 为交易
func deserializeTransaction(data []byte) (*ethgo.Transaction, error) {
    tx := &ethgo.Transaction{}
    err := tx.UnmarshalRLP(data)
    return tx, err
}

// 序列化为十六进制字符串（用于发送原始交易）
func toRawTransaction(tx *ethgo.Transaction) (string, error) {
    data, err := serializeTransaction(tx)
    if err != nil {
        return "", err
    }
    return "0x" + hex.EncodeToString(data), nil
}
```

### 计算交易哈希
```go
func getTransactionHash(tx *ethgo.Transaction) ethgo.Hash {
    return tx.GetHash()
}
```

## 4. 签名功能

### 创建密钥和签名
```go
import (
    "github.com/umbracle/ethgo/wallet"
    "github.com/umbracle/ethgo"
)

func signTransaction() error {
    // 生成新密钥
    key, err := wallet.GenerateKey()
    if err != nil {
        return err
    }

    // 获取地址
    address := key.Address()
    fmt.Printf("Address: %s\n", address)

    // 创建交易
    tx := createLegacyTransaction()
    tx.From = address

    // 创建签名器
    signer := wallet.NewEIP155Signer(big.NewInt(1)) // 主网

    // 签名交易
    err = signer.SignTx(tx, key)
    if err != nil {
        return err
    }

    return nil
}
```

### 从私钥恢复密钥
```go
func keyFromPrivateKey(privateKeyHex string) (ethgo.Key, error) {
    // 从十六进制私钥创建密钥
    privateKeyBytes, err := hex.DecodeString(privateKeyHex)
    if err != nil {
        return nil, err
    }

    key, err := wallet.NewWalletFromPrivKey(privateKeyBytes)
    if err != nil {
        return nil, err
    }

    return key, nil
}
```

### 验证签名和恢复发送方
```go
func verifyAndRecover(tx *ethgo.Transaction) (ethgo.Address, error) {
    signer := wallet.NewEIP155Signer(tx.ChainID)

    // 恢复发送方地址
    sender, err := signer.RecoverSender(tx)
    if err != nil {
        return ethgo.Address{}, err
    }

    fmt.Printf("Recovered sender: %s\n", sender)
    return sender, nil
}
```

## 5. JSON-RPC 集成

### 发送原始交易
```go
func sendRawTransaction(client *jsonrpc.Client, tx *ethgo.Transaction) (ethgo.Hash, error) {
    // 序列化交易
    rawTx, err := toRawTransaction(tx)
    if err != nil {
        return ethgo.Hash{}, err
    }

    // 发送原始交易
    var txHash ethgo.Hash
    err = client.Call("eth_sendRawTransaction", &txHash, rawTx)
    if err != nil {
        return ethgo.Hash{}, err
    }

    return txHash, nil
}
```

### 查询交易状态
```go
func getTransactionReceipt(client *jsonrpc.Client, txHash ethgo.Hash) (*ethgo.Receipt, error) {
    var receipt ethgo.Receipt
    err := client.Call("eth_getTransactionReceipt", &receipt, txHash)
    if err != nil {
        return nil, err
    }

    return &receipt, nil
}
```

### 获取账户余额
```go
func getBalance(client *jsonrpc.Client, address ethgo.Address) (*big.Int, error) {
    var balanceStr string
    err := client.Call("eth_getBalance", &balanceStr, address, "latest")
    if err != nil {
        return nil, err
    }

    balance, ok := new(big.Int).SetString(balanceStr[2:], 16) // 去掉 0x 前缀
    if !ok {
        return nil, fmt.Errorf("failed to parse balance")
    }

    return balance, nil
}
```

## 6. 巧妙用法

### 批量交易处理
```go
func processBatchTransactions(client *jsonrpc.Client, txs []*ethgo.Transaction) ([]ethgo.Hash, error) {
    var txHashes []ethgo.Hash

    for _, tx := range txs {
        txHash, err := sendRawTransaction(client, tx)
        if err != nil {
            // 可以选择继续处理其他交易或返回错误
            fmt.Printf("Failed to send transaction: %v\n", err)
            continue
        }
        txHashes = append(txHashes, txHash)
    }

    return txHashes, nil
}
```

### 交易监控
```go
func monitorTransaction(client *jsonrpc.Client, txHash ethgo.Hash, maxAttempts int) (*ethgo.Receipt, error) {
    for i := 0; i < maxAttempts; i++ {
        receipt, err := getTransactionReceipt(client, txHash)
        if err != nil {
            return nil, err
        }

        if receipt != nil && receipt.BlockHash != (ethgo.Hash{}) {
            // 交易已确认
            return receipt, nil
        }

        // 等待一段时间再重试
        time.Sleep(5 * time.Second)
    }

    return nil, fmt.Errorf("transaction not confirmed after %d attempts", maxAttempts)
}
```

### 智能合约交互
```go
import (
    "github.com/umbracle/ethgo/abi"
    "github.com/umbracle/ethgo/contract"
)

func interactWithContract(client *jsonrpc.Client, contractAddress ethgo.Address) error {
    // 定义 ABI（简化示例）
    abiDef := `[{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"type":"function"}]`

    // 创建合约实例
    c := contract.NewContract(
        contractAddress,
        abi.MustNewABI(abiDef),
        contract.WithJsonRPC(client.Eth()),
    )

    // 调用合约方法
    res, err := c.Call("totalSupply", ethgo.Latest)
    if err != nil {
        return err
    }

    // 处理返回值
    totalSupply := res["totalSupply"].(*big.Int)
    fmt.Printf("Total supply: %s\n", totalSupply.String())

    return nil
}
```

## 7. 注意事项

### 常见错误和避免方法

1. **Gas 设置不足**
   ```go
   // 错误：Gas 设置过低
   tx.Gas = 21000 // 对于复杂合约调用可能不足

   // 正确：使用 eth_estimateGas 估算
   var gasEstimate string
   client.Call("eth_estimateGas", &gasEstimate, map[string]interface{}{
       "from": tx.From,
       "to":   tx.To,
       "data": tx.Input,
   })
   ```

2. **ChainID 不匹配**
   ```go
   // 错误：签名器 ChainID 与交易 ChainID 不匹配
   signer := wallet.NewEIP155Signer(big.NewInt(1))
   tx.ChainID = big.NewInt(5) // Goerli 测试网

   // 正确：确保 ChainID 一致
   signer := wallet.NewEIP155Signer(tx.ChainID)
   ```

3. **Nonce 管理**
   ```go
   // 错误：手动管理 Nonce 可能导致冲突
   tx.Nonce = lastNonce + 1 // 可能与其他交易冲突

   // 正确：从节点获取当前 Nonce
   var nonceStr string
   client.Call("eth_getTransactionCount", &nonceStr, address, "pending")
   nonce, _ := strconv.ParseUint(nonceStr[2:], 16, 64)
   tx.Nonce = nonce
   ```

### 性能优化建议

1. **连接池管理**
   - 重用 JSON-RPC 客户端连接
   - 设置合理的超时时间
   - 使用连接池处理高并发请求

2. **批量请求**
   ```go
   // 使用批处理减少 RPC 调用
   batch := client.NewBatch()
   var results []interface{}

   for i := 0; i < 10; i++ {
       var result string
       batch.AddCall("eth_blockNumber", &result)
       results = append(results, &result)
   }

   if err := batch.Execute(); err != nil {
       // 处理错误
   }
   ```

3. **缓存常用数据**
   - 缓存 ChainID、Gas 价格等不常变化的数据
   - 实现本地 Nonce 管理减少 RPC 调用

### 安全注意事项

1. **私钥管理**
   - 永远不要在代码中硬编码私钥
   - 使用环境变量或安全的密钥管理系统
   - 考虑使用硬件安全模块（HSM）或 MPC-KMS

2. **交易验证**
   - 验证所有输入参数
   - 检查地址格式和有效性
   - 验证金额不超过账户余额

3. **错误处理**
   - 不要泄露敏感信息在错误消息中
   - 实现适当的重试逻辑
   - 记录所有交易操作用于审计

## 8. 真实代码片段

### 从 GitHub 项目提取的优秀实践

以下是从 ethgo 示例和其他项目中提取的实用代码模式：

```go
// 完整的交易构建、签名和发送流程
func sendETHTransaction(privateKeyHex, toAddress string, amount *big.Int) (ethgo.Hash, error) {
    // 1. 从私钥创建密钥
    key, err := keyFromPrivateKey(privateKeyHex)
    if err != nil {
        return ethgo.Hash{}, fmt.Errorf("failed to create key: %v", err)
    }

    // 2. 连接到节点
    client, err := jsonrpc.NewClient("https://mainnet.infura.io/v3/YOUR-PROJECT-ID")
    if err != nil {
        return ethgo.Hash{}, fmt.Errorf("failed to connect: %v", err)
    }

    // 3. 获取当前 Nonce
    var nonceStr string
    err = client.Call("eth_getTransactionCount", &nonceStr, key.Address(), "pending")
    if err != nil {
        return ethgo.Hash{}, fmt.Errorf("failed to get nonce: %v", err)
    }
    nonce, _ := strconv.ParseUint(nonceStr[2:], 16, 64)

    // 4. 获取当前 Gas 价格
    var gasPriceStr string
    err = client.Call("eth_gasPrice", &gasPriceStr)
    if err != nil {
        return ethgo.Hash{}, fmt.Errorf("failed to get gas price: %v", err)
    }
    gasPrice, _ := new(big.Int).SetString(gasPriceStr[2:], 16)

    // 5. 构建交易
    to := ethgo.HexToAddress(toAddress)
    tx := &ethgo.Transaction{
        Type:     ethgo.TransactionLegacy,
        To:       &to,
        Value:    amount,
        Gas:      21000,
        GasPrice: gasPrice.Uint64(),
        Nonce:    nonce,
        ChainID:  big.NewInt(1),
        From:     key.Address(),
    }

    // 6. 签名交易
    signer := wallet.NewEIP155Signer(big.NewInt(1))
    if err := signer.SignTx(tx, key); err != nil {
        return ethgo.Hash{}, fmt.Errorf("failed to sign: %v", err)
    }

    // 7. 发送交易
    txHash, err := sendRawTransaction(client, tx)
    if err != nil {
        return ethgo.Hash{}, fmt.Errorf("failed to send: %v", err)
    }

    fmt.Printf("Transaction sent: %s\n", txHash)
    return txHash, nil
}
```

**为什么这是好的实践：**
1. **完整的错误处理**：每个步骤都有明确的错误检查和描述性错误消息
2. **安全的密钥管理**：私钥作为参数传入，不在代码中硬编码
3. **实时数据获取**：从节点获取最新的 Nonce 和 Gas 价格
4. **清晰的步骤分离**：每个功能都有明确的职责
5. **适当的日志记录**：在关键步骤提供状态反馈

## 9. 引用来源

### 官方文档
- **GitHub 仓库**: https://github.com/umbracle/ethgo
- **Go 包文档**: https://pkg.go.dev/github.com/umbracle/ethgo
- **示例代码**: https://github.com/umbracle/ethgo/tree/main/examples

### 核心源代码文件
1. **数据结构定义**: `structs.go` - Transaction、Address、Hash 等核心类型
2. **序列化处理**: `structs_marshal.go`、`structs_marshal_rlp.go` - JSON 和 RLP 序列化
3. **签名实现**: `wallet/key.go`、`wallet/signer.go` - 密钥管理和交易签名
4. **JSON-RPC 客户端**: `jsonrpc/client.go` - RPC 通信层
5. **编码工具**: `encoding.go` - 编码解码辅助函数

### 社区资源
- **以太坊官方文档**: https://ethereum.org/en/developers/docs/
- **EIP 标准**:
  - EIP-155: 重放攻击保护
  - EIP-2930: 访问列表交易
  - EIP-1559: 动态费用市场

## 总结

ethgo 是一个设计良好的 Go 语言以太坊 SDK，特别适合需要精细控制交易处理的应用。它的主要优势包括：

1. **简洁的 API 设计**：直观的接口，易于理解和使用
2. **完整的交易支持**：覆盖所有以太坊交易类型
3. **强大的签名功能**：支持 EIP-155、EIP-2930、EIP-1559 签名
4. **灵活的集成方式**：既可作为库使用，也提供 CLI 工具

对于 web3signer-go 项目，ethgo 可以作为处理以太坊交易格式和序列化的基础库，但需要注意：
- ethgo 主要关注客户端功能，需要与 MPC-KMS 签名服务集成
- 需要根据项目需求封装适当的接口层
- 考虑性能和安全性的最佳实践

建议在项目初期使用 ethgo 进行原型开发，然后根据实际需求进行定制和优化。