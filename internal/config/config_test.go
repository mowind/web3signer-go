package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  HTTPConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: HTTPConfig{
				Host: "localhost",
				Port: 8080,
			},
			wantErr: false,
		},
		{
			name: "valid TLS config",
			config: HTTPConfig{
				Host:        "localhost",
				Port:        8443,
				TLSCertFile: createTempFile(t, "cert.pem", []byte("cert content")),
				TLSKeyFile:  createTempFile(t, "key.pem", []byte("key content")),
			},
			wantErr: false,
		},
		{
			name: "valid TLS with auto-redirect",
			config: HTTPConfig{
				Host:            "localhost",
				Port:            8443,
				TLSCertFile:     createTempFile(t, "cert.pem", []byte("cert content")),
				TLSKeyFile:      createTempFile(t, "key.pem", []byte("key content")),
				TLSAutoRedirect: true,
			},
			wantErr: false,
		},
		{
			name: "empty host",
			config: HTTPConfig{
				Host: "",
				Port: 8080,
			},
			wantErr: true,
		},
		{
			name: "port too low",
			config: HTTPConfig{
				Host: "localhost",
				Port: 0,
			},
			wantErr: true,
		},
		{
			name: "port too high",
			config: HTTPConfig{
				Host: "localhost",
				Port: MaxPort + 1,
			},
			wantErr: true,
		},
		{
			name: "TLS cert without key",
			config: HTTPConfig{
				Host:        "localhost",
				Port:        8443,
				TLSCertFile: "/path/to/cert.pem",
				TLSKeyFile:  "",
			},
			wantErr: true,
		},
		{
			name: "TLS key without cert",
			config: HTTPConfig{
				Host:        "localhost",
				Port:        8443,
				TLSCertFile: "",
				TLSKeyFile:  "/path/to/key.pem",
			},
			wantErr: true,
		},
		{
			name: "TLS auto-redirect without cert/key",
			config: HTTPConfig{
				Host:            "localhost",
				Port:            8443,
				TLSAutoRedirect: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("HTTPConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKMSConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  KMSConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: KMSConfig{
				Endpoint:    "http://localhost:8080",
				AccessKeyID: "ak",
				SecretKey:   "sk",
				KeyID:       "key123",
				Address:     "0x1234567890123456789012345678901234567890",
			},
			wantErr: false,
		},
		{
			name: "missing endpoint",
			config: KMSConfig{
				Endpoint:    "",
				AccessKeyID: "ak",
				SecretKey:   "sk",
				KeyID:       "key123",
				Address:     "0x1234567890123456789012345678901234567890",
			},
			wantErr: true,
		},
		{
			name: "missing access key",
			config: KMSConfig{
				Endpoint:    "http://localhost:8080",
				AccessKeyID: "",
				SecretKey:   "sk",
				KeyID:       "key123",
				Address:     "0x1234567890123456789012345678901234567890",
			},
			wantErr: true,
		},
		{
			name: "missing address",
			config: KMSConfig{
				Endpoint:    "http://localhost:8080",
				AccessKeyID: "ak",
				SecretKey:   "sk",
				KeyID:       "key123",
				Address:     "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("KMSConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDownstreamConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  DownstreamConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: DownstreamConfig{
				HTTPHost: "http://localhost",
				HTTPPort: 8545,
				HTTPPath: "/",
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: DownstreamConfig{
				HTTPHost: "",
				HTTPPort: 8545,
				HTTPPath: "/",
			},
			wantErr: true,
		},
		{
			name: "missing path",
			config: DownstreamConfig{
				HTTPHost: "localhost",
				HTTPPort: 8545,
				HTTPPath: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DownstreamConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLogConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  LogConfig
		wantErr bool
	}{
		{
			name:    "valid debug",
			config:  LogConfig{Level: LogLevelDebug},
			wantErr: false,
		},
		{
			name:    "valid info",
			config:  LogConfig{Level: LogLevelInfo},
			wantErr: false,
		},
		{
			name:    "invalid level",
			config:  LogConfig{Level: "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("LogConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	validConfig := Config{
		HTTP: HTTPConfig{
			Host: "localhost",
			Port: 9000,
		},
		KMS: KMSConfig{
			Endpoint:    "http://localhost:8080",
			AccessKeyID: "ak",
			SecretKey:   "sk",
			KeyID:       "key123",
			Address:     "0x1234567890123456789012345678901234567890",
		},
		Downstream: DownstreamConfig{
			HTTPHost: "http://localhost",
			HTTPPort: 8545,
			HTTPPath: "/",
		},
		Log: LogConfig{Level: LogLevelInfo},
	}

	t.Run("valid config", func(t *testing.T) {
		cfg := validConfig
		if err := cfg.Validate(); err != nil {
			t.Errorf("Config.Validate() error = %v", err)
		}
	})

	t.Run("sets default log level", func(t *testing.T) {
		cfg := validConfig
		cfg.Log.Level = ""
		if err := cfg.Validate(); err != nil {
			t.Errorf("Config.Validate() error = %v", err)
		}
		if cfg.Log.Level != DefaultLogLevel {
			t.Errorf("expected log level %s, got %s", DefaultLogLevel, cfg.Log.Level)
		}
	})

	t.Run("invalid http config", func(t *testing.T) {
		cfg := validConfig
		cfg.HTTP.Host = ""
		if err := cfg.Validate(); err == nil {
			t.Error("expected error for invalid http config")
		}
	})
}

func TestDownstreamConfig_BuildURL(t *testing.T) {
	tests := []struct {
		name     string
		config   DownstreamConfig
		expected string
	}{
		{
			name: "basic URL with port",
			config: DownstreamConfig{
				HTTPHost: "http://localhost",
				HTTPPort: 8545,
				HTTPPath: "/api",
			},
			expected: "http://localhost:8545/api",
		},
		{
			name: "URL without port",
			config: DownstreamConfig{
				HTTPHost: "http://localhost",
				HTTPPort: 0,
				HTTPPath: "/",
			},
			expected: "http://localhost/",
		},
		{
			name: "HTTPS URL with port",
			config: DownstreamConfig{
				HTTPHost: "https://api.example.com",
				HTTPPort: 443,
				HTTPPath: "/jsonrpc",
			},
			expected: "https://api.example.com:443/jsonrpc",
		},
		{
			name: "URL already has port",
			config: DownstreamConfig{
				HTTPHost: "http://localhost:8080",
				HTTPPort: 8545, // Should be ignored
				HTTPPath: "/api",
			},
			expected: "http://localhost:8080/api",
		},
		{
			name: "URL with trailing slash in host",
			config: DownstreamConfig{
				HTTPHost: "http://localhost/",
				HTTPPort: 8545,
				HTTPPath: "/api",
			},
			expected: "http://localhost:8545/api",
		},
		{
			name: "complex path",
			config: DownstreamConfig{
				HTTPHost: "http://localhost",
				HTTPPort: 8545,
				HTTPPath: "/api/v1/jsonrpc",
			},
			expected: "http://localhost:8545/api/v1/jsonrpc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.BuildURL()
			if result != tt.expected {
				t.Errorf("BuildURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasPort(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "URL with port",
			url:      "http://localhost:8080",
			expected: true,
		},
		{
			name:     "URL without port",
			url:      "http://localhost",
			expected: false,
		},
		{
			name:     "HTTPS URL with port",
			url:      "https://api.example.com:443",
			expected: true,
		},
		{
			name:     "HTTPS URL without port",
			url:      "https://api.example.com",
			expected: false,
		},
		{
			name:     "URL with port and path",
			url:      "http://localhost:8080/api",
			expected: true, // net/url.Parse correctly extracts port 8080
		},
		{
			name:     "URL with colon in path",
			url:      "http://localhost/api:v1",
			expected: false,
		},
		{
			name:     "URL with invalid port characters",
			url:      "http://localhost:abc",
			expected: false,
		},
		{
			name:     "URL with port-like segment in path",
			url:      "http://localhost/api:8080",
			expected: false, // net/url.Parse correctly treats this as path, not port
		},
		{
			name:     "Empty URL",
			url:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasPort(tt.url)
			if result != tt.expected {
				t.Errorf("hasPort(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestConfig_String(t *testing.T) {
	config := Config{
		HTTP: HTTPConfig{
			Host: "localhost",
			Port: 9000,
		},
		KMS: KMSConfig{
			Endpoint:    "http://kms.example.com:8080",
			AccessKeyID: "test-access-key",
			SecretKey:   "test-secret-key",
			KeyID:       "test-key-id",
		},
		Downstream: DownstreamConfig{
			HTTPHost: "http://localhost",
			HTTPPort: 8545,
			HTTPPath: "/",
		},
		Log: LogConfig{Level: "info"},
	}

	result := config.String()

	// Check that sensitive information is redacted
	if strings.Contains(result, "test-access-key") {
		t.Error("String() should redact AccessKeyID")
	}
	if strings.Contains(result, "test-secret-key") {
		t.Error("String() should redact SecretKey")
	}

	// Check that non-sensitive information is included
	if !strings.Contains(result, "localhost") {
		t.Error("String() should include host information")
	}
	if !strings.Contains(result, "9000") {
		t.Error("String() should include port information")
	}
	if !strings.Contains(result, "test-key-id") {
		t.Error("String() should include KeyID")
	}
	if !strings.Contains(result, "http://kms.example.com:8080") {
		t.Error("String() should include KMS endpoint")
	}
	if !strings.Contains(result, "[REDACTED]") {
		t.Error("String() should show [REDACTED] for sensitive fields")
	}
}

func TestDownstreamConfig_Validate_MoreCases(t *testing.T) {
	tests := []struct {
		name    string
		config  DownstreamConfig
		wantErr bool
	}{
		{
			name: "invalid protocol",
			config: DownstreamConfig{
				HTTPHost: "ftp://localhost",
				HTTPPort: 8545,
				HTTPPath: "/",
			},
			wantErr: true,
		},
		{
			name: "port too high",
			config: DownstreamConfig{
				HTTPHost: "http://localhost",
				HTTPPort: MaxPort + 1,
				HTTPPath: "/",
			},
			wantErr: true,
		},
		{
			name: "negative port",
			config: DownstreamConfig{
				HTTPHost: "http://localhost",
				HTTPPort: -1,
				HTTPPath: "/",
			},
			wantErr: true,
		},
		{
			name: "path without leading slash gets fixed",
			config: DownstreamConfig{
				HTTPHost: "http://localhost",
				HTTPPort: 8545,
				HTTPPath: "api",
			},
			wantErr: false,
		},
		{
			name: "https with valid port",
			config: DownstreamConfig{
				HTTPHost: "https://localhost",
				HTTPPort: 443,
				HTTPPath: "/api",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DownstreamConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check that path gets fixed if needed
			if !tt.wantErr && tt.config.HTTPPath == "api" {
				if !strings.HasPrefix(tt.config.HTTPPath, "/") {
					t.Error("Validate() should add leading slash to path")
				}
			}
		})
	}
}

func TestKMSConfig_Validate_MoreCases(t *testing.T) {
	tests := []struct {
		name    string
		config  KMSConfig
		wantErr bool
	}{
		{
			name: "missing secret key",
			config: KMSConfig{
				Endpoint:    "http://localhost:8080",
				AccessKeyID: "ak",
				SecretKey:   "",
				KeyID:       "key123",
			},
			wantErr: true,
		},
		{
			name: "missing key id",
			config: KMSConfig{
				Endpoint:    "http://localhost:8080",
				AccessKeyID: "ak",
				SecretKey:   "sk",
				KeyID:       "",
			},
			wantErr: true,
		},
		{
			name: "all fields empty",
			config: KMSConfig{
				Endpoint:    "",
				AccessKeyID: "",
				SecretKey:   "",
				KeyID:       "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("KMSConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPConfig_Validate_TLSFileExistence(t *testing.T) {
	tests := []struct {
		name        string
		config      HTTPConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid TLS with existing files",
			config: HTTPConfig{
				Host:        "localhost",
				Port:        8443,
				TLSCertFile: createTempFile(t, "cert.pem", []byte("cert content")),
				TLSKeyFile:  createTempFile(t, "key.pem", []byte("key content")),
			},
			wantErr: false,
		},
		{
			name: "non-existent TLS cert file",
			config: HTTPConfig{
				Host:        "localhost",
				Port:        8443,
				TLSCertFile: "/nonexistent/cert.pem",
				TLSKeyFile:  "/nonexistent/key.pem",
			},
			wantErr:     true,
			errContains: "tls-cert-file does not exist",
		},
		{
			name: "non-existent TLS key file",
			config: HTTPConfig{
				Host:        "localhost",
				Port:        8443,
				TLSCertFile: createTempFile(t, "cert.pem", []byte("cert content")),
				TLSKeyFile:  "/nonexistent/key.pem",
			},
			wantErr:     true,
			errContains: "tls-key-file does not exist",
		},
		{
			name: "both TLS files exist",
			config: HTTPConfig{
				Host:        "localhost",
				Port:        8443,
				TLSCertFile: createTempFile(t, "cert.pem", []byte("cert content")),
				TLSKeyFile:  createTempFile(t, "key.pem", []byte("key content")),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("HTTPConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got %v", tt.errContains, err)
				}
			}
		})
	}
}

func createTempFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	return path
}
