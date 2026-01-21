# web3signer-go - Agent Development Guide

## Build & Test Commands

### Building
```bash
make build                       # Build all binaries
go build -o build/web3signer ./cmd/web3signer
```

### Testing
```bash
make test                        # Run all tests
go test -v -run TestFunc ./path  # Run single test
go test -v ./internal/kms/...    # Package tests
go test -race ./...              # With race detector
make test-coverage               # With coverage
make integration-test            # Integration tests
make coverage                    # HTML coverage report
```

### Linting & Formatting
```bash
make fmt                         # Format code (goimports)
make lint                        # Run golangci-lint
make vet                         # Vet code
make tidy                        # Tidy dependencies
make check                       # Test + lint
```

## Code Style Guidelines

### File Structure
```
internal/<module>/
├── <module>.go          # Main types
├── <module>_test.go     # Unit tests
└── README.md            # Module docs
```

### Imports
- Order: stdlib, third-party, internal
- Use `goimports` via `make fmt`

### Naming Conventions
- **Packages**: lowercase, single word, no underscores (`kms`, `signer`)
- **Types**: PascalCase (`KMSConfig`, `MPCKMSSigner`)
- **Interfaces**: PascalCase, end with `Interface` for single-method (`ClientInterface`)
- **Functions**: PascalCase (exported), camelCase (private)
- **Variables**: camelCase

### Error Handling
- Wrap with context: `fmt.Errorf("failed to do X: %w", err)`
- Error messages: lowercase, no trailing punctuation
- Validate early, fail fast

### Logging
- Use `logrus` (structured)
- Debug: request/response bodies
- Info: important events
- **Never log**: keys, secrets, signatures

### Testing
- Table-driven tests for multiple cases
- Mock HTTP with `httptest`
- `t.Skip()` with clear reason
- Test coverage required for new features

### Configuration
- Viper: CLI flags > env vars > config file > defaults
- Env var prefix: `WEB3SIGNER_`
- Config file: `~/.web3signer.yaml` (YAML)

### Key Dependencies
- `gin-gonic/gin` - HTTP server
- `logrus` - Logging
- `ethgo` - Ethereum utilities
- `cobra` - CLI framework
- `viper` - Config management

### Commit Messages
Follow Conventional Commits:
- `feat(kms): add support for X`
- `fix(signer): correct hash calculation`
- `test(router): add integration test`
- `docs(readme): update instructions`

### Common Patterns
```go
// Constructor
func NewClient(cfg *Config) *Client {
    return &Client{config: cfg}
}

// Validation
func (c *Config) Validate() error {
    if c.Endpoint == "" {
        return fmt.Errorf("endpoint is required")
    }
    return nil
}

// Error wrapping
result, err := doSomething()
if err != nil {
    return nil, fmt.Errorf("failed: %w", err)
}

// Context with timeout
ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
defer cancel()
```

### Known Issues / TODOs
- `trimBytesZeros` in signer.go (line 197-215): Should be at KMS client layer
- `hasPort` in config.go: Should use `url.Parse()` instead of string manipulation

### Architecture
- **Router**: JSON-RPC routing (sign vs forward)
- **Signer**: Implements `ethgo.Key`, delegates to KMS
- **KMS**: HTTP client with HMAC-SHA256 auth
- **Downstream**: Transparent proxy to Ethereum node
- **Server**: Gin-based with CORS support

### Before Submitting
1. Run `make check` (test + lint)
2. Ensure new code has tests
3. Update documentation if needed
4. No sensitive data in logs
5. Clear, actionable error messages
