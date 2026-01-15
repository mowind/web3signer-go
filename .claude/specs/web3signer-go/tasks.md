# web3signer-go 实施计划

## 概述

本计划将 web3signer-go 的功能设计转换为一系列具体的编码任务，以测试驱动的方式逐步实现。每个任务都建立在之前的任务基础上，确保渐进式进展和早期测试。

## 实施任务

### 1. 项目初始化和基础结构

- [ ] **1.1 初始化 Go 模块和项目结构**
  - 运行 `go mod init github.com/mowind/web3signer-go`
  - 创建标准 Go 项目目录结构：`cmd/`, `pkg/`, `internal/`, `configs/`
  - 添加 `.gitignore` 和必要的配置文件
  - 引用需求：5.1（配置管理）

- [ ] **1.2 添加项目依赖**
  - 添加 gin、viper、cobra、logrus、ethgo 到 go.mod
  - 运行 `go mod tidy` 下载依赖
  - 验证所有依赖正确安装
  - 引用需求：技术约束1-5

### 2. 配置管理系统

- [ ] **2.1 实现配置结构体**
  - 在 `internal/config` 中创建 `config.go`
  - 定义 Config、KMSConfig、DownstreamConfig 结构体
  - 添加环境变量支持标记
  - 引用需求：2.2, 2.3, 5.1

- [ ] **2.2 实现命令行参数解析**
  - 在 `cmd/web3signer` 中创建 `main.go` 和 `root.go`
  - 使用 cobra 实现命令行参数解析
  - 支持所有必需的参数：`--http-host`, `--http-port`, `--kms-endpoint`, `--kms-access-key-id`, `--kms-secret-key`, `--kms-key-id`, `--downstream-http-host`, `--downstream-http-port`, `--downstream-http-path`, `--log-level`
  - 引用需求：1.1-1.5, 2.1-2.4, 5.2

- [ ] **2.3 实现配置验证和初始化**
  - 添加配置验证逻辑，确保必需参数不为空
  - 实现配置初始化函数
  - 添加配置摘要输出
  - 编写单元测试验证配置解析
  - 引用需求：5.2, 5.3, 5.4

### 3. HTTP 服务器和 JSON-RPC 框架

- [ ] **3.1 实现 HTTP 服务器**
  - 在 `internal/server` 中创建 HTTP 服务器
  - 使用 gin 框架设置路由
  - 实现健康检查端点
  - 绑定到配置的主机和端口
  - 引用需求：1.6, 1.7

- [ ] **3.2 实现 JSON-RPC 请求解析器**
  - 在 `internal/jsonrpc` 中创建请求/响应结构体
  - 实现单个 JSON-RPC 请求解析
  - 实现批量 JSON-RPC 请求解析
  - 添加请求验证逻辑
  - 编写单元测试验证解析逻辑
  - 引用需求：3.6, 3.7, 3.8

- [ ] **3.3 实现 JSON-RPC 错误处理**
  - 定义 JSON-RPC 错误结构体
  - 实现标准错误码常量
  - 添加错误响应生成函数
  - 编写错误处理测试
  - 引用需求：6.1, 6.2

### 4. MPC-KMS 客户端

- [ ] **4.1 实现 MPC-KMS HTTP 签名认证**
  - 在 `internal/kms` 中创建客户端
  - 实现 `Authorization` 头生成逻辑
  - 支持请求时间戳生成和验证
  - 实现 Content-SHA256 计算
  - 编写单元测试验证签名算法
  - 引用需求：2.5

- [ ] **4.2 实现 MPC-KMS 签名客户端**
  - 创建 KMSClient 接口和实现
  - 实现 `Sign` 方法，调用 `/api/v1/keys/{key_id}/sign` 端点
  - 处理 MPC-KMS 响应和错误
  - 添加 HTTP 超时和重试逻辑
  - 编写集成测试（使用 mock 服务）
  - 引用需求：2.6, 2.7

### 5. 下游服务客户端

- [ ] **5.1 实现下游服务转发客户端**
  - 在 `internal/downstream` 中创建客户端
  - 实现 HTTP 客户端，转发 JSON-RPC 请求
  - 保持请求 ID 不变
  - 处理下游服务响应和错误
  - 添加超时和连接池配置
  - 引用需求：4.1-4.5

### 6. 交易处理和签名逻辑

- [ ] **6.1 实现 ethgo 交易构建器**
  - 在 `internal/signer` 中创建交易工具
  - 实现 JSON-RPC 参数到 ethgo.Transaction 的转换
  - 支持所有交易类型：Legacy、EIP-2930、EIP-1559
  - 添加交易序列化和反序列化辅助函数
  - 编写单元测试验证交易构建
  - 引用需求：3.5

- [ ] **6.2 实现 MPC-KMS Signer 接口**
  - 创建 MPCKMSSigner 结构体，实现 ethgo 的 Signer 接口
  - 实现 `Sign` 方法，调用 MPC-KMS 客户端
  - 处理交易哈希计算和签名数据准备
  - 将签名结果填充到交易结构
  - 编写单元测试验证签名流程
  - 引用需求：2.6, 3.5

### 7. JSON-RPC 路由器和处理器

- [ ] **7.1 实现 JSON-RPC 路由器**
  - 在 `internal/router` 中创建路由器
  - 实现请求分发逻辑，根据方法名路由
  - 支持批量请求处理（并发或顺序）
  - 保持响应顺序和ID
  - 编写单元测试验证路由逻辑
  - 引用需求：3.8

- [ ] **7.2 实现签名处理器**
  - 创建 SignHandler 处理 `eth_sign`, `eth_signTransaction`, `eth_sendTransaction`
  - 实现 `eth_sign`：直接对数据进行签名
  - 实现 `eth_signTransaction`：构建交易并签名
  - 实现 `eth_sendTransaction`：签名并转发到下游服务
  - 编写单元测试验证各个方法
  - 引用需求：3.2, 3.3, 3.4

- [ ] **7.3 实现转发处理器**
  - 创建 ForwardHandler 处理不支持的 JSON-RPC 方法
  - 调用下游服务客户端转发请求
  - 透传下游服务响应和错误
  - 实现 `eth_accounts` 空响应
  - 编写单元测试验证转发逻辑
  - 引用需求：3.1, 4.1-4.3

### 8. 错误处理和日志系统

- [ ] **8.1 实现统一的错误处理**
  - 创建错误包装和转换工具
  - 实现 MPC-KMS 错误到 JSON-RPC 错误的映射
  - 添加错误日志记录
  - 编写错误处理测试
  - 引用需求：6.3, 6.4

- [ ] **8.2 实现结构化日志系统**
  - 配置 logrus 为 JSON 格式输出
  - 添加请求ID追踪
  - 实现不同日志级别配置
  - 添加关键操作日志：HTTP请求、MPC-KMS调用、下游转发
  - 编写日志配置测试
  - 引用需求：7.1-7.4

### 9. 集成和端到端测试

- [ ] **9.1 创建 mock MPC-KMS 服务**
  - 实现简单的 HTTP 服务器模拟 MPC-KMS
  - 支持签名端点 `/api/v1/keys/{key_id}/sign`
  - 模拟成功响应和错误响应
  - 验证签名认证逻辑

- [ ] **9.2 创建 mock 下游服务**
  - 实现 JSON-RPC 服务器模拟下游服务
  - 支持常见的 ETH JSON-RPC 方法
  - 模拟响应和错误

- [ ] **9.3 实现端到端集成测试**
  - 编写测试启动完整的 web3signer-go 服务
  - 测试 `eth_sign`、`eth_signTransaction`、`eth_sendTransaction`
  - 测试不支持的 JSON-RPC 方法转发
  - 测试批量请求处理
  - 验证错误处理场景
  - 引用需求：3.2-3.4, 3.8, 4.1

### 10. 构建和部署准备

- [ ] **10.1 优化构建配置**
  - 添加 Makefile 或 Taskfile 构建脚本
  - 配置版本信息和编译标志
  - 添加跨平台编译支持
  - 优化二进制文件大小

- [ ] **10.2 创建部署文档**
  - 编写详细的启动命令示例
  - 添加环境变量配置说明
  - 创建 dockerfile 容器化配置
  - 添加健康检查和监控建议

### 11. 代码质量保证

- [ ] **11.1 设置代码质量工具**
  - 配置 golangci-lint 或类似工具
  - 添加 pre-commit 钩子
  - 设置代码格式化检查
  - 配置测试覆盖率要求

- [ ] **11.2 完善测试覆盖**
  - 确保核心组件单元测试覆盖率 >80%
  - 添加集成测试覆盖关键流程
  - 编写压力测试验证并发处理
  - 添加性能基准测试

## 实施原则

1. **测试驱动开发**：每个功能先写测试，再实现代码
2. **渐进式实现**：从简单到复杂，逐步构建功能
3. **早期集成**：尽早集成各个组件，发现问题
4. **持续验证**：每个步骤完成后验证功能正确性
5. **代码质量**：保持代码简洁，遵循 Go 最佳实践

## 成功标准

完成所有任务后，web3signer-go 应该能够：
1. 成功启动并监听配置的端口
2. 正确处理 `eth_sign`、`eth_signTransaction`、`eth_sendTransaction` 请求
3. 正确转发不支持的 JSON-RPC 方法到下游服务
4. 成功调用 MPC-KMS 进行签名操作
5. 支持批量 JSON-RPC 请求
6. 返回符合规范的 JSON-RPC 响应
7. 提供清晰的错误信息和日志