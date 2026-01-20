[根目录](../../CLAUDE.md) > [internal](../) > **kms (MPC-KMS 客户端)**

---

# internal/kms - MPC-KMS 客户端模块

> **最后更新**: 2026-01-20 11:07:09
> **模块状态**: 🟢 完成
> **测试覆盖**: ✅ 完整

---

## 模块职责

MPC-KMS 客户端模块负责：

1. **HTTP 签名认证** - 实现 MPC-KMS 服务的 HTTP 签名认证机制
2. **签名请求** - 调用 MPC-KMS 签名接口
3. **任务轮询** - 处理异步签名审批流程
4. **错误处理** - 解析并返回 KMS 错误信息

---

## 入口与启动

### 文件结构

```
internal/kms/
├── client.go           # 客户端实现
├── http_client.go      # HTTP 认证客户端
├── interface.go        # 接口定义
├── types.go            # 数据类型定义
├── signing.go          # 签名相关类型
├── client_test.go      # 客户端测试
└── interface_test.go   # 接口测试
```

### 创建客户端

```go
import "github.com/mowind/web3signer-go/internal/kms"

// 方式 1: 使用默认 HTTP 客户端
client := kms.NewClient(cfg)

// 方式 2: 使用自定义 HTTP 客户端（测试时常用）
mockClient := &MockHTTPClient{}
client := kms.NewClientWithHTTPClient(cfg, mockClient)
```

---

## 对外接口

### ClientInterface 接口

```go
type ClientInterface interface {
    // Sign 基础签名方法
    Sign(ctx context.Context, keyID string, message []byte) ([]byte, error)

    // SignWithOptions 支持更多选项（摘要、回调 URL）
    SignWithOptions(
        ctx context.Context,
        keyID string,
        message []byte,
        encoding DataEncoding,
        summary *SignSummary,
        callbackURL string,
    ) ([]byte, error)

    // GetTaskResult 获取异步任务结果
    GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error)

    // WaitForTaskCompletion 轮询等待任务完成
    WaitForTaskCompletion(
        ctx context.Context,
        taskID string,
        interval time.Duration,
    ) (*TaskResult, error)
}
```

### HTTPClientInterface 接口

```go
type HTTPClientInterface interface {
    Do(req *http.Request) (*http.Response, error)
}
```

---

## 关键依赖与配置

### 核心依赖

```go
import (
    "bytes"
    "context"
    "encoding/json"
    "io"
    "net/http"
    "time"

    "github.com/mowind/web3signer-go/internal/config"
)
```

### 配置需求

```go
type KMSConfig struct {
    Endpoint    string // KMS 服务端点
    AccessKeyID string // 访问密钥 ID
    SecretKey   string // 密钥
    KeyID       string // 密钥 ID
    Address     string // 以太坊地址
}
```

---

## 数据模型

### 签名请求

```go
type SignRequest struct {
    Data      string        `json:"data"`       // 待签名数据（hex 编码）
    Encoding  DataEncoding  `json:"encoding"`   // 数据编码方式
    Summary   *SignSummary  `json:"summary"`    // 交易摘要（可选）
    CallbackURL string      `json:"callback_url"` // 回调 URL（可选）
}

type DataEncoding string

const (
    DataEncodingHex DataEncoding = "hex" // 十六进制编码
    DataEncodingBase64 DataEncoding = "base64" // Base64 编码
)
```

### 交易摘要

```go
type SignSummary struct {
    Type   string `json:"type"`   // 操作类型 (transfer, contract_call, etc.)
    From   string `json:"from"`   // 发送方地址
    To     string `json:"to"`     // 接收方地址
    Amount string `json:"amount"` // 金额
    Token  string `json:"token"`  // 代币符号
    Remark string `json:"remark"` // 备注
}

// 创建转账摘要
func NewTransferSummary(from, to, amount, token, remark string) *SignSummary
```

### 签名响应

```go
// 成功响应 (HTTP 200)
type SignResponse struct {
    Signature string `json:"signature"` // 签名结果（hex 编码）
}

// 审批中响应 (HTTP 201)
type TaskResponse struct {
    TaskID string `json:"task_id"` // 任务 ID
}

// 错误响应
type ErrorResponse struct {
    Code    int    `json:"code"`    // 错误码
    Message string `json:"message"` // 错误信息
}
```

### 任务状态

```go
type TaskStatus string

const (
    TaskStatusPendingApproval TaskStatus = "pending_approval" // 待审批
    TaskStatusApproved        TaskStatus = "approved"         // 已批准
    TaskStatusDone            TaskStatus = "done"             // 完成
    TaskStatusFailed          TaskStatus = "failed"           // 失败
    TaskStatusRejected        TaskStatus = "rejected"         // 已拒绝
)

type TaskResult struct {
    Status   TaskStatus `json:"status"`   // 任务状态
    Response string     `json:"response"` // 响应数据（签名结果的 JSON）
    Message string      `json:"message"`  // 消息
}
```

---

## 测试与质量

### 测试文件

- `client_test.go` - 客户端功能测试
- `interface_test.go` - 接口实现验证

### 测试覆盖

- ✅ 基础签名
- ✅ 带选项的签名
- ✅ 任务轮询
- ✅ 错误处理
- ✅ HTTP 认证

### 代码质量

- **职责单一**: ✅ 客户端只负责 KMS 通信
- **接口抽象**: ✅ 通过接口支持 Mock
- **错误处理**: ✅ 直接透传 KMS 错误
- **函数长度**: ✅ 所有函数简洁清晰

---

## 常见问题 (FAQ)

### Q: 签名返回 201 状态码是什么意思？

A: 201 表示签名需要审批。KMS 会返回任务 ID，你需要轮询任务结果：

```go
result, err := client.Sign(ctx, keyID, message)
// 如果 err 包含 "signature requires approval"，提取 task_id 并轮询

taskID := extractTaskID(err)
result, err := client.WaitForTaskCompletion(ctx, taskID, 5*time.Second)
```

### Q: 如何添加交易摘要？

A: 使用 `SignWithOptions` 方法：

```go
summary := kms.NewTransferSummary(
    "0xFromAddress...",
    "0xToAddress...",
    "1000000000000000000", // 1 ETH in Wei
    "ETH",
    "Payment for services",
)

signature, err := client.SignWithOptions(
    ctx,
    keyID,
    message,
    kms.DataEncodingHex,
    summary,
    "", // 不使用回调
)
```

### Q: 如何在测试中 Mock KMS 客户端？

A: 实现 `HTTPClientInterface`：

```go
type MockHTTPClient struct {
    Response *http.Response
    Err      error
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
    return m.Response, m.Err
}

// 使用 Mock
mockClient := &MockHTTPClient{
    Response: &http.Response{
        StatusCode: http.StatusOK,
        Body:       io.NopCloser(bytes.NewReader([]byte(`{"signature":"0x..."}`))),
    },
}
client := kms.NewClientWithHTTPClient(cfg, mockClient)
```

### Q: 数据编码方式有什么区别？

A: 支持两种编码方式：

- `hex`: 十六进制编码（Ethereum 默认）
- `base64`: Base64 编码

推荐使用 `hex`，因为以太坊生态系统使用 hex 编码。

---

## 相关文件清单

### 核心文件

- `client.go` (199 行) - 客户端实现
- `http_client.go` (150+ 行) - HTTP 认证客户端
- `interface.go` (40 行) - 接口定义
- `types.go` (100+ 行) - 数据类型
- `signing.go` (80 行) - 签名相关类型

### 测试文件

- `client_test.go` - 客户端测试
- `interface_test.go` - 接口测试

### 依赖模块

- `internal/config` - 配置定义

---

## 代码审查要点

### ✅ 好的设计

1. **接口抽象** - `HTTPClientInterface` 让客户端可测试
2. **错误透传** - 不掩盖 KMS 错误，让问题在测试中暴露
3. **职责单一** - 只负责 KMS 通信，不处理业务逻辑

### ⚠️ 需要注意

1. **异步审批** - 当前已支持，但需要使用者主动轮询
2. **超时处理** - 需要在 context 中设置合理的超时时间
3. **并发安全** - 客户端可以安全地在多个 goroutine 中使用

---

## 变更记录 (Changelog)

### 2026-01-20
- 初始化模块文档
- 添加 HTTP 客户端抽象层
- 完成接口定义与测试

---

**文档版本**: 1.0.0
**维护者**: mowind
