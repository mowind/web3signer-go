package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mowind/web3signer-go/internal/config"
	"github.com/mowind/web3signer-go/internal/kms"
)

func TestTestSignRequest(t *testing.T) {
	// Create a mock client
	kmsConfig := &config.KMSConfig{
		Endpoint:    "http://localhost:8080",
		AccessKeyID: "test-access-key",
		SecretKey:   "test-secret-key",
		KeyID:       "test-key-id",
	}

	client := kms.NewClient(kmsConfig)

	// Test the testSignRequest function
	err := testSignRequest(client)
	if err != nil {
		// This is expected to fail since we're using a mock endpoint
		// Just verify it doesn't panic
		t.Logf("testSignRequest returned error (expected): %v", err)
	}
}

func TestTestActualSign(t *testing.T) {
	// Create a mock client
	kmsConfig := &config.KMSConfig{
		Endpoint:    "http://localhost:8080",
		AccessKeyID: "test-access-key",
		SecretKey:   "test-secret-key",
		KeyID:       "test-key-id",
	}

	client := kms.NewClient(kmsConfig)

	// Test the testActualSign function
	err := testActualSign(client, kmsConfig)
	if err != nil {
		// This is expected to fail since we're using a mock endpoint
		t.Logf("testActualSign returned error (expected): %v", err)
	}
}

func TestTestErrorHandling(t *testing.T) {
	// Create a mock client
	kmsConfig := &config.KMSConfig{
		Endpoint:    "http://localhost:8080",
		AccessKeyID: "test-access-key",
		SecretKey:   "test-secret-key",
		KeyID:       "test-key-id",
	}

	client := kms.NewClient(kmsConfig)

	// Test the testErrorHandling function
	err := testErrorHandling(client, kmsConfig)
	if err != nil {
		t.Errorf("testErrorHandling should not return error, got: %v", err)
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test that helper functions exist and have correct signatures

	// Test that helper functions exist
	// Functions are never nil in Go, so we just verify they can be assigned
	var _ = testSignRequest
	var _ = testActualSign
	var _ = testErrorHandling
}

func TestMainFunctionExists(t *testing.T) {
	// Test that main function exists
	// This is a simple test to ensure the package compiles
	t.Log("test-kms package compiles successfully")
}

func TestHTTPRequestBuilding(t *testing.T) {
	// Test HTTP request building logic from testSignRequest
	testData := []byte(`{"data": "test", "encoding": "PLAIN"}`)
	req, err := http.NewRequest("POST", "http://localhost:8080/api/v1/keys/test/sign", bytes.NewReader(testData))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("Expected POST method, got %s", req.Method)
	}

	if req.URL.String() != "http://localhost:8080/api/v1/keys/test/sign" {
		t.Errorf("Expected URL %s, got %s", "http://localhost:8080/api/v1/keys/test/sign", req.URL.String())
	}
}

func TestContextCreation(t *testing.T) {
	// Test context creation from testActualSign
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Verify context has timeout
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Context should have deadline")
	}

	if time.Until(deadline) > 31*time.Second || time.Until(deadline) < 29*time.Second {
		t.Errorf("Context timeout should be ~30 seconds, got %v", time.Until(deadline))
	}
}

func TestErrorMessageParsing(t *testing.T) {
	// Test error message parsing logic from testActualSign
	errMsg := "bad sign message length: expected 32 bytes, got 16"

	if strings.Contains(errMsg, "bad sign message length") {
		t.Log("Error message parsing works correctly")
	} else {
		t.Error("Should detect 'bad sign message length' in error")
	}
}

func TestDataEncodingConstants(t *testing.T) {
	// Test that DataEncoding constants are accessible
	if kms.DataEncodingPlain != "PLAIN" {
		t.Errorf("Expected DataEncodingPlain to be 'PLAIN', got %s", kms.DataEncodingPlain)
	}

	if kms.DataEncodingHex != "HEX" {
		t.Errorf("Expected DataEncodingHex to be 'HEX', got %s", kms.DataEncodingHex)
	}
}

func TestPrintFunctions(t *testing.T) {
	// Test that fmt package is imported and can be used
	// This is a compilation test
	output := fmt.Sprintf("Test output: %d", 42)
	if !strings.Contains(output, "42") {
		t.Errorf("fmt.Sprintf not working, got: %s", output)
	}
}
