package signer

import (
	"fmt"
	"math/big"

	"github.com/mowind/web3signer-go/internal/kms"
	"github.com/sirupsen/logrus"
	"github.com/umbracle/ethgo"
)

// Client is an interface for signing operations.
// This matches the ethgo.Key interface, allowing us to use
// MPCKMSSigner or any other signer implementation.
type Client interface {
	Address() ethgo.Address
	Sign(hash []byte) ([]byte, error)
	SignTransaction(tx *ethgo.Transaction) (*ethgo.Transaction, error)
}

// MultiKeySigner manages multiple KMS clients with dynamic key selection.
//
// This signer implements the ethgo.Key interface and allows:
//   - Multiple key IDs to be registered with their respective clients
//   - Dynamic addition and removal of keys
//   - A default key for backward compatibility
//   - Per-transaction key selection via SignTransactionWithKeyID
type MultiKeySigner struct {
	clients      map[string]Client // keyID -> Client mapping
	defaultKeyID string            // default key ID for backward compatibility
	logger       *logrus.Logger
	chainID      *big.Int
}

// NewMultiKeySigner creates a new MultiKeySigner instance.
//
// Parameters:
//   - defaultKeyID: The default key ID to use when no specific key is provided
//   - chainID: The chain ID for transaction signing (for EIP-1559)
//   - logger: Logger for operation tracking
//
// Returns:
//   - *MultiKeySigner: A new MultiKeySigner instance ready for client registration
func NewMultiKeySigner(defaultKeyID string, chainID *big.Int, logger *logrus.Logger) *MultiKeySigner {
	return &MultiKeySigner{
		clients:      make(map[string]Client),
		defaultKeyID: defaultKeyID,
		logger:       logger,
		chainID:      chainID,
	}
}

// AddClient registers a new KMS client for a specific key ID.
//
// Parameters:
//   - keyID: The KMS key identifier to associate with this client
//   - client: The signing client to register (must implement Client interface)
//
// Returns:
//   - error: An error if keyID is empty or client is nil, or if keyID already exists
func (m *MultiKeySigner) AddClient(keyID string, client Client) error {
	if keyID == "" {
		return fmt.Errorf("keyID cannot be empty")
	}
	if client == nil {
		return fmt.Errorf("client cannot be nil")
	}

	if _, exists := m.clients[keyID]; exists {
		return fmt.Errorf("keyID %s already registered", keyID)
	}

	m.clients[keyID] = client
	m.logger.WithField("key_id", keyID).Info("Client added to MultiKeySigner")

	return nil
}

// RemoveClient removes a KMS client from the registry.
//
// Parameters:
//   - keyID: The KMS key identifier to remove
//
// Returns:
//   - error: An error if keyID is not found or is the default key
func (m *MultiKeySigner) RemoveClient(keyID string) error {
	if keyID == m.defaultKeyID {
		return fmt.Errorf("cannot remove default keyID: %s", keyID)
	}

	if _, exists := m.clients[keyID]; !exists {
		return fmt.Errorf("keyID %s not found", keyID)
	}

	delete(m.clients, keyID)
	m.logger.WithField("key_id", keyID).Info("Client removed from MultiKeySigner")

	return nil
}

// GetClient retrieves a registered client by key ID.
//
// Parameters:
//   - keyID: The KMS key identifier to look up
//
// Returns:
//   - Client: The registered client
//   - error: An error if keyID is not found
func (m *MultiKeySigner) GetClient(keyID string) (Client, error) {
	client, exists := m.clients[keyID]
	if !exists {
		return nil, fmt.Errorf("keyID %s not found", keyID)
	}
	return client, nil
}

// Address returns the default key's Ethereum address.
//
// This implements the ethgo.Key interface.
//
// Returns:
//   - ethgo.Address: The address of the default key
func (m *MultiKeySigner) Address() ethgo.Address {
	client, err := m.GetClient(m.defaultKeyID)
	if err != nil {
		m.logger.WithError(err).Error("Failed to get default client for Address")
		return ethgo.Address{}
	}
	return client.Address()
}

// Sign signs a 32-byte hash using the default key.
//
// This implements the ethgo.Key interface.
//
// Parameters:
//   - hash: 32-byte hash to sign (typically Keccak-256)
//
// Returns:
//   - []byte: The signature bytes
//   - error: An error if signing fails
func (m *MultiKeySigner) Sign(hash []byte) ([]byte, error) {
	client, err := m.GetClient(m.defaultKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default client: %w", err)
	}
	return client.Sign(hash)
}

// SignTransaction signs an Ethereum transaction using the default key.
//
// This implements the ethgo.Key interface.
//
// Parameters:
//   - tx: The transaction to sign
//
// Returns:
//   - *ethgo.Transaction: A new transaction with signature applied
//   - error: An error if signing fails
func (m *MultiKeySigner) SignTransaction(tx *ethgo.Transaction) (*ethgo.Transaction, error) {
	client, err := m.GetClient(m.defaultKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default client: %w", err)
	}
	return client.SignTransaction(tx)
}

// SignTransactionWithKeyID signs an Ethereum transaction using a specific key ID.
//
// This method enables dynamic key selection per transaction, allowing
// applications to use different keys for different operations.
//
// Parameters:
//   - tx: The transaction to sign
//   - keyID: The specific key ID to use for signing
//
// Returns:
//   - *ethgo.Transaction: A new transaction with signature applied
//   - error: An error if the keyID is not found or signing fails
func (m *MultiKeySigner) SignTransactionWithKeyID(tx *ethgo.Transaction, keyID string) (*ethgo.Transaction, error) {
	client, err := m.GetClient(keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client for keyID %s: %w", keyID, err)
	}
	return client.SignTransaction(tx)
}

// SignTransactionWithSummary signs an Ethereum transaction using a specific key ID with approval summary.
//
// This method provides support for KMS approval workflow by including transaction summary.
//
// Parameters:
//   - tx: The transaction to sign
//   - keyID: The specific key ID to use for signing
//   - summary: Transaction summary for approval display
//
// Returns:
//   - *ethgo.Transaction: A new transaction with signature applied
//   - error: An error if the keyID is not found, client is not MPCKMSSigner, or signing fails
func (m *MultiKeySigner) SignTransactionWithSummary(tx *ethgo.Transaction, keyID string, summary *kms.SignSummary) (*ethgo.Transaction, error) {
	client, err := m.GetClient(keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client for keyID %s: %w", keyID, err)
	}

	// Check if client supports SignTransactionWithSummary (MPCKMSSigner)
	mpcSigner, ok := client.(*MPCKMSSigner)
	if !ok {
		return nil, fmt.Errorf("client for keyID %s does not support SignTransactionWithSummary", keyID)
	}

	return mpcSigner.SignTransactionWithSummary(tx, summary)
}

// CreateTransferSummary creates a transfer summary from transaction details for a specific key.
//
// This method extracts relevant transaction information for approval display.
//
// Parameters:
//   - tx: The transaction to extract details from
//   - keyID: The specific key ID to use for address extraction
//   - token: The token symbol (e.g., "ETH", "USDT"). Defaults to "ETH" if empty.
//   - remark: Optional transaction description/remark
//
// Returns:
//   - *kms.SignSummary: A transaction summary for KMS approval workflow
//   - error: An error if the keyID is not found or client is not MPCKMSSigner
func (m *MultiKeySigner) CreateTransferSummary(tx *ethgo.Transaction, keyID string, token string, remark string) (*kms.SignSummary, error) {
	client, err := m.GetClient(keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client for keyID %s: %w", keyID, err)
	}

	// Check if client supports CreateTransferSummary (MPCKMSSigner)
	mpcSigner, ok := client.(*MPCKMSSigner)
	if !ok {
		return nil, fmt.Errorf("client for keyID %s does not support CreateTransferSummary", keyID)
	}

	return mpcSigner.CreateTransferSummary(tx, token, remark), nil
}

// VerifyInterface verifies that MultiKeySigner implements the required interfaces.
var _ ethgo.Key = (*MultiKeySigner)(nil)
