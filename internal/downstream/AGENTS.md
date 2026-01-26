# internal/downstream - Agent Development Guide

**Generated:** Mon, Jan 26 2026
**Package:** Ethereum node proxy with JSON-RPC forwarding

---

## OVERVIEW
Transparent JSON-RPC proxy to Ethereum nodes with connection pooling, single/batch request forwarding, and comprehensive error handling.

---

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Request forwarding | `client.go` | ForwardRequest, ForwardBatchRequest |
| Error types | `errors.go` | ConnectionError, RequestError, InvalidResponseError |
| Response validation | `client.go` | ID matching, HTTP status checks |
| Interfaces | `interface.go` | ClientInterface, Forwarder, SimpleForwarder |
| Connection pool config | `client.go:49` | createTransport() - 100 idle conns, 30s timeout |
| Health check | `client.go:268` | TestConnection() via web3_clientVersion |

---

## CONVENTIONS

**Package-Specific Rules:**

- **Request headers:** Always set `Content-Type: application/json` and `Accept: application/json`
- **ID validation:** Log warnings on mismatch (compareIDs), never fail on ID mismatch
- **Batch handling:** Preserve response order, validate response count matches request count
- **Error wrapping:** Use WrapError() for all errors with proper error codes
- **Connection pooling:** MaxIdleConns=100, MaxIdleConnsPerHost=100, IdleConnTimeout=90s
- **Timeouts:** 30-second HTTP timeout, 10-second response header timeout
- **Context usage:** All forward methods accept context for cancellation/timeout

**Error Type Checks:**
- `IsConnectionError()` - Network/connection failures
- `IsInvalidResponseError()` - JSON parsing/response format errors
- Never check errors with type assertions directly - use helper functions

---

## ANTI-PATTERNS (THIS PACKAGE)

**Technical Debt:**
- ID mismatch validation only logs warnings - should be configurable to fail in strict mode
- Close() is a no-op - HTTP client auto-manages, but future cleanup needs it

**Anti-Patterns in Code:**
- Never modify request/response during forwarding (transparent proxy)
- Never swallow errors - always wrap with WrapError()
- Never hardcode timeouts - use context-based timeouts instead
- Never validate JSON-RPC method names - forward all methods transparently
- Never log request/response bodies in production (security risk)
