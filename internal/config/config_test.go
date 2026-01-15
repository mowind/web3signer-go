package config

import (
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
				Endpoint:      "http://localhost:8080",
				AccessKeyID:   "ak",
				SecretKey:     "sk",
				KeyID:         "key123",
			},
			wantErr: false,
		},
		{
			name: "missing endpoint",
			config: KMSConfig{
				Endpoint:      "",
				AccessKeyID:   "ak",
				SecretKey:     "sk",
				KeyID:         "key123",
			},
			wantErr: true,
		},
		{
			name: "missing access key",
			config: KMSConfig{
				Endpoint:      "http://localhost:8080",
				AccessKeyID:   "",
				SecretKey:     "sk",
				KeyID:         "key123",
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
				HTTPHost: "localhost",
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
			Endpoint:      "http://localhost:8080",
			AccessKeyID:   "ak",
			SecretKey:     "sk",
			KeyID:         "key123",
		},
		Downstream: DownstreamConfig{
			HTTPHost: "localhost",
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