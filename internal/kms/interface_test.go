package kms

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
)

// mockClient 模拟 MPC-KMS 客户端
type mockClient struct {
	signFunc            func(ctx context.Context, keyID string, message []byte) ([]byte, error)
	signWithOptionsFunc func(ctx context.Context, keyID string, message []byte, encoding DataEncoding, summary *SignSummary, callbackURL string) ([]byte, error)
	getTaskResultFunc   func(ctx context.Context, taskID string) (*TaskResult, error)
	waitForTaskFunc     func(ctx context.Context, taskID string, interval time.Duration) (*TaskResult, error)
}

func (m *mockClient) Sign(ctx context.Context, keyID string, message []byte) ([]byte, error) {
	if m.signFunc != nil {
		return m.signFunc(ctx, keyID, message)
	}
	return nil, nil
}

func (m *mockClient) SignWithOptions(ctx context.Context, keyID string, message []byte, encoding DataEncoding, summary *SignSummary, callbackURL string) ([]byte, error) {
	if m.signWithOptionsFunc != nil {
		return m.signWithOptionsFunc(ctx, keyID, message, encoding, summary, callbackURL)
	}
	return nil, nil
}

func (m *mockClient) GetTaskResult(ctx context.Context, taskID string) (*TaskResult, error) {
	if m.getTaskResultFunc != nil {
		return m.getTaskResultFunc(ctx, taskID)
	}
	return nil, nil
}

func (m *mockClient) WaitForTaskCompletion(ctx context.Context, taskID string, interval time.Duration) (*TaskResult, error) {
	if m.waitForTaskFunc != nil {
		return m.waitForTaskFunc(ctx, taskID, interval)
	}
	return nil, nil
}

func TestMPCKMSSigner_SignMessage(t *testing.T) {
	tests := []struct {
		name        string
		keyID       string
		message     []byte
		expectedSig []byte
		shouldError bool
	}{
		{
			name:        "successful signature",
			keyID:       "test-key-id",
			message:     []byte("test message"),
			expectedSig: []byte("0x1234567890abcdef"),
			shouldError: false,
		},
		{
			name:        "empty message",
			keyID:       "test-key-id",
			message:     []byte(""),
			expectedSig: []byte("0xempty"),
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClient{
				signFunc: func(ctx context.Context, keyID string, message []byte) ([]byte, error) {
					if keyID != tt.keyID {
						t.Errorf("Expected keyID %s, got %s", tt.keyID, keyID)
					}
					if string(message) != string(tt.message) {
						t.Errorf("Expected message %s, got %s", tt.message, message)
					}
					return tt.expectedSig, nil
				},
			}

			signer := NewMPCKMSSigner(mock, tt.keyID)
			signature, err := signer.SignMessage(context.Background(), tt.message)

			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.shouldError && string(signature) != string(tt.expectedSig) {
				t.Errorf("Expected signature %s, got %s", tt.expectedSig, signature)
			}
		})
	}
}

func TestMPCKMSSigner_SignTransaction(t *testing.T) {
	tests := []struct {
		name        string
		keyID       string
		transaction []byte
		expectedSig []byte
		shouldError bool
	}{
		{
			name:        "successful transaction signature",
			keyID:       "test-key-id",
			transaction: []byte("0x1234567890abcdef"),
			expectedSig: []byte("0xabcdef1234567890"),
			shouldError: false,
		},
		{
			name:        "hex encoded transaction",
			keyID:       "test-key-id",
			transaction: []byte("deadbeef"),
			expectedSig: []byte("0xsignature"),
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockClient{
				signWithOptionsFunc: func(ctx context.Context, keyID string, message []byte, encoding DataEncoding, summary *SignSummary, callbackURL string) ([]byte, error) {
					if keyID != tt.keyID {
						t.Errorf("Expected keyID %s, got %s", tt.keyID, keyID)
					}
					if string(message) != string(tt.transaction) {
						t.Errorf("Expected transaction %s, got %s", tt.transaction, message)
					}
					if encoding != DataEncodingHex {
						t.Errorf("Expected encoding HEX, got %s", encoding)
					}
					if summary == nil {
						t.Error("Expected summary but got nil")
					} else if summary.Type != string(SummaryTypeTransfer) {
						t.Errorf("Expected summary type TRANSFER, got %s", summary.Type)
					}
					return tt.expectedSig, nil
				},
			}

			signer := NewMPCKMSSigner(mock, tt.keyID)
			signature, err := signer.SignTransaction(context.Background(), tt.transaction)

			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.shouldError && string(signature) != string(tt.expectedSig) {
				t.Errorf("Expected signature %s, got %s", tt.expectedSig, signature)
			}
		})
	}
}

func TestNewMPCKMSSigner(t *testing.T) {
	cfg := &config.KMSConfig{
		Endpoint:    "https://kms.example.com",
		AccessKeyID: "AK1234567890",
		SecretKey:   "test-secret-key",
		KeyID:       "test-key-id",
	}

	client := NewClient(cfg)
	signer := NewMPCKMSSigner(client, cfg.KeyID)

	if signer == nil {
		t.Fatal("NewMPCKMSSigner returned nil")
	}
	if signer.keyID != cfg.KeyID {
		t.Errorf("Expected keyID %s, got %s", cfg.KeyID, signer.keyID)
	}
	if signer.client == nil {
		t.Error("Client is nil")
	}
}

func TestInterfaceImplementation(t *testing.T) {
	// 验证 Client 实现了 ClientInterface
	var _ ClientInterface = (*Client)(nil)

	// 验证 MPCKMSSigner 实现了 Signer
	var _ Signer = (*MPCKMSSigner)(nil)

	cfg := &config.KMSConfig{
		Endpoint:    "https://kms.example.com",
		AccessKeyID: "AK1234567890",
		SecretKey:   "test-secret-key",
		KeyID:       "test-key-id",
	}

	client := NewClient(cfg)
	signer := NewMPCKMSSigner(client, cfg.KeyID)

	// 类型断言验证
	if _, ok := interface{}(client).(ClientInterface); !ok {
		t.Error("Client does not implement ClientInterface")
	}

	if _, ok := interface{}(signer).(Signer); !ok {
		t.Error("MPCKMSSigner does not implement Signer")
	}
}

func TestSignRequest_Marshal(t *testing.T) {
	tests := []struct {
		name     string
		request  *SignRequest
		expected string
	}{
		{
			name: "simple request",
			request: &SignRequest{
				Data:         "test data",
				DataEncoding: "PLAIN",
			},
			expected: `{"data":"test data","data_encoding":"PLAIN"}`,
		},
		{
			name: "request with summary",
			request: &SignRequest{
				Data:         "0x123456",
				DataEncoding: "HEX",
				Summary: &SignSummary{
					Type:   "TRANSFER",
					From:   "0x1111111111111111111111111111111111111111",
					To:     "0x2222222222222222222222222222222222222222",
					Amount: "1.5",
					Token:  "ETH",
					Remark: "test transfer",
				},
			},
			expected: `{"data":"0x123456","data_encoding":"HEX","summary":{"type":"TRANSFER","from":"0x1111111111111111111111111111111111111111","to":"0x2222222222222222222222222222222222222222","amount":"1.5","remark":"test transfer","token":"ETH"}}`,
		},
		{
			name: "request with callback",
			request: &SignRequest{
				Data:         "test",
				DataEncoding: "PLAIN",
				CallbackURL:  "https://example.com/callback",
			},
			expected: `{"data":"test","data_encoding":"PLAIN","callback_url":"https://example.com/callback"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.request.Marshal()
			if err != nil {
				t.Errorf("Marshal failed: %v", err)
			}

			// 解析 JSON 验证结构
			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Errorf("Unmarshal failed: %v", err)
			}

			// 验证数据字段
			if dataStr, ok := parsed["data"].(string); !ok || dataStr != tt.request.Data {
				t.Errorf("Data field mismatch: got %v, want %s", parsed["data"], tt.request.Data)
			}

			// 验证编码字段
			if encoding, ok := parsed["data_encoding"].(string); !ok || encoding != tt.request.DataEncoding {
				t.Errorf("DataEncoding field mismatch: got %v, want %s", parsed["data_encoding"], tt.request.DataEncoding)
			}

			// 验证摘要字段
			if tt.request.Summary != nil {
				summary, ok := parsed["summary"].(map[string]interface{})
				if !ok {
					t.Error("Summary field missing or wrong type")
				}
				if summary["type"] != tt.request.Summary.Type {
					t.Errorf("Summary type mismatch: got %v, want %s", summary["type"], tt.request.Summary.Type)
				}
			}

			// 验证回调字段
			if tt.request.CallbackURL != "" {
				if callback, ok := parsed["callback_url"].(string); !ok || callback != tt.request.CallbackURL {
					t.Errorf("CallbackURL field mismatch: got %v, want %s", parsed["callback_url"], tt.request.CallbackURL)
				}
			}
		})
	}
}

func TestNewSignRequest(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		encoding DataEncoding
		expected *SignRequest
	}{
		{
			name:     "plain encoding",
			data:     []byte("test data"),
			encoding: DataEncodingPlain,
			expected: &SignRequest{
				Data:         "test data",
				DataEncoding: "PLAIN",
			},
		},
		{
			name:     "hex encoding",
			data:     []byte{0x12, 0x34, 0x56}, // 原始字节数据
			encoding: DataEncodingHex,
			expected: &SignRequest{
				Data:         "123456", // HEX编码，不带0x前缀
				DataEncoding: "HEX",
			},
		},
		{
			name:     "base64 encoding",
			data:     []byte("test data"), // 原始字符串数据
			encoding: DataEncodingBase64,
			expected: &SignRequest{
				Data:         "dGVzdCBkYXRh", // Base64编码结果
				DataEncoding: "BASE64",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewSignRequest(tt.data, tt.encoding)
			if req.Data != tt.expected.Data {
				t.Errorf("Data mismatch: got %s, want %s", req.Data, tt.expected.Data)
			}
			if req.DataEncoding != tt.expected.DataEncoding {
				t.Errorf("DataEncoding mismatch: got %s, want %s", req.DataEncoding, tt.expected.DataEncoding)
			}
		})
	}
}

func TestWithSummary(t *testing.T) {
	req := NewSignRequest([]byte("test"), DataEncodingPlain)
	summary := &SignSummary{
		Type:   "TRANSFER",
		From:   "0x1111",
		To:     "0x2222",
		Amount: "1.0",
		Token:  "ETH",
	}

	req.WithSummary(summary)

	if req.Summary != summary {
		t.Error("Summary not set correctly")
	}
	if req.Summary.Type != "TRANSFER" {
		t.Errorf("Summary type mismatch: got %s, want TRANSFER", req.Summary.Type)
	}
}

func TestWithCallbackURL(t *testing.T) {
	req := NewSignRequest([]byte("test"), DataEncodingPlain)
	callbackURL := "https://example.com/callback"

	req.WithCallbackURL(callbackURL)

	if req.CallbackURL != callbackURL {
		t.Errorf("CallbackURL mismatch: got %s, want %s", req.CallbackURL, callbackURL)
	}
}
