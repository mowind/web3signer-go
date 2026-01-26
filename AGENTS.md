# web3signer-go - Agent Development Guide

**Generated:** Mon, Jan 26 2026
**Project:** Ethereum JSON-RPC signing service with MPC-KMS
**Stack:** Go 1.25, Gin, Cobra, Viper, ethgo

---

## OVERVIEW
Drop-in Ethereum JSON-RPC signing service. Routes sign requests to MPC-KMS (keys never exposed), forwards all other requests to downstream nodes transparently.

---

## STRUCTURE
```
web3signer-go/
├── cmd/                  # Entry points
│   ├── web3signer/       # Main binary (CLI)
│   └── test-kms/         # KMS testing utility
├── internal/             # Private packages
│   ├── router/           # JSON-RPC routing (sign vs forward)
│   ├── signer/           # Signing implementation (ethgo.Key)
│   ├── kms/              # MPC-KMS HTTP client
│   ├── downstream/        # Ethereum node proxy
│   ├── jsonrpc/          # JSON-RPC types/errors
│   ├── errors/           # Error definitions
│   ├── server/           # HTTP server (Gin)
│   └── config/           # Configuration (Viper)
├── test/                 # Integration tests & mocks
└── build/                # Build artifacts
```

---

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| JSON-RPC routing | `internal/router/` | SignHandler vs ForwardHandler |
| Transaction signing | `internal/signer/` | MPCKMSSigner.SignTransaction |
| KMS auth flow | `internal/kms/client.go` | HMAC-SHA256, task polling |
| Ethereum interaction | `internal/downstream/` | Proxy to node, nonce/gas/estimateGas |
| CLI setup | `cmd/web3signer/` | Cobra flags, Viper config |
| Error handling | `internal/errors/` | WrapError, DownstreamError |

---

## CODE MAP

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| Router | struct | internal/router/router.go | JSON-RPC request router |
| SignHandler | struct | internal/router/handler.go | Handles sign requests |
| ForwardHandler | struct | internal/router/handler.go | Proxies non-sign requests |
| MPCKMSSigner | struct | internal/signer/signer.go | ethgo.Key implementation |
| KMSClient | struct | internal/kms/client.go | MPC-KMS HTTP client |
| DownstreamClient | struct | internal/downstream/client.go | Ethereum node proxy |
| Config | struct | internal/config/config.go | All configuration |

---

## CONVENTIONS

**Project-Specific Rules:**

- **Entry points:** Root-level binary at `./`, test utilities at `./cmd/`
- **Testing:** Integration tests and mocks in `./test/` (not in internal/)
- **Logging:** Structured with logrus, debug level for request/response bodies, info for events
- **Never log:** Keys, secrets, signatures

**Go Standards Followed:**
- Effective Go conventions
- Lowercase single-word packages (`kms`, `signer`, `router`)
- PascalCase types, camelCase functions/variables
- Interfaces end with `Interface` for single-method

---

## ANTI-PATTERNS (THIS PROJECT)

**Deprecated:**
- `NewClientWithLogger` in KMS - use `NewClient` instead

**Technical Debt:**
- `trimBytesZeros` in `signer.go` (lines 197-215) - should be at KMS client layer
- TODO in `SignSummary` - extract transaction fields from data blob

**Security Issues:**
- `test-kms/main.go` contains hardcoded credentials for testing

**Anti-Patterns in Code:**
- Empty catch blocks prohibited
- Type error suppression (`as any`, `@ts-ignore`) forbidden
- Shotgun debugging (random changes) prohibited

---

## COMMANDS

**Build:**
```bash
make build              # Build web3signer and test-kms
go build -o build/web3signer ./cmd/web3signer
```

**Test:**
```bash
make test               # Run all tests
make test-coverage      # With coverage report
make integration-test   # Integration tests
make coverage           # HTML coverage report
```

**Lint:**
```bash
make fmt                # Format with goimports
make lint               # golangci-lint (gosec, gocritic, gocyclo)
make vet                # go vet
make check              # test + lint combined
```

**Docker:**
```bash
docker build -t web3signer .
```

---

## NOTES

**KMS Signing Flow:**
1. SignRequest (data/encoding/summary/callback_url)
2. HMAC-SHA256 auth (GMT timestamp → Content-SHA256 → sign string → Authorization: "MPC-KMS AK:Signature")
3. HTTP 200: direct signature, HTTP 201: task_id (approval workflow)
4. Poll WaitForTaskCompletion every 5s (PENDING_APPROVAL/APPROVED → DONE, 5min timeout)

**eth_sendTransaction Flow:**
1. Parse + validate from address
2. Get nonce from downstream
3. Get gasPrice (Legacy/AccessList → gasPrice, EIP-1559 → maxFeePerGas/maxPriorityFeePerGas)
4. Estimate gas if 0 (add 20% safety margin)
5. Sign → RLP encode → eth_sendRawTransaction

**Configuration Priority:**
CLI flags > env vars (`WEB3SIGNER_` prefix) > config file (`~/.web3signer.yaml`) > defaults

**Required KMS Config:** endpoint, access-key-id, secret-key, key-id, address
**Required Downstream Config:** http-host, http-port (default: 8545), http-path (default: /)

**HTTP Server:**
- Default: localhost:9000
- CORS enabled
- 30s request timeout
- Connection pool: MaxIdleConns=100, IdleConnTimeout=90s
