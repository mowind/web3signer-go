# web3signer-go

> 🚀 **A production-ready Go implementation of Web3 signer with MPC-KMS signing support**

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![CI](https://github.com/mowind/web3signer-go/workflows/CI/CD%20Pipeline/badge.svg)](https://github.com/mowind/web3signer-go/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/mowind/web3signer-go)](https://goreportcard.com/report/github.com/mowind/web3signer-go)
[![GoDoc](https://godoc.org/github.com/mowind/web3signer-go?status.svg)](https://godoc.org/github.com/mowind/web3signer-go)

## Overview

`web3signer-go` is a lightweight, secure signing service inspired by [Consensys/web3signer](https://github.com/Consensys/web3signer), specifically focusing on **MPC-KMS (Multi-Party Computation Key Management Service)** signing.

It provides a transparent HTTP JSON-RPC proxy that:
- ✅ **Signs transactions** using secure MPC-KMS (key never exposed)
- ✅ **Forwards all other requests** to downstream Ethereum nodes
- ✅ **Standard Ethereum JSON-RPC** - drop-in replacement for direct node access

### Key Differentiators

| Feature | web3signer-go | Original web3signer |
|---------|---------------|---------------------|
| **Signing Methods** | MPC-KMS only | File, Vault, AWS KMS, YubiHSM, etc. |
| **Architecture** | Simple, focused | Complex, extensible |
| **Language** | Go | Java |
| **Deployment** | Single binary, minimal deps | JVM, heavier footprint |
| **Use Case** | Production MPC-KMS deployments | Multi-backend signing scenarios |

## Features

- ✅ **MPC-KMS Integration** - Secure multi-party computation key management
- ✅ **JSON-RPC Server** - HTTP server with JSON-RPC 2.0 support
- ✅ **Transaction Signing** - Supports `eth_sign`, `eth_signTransaction`, `eth_sendTransaction`
- ✅ **Smart Contract Support** - EIP-1559 and legacy transaction types
- ✅ **Downstream Forwarding** - Transparent proxy to Ethereum nodes
- ✅ **Health Checks** - `/health` and `/ready` endpoints for monitoring
- ✅ **CORS Support** - Configurable CORS headers for web applications
- ✅ **Configuration Management** - CLI flags, config files, and environment variables
- ✅ **Structured Logging** - Logrus-based logging with configurable levels
- ✅ **Comprehensive Testing** - Unit tests and integration tests with high coverage
- ✅ **Docker Support** - Multi-stage Dockerfile for production deployments
- ✅ **CI/CD Pipeline** - Automated testing, linting, and security scanning

## Quick Start

### Prerequisites

- Go 1.25 or later
- MPC-KMS service endpoint
- Downstream JSON-RPC service (e.g., Ethereum node)

### Installation

#### From Source

```bash
# Clone the repository
git clone https://github.com/mowind/web3signer-go.git
cd web3signer-go

# Build the binary
make build

# Or install directly
go install ./cmd/web3signer/
```

#### Using Docker

```bash
# Build the Docker image
docker build -t web3signer:latest .

# Run the container
docker run -d \
  --name web3signer \
  -p 9000:9000 \
  -e WEB3SIGNER_HTTP_HOST=0.0.0.0 \
  -e WEB3SIGNER_HTTP_PORT=9000 \
  -e WEB3SIGNER_KMS_ENDPOINT=http://kms.example.com:8080 \
  -e WEB3SIGNER_KMS_ACCESS_KEY_ID=YOUR_ACCESS_KEY \
  -e WEB3SIGNER_KMS_SECRET_KEY=YOUR_SECRET_KEY \
  -e WEB3SIGNER_KMS_KEY_ID=YOUR_KEY_ID \
  -e WEB3SIGNER_DOWNSTREAM_HTTP_HOST=http://localhost \
  -e WEB3SIGNER_DOWNSTREAM_HTTP_PORT=8545 \
  web3signer:latest
```

For detailed deployment instructions, see [DEPLOYMENT.md](DEPLOYMENT.md).

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
│   └── test-kms/           # KMS test utilities
├── internal/               # Private application code
│   ├── config/             # Configuration types and validation
│   ├── kms/                # MPC-KMS HTTP client
│   ├── server/             # HTTP server with Gin
│   ├── router/             # JSON-RPC routing and handlers
│   ├── jsonrpc/            # JSON-RPC types and utilities
│   ├── downstream/         # Downstream service HTTP client
│   ├── signer/             # Signing logic (implements ethgo.Key)
│   └── errors/             # Error types and handling
├── test/                   # Integration tests and mocks
├── scripts/                # Build and deployment scripts
├── .github/                # GitHub workflows and CI/CD
├── Dockerfile              # Multi-stage Docker build
├── Makefile                # Build and test commands
├── DEPLOYMENT.md           # Detailed deployment guide
├── CLAUDE.md               # AI-assisted development context
└── README.md               # This file
```

## Development

### Prerequisites

- Go 1.25 or later
- Docker (optional, for containerized deployment)
- golangci-lint (install with `make install-tools`)

### Build Commands

```bash
# Build all binaries
make build

# Build specific binary
go build -o web3signer ./cmd/web3signer

# Clean build artifacts
make clean

# Check development environment
make env
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Generate HTML coverage report
make coverage

# Run integration tests
make integration-test

# Run tests for specific package
go test ./internal/kms/...

# Run tests with verbose output
go test -v ./...

# Run tests with race detector
go test -race ./...
```

### Code Quality

```bash
# Format code
make fmt

# Run vet
make vet

# Run linter
make lint

# Tidy dependencies
make tidy

# Run all checks (test + lint)
make check

# Install development tools
make install-tools
```

### Using the Test KMS Tool

The project includes a test KMS client for development:

```bash
# Build the test-kms tool
make build

# Test a signing operation
./build/test-kms \
  --endpoint http://localhost:8080 \
  --access-key-id YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --key-id YOUR_KEY_ID
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
- `--kms-address` - Ethereum address associated with the key (required)

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

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Service health check |
| `/ready` | GET | Service readiness check |

**Response Example:**

```json
{
  "status": "healthy",
  "time": "2026-01-20T08:00:00Z"
}
```

### JSON-RPC Endpoint

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | POST | JSON-RPC 2.0 endpoint |

### Supported Signing Methods

| Method | Description |
|--------|-------------|
| `eth_sign` | Sign arbitrary data with the configured key |
| `eth_signTransaction` | Sign a transaction (returns signed transaction) |
| `eth_sendTransaction` | Sign and send a transaction to the network |
| `eth_accounts` | Returns the configured Ethereum address |

### Example Requests

#### Sign a Transaction

```bash
curl -X POST http://localhost:9000/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "eth_signTransaction",
    "params": [{
      "from": "0xYourAddress",
      "to": "0xRecipientAddress",
      "gas": "0x5208",
      "gasPrice": "0x4a817c800",
      "nonce": "0x0",
      "value": "0xde0b6b3a7640000",
      "chainId": "0x1"
    }]
  }'
```

#### Send a Transaction

```bash
curl -X POST http://localhost:9000/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "eth_sendTransaction",
    "params": [{
      "from": "0xYourAddress",
      "to": "0xRecipientAddress",
      "gas": "0x5208",
      "maxFeePerGas": "0x4a817c800",
      "maxPriorityFeePerGas": "0x4a817c800",
      "nonce": "0x1",
      "value": "0xde0b6b3a7640000",
      "chainId": "0x1"
    }]
  }'
```

## Contributing

We welcome contributions! Please see our development guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all checks pass (`make check`)
6. Commit with [Conventional Commits](https://www.conventionalcommits.org/)
7. Push and create a pull request

### Development Workflow

```bash
# 1. Fork and clone
git clone https://github.com/YOUR_USERNAME/web3signer-go.git
cd web3signer-go

# 2. Create feature branch
git checkout -b feat/amazing-feature

# 3. Make changes and test
make test
make lint

# 4. Commit your changes
git add .
git commit -m "feat(module): add amazing feature"

# 5. Push and create PR
git push origin feat/amazing-feature
```

### Code Style Guidelines

- Follow standard Go conventions and [Effective Go](https://golang.org/doc/effective_go)
- Run `make fmt` before committing
- Ensure `make lint` passes without errors
- Maintain test coverage for all changes
- Write clear, self-documenting code

### Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(kms): add support for multiple key IDs
fix(signer): correct EIP-1559 transaction calculation
docs(readme): update deployment instructions
test(router): add integration test for batch requests
ci(docker): optimize multi-stage build
```

## Roadmap

### In Progress
- [ ] Multi-key support (currently single key-id)
- [ ] Asynchronous signing approval workflow (MPC-KMS task polling)

### Planned
- [ ] Prometheus metrics endpoint
- [ ] Kubernetes deployment manifests
- [ ] Performance benchmarking
- [ ] Webhook notifications for signing events
- [ ] Enhanced logging and tracing support
- [ ] Rate limiting and request throttling

## Documentation

- 📖 **[CLAUDE.md](CLAUDE.md)** - AI-assisted development context and architecture guide
- 📦 **[DEPLOYMENT.md](DEPLOYMENT.md)** - Detailed deployment guide with Docker and production configurations
- 🔧 **[API Documentation](#api-documentation)** - JSON-RPC endpoints and examples

## Support

- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/mowind/web3signer-go/issues)
- 💡 **Feature Requests**: [GitHub Discussions](https://github.com/mowind/web3signer-go/discussions)
- 📧 **Security Issues**: Please report security vulnerabilities privately via GitHub's security advisory features

## License

This project is licensed under the GNU General Public License v3.0 (GPLv3). See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [Consensys/web3signer](https://github.com/Consensys/web3signer)
- Built with [Gin](https://github.com/gin-gonic/gin) for HTTP routing
- Uses [ethgo](https://github.com/umbracle/ethgo) for Ethereum utilities
- Configuration via [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper)
- Logging with [Logrus](https://github.com/sirupsen/logrus)

---

**Made with ❤️ by the web3signer-go team**
