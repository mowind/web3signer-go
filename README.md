# web3signer-go

A Go implementation of web3signer with MPC-KMS (Multi-Party Computation - Key Management Service) signing support.

## Overview

`web3signer-go` is inspired by [Consensys/web3signer](https://github.com/Consensys/web3signer) but specifically focuses on **MPC-KMS signing**. It provides an HTTP JSON-RPC interface that:

1. **Signs transactions using MPC-KMS** - Supports cryptographic operations via MPC-KMS service
2. **Forwards other JSON-RPC methods** - Routes non-signing requests to downstream services
3. **Provides JSON-RPC compatibility** - Implements standard Ethereum JSON-RPC methods

## Features

- ✅ **MPC-KMS Integration** - Secure multi-party computation key management
- ✅ **JSON-RPC Server** - HTTP server with JSON-RPC 2.0 support
- ✅ **Transaction Signing** - Supports `eth_sign`, `eth_signTransaction`, `eth_sendTransaction`
- ✅ **Downstream Forwarding** - Routes other methods to configured downstream services
- ✅ **Configuration Management** - CLI flags and config file support via Cobra/Viper
- ✅ **Structured Logging** - Logrus-based logging with configurable levels
- ✅ **Health Checks** - `/health` and `/ready` endpoints for monitoring
- ✅ **Comprehensive Testing** - Unit tests, integration tests, and mock services

## Quick Start

### Prerequisites

- Go 1.25 or later
- MPC-KMS service endpoint
- Downstream JSON-RPC service (e.g., Ethereum node)

### Installation

```bash
# Clone the repository
git clone https://github.com/mowind/web3signer-go.git
cd web3signer-go

# Build the binary
make build

# Or install directly
go install ./cmd/web3signer/
```

### Basic Usage

```bash
# Run with command-line flags
./build/web3signer \
  --http-host localhost \
  --http-port 9000 \
  --kms-endpoint http://kms.example.com:8080 \
  --kms-access-key-id YOUR_ACCESS_KEY \
  --kms-secret-key YOUR_SECRET_KEY \
  --kms-key-id YOUR_KEY_ID \
  --downstream-http-host http://localhost \
  --downstream-http-port 8545 \
  --downstream-http-path / \
  --log-level info
```

### Configuration File

Create a configuration file `~/.web3signer.yaml`:

```yaml
http:
  host: localhost
  port: 9000

kms:
  endpoint: http://kms.example.com:8080
  access-key-id: YOUR_ACCESS_KEY
  secret-key: YOUR_SECRET_KEY
  key-id: YOUR_KEY_ID

downstream:
  http-host: http://localhost
  http-port: 8545
  http-path: /

log:
  level: info
```

Then run with:
```bash
./build/web3signer --config ~/.web3signer.yaml
```

## JSON-RPC Methods

### Supported Signing Methods

- `eth_sign` - Sign arbitrary data
- `eth_signTransaction` - Sign a transaction
- `eth_sendTransaction` - Sign and send a transaction

### Forwarded Methods

All other JSON-RPC methods are forwarded to the configured downstream service, including:
- `eth_getBalance`
- `eth_getTransactionCount`
- `eth_call`
- `eth_getBlockByNumber`
- `net_version`
- `web3_clientVersion`
- And more...

## Project Structure

```
web3signer-go/
├── cmd/                    # Application entry points
│   ├── web3signer/         # Main application
│   └── test-kms/           # Test utilities
├── internal/               # Private application code
│   ├── config/             # Configuration types and validation
│   ├── kms/                # MPC-KMS client implementation
│   ├── server/             # HTTP server with Gin
│   ├── router/             # JSON-RPC routing
│   ├── jsonrpc/            # JSON-RPC types and utilities
│   ├── downstream/         # Downstream service client
│   ├── signer/             # Signing logic
│   └── errors/             # Error types and handling
├── test/                   # Integration tests and mocks
├── api/                    # API definitions
├── configs/                # Configuration templates
├── scripts/                # Build and deployment scripts
└── build/                  # Build output directory
```

## Development

### Building

```bash
# Build using Makefile
make build          # Build to build/web3signer
make clean          # Clean build artifacts

# Or use Go directly
go build ./cmd/web3signer/
go build -o web3signer ./cmd/web3signer/
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with race detector
go test -race ./...

# Run tests for specific package
go test ./internal/kms/...

# Run single test
go test -run TestClient_Sign
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run vet
go vet ./...

# Clean dependencies
go mod tidy
```

### Running Tests

```bash
# Integration test with mock services
go test ./test/...

# Test KMS client
go test ./internal/kms/...

# Test JSON-RPC types
go test ./internal/jsonrpc/...
```

## Configuration Reference

### HTTP Server Configuration
- `--http-host` - Server host (default: `localhost`)
- `--http-port` - Server port (default: `9000`)

### MPC-KMS Configuration
- `--kms-endpoint` - MPC-KMS endpoint URL (required)
- `--kms-access-key-id` - Access key ID (required)
- `--kms-secret-key` - Secret key (required)
- `--kms-key-id` - Key ID for signing (required)

### Downstream Service Configuration
- `--downstream-http-host` - Downstream service host (default: `http://localhost`)
- `--downstream-http-port` - Downstream service port (default: `8545`)
- `--downstream-http-path` - Downstream service path (default: `/`)

### Logging Configuration
- `--log-level` - Log level: debug, info, warn, error, fatal (default: `info`)

## Environment Variables

All configuration options can be set via environment variables using the `WEB3SIGNER_` prefix:

```bash
export WEB3SIGNER_HTTP_HOST=0.0.0.0
export WEB3SIGNER_HTTP_PORT=9000
export WEB3SIGNER_KMS_ENDPOINT=http://kms.example.com:8080
export WEB3SIGNER_KMS_ACCESS_KEY_ID=your_access_key
export WEB3SIGNER_KMS_SECRET_KEY=your_secret_key
export WEB3SIGNER_KMS_KEY_ID=your_key_id
```

## API Documentation

### Health Endpoints
- `GET /health` - Health check endpoint
- `GET /ready` - Readiness check endpoint

### JSON-RPC Endpoint
- `POST /` - JSON-RPC 2.0 endpoint

### Example Request

```bash
curl -X POST http://localhost:9000/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "eth_sign",
    "params": ["0x9b2055d370f73ec7d8a03e965129118dc8f5bf83", "0xdeadbeef"]
  }'
```

## License

This project is licensed under the GNU General Public License v3.0 (GPLv3). See the [LICENSE](LICENSE) file for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

### Development Setup

#### Code Quality Tools

This project uses several code quality tools:

1. **golangci-lint** - Comprehensive Go linter
   ```bash
   make install-tools  # Install golangci-lint
   make lint           # Run linter
   ```

2. **pre-commit hooks** - Git hooks for code quality
   ```bash
   # Install pre-commit
   pip install pre-commit
   
   # Install hooks
   pre-commit install
   
   # Run hooks on all files
   pre-commit run --all-files
   ```

3. **Testing**
   ```bash
   make test           # Run tests
   make test-coverage  # Run tests with coverage
   make coverage       # Generate HTML coverage report
   ```

#### Code Style Guidelines

- Follow standard Go conventions
- Run `make fmt` before committing
- Ensure `make lint` passes without errors
- Maintain test coverage >80% for core components

## Acknowledgments

- Inspired by [Consensys/web3signer](https://github.com/Consensys/web3signer)
- Built with [Gin](https://github.com/gin-gonic/gin) for HTTP routing
- Uses [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper) for CLI and configuration
- [Logrus](https://github.com/sirupsen/logrus) for structured logging