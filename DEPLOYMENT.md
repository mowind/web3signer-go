# web3signer-go 部署文档

## 目录

- [构建说明](#构建说明)
- [快速开始](#快速开始)
- [Docker 部署](#docker-部署)
- [直接运行](#直接运行)
- [配置说明](#配置说明)
- [健康检查](#健康检查)
- [监控和日志](#监控和日志)
- [故障排除](#故障排除)

---

## 构建说明

### 使用 Makefile

项目使用 Makefile 管理构建和测试过程：

```bash
# 构建项目
make build

# 运行测试
make test

# 格式化代码
make fmt

# 代码检查
make vet

# 跨平台编译
make build-all

# 清理构建文件
make clean

# 查看帮助
make help
```

### 手动构建

如果需要手动构建：

```bash
# 构建
go build -o web3signer ./cmd/web3signer

# 运行
./web3signer --help

# 指定配置文件
./web3signer --config /path/to/config.yaml

# 查看版本信息
./web3signer --version
```

---

## 快速开始

### 本地运行（开发环境）

最简单的启动方式：

```bash
# 使用默认配置（假设配置文件在当前目录或使用环境变量）
./web3signer

# 或者使用 make 运行
make run
```

### 基础配置示例

```bash
# 完整配置示例
./web3signer \
  --http-host 0.0.0.0 \
  --http-port 8545 \
  --kms-endpoint https://kms.example.com \
  --kms-access-key-id YOUR_ACCESS_KEY \
  --kms-secret-key YOUR_SECRET_KEY \
  --kms-key-id YOUR_KEY_ID \
  --log-level debug
```

---

## Docker 部署

### 使用 Docker Compose（推荐）

```yaml
version: '3.8'

services:
  web3signer:
    build: .
    container_name: web3signer
    restart: unless-stopped
    ports:
      - "8545:8545"
    environment:
      - HTTP_HOST=0.0.0.0
      - HTTP_PORT=8545
      - KMS_ENDPOINT=https://kms.example.com
      - KMS_ACCESS_KEY_ID=YOUR_ACCESS_KEY
      - KMS_SECRET_KEY=YOUR_SECRET_KEY
      - KMS_KEY_ID=YOUR_KEY_ID
      - LOG_LEVEL=info
    volumes:
      # 挂载配置文件（可选）
      - ./configs:/app/configs
      # 挂载日志（可选）
      - ./logs:/app/logs
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8545/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### 使用 Docker 命令

```bash
# 构建镜像
docker build -t web3signer:latest .

# 运行容器（基础）
docker run -d \
  --name web3signer \
  -p 8545:8545 \
  -e HTTP_HOST=0.0.0.0 \
  -e HTTP_PORT=8545 \
  -e KMS_ENDPOINT=https://kms.example.com \
  -e KMS_ACCESS_KEY_ID=YOUR_ACCESS_KEY \
  -e KMS_SECRET_KEY=YOUR_SECRET_KEY \
  -e KMS_KEY_ID=YOUR_KEY_ID \
  web3signer:latest

# 运行容器（指定配置文件）
docker run -d \
  --name web3signer \
  -p 8545:8545 \
  -v /path/to/config:/app/configs:ro \
  web3signer:latest

# 使用自定义网络
docker run -d \
  --name web3signer \
  --network my-network \
  -p 8545:8545 \
  web3signer:latest

# 资源限制
docker run -d \
  --name web3signer \
  --cpus="0.5" \
  --memory="512m" \
  -p 8545:8545 \
  web3signer:latest

# 后台运行（生产）
docker run -d \
  --name web3signer \
  --restart unless-stopped \
  --log-driver json-file \
  --log-opt max-size=10m \
  --log-opt max-file=5 \
  -p 8545:8545 \
  web3signer:latest
```

### Docker 进阶配置

#### 多阶段构建优化

```dockerfile
# Dockerfile 已支持多阶段构建，实现更小的镜像

# 构建多阶段镜像
docker build --target builder -t web3signer-builder .
docker build -t web3signer:latest .
```

#### 健康检查

```bash
# 检查容器状态
docker ps

# 查看容器日志
docker logs web3signer

# 检查容器健康状态
docker inspect --format='{{.State.Health.Status}}' web3signer

# 进入容器调试
docker exec -it web3signer sh

# 停止容器
docker stop web3signer

# 启动已停止的容器
docker start web3signer

# 重启容器
docker restart web3signer

# 移除容器
docker rm -f web3signer
```

---

## 直接运行

### 生产环境运行

```bash
# 使用配置文件
./web3signer --config /etc/web3signer/config.yaml

# 使用环境变量
export HTTP_HOST=0.0.0.0
export HTTP_PORT=8545
export KMS_ENDPOINT=https://kms.example.com
export KMS_ACCESS_KEY_ID=${KMS_ACCESS_KEY_ID}
export KMS_SECRET_KEY=${KMS_SECRET_KEY}
export KMS_KEY_ID=${KMS_KEY_ID}
export LOG_LEVEL=info

./web3signer

# 使用 systemd 服务（生产）
sudo systemctl start web3signer
sudo systemctl status web3signer
sudo systemctl enable web3signer
```

### 后台运行

```bash
# 使用 nohup
nohup ./web3signer > /var/log/web3signer.log 2>&1 &

# 使用 systemd
# 需要创建 systemd service 文件

# 使用 screen 或 tmux
screen -dmS web3signer ./web3signer
tmux new -s -d -n web3signer "./web3signer"
```

### 性能优化运行

```bash
# 设置 Go 运行时参数
export GOMAXPROCS=4
export GOMEMLIMIT=512MiB

# 使用 pprof 进行性能分析
./web3signer --cpuprofile=/tmp/cpu.prof --memprofile=/tmp/mem.prof
```

---

## 配置说明

### 必需配置参数

#### HTTP 服务器配置

| 参数 | 默认值 | 说明 | 环境变量 |
|------|---------|------|-----------|
| `--http-host` | 0.0.0.0 | HTTP 服务器监听地址 | HTTP_HOST |
| `--http-port` | 8545 | HTTP 服务器监听端口 | HTTP_PORT |
| `--log-level` | info | 日志级别 (debug/info/warn/error) | LOG_LEVEL |

#### MPC-KMS 配置

| 参数 | 默认值 | 说明 | 环境变量 |
|------|---------|------|-----------|
| `--kms-endpoint` | - | MPC-KMS 服务端点 URL | KMS_ENDPOINT |
| `--kms-access-key-id` | - | MPC-KMS 访问密钥 ID | KMS_ACCESS_KEY_ID |
| `--kms-secret-key` | - | MPC-KMS 密钥（生产环境建议使用密钥管理） | KMS_SECRET_KEY |
| `--kms-key-id` | - | 要使用的密钥 ID | KMS_KEY_ID |

#### 下游服务配置

| 参数 | 默认值 | 说明 | 环境变量 |
|------|---------|------|-----------|
| `--downstream-http-host` | 127.0.0.1 | 下游 HTTP 服务主机 | DOWNSTREAM_HTTP_HOST |
| `--downstream-http-port` | 8545 | 下游 HTTP 服务端口 | DOWNSTREAM_HTTP_PORT |
| `--downstream-http-path` | / | 下游 HTTP 服务路径 | DOWNSTREAM_HTTP_PATH |

### 配置文件示例

创建配置文件 `configs/production.yaml`：

```yaml
http:
  host: 0.0.0.0
  port: 8545

kms:
  endpoint: https://kms.example.com
  access_key_id: ${KMS_ACCESS_KEY_ID}
  secret_key: ${KMS_SECRET_KEY}
  key_id: ${KMS_KEY_ID}

downstream:
  http_host: 127.0.0.1
  http_port: 8545
  http_path: /

logging:
  level: info
  format: json
  output: /var/log/web3signer/web3signer.log
```

创建开发环境配置 `configs/development.yaml`：

```yaml
http:
  host: 0.0.0.0
  port: 8545

kms:
  endpoint: http://localhost:8080
  access_key_id: test-key
  secret_key: test-secret
  key_id: test-key-id

downstream:
  http_host: localhost
  http_port: 8545

logging:
  level: debug
  format: text
```

### 环境变量优先级

1. 命令行参数（最高优先级）
2. 环境变量
3. 配置文件

示例 `.env` 文件：

```bash
# HTTP 服务器
HTTP_HOST=0.0.0.0
HTTP_PORT=8545
LOG_LEVEL=info

# MPC-KMS 配置
KMS_ENDPOINT=https://kms.example.com
KMS_ACCESS_KEY_ID=AK1234567890
KMS_SECRET_KEY=your-secret-key-here
KMS_KEY_ID=key-id-for-signing

# 下游服务
DOWNSTREAM_HTTP_HOST=127.0.0.1
DOWNSTREAM_HTTP_PORT=8545
DOWNSTREAM_HTTP_PATH=/

# 运行时配置
GOMAXPROCS=4
GOMEMLIMIT=512MiB
```

### systemd Service 示例

创建 `/etc/systemd/system/web3signer.service`：

```ini
[Unit]
Description=web3signer Ethereum Signer Service
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
User=web3signer
WorkingDirectory=/opt/web3signer
ExecStart=/opt/web3signer/web3signer --config /opt/web3signer/config.yaml
Restart=always
RestartSec=10s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=web3signer

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
# 启用服务
sudo systemctl enable web3signer
sudo systemctl start web3signer

# 查看状态
sudo systemctl status web3signer

# 查看日志
sudo journalctl -u web3signer -f

# 停止服务
sudo systemctl stop web3signer
```

---

## 健康检查

### 健康检查端点

- **GET /health** - 服务健康检查
- **GET /ready** - 就绪检查

### 健康检查响应

```json
{
  "status": "healthy",
  "time": "2026-01-16T08:00:00Z",
  "services": {
    "kms": {
      "status": "connected",
      "endpoint": "https://kms.example.com"
    },
    "downstream": {
      "status": "connected",
      "endpoint": "http://127.0.0.1:8545"
    }
  }
}
```

### 健康检查使用

```bash
# 使用 curl 检查
curl http://localhost:8545/health
curl http://localhost:8545/ready

# 使用 wget
wget -qO- http://localhost:8545/health

# 监控健康状态
watch -n 5 'curl -s http://localhost:8545/health | jq'
```

### Kubernetes 健康检查

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8545
    initialDelaySeconds: 5
    periodSeconds: 10
    timeoutSeconds: 3
    failureThreshold: 3

readinessProbe:
  httpGet:
    path: /ready
    port: 8545
    initialDelaySeconds: 10
    periodSeconds: 5
    timeoutSeconds: 3
    failureThreshold: 3
```

---

## 监控和日志

### 日志配置

日志级别说明：
- **debug**: 详细调试信息，包括请求/响应详情
- **info**: 一般信息，包括启动、停止、正常操作
- **warn**: 警告信息，但不影响服务运行
- **error**: 错误信息，需要立即关注

日志格式：

```bash
# JSON 格式（结构化，推荐生产环境）
./web3signer --log-level info --log-format json

# 文本格式（开发环境友好）
./web3signer --log-level debug --log-format text
```

### 日志输出示例

```json
{
  "level": "info",
  "time": "2026-01-16T08:30:00.123Z",
  "msg": "Starting HTTP server",
  "service": "web3signer",
  "address": "0.0.0.0:8545"
}
```

### 结构化日志字段

- `level`: 日志级别
- `time`: ISO8601 时间戳
- `msg`: 日志消息
- `service`: 服务名称
- `request_id`: 请求 ID（用于追踪）
- `method`: JSON-RPC 方法名
- `error`: 错误详情（如果有）
- `duration`: 请求处理时间（毫秒）
- `upstream`: 上游服务名称（kms/downstream）

### Prometheus 监控（推荐）

```go
// 需要在代码中添加 Prometheus metrics
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/promhttp"
)

var (
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "web3signer_http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path"},
    )
    
    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "web3signer_http_request_duration_seconds",
            Help: "HTTP request latency in seconds",
        },
        []string{"method"},
    )
    
    kmsCallsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "web3signer_kms_calls_total",
            Help: "Total number of KMS calls",
        },
        []string{"method", "status"},
    )
)
```

添加到主程序：

```bash
# 在启动参数中添加
--metrics-port=9090
--metrics-enabled=true
```

### 日志收集

```bash
# 使用 Loki 收集日志
# 1. 安装 promtail
# 2. 配置 loki.yaml
# 3. 启动服务

# 使用 ELK Stack
# 1. 安装 Elasticsearch + Logstash + Kibana
# 2. 配置 Logstash 收集应用日志
# 3. 使用 Kibana 可视化

# 使用 Grafana
# 1. 配置 Prometheus 数据源
# 2. 添加日志查询仪表板
```

### 告警配置

示例告警规则：

```yaml
groups:
  - name: web3signer
    rules:
      - alert: HighErrorRate
        expr: rate(web3signer_http_requests_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          
      - alert: KMSDown
        expr: up{job="kms-service"} == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "KMS service is down"
          
      - alert: HighLatency
        expr: histogram_quantile(0.95)(web3signer_http_request_duration_seconds[5m]) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High request latency detected"
```

---

## 故障排除

### 常见问题

#### 1. 端口被占用

**症状**: 启动失败，提示 "address already in use"

**解决方案**:

```bash
# 查找占用端口的进程
sudo lsof -i :8545
sudo netstat -tulpn | grep :8545

# 杀死进程
sudo kill -9 <PID>

# 或者更换端口
./web3signer --http-port 8546
```

#### 2. KMS 连接失败

**症状**: 日志显示 "failed to sign with MPC-KMS"

**检查和解决**:

```bash
# 测试 KMS 连通性
curl -X POST https://kms.example.com/api/v1/keys/test/sign \
  -H "Content-Type: application/json" \
  -H "Authorization: MPC-KMS YOUR_KEY:YOUR_SIGNATURE" \
  -d '{"data":"test","data_encoding":"PLAIN"}'

# 检查 DNS 解析
nslookup kms.example.com

# 使用 telnet 测试连接
telnet kms.example.com 443

# 检查证书
openssl s_client -connect kms.example.com:443 -servername kms.example.com
```

#### 3. 权限错误

**症状**: "permission denied" 或 "access denied"

**解决方案**:

```bash
# 检查文件权限
ls -la web3signer
ls -la configs/

# 修复权限
chmod +x web3signer
sudo chown -R $USER:$USER configs/

# Docker 权限
docker run --user $UID:$GID web3signer:latest
```

#### 4. 内存不足

**症状**: "out of memory" 或 "cannot allocate memory"

**解决方案**:

```bash
# 检查内存使用
docker stats web3signer
free -h
cat /proc/meminfo

# 增加 Docker 内存限制
docker run -d \
  --memory="1g" \
  --memory-swap="1g" \
  web3signer:latest

# 优化 Go 内存
export GOMEMLIMIT=256MiB

# 使用 swap
sudo swapon /swapfile
```

#### 5. 下游服务连接超时

**症状**: "downstream service timeout" 或 "connection refused"

**解决方案**:

```bash
# 增加超时时间（如果可配置）
./web3signer --downstream-timeout 30

# 检查网络连接
ping 127.0.0.1

# 测试下游服务
curl http://127.0.0.1:8545/health

# 使用负载均衡（如果有多个下游实例）
# 在配置文件中指定多个下游地址
```

### 调试技巧

```bash
# 启用调试日志
./web3signer --log-level debug

# 使用 strace 追踪系统调用（Linux）
strace -f -e trace=network -p $$ ./web3signer

# 使用 Go pprof
./web3signer --cpuprofile=/tmp/cpu.prof
```

### 日志文件位置

```bash
# Docker 日志
docker logs -f web3signer
docker inspect --format='{{.LogPath}}' web3signer

# 持载日志目录
docker run -v $(pwd)/logs:/var/log/web3signer -d web3signer

# systemd 日志
sudo journalctl -u web3signer -f
sudo journalctl -u web3signer -e
```

### 性能优化建议

1. **连接池**: 复用 HTTP 连接
2. **并发控制**: 使用工作池限制并发请求数
3. **缓存**: 缓存频繁访问的配置和数据
4. **压缩**: 启用 HTTP 响应压缩
5. **Go 版本**: 定期升级 Go 版本以获得性能改进

---

## 安全建议

### 生产环境配置

1. **使用非 root 用户**: Docker 容器使用专用用户
2. **限制权限**: 文件系统只读，只写必要目录
3. **网络隔离**: 使用 Docker 网络或 Kubernetes network policies
4. **密钥管理**: 
   - 使用环境变量或密钥管理服务（AWS Secrets Manager、Vault）
   - 不要在配置文件中硬编码密钥
   - 定期轮换访问密钥
5. **TLS 配置**: 使用 HTTPS，配置有效证书
6. **防火墙规则**: 只开放必要端口 (8545)

### 密钥管理

```bash
# 使用 AWS Secrets Manager
aws secretsmanager get-secret-value --secret-id web3signer/kms-secret-key

# 使用 HashiCorp Vault
vault kv get -field=secret_key secret/web3signer

# 环境变量
export KMS_SECRET_KEY=$(/path/to/secret)
```

### 安全加固

```bash
# 配置防火墙
sudo ufw allow 8545/tcp
sudo firewall-cmd --zone=public --add-port=8545/tcp --permanent

# 限制 Docker 能力
--cap-drop=ALL
--cap-add=NET_BIND_SERVICE
--security-opt=no-new-privileges

# 只读文件系统
--read-only
--tmpfs /tmp
```

---

## 备份和恢复

### 配置备份

```bash
# 备份配置文件
tar -czf web3signer-configs-$(date +%Y%m%d).tar.gz configs/

# 备份密钥
# 建议使用密钥管理系统导出

# Docker 备份
docker export web3signer > web3signer-backup.tar
```

### 恢复流程

```bash
# 停止服务
docker stop web3signer

# 恢复配置
tar -xzf web3signer-configs-20260116.tar.gz -C /

# 重启服务
docker start web3signer

# 验证
curl http://localhost:8545/health
```

---

## 升级流程

### 滚动升级

```bash
# 1. 备份当前版本
./web3signer --version > /tmp/version-backup.txt

# 2. 下载新版本
wget https://github.com/mowind/web3signer-go/releases/download/v1.0.1/web3signer-linux-amd64

# 3. 验证新版本
./web3signer --version

# 4. 停止旧服务
sudo systemctl stop web3signer
docker stop web3signer

# 5. 替换二进制
cp web3signer /usr/local/bin/web3signer

# 6. 启动新版本
sudo systemctl start web3signer
docker start web3signer

# 7. 验证
curl http://localhost:8545/ready
```

### 蓝绿部署

```bash
# 使用蓝绿部署策略
# Blue: 当前生产版本
# Green: 新版本正在测试

# 切换命令
kubectl set image deployment/web3signer web3signer:green
kubectl rollout status deployment/web3signer
```

---

## 联系和支持

如有问题，请通过以下方式获取支持：

- GitHub Issues: https://github.com/mowind/web3signer-go/issues
- 文档: https://github.com/mowind/web3signer-go/wiki
- 讨论: https://github.com/mowind/web3signer-go/discussions

---

## 快速参考

### 常用命令速查

```bash
# 查看版本
./web3signer --version

# 查看帮助
./web3signer --help

# 健康检查
curl http://localhost:8545/health

# 查看日志
docker logs -f web3signer

# 进入容器
docker exec -it web3signer sh

# 重启服务
docker restart web3signer

# 查看端口占用
sudo lsof -i :8545
```

### 配置检查清单

部署前确认：

- [ ] 所有必需参数已配置（KMS endpoint、密钥等）
- [ ] 防火墙规则已配置（开放必要端口）
- [ ] 磁盘空间充足（至少 1GB 可用）
- [ ] 网络连接正常（可访问 KMS 和下游服务）
- [ ] 日志目录已创建且有写权限
- [ ] 监控系统已配置（如 Prometheus）
- [ ] 密钥管理方案已确定
- [ ] 升级流程已制定
- [ ] 备份策略已定义