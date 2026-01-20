[根目录](../../CLAUDE.md) > [internal](../) > **router (JSON-RPC 路由)**

---

# internal/router - JSON-RPC 路由模块

> **最后更新**: 2026-01-20 11:07:09
> **模块状态**: 🟢 完成
> **测试覆盖**: ✅ 完整

---

## 模块职责

JSON-RPC 路由模块负责：

1. **请求路由** - 根据 JSON-RPC 方法名分发到对应处理器
2. **批量请求** - 支持 JSON-RPC 批量请求规范
3. **处理器管理** - 注册、注销、查找处理器
4. **默认转发** - 将未注册的方法转发到下游服务

---

## 入口与启动

### 文件结构

```
internal/router/
├── router.go              # 路由器实现
├── base.go                # 处理器基类
├── sign_handler.go        # 签名处理器
├── forward_handler.go     # 转发处理器
├── factory.go             # 处理器工厂
├── router_test.go         # 路由器测试
├── integration_test.go    # 集成测试
└── simple_integration_test.go # 简单集成测试
```

### 创建路由器

```go
import "github.com/mowind/web3signer-go/internal/router"

// 创建路由器
logger := logrus.New()
router := router.NewRouter(logger)

// 注册处理器
router.Register(signHandler)
router.Register(healthHandler)

// 设置默认处理器（转发未注册的方法）
router.SetDefaultHandler(forwardHandler)
```

---

## 对外接口

### Handler 接口

```go
type Handler interface {
    // Handle 处理 JSON-RPC 请求
    Handle(ctx context.Context, request *jsonrpc.Request) (*jsonrpc.Response, error)

    // Method 返回处理器支持的方法名
    Method() string
}
```

### Router 接口

```go
type Router struct {
    handlers       map[string]Handler  // 已注册的处理器
    defaultHandler Handler             // 默认处理器
    mu             sync.RWMutex        // 读写锁
    logger         *logrus.Logger      // 日志记录器
}

// 创建路由器
func NewRouter(logger *logrus.Logger) *Router

// 注册处理器
func (r *Router) Register(handler Handler) error

// 注销处理器
func (r *Router) Unregister(method string)

// 设置默认处理器
func (r *Router) SetDefaultHandler(handler Handler)

// 路由单个请求
func (r *Router) Route(ctx context.Context, request *jsonrpc.Request) *jsonrpc.Response

// 路由批量请求
func (r *Router) RouteBatch(ctx context.Context, requests []jsonrpc.Request) []*jsonrpc.Response

// 处理 HTTP 请求（用于 Gin 集成）
func (r *Router) HandleHTTPRequest(w http.ResponseWriter, req *http.Request)
```

---

## 关键依赖与配置

### 核心依赖

```go
import (
    "context"
    "io"
    "net/http"
    "sync"

    "github.com/mowind/web3signer-go/internal/jsonrpc"
    "github.com/sirupsen/logrus"
)
```

### 处理器依赖

- **SignHandler**: 需要签名器和 KMS 客户端
- **ForwardHandler**: 需要下游服务客户端
- **HealthHandler**: 无额外依赖

---

## 数据模型

### 请求流程

```
HTTP Request
    ↓
HandleHTTPRequest (解析 body)
    ↓
ParseRequest (解析 JSON-RPC)
    ↓
    ├─→ 单个请求
    │       ↓
    │   Route (路由到处理器)
    │       ↓
    │   Handler.Handle (处理)
    │       ↓
    │   Response (响应)
    │
    └─→ 批量请求
            ↓
        RouteBatch (遍历处理)
            ↓
        []Response (响应数组)
    ↓
MarshalResponse (序列化)
    ↓
HTTP Response
```

### 支持的方法

#### 签名方法（由 SignHandler 处理）

- `eth_sign` - 签名数据
- `eth_signTransaction` - 签名交易
- `eth_sendTransaction` - 签名并发送交易

#### 健康检查方法（可选）

- `eth_chainId` - 返回链 ID
- `web3_clientVersion` - 返回客户端版本

#### 转发方法（由 ForwardHandler 处理）

- 所有其他方法都转发到下游服务

---

## 测试与质量

### 测试文件

- `router_test.go` - 路由器单元测试
- `integration_test.go` - 完整集成测试
- `simple_integration_test.go` - 简化集成测试

### 测试覆盖

- ✅ 处理器注册与注销
- ✅ 单个请求路由
- ✅ 批量请求路由
- ✅ 默认处理器
- ✅ 错误处理
- ✅ 并发安全

### 代码质量

- **职责单一**: ✅ 路由器只负责路由，不处理业务逻辑
- **并发安全**: ✅ 使用读写锁保护 handlers map
- **错误处理**: ✅ 正确处理各种错误情况

---

## 常见问题 (FAQ)

### Q: 如何添加新的 JSON-RPC 方法？

A: 创建一个新的 Handler 并注册：

```go
// 1. 定义 Handler
type MyHandler struct {
    logger *logrus.Logger
}

func (h *MyHandler) Method() string {
    return "my_method"
}

func (h *MyHandler) Handle(ctx context.Context, req *jsonrpc.Request) (*jsonrpc.Response, error) {
    // 处理逻辑
    return jsonrpc.NewSuccessResponse(req.ID, result), nil
}

// 2. 注册到路由器
router.Register(&MyHandler{logger: logger})
```

### Q: 批量请求如何保证顺序？

A: `RouteBatch` 方法会按顺序处理请求，保持响应顺序：

```go
// 请求
[
    {"id": 1, "method": "eth_getBalance"},
    {"id": 2, "method": "eth_signTransaction"}
]

// 响应（顺序一致）
[
    {"id": 1, "result": "..."},
    {"id": 2, "result": "..."}
]
```

### Q: 如何实现异步处理？

A: Handler 的 `Handle` 方法是同步的。如果需要异步处理，可以在 Handler 内部启动 goroutine：

```go
func (h *MyHandler) Handle(ctx context.Context, req *jsonrpc.Request) (*jsonrpc.Response, error) {
    // 启动异步任务
    go func() {
        result := doAsyncWork()
        // 通过回调或其他方式返回结果
    }()

    // 立即返回任务 ID
    return jsonrpc.NewSuccessResponse(req.ID, taskID), nil
}
```

### Q: 路由器是线程安全的吗？

A: 是的。路由器使用 `sync.RWMutex` 保护 `handlers` map：

- 读操作（路由请求）使用读锁
- 写操作（注册/注销）使用写锁

多个 goroutine 可以安全地并发路由请求。

### Q: 如何调试路由问题？

A: 路由器会记录详细日志：

```go
// 启用 Debug 日志
logger.SetLevel(logrus.DebugLevel)

// 日志示例
router.WithFields(logrus.Fields{
    "method": "eth_sign",
    "id": 1,
}).Debug("Routing JSON-RPC request")
```

---

## 相关文件清单

### 核心文件

- `router.go` (227 行) - 路由器核心实现
- `base.go` (60 行) - 处理器基类
- `sign_handler.go` (150+ 行) - 签名处理器
- `forward_handler.go` (80 行) - 转发处理器
- `factory.go` (100 行) - 处理器工厂

### 测试文件

- `router_test.go` - 路由器测试
- `integration_test.go` - 集成测试
- `simple_integration_test.go` - 简化集成测试

### 依赖模块

- `internal/jsonrpc` - JSON-RPC 类型定义
- `internal/signer` - 签名器
- `internal/downstream` - 下游客户端

---

## 代码审查要点

### ✅ 好的设计

1. **接口简洁** - `Handler` 接口只有两个方法
2. **职责分离** - 路由器不处理业务逻辑
3. **并发安全** - 正确使用读写锁
4. **默认处理器** - 优雅地处理未注册的方法

### ⚠️ 需要注意

1. **错误处理** - 错误响应已经标准化，不需要额外包装
2. **日志级别** - 生产环境建议使用 Info 或 Warn 级别
3. **性能** - 批量请求目前是顺序处理，可以优化为并发

### 🔴 潜在优化

1. **并发批量处理** - 当前批量请求是顺序处理，可以并发：
   ```go
   // 当前（顺序）
   for i, request := range requests {
       responses[i] = r.Route(ctx, &request)
   }

   // 优化（并发）
   var wg sync.WaitGroup
   for i, request := range requests {
       wg.Add(1)
       go func(idx int, req *jsonrpc.Request) {
           defer wg.Done()
           responses[idx] = r.Route(ctx, req)
       }(i, &request)
   }
   wg.Wait()
   ```

2. **处理器池** - 如果处理器创建成本高，可以使用对象池

---

## 使用示例

### 基本使用

```go
// 创建路由器
logger := logrus.New()
router := router.NewRouter(logger)

// 创建处理器
signHandler := router.NewSignHandler(signer, logger)
forwardHandler := router.NewForwardHandler(downstreamClient, logger)

// 注册处理器
router.Register(signHandler)
router.SetDefaultHandler(forwardHandler)

// 路由请求
request := &jsonrpc.Request{
    JSONRPC: "2.0",
    Method:  "eth_sign",
    Params:  []interface{}{...},
    ID:      1,
}
response := router.Route(context.Background(), request)
```

### 集成到 Gin

```go
// 在 Gin 中使用
router := setupRouter()

engine.POST("/", func(c *gin.Context) {
    router.HandleHTTPRequest(c.Writer, c.Request)
})
```

---

## 变更记录 (Changelog)

### 2026-01-20
- 初始化模块文档
- 完成代码审查
- 识别批量请求优化机会

---

**文档版本**: 1.0.0
**维护者**: mowind
