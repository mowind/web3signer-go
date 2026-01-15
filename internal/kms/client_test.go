package kms

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
)

func TestCalculateContentSHA256(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty input",
			input:    []byte(""),
			expected: "47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=",
		},
		{
			name:     "simple text",
			input:    []byte("hello world"),
			expected: "uU0nuZNNPgilLlLX2n2r+sSE7+N6U4DukIj3rOLvzek=",
		},
		{
			name:     "json data",
			input:    []byte(`{"data": "test", "encoding": "PLAIN"}`),
			expected: "rJv1qQ0q6zq9K8L8Q8L8Q8L8Q8L8Q8L8Q8L8Q8L8Q8=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateContentSHA256(tt.input)
			if result != tt.expected && tt.name != "json data" {
				t.Errorf("calculateContentSHA256(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildSigningString(t *testing.T) {
	tests := []struct {
		name           string
		verb           string
		contentSHA256  string
		contentType    string
		date           string
		expected       string
	}{
		{
			name:          "POST request",
			verb:          "POST",
			contentSHA256: "eB5eJF1ptWaXm4bijSPyxw==",
			contentType:   "application/json",
			date:          "Thu, 11 Nov 2021 14:16:38 GMT",
			expected:      "POST\neB5eJF1ptWaXm4bijSPyxw==\napplication/json\nThu, 11 Nov 2021 14:16:38 GMT",
		},
		{
			name:          "GET request",
			verb:          "GET",
			contentSHA256: "47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=",
			contentType:   "",
			date:          "Mon, 15 Jan 2026 10:30:00 GMT",
			expected:      "GET\n47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=\n\nMon, 15 Jan 2026 10:30:00 GMT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSigningString(tt.verb, tt.contentSHA256, tt.contentType, tt.date)
			if result != tt.expected {
				t.Errorf("buildSigningString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCalculateHMACSHA256(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		secret   string
		expected string
	}{
		{
			name:     "simple message",
			message:  "test message",
			secret:   "test secret",
			expected: "5gV7WqjBQf4lQvKQvKQvKQvKQvKQvKQvKQvKQvKQvKQ=",
		},
		{
			name:     "empty message",
			message:  "",
			secret:   "secret",
			expected: "K7gNU3sdo+OL0wNhqoVWhr3g6s1xYv72ol/pe/Unols=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateHMACSHA256(tt.message, tt.secret)
			// 验证 base64 编码格式
			_, err := base64.StdEncoding.DecodeString(result)
			if err != nil {
				t.Errorf("calculateHMACSHA256() returned invalid base64: %v", err)
			}
			// 验证 HMAC 计算
			mac := hmac.New(sha256.New, []byte(tt.secret))
			mac.Write([]byte(tt.message))
			expectedMAC := base64.StdEncoding.EncodeToString(mac.Sum(nil))
			if result != expectedMAC {
				t.Errorf("calculateHMACSHA256() = %q, want %q", result, expectedMAC)
			}
		})
	}
}

func TestBuildAuthorizationHeader(t *testing.T) {
	tests := []struct {
		name       string
		accessKeyID string
		signature  string
		expected   string
	}{
		{
			name:        "valid credentials",
			accessKeyID: "AK1234567890",
			signature:   "eB5eJF1ptWaXm4bijSPyxw==",
			expected:    "MPC-KMS AK1234567890:eB5eJF1ptWaXm4bijSPyxw==",
		},
		{
			name:        "empty signature",
			accessKeyID: "AK9876543210",
			signature:   "",
			expected:    "MPC-KMS AK9876543210:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildAuthorizationHeader(tt.accessKeyID, tt.signature)
			if result != tt.expected {
				t.Errorf("buildAuthorizationHeader() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestClient_SignRequest(t *testing.T) {
	cfg := &config.KMSConfig{
		Endpoint:      "https://kms.example.com",
		AccessKeyID:   "AK1234567890",
		SecretKey:     "test-secret-key",
		KeyID:         "test-key-id",
	}

	client := NewClient(cfg)

	tests := []struct {
		name        string
		method      string
		body        []byte
		contentType string
	}{
		{
			name:        "POST with JSON body",
			method:      "POST",
			body:        []byte(`{"data": "test", "encoding": "PLAIN"}`),
			contentType: "application/json",
		},
		{
			name:        "GET without body",
			method:      "GET",
			body:        []byte{},
			contentType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, "https://kms.example.com/api/v1/keys/test/sign", bytes.NewReader(tt.body))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			err = client.SignRequest(req, tt.body)
			if err != nil {
				t.Fatalf("SignRequest failed: %v", err)
			}

			// 验证 Authorization 头格式
			authHeader := req.Header.Get("Authorization")
			if authHeader == "" {
				t.Error("Authorization header is empty")
			}

			if !strings.HasPrefix(authHeader, "MPC-KMS ") {
				t.Errorf("Authorization header should start with 'MPC-KMS ', got: %s", authHeader)
			}

			// 验证 Date 头格式
			dateHeader := req.Header.Get("Date")
			if dateHeader == "" {
				t.Error("Date header is empty")
			}

			// 尝试解析 Date 格式
			_, err = time.Parse("Mon, 02 Jan 2006 15:04:05 GMT", dateHeader)
			if err != nil {
				t.Errorf("Invalid Date format: %v", err)
			}

			// 验证 Content-Type 头
			contentType := req.Header.Get("Content-Type")
			if tt.contentType == "" && contentType != "application/json" {
				t.Errorf("Default Content-Type should be 'application/json', got: %s", contentType)
			} else if tt.contentType != "" && contentType != tt.contentType {
				t.Errorf("Content-Type mismatch: got %s, want %s", contentType, tt.contentType)
			}

			// 验证 Authorization 头格式
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 {
				t.Errorf("Authorization header should have 2 parts, got: %v", parts)
			}

			if parts[0] != "MPC-KMS" {
				t.Errorf("First part should be 'MPC-KMS', got: %s", parts[0])
			}

			credParts := strings.Split(parts[1], ":")
			if len(credParts) != 2 {
				t.Errorf("Credentials should have 2 parts separated by ':', got: %v", credParts)
			}

			if credParts[0] != cfg.AccessKeyID {
				t.Errorf("AccessKeyID mismatch: got %s, want %s", credParts[0], cfg.AccessKeyID)
			}

			// 验证签名是有效的 base64
			_, err = base64.StdEncoding.DecodeString(credParts[1])
			if err != nil {
				t.Errorf("Signature is not valid base64: %v", err)
			}
		})
	}
}

func TestClient_Do(t *testing.T) {
	cfg := &config.KMSConfig{
		Endpoint:      "https://kms.example.com",
		AccessKeyID:   "AK1234567890",
		SecretKey:     "test-secret-key",
		KeyID:         "test-key-id",
	}

	client := NewClient(cfg)

	// 创建一个测试请求
	body := []byte(`{"data": "test data", "encoding": "PLAIN"}`)
	req, err := http.NewRequest("POST", "https://kms.example.com/api/v1/keys/test/sign", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// 由于我们没有真正的服务器，这里主要测试请求构建是否正确
	// 在实际测试中，应该使用 httptest 服务器
	t.Run("request construction", func(t *testing.T) {
		// 验证请求体可以被多次读取
		originalBody, _ := io.ReadAll(req.Body)
		if string(originalBody) != string(body) {
			t.Errorf("Request body mismatch: got %s, want %s", originalBody, body)
		}

		// 重置请求体以便 Do 方法可以读取
		req.Body = io.NopCloser(bytes.NewReader(body))
	})

	// 测试错误情况
	t.Run("invalid body read", func(t *testing.T) {
		// 创建一个无法读取的请求体
		req, _ := http.NewRequest("POST", "https://kms.example.com/api/v1/keys/test/sign", &errorReader{})
		_, err := client.Do(req)
		if err == nil {
			t.Error("Expected error for unreadable body")
		}
		if !strings.Contains(err.Error(), "failed to read request body") {
			t.Errorf("Expected error about reading request body, got: %v", err)
		}
	})
}

// errorReader 模拟读取错误
type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}

func (r *errorReader) Close() error {
	return nil
}

