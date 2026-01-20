[根目录](../../CLAUDE.md) > [internal](../) > **signer (签名逻辑)**

---

# internal/signer - 签名逻辑模块

> **最后更新**: 2026-01-20 11:07:09
> **模块状态**: 🟢 完成
> **测试覆盖**: ✅ 完整

---

## 模块职责

签名器模块负责：

1. **实现 ethgo.Key 接口** - 让 MPC-KMS 签名器可以作为 ethgo 的密钥使用
2. **交易签名** - 支持所有以太坊交易类型（Legacy, EIP-2930, EIP-1559）
3. **交易哈希计算** - 正确计算签名哈希
4. **签名结果组装** - 将 KMS 返回的签名组装到交易中

---

## 入口与启动

### 文件结构

```
internal/signer/
├── signer.go          # 签名器实现
├── builder.go         # 签名器构建器
├── signer_test.go     # 签名器测试
└── builder_test.go    # 构建器测试
```

### 创建签名器

```go
import (
    "github.com/mowind/web3signer-go/internal/signer"
    "github.com/mowind/web3signer-go/internal/kms"
    "github.com/umbracle/ethgo"
)

// 使用构建器创建
signer := signer.NewBuilder().
    WithClient(kmsClient).
    WithKeyID("key-123").
    WithAddress(ethgo.Address(...)).
    WithChainID(big.NewInt(1)).
    Build()
```

---

## 对外接口

### MPCKMSSigner 结构

```go
type MPCKMSSigner struct {
    client  kms.ClientInterface // KMS 客户端
    keyID   string              // 密钥 ID
    address ethgo.Address        // 以太坊地址
    chainID *big.Int             // 链 ID
}
```

### ethgo.Key 接口实现

```go
// Address 返回签名器地址
func (s *MPCKMSSigner) Address() ethgo.Address

// Sign 对哈希进行签名（实现 ethgo.Key 接口）
func (s *MPCKMSSigner) Sign(hash []byte) ([]byte, error)

// SignTransaction 对交易进行签名
func (s *MPCKMSSigner) SignTransaction(tx *ethgo.Transaction) (*ethgo.Transaction, error)
```

### 扩展方法

```go
// SignTransactionWithSummary 对交易进行签名，并包含交易摘要
func (s *MPCKMSSigner) SignTransactionWithSummary(
    tx *ethgo.Transaction,
    summary *kms.SignSummary,
) (*ethgo.Transaction, error)

// CreateTransferSummary 从交易创建转账摘要
func (s *MPCKMSSigner) CreateTransferSummary(
    tx *ethgo.Transaction,
    token string,
    remark string,
) *kms.SignSummary
```

---

## 关键依赖与配置

### 核心依赖

```go
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
```

### 配置需求

```go
type SignerConfig struct {
    Client  kms.ClientInterface // KMS 客户端（必需）
    KeyID   string              // 密钥 ID（必需）
    Address ethgo.Address        // 以太坊地址（必需）
    ChainID *big.Int             // 链 ID（必需）
}
```

---

## 数据模型

### 交易类型支持

```go
// Legacy 交易 (Type 0)
type TransactionLegacy TransactionType = 0

// EIP-2930 交易 (Type 1)
type TransactionAccessList TransactionType = 1

// EIP-1559 交易 (Type 2)
type TransactionDynamicFee TransactionType = 2
```

### 签名过程

```go
// 1. 复制交易（避免修改原始交易）
txCopy := tx.Copy()

// 2. 计算签名哈希
hash := s.signHash(txCopy)

// 3. 调用 KMS 签名
signature, err := s.Sign(hash)

// 4. 解析签名结果 (R, S, V)
r := signature[0:32]
s := signature[32:64]
v := signature[64]

// 5. 调整 V 值（Legacy 交易需要）
if tx.Type == TransactionLegacy {
    v = v + 35 + chainID * 2
}

// 6. 组装签名后的交易
txCopy.R = r
txCopy.S = s
txCopy.V = v
```

---

## 测试与质量

### 测试文件

- `signer_test.go` - 签名器功能测试
- `builder_test.go` - 构建器测试

### 测试覆盖

- ✅ 地址获取
- ✅ 哈希签名
- ✅ Legacy 交易签名
- ✅ EIP-1559 交易签名
- ✅ 交易摘要创建

### 代码质量

- **职责单一**: ✅ 只负责签名逻辑
- **接口兼容**: ✅ 完全实现 ethgo.Key 接口
- **日志记录**: ⚠️ 日志过多（见 FAQ）

---

## 常见问题 (FAQ)

### Q: 为什么 `SignTransaction` 要复制交易？

A: 为了避免修改原始交易对象。ethgo 的 Transaction 包含指针字段，直接修改会影响原始数据。

### Q: V 值的计算逻辑是什么？

A: 不同交易类型的 V 值不同：

- **Legacy 交易**: `v = signature_v + 35 + chainID * 2`
- **EIP-2930 / EIP-1559**: `v = signature_v`（直接使用）

这是 Ethereum 的签名标准（EIP-155）。

### Q: 为什么日志这么多？

A: 这是个已知问题（在代码审查中已指出）。`SignTransaction` 中有很多调试日志：

```go
logrus.WithFields(logrus.Fields{
    "original_nonce":    tx.Nonce,
    "original_gas":      tx.Gas,
    // ...
}).Info("Original transaction before signing")
```

**改进建议**：将这些日志改为 `Debug` 级别或直接删除。

### Q: 如何支持新的交易类型？

A: 当前已支持所有标准交易类型。如果需要支持新的 EIP，主要修改 `signHash` 方法：

```go
func (s *MPCKMSSigner) signHash(tx *ethgo.Transaction) []byte {
    // 在这里添加新交易类型的哈希计算逻辑
}
```

### Q: `trimBytesZeros` 是做什么的？

A: 移除字节切片的前导零。这是因为 KMS 返回的签名是固定长度（65 字节），但 RLP 编码需要去除前导零。

**注意**：这是一个特殊情况处理，理想情况下应该在上游数据源就正确处理。

---

## 相关文件清单

### 核心文件

- `signer.go` (356 行) - 签名器实现
- `builder.go` (80 行) - 签名器构建器

### 测试文件

- `signer_test.go` - 签名器测试
- `builder_test.go` - 构建器测试

### 依赖模块

- `internal/kms` - KMS 客户端
- `github.com/umbracle/ethgo` - 以太坊工具库

---

## 代码审查要点

### ✅ 好的设计

1. **接口实现** - 完全实现 `ethgo.Key` 接口，可替换
2. **数据安全** - 复制交易避免修改原始数据
3. **类型支持** - 支持所有标准交易类型

### ⚠️ 需要改进

1. **日志过多** - `SignTransaction` 中有大量 `Info` 级别日志
   - **建议**: 改为 `Debug` 级别或删除

2. **特殊情况处理** - `trimBytesZeros` 是对上游数据的补丁
   - **建议**: 在 KMS 客户端层面处理

3. **函数长度** - `SignTransaction` 有 150+ 行
   - **建议**: 可以拆分为更小的函数

### 🔴 潜在问题

- **V 值计算** - Legacy 交易的 V 值计算可能溢出
  - **当前**: 使用 `uint64` 计算
  - **建议**: 使用 `big.Int` 避免溢出

---

## 使用示例

### 基本签名

```go
// 创建签名器
signer := signer.NewMPCKMSSigner(
    kmsClient,
    "key-123",
    ethgo.HexToAddress("0x..."),
    big.NewInt(1), // Mainnet
)

// 签名交易
signedTx, err := signer.SignTransaction(tx)
if err != nil {
    log.Fatal(err)
}

// 获取原始交易
rlp, err := ethgo.RLPMarshalToBytes(signedTx)
```

### 带摘要的签名

```go
// 创建摘要
summary := signer.CreateTransferSummary(
    tx,
    "USDT", // 代币符号
    "Payment", // 备注
)

// 签名（带摘要）
signedTx, err := signer.SignTransactionWithSummary(tx, summary)
```

---

## 变更记录 (Changelog)

### 2026-01-20
- 初始化模块文档
- 完成代码审查
- 识别日志过多问题

---

**文档版本**: 1.0.0
**维护者**: mowind
