package signer

import (
	"bytes"
	"context"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/mowind/web3signer-go/internal/kms"
	"github.com/sirupsen/logrus"
	"github.com/umbracle/ethgo"
)

// mockClient implements the Client interface for testing.
type mockClient struct {
	address    ethgo.Address
	signFunc   func(hash []byte) ([]byte, error)
	signTxFunc func(tx *ethgo.Transaction) (*ethgo.Transaction, error)
}

func (m *mockClient) Address() ethgo.Address {
	return m.address
}

func (m *mockClient) Sign(hash []byte) ([]byte, error) {
	if m.signFunc != nil {
		return m.signFunc(hash)
	}
	signature := make([]byte, 65)
	for i := 0; i < 65; i++ {
		signature[i] = byte(i + 1)
	}
	return signature, nil
}

func (m *mockClient) SignTransaction(tx *ethgo.Transaction) (*ethgo.Transaction, error) {
	if m.signTxFunc != nil {
		return m.signTxFunc(tx)
	}
	return tx, nil
}

func TestNewMultiKeySigner(t *testing.T) {
	defaultKeyID := "default-key"
	chainID := big.NewInt(1)
	logger := logrus.New()

	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	if signer == nil {
		t.Fatal("Expected non-nil MultiKeySigner")
	}

	if signer.defaultKeyID != defaultKeyID {
		t.Errorf("Expected defaultKeyID %s, got %s", defaultKeyID, signer.defaultKeyID)
	}

	if signer.clients == nil {
		t.Error("Expected clients map to be initialized")
	}

	if len(signer.clients) != 0 {
		t.Errorf("Expected empty clients map, got %d entries", len(signer.clients))
	}
}

func TestMultiKeySigner_AddClient(t *testing.T) {
	defaultKeyID := "default-key"
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	client1 := &mockClient{
		address: ethgo.HexToAddress("0x1234567890123456789012345678901234567890"),
	}

	if err := signer.AddClient("key-1", client1); err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	if len(signer.clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(signer.clients))
	}

	retrieved, err := signer.GetClient("key-1")
	if err != nil {
		t.Fatalf("Failed to get client: %v", err)
	}

	if retrieved != client1 {
		t.Error("Retrieved client is not the same as added client")
	}
}

func TestMultiKeySigner_AddClient_ErrorCases(t *testing.T) {
	defaultKeyID := "default-key"
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	client := &mockClient{
		address: ethgo.HexToAddress("0x1234567890123456789012345678901234567890"),
	}

	tests := []struct {
		name    string
		keyID   string
		client  Client
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty keyID",
			keyID:   "",
			client:  client,
			wantErr: true,
			errMsg:  "keyID cannot be empty",
		},
		{
			name:    "nil client",
			keyID:   "key-1",
			client:  nil,
			wantErr: true,
			errMsg:  "client cannot be nil",
		},
		{
			name:    "duplicate keyID",
			keyID:   "key-1",
			client:  client,
			wantErr: true,
			errMsg:  "already registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "duplicate keyID" {
				if err := signer.AddClient("key-1", client); err != nil {
					t.Fatalf("Failed to add initial client: %v", err)
				}
			}

			err := signer.AddClient(tt.keyID, tt.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}

			if tt.wantErr && err != nil {
				if !bytes.Contains([]byte(err.Error()), []byte(tt.errMsg)) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestMultiKeySigner_RemoveClient(t *testing.T) {
	defaultKeyID := "default-key"
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	client1 := &mockClient{
		address: ethgo.HexToAddress("0x1234567890123456789012345678901234567890"),
	}
	client2 := &mockClient{
		address: ethgo.HexToAddress("0x0987654321098765432109876543210987654321"),
	}

	if err := signer.AddClient(defaultKeyID, client1); err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}
	if err := signer.AddClient("key-2", client2); err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	err := signer.RemoveClient("key-2")
	if err != nil {
		t.Fatalf("Failed to remove client: %v", err)
	}

	if len(signer.clients) != 1 {
		t.Errorf("Expected 1 client after removal, got %d", len(signer.clients))
	}

	_, err = signer.GetClient("key-2")
	if err == nil {
		t.Error("Expected error when getting removed client")
	}
}

func TestMultiKeySigner_RemoveClient_ErrorCases(t *testing.T) {
	defaultKeyID := "default-key"
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	client := &mockClient{
		address: ethgo.HexToAddress("0x1234567890123456789012345678901234567890"),
	}
	if err := signer.AddClient(defaultKeyID, client); err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	tests := []struct {
		name    string
		keyID   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "remove default key",
			keyID:   defaultKeyID,
			wantErr: true,
			errMsg:  "cannot remove default key",
		},
		{
			name:    "remove non-existent key",
			keyID:   "non-existent",
			wantErr: true,
			errMsg:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := signer.RemoveClient(tt.keyID)
			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if !bytes.Contains([]byte(err.Error()), []byte(tt.errMsg)) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestMultiKeySigner_GetClient(t *testing.T) {
	defaultKeyID := "default-key"
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	client1 := &mockClient{
		address: ethgo.HexToAddress("0x1234567890123456789012345678901234567890"),
	}

	if err := signer.AddClient("key-1", client1); err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	retrieved, err := signer.GetClient("key-1")
	if err != nil {
		t.Fatalf("Failed to get client: %v", err)
	}

	if retrieved != client1 {
		t.Error("Retrieved client is not the same as added client")
	}
}

func TestMultiKeySigner_GetClient_NotFound(t *testing.T) {
	defaultKeyID := "default-key"
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	_, err := signer.GetClient("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent client")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("not found")) {
		t.Errorf("Expected error containing 'not found', got '%s'", err.Error())
	}
}

func TestMultiKeySigner_Address(t *testing.T) {
	defaultKeyID := "default-key"
	expectedAddress := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	client := &mockClient{
		address: expectedAddress,
	}
	if err := signer.AddClient(defaultKeyID, client); err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	address := signer.Address()
	if address != expectedAddress {
		t.Errorf("Expected address %s, got %s", expectedAddress.String(), address.String())
	}
}

func TestMultiKeySigner_Sign(t *testing.T) {
	defaultKeyID := "default-key"
	expectedAddress := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	expectedHash := make([]byte, 32)
	for i := 0; i < 32; i++ {
		expectedHash[i] = byte(i)
	}

	expectedSignature := make([]byte, 65)
	for i := 0; i < 65; i++ {
		expectedSignature[i] = byte(i + 50)
	}

	client := &mockClient{
		address: expectedAddress,
		signFunc: func(hash []byte) ([]byte, error) {
			if !bytes.Equal(hash, expectedHash) {
				t.Errorf("Expected hash %x, got %x", expectedHash, hash)
			}
			return expectedSignature, nil
		},
	}
	if err := signer.AddClient(defaultKeyID, client); err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	signature, err := signer.Sign(expectedHash)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	if !bytes.Equal(signature, expectedSignature) {
		t.Error("Returned signature does not match expected signature")
	}
}

func TestMultiKeySigner_SignTransaction(t *testing.T) {
	defaultKeyID := "default-key"
	expectedAddress := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	toAddr := ethgo.HexToAddress("0x0987654321098765432109876543210987654321")
	tx := &ethgo.Transaction{
		To:       &toAddr,
		Nonce:    5,
		GasPrice: 20000000000,
		Gas:      21000,
		Value:    big.NewInt(1000000000000000000),
		Input:    []byte{},
	}

	client := &mockClient{
		address: expectedAddress,
		signTxFunc: func(tx *ethgo.Transaction) (*ethgo.Transaction, error) {
			signedTx := tx.Copy()
			signedTx.R = []byte{1, 2, 3}
			signedTx.S = []byte{4, 5, 6}
			signedTx.V = []byte{7}
			return signedTx, nil
		},
	}
	if err := signer.AddClient(defaultKeyID, client); err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	signedTx, err := signer.SignTransaction(tx)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	if len(signedTx.R) == 0 {
		t.Error("Expected R to be set")
	}

	if len(signedTx.S) == 0 {
		t.Error("Expected S to be set")
	}

	if len(signedTx.V) == 0 {
		t.Error("Expected V to be set")
	}
}

func TestMultiKeySigner_SignTransactionWithKeyID(t *testing.T) {
	defaultKeyID := "default-key"
	defaultAddress := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	keyID1 := "key-1"
	address1 := ethgo.HexToAddress("0x1111111111111111111111111111111111111111")
	keyID2 := "key-2"
	address2 := ethgo.HexToAddress("0x2222222222222222222222222222222222222222")

	client1 := &mockClient{
		address: address1,
		signTxFunc: func(tx *ethgo.Transaction) (*ethgo.Transaction, error) {
			signedTx := tx.Copy()
			signedTx.From = address1
			signedTx.R = []byte{1, 1, 1}
			return signedTx, nil
		},
	}
	client2 := &mockClient{
		address: address2,
		signTxFunc: func(tx *ethgo.Transaction) (*ethgo.Transaction, error) {
			signedTx := tx.Copy()
			signedTx.From = address2
			signedTx.R = []byte{2, 2, 2}
			return signedTx, nil
		},
	}

	if err := signer.AddClient(defaultKeyID, &mockClient{address: defaultAddress}); err != nil {
		t.Fatalf("Failed to add default client: %v", err)
	}
	if err := signer.AddClient(keyID1, client1); err != nil {
		t.Fatalf("Failed to add client1: %v", err)
	}
	if err := signer.AddClient(keyID2, client2); err != nil {
		t.Fatalf("Failed to add client2: %v", err)
	}

	toAddr := ethgo.HexToAddress("0x0987654321098765432109876543210987654321")
	tx := &ethgo.Transaction{
		To:    &toAddr,
		Nonce: 5,
		Gas:   21000,
		Value: big.NewInt(1000000000000000000),
		Input: []byte{},
	}

	signedTx1, err := signer.SignTransactionWithKeyID(tx, keyID1)
	if err != nil {
		t.Fatalf("Failed to sign transaction with keyID %s: %v", keyID1, err)
	}

	if signedTx1.From != address1 {
		t.Errorf("Expected From address %s, got %s", address1.String(), signedTx1.From.String())
	}

	if !bytes.Equal(signedTx1.R, []byte{1, 1, 1}) {
		t.Error("Expected R from client1")
	}

	signedTx2, err := signer.SignTransactionWithKeyID(tx, keyID2)
	if err != nil {
		t.Fatalf("Failed to sign transaction with keyID %s: %v", keyID2, err)
	}

	if signedTx2.From != address2 {
		t.Errorf("Expected From address %s, got %s", address2.String(), signedTx2.From.String())
	}

	if !bytes.Equal(signedTx2.R, []byte{2, 2, 2}) {
		t.Error("Expected R from client2")
	}
}

func TestMultiKeySigner_SignTransactionWithKeyID_NotFound(t *testing.T) {
	defaultKeyID := "default-key"
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	client := &mockClient{
		address: ethgo.HexToAddress("0x1234567890123456789012345678901234567890"),
	}
	if err := signer.AddClient(defaultKeyID, client); err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	toAddr := ethgo.HexToAddress("0x0987654321098765432109876543210987654321")
	tx := &ethgo.Transaction{
		To:    &toAddr,
		Nonce: 5,
		Gas:   21000,
		Input: []byte{},
	}

	_, err := signer.SignTransactionWithKeyID(tx, "non-existent-key")
	if err == nil {
		t.Error("Expected error when signing with non-existent keyID")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("not found")) {
		t.Errorf("Expected error containing 'not found', got '%s'", err.Error())
	}
}

func TestMultiKeySigner_SignTransactionWithSummary(t *testing.T) {
	defaultKeyID := "default-key"
	address := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	expectedSummary := &kms.SignSummary{
		Type:   "TRANSFER",
		From:   address.String(),
		To:     "0x0987654321098765432109876543210987654321",
		Amount: "1000000000000000000",
		Token:  "ETH",
		Remark: "test",
	}

	kmsClient := &mockKMSClient{
		signWithOptionsFunc: func(ctx context.Context, keyID string, message []byte, encoding kms.DataEncoding, summary *kms.SignSummary, callbackURL string) ([]byte, error) {
			if summary.Type != expectedSummary.Type {
				t.Errorf("Expected summary type %s, got %s", expectedSummary.Type, summary.Type)
			}
			signature := make([]byte, 65)
			for i := 0; i < 65; i++ {
				signature[i] = byte(i + 1)
			}
			return []byte(hex.EncodeToString(signature)), nil
		},
	}

	mpcSigner := NewMPCKMSSigner(kmsClient, defaultKeyID, address, chainID)
	if err := signer.AddClient(defaultKeyID, mpcSigner); err != nil {
		t.Fatalf("Failed to add mpcSigner: %v", err)
	}

	toAddr := ethgo.HexToAddress(expectedSummary.To)
	tx := &ethgo.Transaction{
		To:    &toAddr,
		Nonce: 5,
		Gas:   21000,
		Input: []byte{},
	}

	signedTx, err := signer.SignTransactionWithSummary(tx, defaultKeyID, expectedSummary)
	if err != nil {
		t.Fatalf("Failed to sign transaction with summary: %v", err)
	}

	if signedTx == nil {
		t.Error("Expected signed transaction, got nil")
	}
}

func TestMultiKeySigner_SignTransactionWithSummary_NotMPCKMSSigner(t *testing.T) {
	defaultKeyID := "default-key"
	address := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	client := &mockClient{address: address}
	if err := signer.AddClient(defaultKeyID, client); err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	toAddr := ethgo.HexToAddress("0x0987654321098765432109876543210987654321")
	tx := &ethgo.Transaction{
		To:    &toAddr,
		Nonce: 5,
		Gas:   21000,
		Input: []byte{},
	}

	summary := &kms.SignSummary{
		Type: "TRANSFER",
	}

	_, err := signer.SignTransactionWithSummary(tx, defaultKeyID, summary)
	if err == nil {
		t.Error("Expected error when client is not MPCKMSSigner")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("does not support")) {
		t.Errorf("Expected error containing 'does not support', got '%s'", err.Error())
	}
}

func TestMultiKeySigner_CreateTransferSummary(t *testing.T) {
	defaultKeyID := "default-key"
	address := ethgo.HexToAddress("0x1234567890123456789012345678901234567890")
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	kmsClient := &mockKMSClient{}
	mpcSigner := NewMPCKMSSigner(kmsClient, defaultKeyID, address, chainID)
	if err := signer.AddClient(defaultKeyID, mpcSigner); err != nil {
		t.Fatalf("Failed to add mpcSigner: %v", err)
	}

	toAddr := ethgo.HexToAddress("0x0987654321098765432109876543210987654321")
	tx := &ethgo.Transaction{
		To:    &toAddr,
		Value: big.NewInt(500000000000000000),
		Input: []byte{},
	}

	summary, err := signer.CreateTransferSummary(tx, defaultKeyID, "ETH", "test")
	if err != nil {
		t.Fatalf("Failed to create transfer summary: %v", err)
	}

	if summary.Type != "TRANSFER" {
		t.Errorf("Expected type TRANSFER, got %s", summary.Type)
	}

	if summary.From != address.String() {
		t.Errorf("Expected from %s, got %s", address.String(), summary.From)
	}

	if summary.To != toAddr.String() {
		t.Errorf("Expected to %s, got %s", toAddr.String(), summary.To)
	}

	if summary.Amount != "500000000000000000" {
		t.Errorf("Expected amount 500000000000000000, got %s", summary.Amount)
	}

	if summary.Token != "ETH" {
		t.Errorf("Expected token ETH, got %s", summary.Token)
	}

	if summary.Remark != "test" {
		t.Errorf("Expected remark 'test', got %s", summary.Remark)
	}
}

func TestMultiKeySigner_MultipleKeys(t *testing.T) {
	defaultKeyID := "default-key"
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	addresses := []string{
		"0x1234567890123456789012345678901234567890",
		"0x1111111111111111111111111111111111111111",
		"0x2222222222222222222222222222222222222222",
		"0x3333333333333333333333333333333333333333",
	}

	keyIDs := []string{"key-1", "key-2", "key-3", "key-4"}

	for i, addr := range addresses {
		client := &mockClient{
			address: ethgo.HexToAddress(addr),
		}
		if err := signer.AddClient(keyIDs[i], client); err != nil {
			t.Fatalf("Failed to add client %s: %v", keyIDs[i], err)
		}
	}

	if len(signer.clients) != 4 {
		t.Errorf("Expected 4 clients, got %d", len(signer.clients))
	}

	for i, keyID := range keyIDs {
		client, err := signer.GetClient(keyID)
		if err != nil {
			t.Errorf("Failed to get client %s: %v", keyID, err)
		}
		if client.Address() != ethgo.HexToAddress(addresses[i]) {
			t.Errorf("Expected address %s, got %s", addresses[i], client.Address().String())
		}
	}
}
