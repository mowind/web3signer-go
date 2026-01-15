#!/bin/bash

# 启动服务器（在后台）
echo "Starting web3signer-go server..."
./web3signer --kms-endpoint "http://localhost:8080" \
            --kms-access-key-id "test" \
            --kms-secret-key "secret" \
            --kms-key-id "key123" \
            --downstream-http-host "localhost" \
            --downstream-http-port 8545 \
            --downstream-http-path "/" \
            --log-level info &
SERVER_PID=$!

# 等待服务器启动
echo "Waiting for server to start..."
sleep 2

# 测试健康检查
echo -e "\n=== Testing health check ==="
curl -s http://localhost:9000/health | jq .

# 测试就绪检查
echo -e "\n=== Testing readiness check ==="
curl -s http://localhost:9000/ready | jq .

# 测试单个 JSON-RPC 请求
echo -e "\n=== Testing single JSON-RPC request ==="
curl -s -X POST http://localhost:9000/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' | jq .

# 测试批量 JSON-RPC 请求
echo -e "\n=== Testing batch JSON-RPC request ==="
curl -s -X POST http://localhost:9000/ \
  -H "Content-Type: application/json" \
  -d '[{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x1234","latest"],"id":1},{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":2}]' | jq .

# 测试无效 JSON-RPC 请求
echo -e "\n=== Testing invalid JSON-RPC request ==="
curl -s -X POST http://localhost:9000/ \
  -H "Content-Type: application/json" \
  -d 'invalid json' | jq .

# 停止服务器
echo -e "\nStopping server..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null

echo "Test complete"