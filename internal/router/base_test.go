package router

import (
	"encoding/json"
	"testing"

	"github.com/mowind/web3signer-go/internal/jsonrpc"
	"github.com/sirupsen/logrus"
)

func TestBaseHandler_Method(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test_method", logger)

	if handler.Method() != "test_method" {
		t.Errorf("Expected method 'test_method', got '%s'", handler.Method())
	}
}

func TestBaseHandler_ValidateParams(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test", logger)

	testCases := []struct {
		name           string
		params         json.RawMessage
		expectedLength int
		expectError    bool
		expectedResult []interface{}
	}{
		{
			name:           "valid params",
			params:         json.RawMessage(`["param1", "param2", "param3"]`),
			expectedLength: 3,
			expectError:    false,
			expectedResult: []interface{}{"param1", "param2", "param3"},
		},
		{
			name:           "empty params",
			params:         json.RawMessage(`[]`),
			expectedLength: 0,
			expectError:    false,
			expectedResult: []interface{}{},
		},
		{
			name:           "wrong length",
			params:         json.RawMessage(`["param1", "param2"]`),
			expectedLength: 3,
			expectError:    true,
		},
		{
			name:           "not array",
			params:         json.RawMessage(`{"key": "value"}`),
			expectedLength: 1,
			expectError:    true,
		},
		{
			name:           "empty params message",
			params:         json.RawMessage(``),
			expectedLength: 1,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := handler.ValidateParams(tc.params, tc.expectedLength)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tc.expectedResult) {
				t.Errorf("Expected %d params, got %d", len(tc.expectedResult), len(result))
				return
			}

			for i, expected := range tc.expectedResult {
				if result[i] != expected {
					t.Errorf("Param %d: expected %v, got %v", i, expected, result[i])
				}
			}
		})
	}
}

func TestBaseHandler_ParseParams(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test", logger)

	type TestParams struct {
		From  string `json:"from"`
		To    string `json:"to"`
		Value string `json:"value"`
	}

	testCases := []struct {
		name        string
		params      json.RawMessage
		target      interface{}
		expectError bool
		validate    func(t *testing.T, target interface{})
	}{
		{
			name:        "array format",
			params:      json.RawMessage(`[{"from": "0x123", "to": "0x456", "value": "100"}]`),
			target:      &TestParams{},
			expectError: false,
			validate: func(t *testing.T, target interface{}) {
				params := target.(*TestParams)
				if params.From != "0x123" {
					t.Errorf("Expected From '0x123', got '%s'", params.From)
				}
				if params.To != "0x456" {
					t.Errorf("Expected To '0x456', got '%s'", params.To)
				}
				if params.Value != "100" {
					t.Errorf("Expected Value '100', got '%s'", params.Value)
				}
			},
		},
		{
			name:        "object format",
			params:      json.RawMessage(`{"from": "0x789", "to": "0xabc", "value": "200"}`),
			target:      &TestParams{},
			expectError: false,
			validate: func(t *testing.T, target interface{}) {
				params := target.(*TestParams)
				if params.From != "0x789" {
					t.Errorf("Expected From '0x789', got '%s'", params.From)
				}
				if params.To != "0xabc" {
					t.Errorf("Expected To '0xabc', got '%s'", params.To)
				}
				if params.Value != "200" {
					t.Errorf("Expected Value '200', got '%s'", params.Value)
				}
			},
		},
		{
			name:        "empty params",
			params:      json.RawMessage(``),
			target:      &TestParams{},
			expectError: true,
		},
		{
			name:        "invalid json",
			params:      json.RawMessage(`{invalid json}`),
			target:      &TestParams{},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := handler.ParseParams(tc.params, tc.target)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tc.validate != nil {
				tc.validate(t, tc.target)
			}
		})
	}
}

func TestBaseHandler_CreateSuccessResponse(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test", logger)

	result := map[string]interface{}{
		"status": "success",
		"data":   "test_data",
	}

	response, err := handler.CreateSuccessResponse("test_id", result)
	if err != nil {
		t.Fatalf("Failed to create success response: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got: %v", response.Error)
	}

	if response.ID != "test_id" {
		t.Errorf("Expected ID 'test_id', got '%v'", response.ID)
	}

	// 验证结果可以被正确解析
	var parsedResult map[string]interface{}
	if err := json.Unmarshal(response.Result, &parsedResult); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if parsedResult["status"] != "success" {
		t.Errorf("Expected status 'success', got '%v'", parsedResult["status"])
	}

	if parsedResult["data"] != "test_data" {
		t.Errorf("Expected data 'test_data', got '%v'", parsedResult["data"])
	}
}

func TestBaseHandler_CreateErrorResponse(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test", logger)

	response := handler.CreateErrorResponse("test_id", jsonrpc.CodeInvalidParams, "Invalid parameters", "details")

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error == nil {
		t.Fatal("Expected error in response")
	}

	if response.Error.Code != jsonrpc.CodeInvalidParams {
		t.Errorf("Expected error code %d, got %d", jsonrpc.CodeInvalidParams, response.Error.Code)
	}

	if response.Error.Message != "Invalid parameters" {
		t.Errorf("Expected error message 'Invalid parameters', got '%s'", response.Error.Message)
	}

	if response.Error.Data != "details" {
		t.Errorf("Expected error data 'details', got '%v'", response.Error.Data)
	}
}

func TestBaseHandler_CreateInvalidParamsResponse(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test", logger)

	response := handler.CreateInvalidParamsResponse("test_id", "Missing required parameter")

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Error == nil {
		t.Fatal("Expected error in response")
	}

	if response.Error.Code != jsonrpc.CodeInvalidParams {
		t.Errorf("Expected error code %d, got %d", jsonrpc.CodeInvalidParams, response.Error.Code)
	}

	if response.Error.Message != "Missing required parameter" {
		t.Errorf("Expected error message 'Missing required parameter', got '%s'", response.Error.Message)
	}
}
