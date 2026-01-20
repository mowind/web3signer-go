package test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/mowind/web3signer-go/internal/kms"
)

// MockKMSServer 模拟 MPC-KMS HTTP 服务
type MockKMSServer struct {
	server      *httptest.Server
	mu          sync.RWMutex
	validKeys   map[string]bool
	signatures  map[string][]string // keyID -> []signatures
	shouldFail  bool
	requireAuth bool
	accessKeyID string
	secretKey   string
}

// NewMockKMSServer 创建新的 mock KMS 服务器
func NewMockKMSServer() *MockKMSServer {
	mock := &MockKMSServer{
		validKeys:   make(map[string]bool),
		signatures:  make(map[string][]string),
		requireAuth: true,
		accessKeyID: "AK1234567890",
		secretKey:   "test-secret-key",
	}

	mock.server = httptest.NewServer(http.HandlerFunc(mock.handleRequest))
	return mock
}

// SetAccessKey 设置访问密钥
func (m *MockKMSServer) SetAccessKey(accessKeyID, secretKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.accessKeyID = accessKeyID
	m.secretKey = secretKey
}

// AddValidKey 添加有效的密钥ID
func (m *MockKMSServer) AddValidKey(keyID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validKeys[keyID] = true
}

// SetShouldFail 设置是否应该失败
func (m *MockKMSServer) SetShouldFail(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
}

// SetRequireAuth 设置是否需要认证
func (m *MockKMSServer) SetRequireAuth(requireAuth bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requireAuth = requireAuth
}

// GetSignatures 获取指定密钥的签名列表
func (m *MockKMSServer) GetSignatures(keyID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.signatures[keyID]
}

// URL 返回服务器 URL
func (m *MockKMSServer) URL() string {
	return m.server.URL
}

// Close 关闭服务器
func (m *MockKMSServer) Close() {
	m.server.Close()
}

// handleRequest 处理 HTTP 请求
func (m *MockKMSServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	// 验证认证
	if m.requireAuth {
		if err := m.validateAuth(r); err != nil {
			m.writeError(w, http.StatusUnauthorized, "Unauthorized", err.Error())
			return
		}
	}

	// 路由处理
	switch {
	case strings.HasPrefix(r.URL.Path, "/api/v1/keys/") && strings.HasSuffix(r.URL.Path, "/sign"):
		m.handleSign(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/v1/tasks/"):
		m.handleTask(w, r)
	case r.URL.Path == "/health":
		m.handleHealth(w, r)
	default:
		m.writeError(w, http.StatusNotFound, "NotFound", "Endpoint not found")
	}
}

// handleSign 处理签名请求
func (m *MockKMSServer) handleSign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		m.writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
		return
	}

	if m.shouldFail {
		m.writeError(w, http.StatusInternalServerError, "InternalError", "Simulated server error")
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		m.writeError(w, http.StatusBadRequest, "InvalidRequest", "Failed to read request body")
		return
	}
	defer func() { _ = r.Body.Close() }()

	// 重新设置 Body，以便后续处理
	r.Body = io.NopCloser(bytes.NewReader(body))

	// 解析请求
	var req kms.SignRequest
	if err := json.Unmarshal(body, &req); err != nil {
		m.writeError(w, http.StatusBadRequest, "InvalidRequest", "Invalid request body")
		return
	}

	// 提取 keyID
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/keys/")
	keyID := strings.TrimSuffix(path, "/sign")

	// 验证密钥
	m.mu.RLock()
	validKey := m.validKeys[keyID]
	m.mu.RUnlock()

	if !validKey {
		m.writeError(w, http.StatusNotFound, "KeyNotFound", "Key not found")
		return
	}

	// 生成模拟签名
	signature := m.generateMockSignature(req.Data, keyID)

	// 保存签名记录
	m.mu.Lock()
	m.signatures[keyID] = append(m.signatures[keyID], signature)
	m.mu.Unlock()

	// 如果需要审批，返回任务ID
	if req.CallbackURL != "" {
		resp := kms.TaskResponse{
			TaskID: fmt.Sprintf("task_%d", time.Now().Unix()),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// 直接返回签名
	resp := kms.SignResponse{
		Signature: signature,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleTask 处理任务查询
func (m *MockKMSServer) handleTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		m.writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
		return
	}

	// 模拟任务结果
	taskID := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
	if taskID == "" {
		m.writeError(w, http.StatusBadRequest, "InvalidTaskID", "Invalid task ID")
		return
	}

	result := kms.TaskResult{
		Status:  kms.TaskStatusDone,
		Message: "Task completed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// handleHealth 处理健康检查
func (m *MockKMSServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

// validateAuth 验证请求认证
func (m *MockKMSServer) validateAuth(r *http.Request) error {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("missing Authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "MPC-KMS" {
		return fmt.Errorf("invalid Authorization header format")
	}

	creds := strings.Split(parts[1], ":")
	if len(creds) != 2 {
		return fmt.Errorf("invalid credentials format")
	}

	accessKeyID := creds[0]
	signature := creds[1]

	m.mu.RLock()
	expectedAccessKeyID := m.accessKeyID
	expectedSecretKey := m.secretKey
	m.mu.RUnlock()

	if accessKeyID != expectedAccessKeyID {
		return fmt.Errorf("invalid access key ID: got %s, expected %s", accessKeyID, expectedAccessKeyID)
	}

	// 验证签名
	contentSHA256 := r.Header.Get("Content-SHA256")
	contentType := r.Header.Get("Content-Type")
	date := r.Header.Get("Date")

	if date == "" {
		return fmt.Errorf("missing Date header")
	}

	// 重新计算请求体的 SHA256 以验证
	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	_ = calculateContentSHA256(bodyBytes)

	// 构建签名字符串
	signingString := buildSigningString(r.Method, contentSHA256, contentType, date)

	// 计算期望的签名
	expectedSignature := calculateHMACSHA256(signingString, expectedSecretKey)
	if signature != expectedSignature {
		return fmt.Errorf("invalid signature: got %s, expected %s", signature, expectedSignature)
	}

	return nil
}

// generateMockSignature 生成模拟签名
func (m *MockKMSServer) generateMockSignature(data, keyID string) string {
	// 生成确定性签名（相同的输入产生相同的输出）
	h := sha256.New()
	h.Write([]byte(data))
	h.Write([]byte(keyID))
	h.Write([]byte("mock-kms-salt"))
	hash := h.Sum(nil)

	// 返回十六进制编码的65字节签名（以太坊签名格式）
	signature := make([]byte, 65)
	copy(signature, hash)
	for i := len(hash); i < 65; i++ {
		signature[i] = byte(i)
	}

	return hex.EncodeToString(signature)
}

// writeError 写入错误响应
func (m *MockKMSServer) writeError(w http.ResponseWriter, statusCode int, code, message string) {
	errResp := kms.ErrorResponse{
		Code:    statusCode,
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errResp)
}

// Helper functions (复制自真实客户端)
func buildSigningString(verb, contentSHA256, contentType, date string) string {
	var result string
	if contentType != "" {
		result = fmt.Sprintf("%s\n%s\n%s\n%s", verb, contentSHA256, contentType, date)
	} else {
		result = fmt.Sprintf("%s\n%s\n\n%s", verb, contentSHA256, date)
	}
	return result
}

func calculateHMACSHA256(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// MockKMSClient 用于测试的 mock KMS 客户端
type MockKMSClient struct {
	server      *MockKMSServer
	accessKeyID string
	secretKey   string
}

// NewMockKMSClient 创建新的 mock KMS 客户端
func NewMockKMSClient(server *MockKMSServer) *MockKMSClient {
	return &MockKMSClient{
		server:      server,
		accessKeyID: "AK1234567890",
		secretKey:   "test-secret-key",
	}
}

// SetCredentials 设置认证凭据
func (c *MockKMSClient) SetCredentials(accessKeyID, secretKey string) {
	c.accessKeyID = accessKeyID
	c.secretKey = secretKey
}

// Sign 实现签名接口
func (c *MockKMSClient) Sign(ctx context.Context, keyID string, message []byte) ([]byte, error) {
	req := kms.SignRequest{
		Data:         string(message),
		DataEncoding: string(kms.DataEncodingPlain),
	}

	resp, err := c.callSignEndpoint(keyID, req)
	if err != nil {
		return nil, err
	}

	return []byte(resp.Signature), nil
}

// SignWithOptions 实现带选项的签名接口
func (c *MockKMSClient) SignWithOptions(ctx context.Context, keyID string, message []byte, encoding kms.DataEncoding, summary *kms.SignSummary, callbackURL string) ([]byte, error) {
	req := kms.SignRequest{
		Data:         string(message),
		DataEncoding: string(encoding),
		Summary:      summary,
		CallbackURL:  callbackURL,
	}

	if callbackURL != "" {
		// 需要审批的情况
		taskResp, err := c.callTaskEndpoint(keyID, req)
		if err != nil {
			return nil, err
		}

		// 等待任务完成
		return c.waitForTaskCompletion(ctx, taskResp.TaskID)
	}

	// 直接签名
	resp, err := c.callSignEndpoint(keyID, req)
	if err != nil {
		return nil, err
	}

	return []byte(resp.Signature), nil
}

// GetTaskResult 获取任务结果
func (c *MockKMSClient) GetTaskResult(ctx context.Context, taskID string) (*kms.TaskResult, error) {
	// 模拟总是返回完成状态
	return &kms.TaskResult{
		Status:  kms.TaskStatusDone,
		Message: "Task completed successfully",
	}, nil
}

// WaitForTaskCompletion 等待任务完成
func (c *MockKMSClient) WaitForTaskCompletion(ctx context.Context, taskID string, interval time.Duration) (*kms.TaskResult, error) {
	// 模拟等待任务完成
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond): // 模拟短暂延迟
		return c.GetTaskResult(ctx, taskID)
	}
}

// do 执行 HTTP 请求（内部方法）
func (c *MockKMSClient) do(req *http.Request) (*http.Response, error) {
	// 简单地将请求转发到 mock 服务器
	return http.DefaultClient.Do(req)
}

// callSignEndpoint 调用签名端点
func (c *MockKMSClient) callSignEndpoint(keyID string, req kms.SignRequest) (*kms.SignResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/keys/%s/sign", c.server.URL(), keyID), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// 设置认证头
	c.signRequest(httpReq, body)

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var errResp kms.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		}
		return nil, fmt.Errorf("KMS error: %s", errResp.Message)
	}

	var signResp kms.SignResponse
	if err := json.NewDecoder(resp.Body).Decode(&signResp); err != nil {
		return nil, err
	}

	return &signResp, nil
}

// callTaskEndpoint 调用任务端点
func (c *MockKMSClient) callTaskEndpoint(keyID string, req kms.SignRequest) (*kms.TaskResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/keys/%s/sign", c.server.URL(), keyID), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// 设置认证头
	c.signRequest(httpReq, body)

	resp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var errResp kms.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		}
		return nil, fmt.Errorf("KMS error: %s", errResp.Message)
	}

	var taskResp kms.TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return nil, err
	}

	return &taskResp, nil
}

// waitForTaskCompletion 等待任务完成并返回签名
func (c *MockKMSClient) waitForTaskCompletion(ctx context.Context, taskID string) ([]byte, error) {
	result, err := c.WaitForTaskCompletion(ctx, taskID, 100*time.Millisecond)
	if err != nil {
		return nil, err
	}

	if result.Status != kms.TaskStatusDone {
		return nil, fmt.Errorf("task failed: %s", result.Message)
	}

	return []byte(result.Response), nil
}

// signRequest 为请求添加认证信息
func (c *MockKMSClient) signRequest(req *http.Request, body []byte) {
	// 计算内容 SHA256
	contentSHA256 := calculateContentSHA256(body)

	// 确保 Content-Type 已设置
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 设置日期
	date := time.Now().Format("Mon, 02 Jan 2006 15:04:05 GMT")

	// 构建签名字符串
	signingString := buildSigningString(req.Method, contentSHA256, req.Header.Get("Content-Type"), date)

	// 计算签名
	signature := calculateHMACSHA256(signingString, c.secretKey)

	// 设置请求头
	req.Header.Set("Authorization", fmt.Sprintf("MPC-KMS %s:%s", c.accessKeyID, signature))
	req.Header.Set("Date", date)
	req.Header.Set("Content-SHA256", contentSHA256)
}

// Helper functions
func calculateContentSHA256(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
