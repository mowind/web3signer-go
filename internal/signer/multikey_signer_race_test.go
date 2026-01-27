package signer

import (
	"fmt"
	"math/big"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/umbracle/ethgo"
)

// TestMultiKeySigner_ConcurrentAccess detects data race conditions
// when multiple goroutines access AddClient and GetClient simultaneously.
// This test should FAIL with -race flag (RED phase of TDD).
func TestMultiKeySigner_ConcurrentAccess(t *testing.T) {
	defaultKeyID := "default-key"
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	var wg sync.WaitGroup

	numWriters := 10
	numReaders := 10

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			keyID := fmt.Sprintf("key-%d", id)
			address := ethgo.HexToAddress(fmt.Sprintf("0x%040x", id+1000))
			client := &mockClient{
				address: address,
			}
			if err := signer.AddClient(keyID, client); err != nil {
				t.Errorf("Failed to add client %s: %v", keyID, err)
			}
		}(i)
	}

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			keyID := fmt.Sprintf("key-%d", id%numWriters)
			_, err := signer.GetClient(keyID)
			_ = err // Key may not exist yet due to race
		}(i)
	}

	wg.Wait()
	expectedClients := numWriters // Verifies final state after fix
	if len(signer.clients) != expectedClients {
		t.Errorf("Expected %d clients, got %d", expectedClients, len(signer.clients))
	}
}

// TestMultiKeySigner_ConcurrentAddRemove tests concurrent AddClient and RemoveClient operations.
// This should also FAIL with -race flag.
func TestMultiKeySigner_ConcurrentAddRemove(t *testing.T) {
	defaultKeyID := "default-key"
	chainID := big.NewInt(1)
	logger := logrus.New()
	signer := NewMultiKeySigner(defaultKeyID, chainID, logger)

	defaultClient := &mockClient{
		address: ethgo.HexToAddress("0x1234567890123456789012345678901234567890"),
	}
	if err := signer.AddClient(defaultKeyID, defaultClient); err != nil {
		t.Fatalf("Failed to add default client: %v", err)
	}

	var wg sync.WaitGroup

	numOperations := 10

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			keyID := fmt.Sprintf("key-%d", id)
			address := ethgo.HexToAddress(fmt.Sprintf("0x%040x", id+1000))
			client := &mockClient{
				address: address,
			}

			if err := signer.AddClient(keyID, client); err != nil {
				t.Logf("Failed to add client %s (may be duplicate): %v", keyID, err)
			}

			// Try to remove client (may fail if already removed)
			_ = signer.RemoveClient(keyID)
		}(i)
	}

	wg.Wait()
}
