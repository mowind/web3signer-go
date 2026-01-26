package signer

import (
	"encoding/json"
	"testing"

	"github.com/umbracle/ethgo"
	"github.com/valyala/fastjson"
)

func TestJSONRPCTransaction_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    JSONRPCTransaction
		wantErr bool
	}{
		{
			name: "Legacy transaction",
			input: `{
				"from": "0x1234567890123456789012345678901234567890",
				"to": "0x0987654321098765432109876543210987654321",
				"gas": "0x5208",
				"gasPrice": "0x4a817c800",
				"nonce": "0x0",
				"value": "0xde0b6b3a7640000"
			}`,
			wantErr: false,
		},
		{
			name: "EIP-1559 transaction",
			input: `{
				"from": "0x1234567890123456789012345678901234567890",
				"to": "0x0987654321098765432109876543210987654321",
				"gas": "0x5208",
				"maxFeePerGas": "0x4a817c800",
				"maxPriorityFeePerGas": "0x4a817c800",
				"nonce": "0x1",
				"value": "0xde0b6b3a7640000",
				"chainId": "0x1"
			}`,
			wantErr: false,
		},
		{
			name: "EIP-2930 transaction with access list",
			input: `{
				"from": "0x1234567890123456789012345678901234567890",
				"to": "0x0987654321098765432109876543210987654321",
				"gas": "0x5208",
				"gasPrice": "0x4a817c800",
				"nonce": "0x2",
				"chainId": "0x1",
				"accessList": [{
					"address": "0x0000000000000000000000000000000000000001",
					"storageKeys": ["0x0000000000000000000000000000000000000000000000000000000000000000"]
				}]
			}`,
			wantErr: false,
		},
		{
			name: "Contract creation (to is null)",
			input: `{
				"from": "0x1234567890123456789012345678901234567890",
				"gas": "0x186a0",
				"gasPrice": "0x4a817c800",
				"nonce": "0x3",
				"value": "0x0",
				"data": "0x60606040526000"
			}`,
			wantErr: false,
		},
		{
			name: "Transaction with data",
			input: `{
				"from": "0x1234567890123456789012345678901234567890",
				"to": "0x0987654321098765432109876543210987654321",
				"gas": "0x5208",
				"gasPrice": "0x4a817c800",
				"nonce": "0x4",
				"value": "0x0",
				"data": "0xa9059cbb00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000abcd"
			}`,
			wantErr: false,
		},
		{
			name: "Missing required gas field",
			input: `{
				"from": "0x1234567890123456789012345678901234567890",
				"to": "0x0987654321098765432109876543210987654321",
				"gasPrice": "0x4a817c800",
				"nonce": "0x5"
			}`,
			wantErr: true,
		},
		{
			name: "Invalid hex format (no 0x prefix)",
			input: `{
				"from": "0x1234567890123456789012345678901234567890",
				"to": "0x0987654321098765432109876543210987654321",
				"gas": "5208"
			}`,
			wantErr: true,
		},
		{
			name: "Transaction with explicit type field",
			input: `{
				"from": "0x1234567890123456789012345678901234567890",
				"to": "0x0987654321098765432109876543210987654321",
				"type": "0x2",
				"gas": "0x5208",
				"maxFeePerGas": "0x4a817c800",
				"maxPriorityFeePerGas": "0x4a817c800",
				"nonce": "0x6",
				"chainId": "0x1"
			}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got JSONRPCTransaction
			err := json.Unmarshal([]byte(tt.input), &got)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify basic fields are decoded
				if got.Gas == 0 && tt.name != "Contract creation (to is null)" {
					t.Error("Gas field not decoded properly")
				}

				// Verify transaction type is set
				if got.Type < ethgo.TransactionLegacy || got.Type > ethgo.TransactionDynamicFee {
					t.Error("Transaction type not set correctly")
				}

				// Verify optional fields when they exist
				if tt.name == "Transaction with data" && len(got.Input) == 0 {
					t.Error("Data field not decoded")
				}

				if tt.name == "Legacy transaction" && got.GasPrice == 0 {
					t.Error("GasPrice not decoded for legacy transaction")
				}

				if tt.name == "EIP-1559 transaction" && (got.MaxFeePerGas == nil || got.MaxPriorityFeePerGas == nil) {
					t.Error("EIP-1559 fields not decoded")
				}

				if tt.name == "Contract creation (to is null)" && got.To != nil {
					t.Error("To should be nil for contract creation")
				}
			}
		})
	}
}

func TestParseJSONRPCTransaction(t *testing.T) {
	tests := []struct {
		name    string
		params  string
		wantErr bool
	}{
		{
			name: "Array format",
			params: `[{
				"from": "0x1234567890123456789012345678901234567890",
				"to": "0x0987654321098765432109876543210987654321",
				"gas": "0x5208",
				"gasPrice": "0x4a817c800",
				"nonce": "0x0"
			}]`,
			wantErr: false,
		},
		{
			name: "Object format",
			params: `{
				"from": "0x1234567890123456789012345678901234567890",
				"to": "0x0987654321098765432109876543210987654321",
				"gas": "0x5208",
				"gasPrice": "0x4a817c800",
				"nonce": "0x0"
			}`,
			wantErr: false,
		},
		{
			name:    "Empty params",
			params:  `[]`,
			wantErr: true,
		},
		{
			name:    "Invalid JSON",
			params:  `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseJSONRPCTransaction([]byte(tt.params))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseJSONRPCTransaction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecodeUint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		key     string
		want    uint64
		wantErr bool
	}{
		{
			name:  "Valid hex",
			input: `{"value": "0x5208"}`,
			key:   "value",
			want:  21000,
		},
		{
			name:  "Zero value",
			input: `{"value": "0x0"}`,
			key:   "value",
			want:  0,
		},
		{
			name:  "Large number",
			input: `{"value": "0xffffffffffffffff"}`,
			key:   "value",
			want:  18446744073709551615,
		},
		{
			name:    "Invalid hex",
			input:   `{"value": "0xxyz"}`,
			key:     "value",
			wantErr: true,
		},
		{
			name:    "No 0x prefix",
			input:   `{"value": "5208"}`,
			key:     "value",
			wantErr: true,
		},
		{
			name:    "Field not found",
			input:   `{"other": "0x123"}`,
			key:     "value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pool fastjsonParserPool
			p, _ := pool.Get().Parse(tt.input)
			got, err := decodeUint(p, tt.key)

			if (err != nil) != tt.wantErr {
				t.Errorf("decodeUint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("decodeUint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecodeBytes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		key     string
		want    []byte
		wantErr bool
	}{
		{
			name:  "Valid bytes",
			input: `{"data": "0xa9059cbb"}`,
			key:   "data",
			want:  []byte{0xa9, 0x05, 0x9c, 0xbb},
		},
		{
			name:  "Empty data",
			input: `{"data": "0x"}`,
			key:   "data",
			want:  []byte{},
		},
		{
			name:    "No 0x prefix",
			input:   `{"data": "a9059cbb"}`,
			key:     "data",
			wantErr: true,
		},
		{
			name:    "Invalid hex",
			input:   `{"data": "0xxyz"}`,
			key:     "data",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pool fastjsonParserPool
			p, _ := pool.Get().Parse(tt.input)
			got, err := decodeBytes(nil, p, tt.key)

			if (err != nil) != tt.wantErr {
				t.Errorf("decodeBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && string(got) != string(tt.want) {
				t.Errorf("decodeBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsKeySet(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		key    string
		result bool
	}{
		{
			name:   "Key exists with value",
			input:  `{"gas": "0x5208"}`,
			key:    "gas",
			result: true,
		},
		{
			name:   "Key is null",
			input:  `{"to": null}`,
			key:    "to",
			result: false,
		},
		{
			name:   "Key does not exist",
			input:  `{"gas": "0x5208"}`,
			key:    "value",
			result: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pool fastjsonParserPool
			p, _ := pool.Get().Parse(tt.input)
			got := isKeySet(p, tt.key)

			if got != tt.result {
				t.Errorf("isKeySet() = %v, want %v", got, tt.result)
			}
		})
	}
}

type fastjsonParserPool struct{}

func (p *fastjsonParserPool) Get() *fastjson.Parser {
	return &fastjson.Parser{}
}

func (p *fastjsonParserPool) Put(parser *fastjson.Parser) {
	// Parser is not pooled in tests
}

func TestIsValidEthAddress(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want bool
	}{
		{
			name: "Valid address with mixed case",
			addr: "0x1234567890123456789012345678901234567890",
			want: true,
		},
		{
			name: "Valid address all lowercase",
			addr: "0x1234567890123456789012345678901234567890",
			want: true,
		},
		{
			name: "Valid address all uppercase",
			addr: "0x1234567890123456789012345678901234567890",
			want: true,
		},
		{
			name: "Empty string",
			addr: "",
			want: false,
		},
		{
			name: "Missing 0x prefix",
			addr: "1234567890123456789012345678901234567890",
			want: false,
		},
		{
			name: "Wrong length - too short",
			addr: "0x123456789012345678901234567890123456789",
			want: false,
		},
		{
			name: "Wrong length - too long",
			addr: "0x12345678901234567890123456789012345678900",
			want: false,
		},
		{
			name: "Invalid hex character - lowercase g",
			addr: "0x123456789012345678901234567890123456789g",
			want: false,
		},
		{
			name: "Invalid hex character - uppercase G",
			addr: "0x123456789012345678901234567890123456789G",
			want: false,
		},
		{
			name: "Invalid hex character - special char",
			addr: "0x123456789012345678901234567890123456789!",
			want: false,
		},
		{
			name: "Invalid hex character - space",
			addr: "0x12345678901234567890123456789012345678 ",
			want: false,
		},
		{
			name: "Wrong prefix - 0X (uppercase)",
			addr: "0X1234567890123456789012345678901234567890",
			want: false,
		},
		{
			name: "Only 0x prefix",
			addr: "0x",
			want: false,
		},
		{
			name: "Valid zeros address",
			addr: "0x0000000000000000000000000000000000000000",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidEthAddress(tt.addr)
			if got != tt.want {
				t.Errorf("isValidEthAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecodeAddrValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		key     string
		wantErr bool
	}{
		{
			name:    "Valid address",
			input:   `{"addr": "0x1234567890123456789012345678901234567890"}`,
			key:     "addr",
			wantErr: false,
		},
		{
			name:    "Empty address",
			input:   `{"addr": ""}`,
			key:     "addr",
			wantErr: true,
		},
		{
			name:    "Missing 0x prefix",
			input:   `{"addr": "1234567890123456789012345678901234567890"}`,
			key:     "addr",
			wantErr: true,
		},
		{
			name:    "Wrong length - too short",
			input:   `{"addr": "0x123456789012345678901234567890123456789"}`,
			key:     "addr",
			wantErr: true,
		},
		{
			name:    "Invalid hex character",
			input:   `{"addr": "0x123456789012345678901234567890123456789g"}`,
			key:     "addr",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pool fastjsonParserPool
			p, _ := pool.Get().Parse(tt.input)

			var addr ethgo.Address
			err := decodeAddr(&addr, p, tt.key)

			if (err != nil) != tt.wantErr {
				t.Errorf("decodeAddr() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
