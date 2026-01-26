# internal/router - Agent Development Guide

**Generated:** Mon, Jan 26 2026

---

## OVERVIEW
Central JSON-RPC routing layer dispatching sign requests to KMS and forwarding all other requests to downstream Ethereum nodes.

---

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Core routing | `router.go` | Router, handlers map, routeRequest, HandleHTTPRequest |
| Sign methods | `sign_handler.go` | SignHandler: eth_accounts/eth_sign/eth_signTransaction/eth_sendTransaction |
| Forward methods | `forward_handler.go` | ForwardHandler: transparent proxy, eth_accounts returns [] |
| Factory pattern | `factory.go` | RouterFactory: router creation + handler registration |
| Routing decision | `sign_handler.go:375` | IsSignMethod(): determines sign vs forward routing |

## ROUTING DECISIONS

**Sign Methods (IsSignMethod → SignHandler):**
- `eth_accounts` - returns KMS-managed address
- `eth_sign` - signs arbitrary data
- `eth_signTransaction` - signs transaction, returns RLP-encoded tx
- `eth_sendTransaction` - signs → RLP → forwards eth_sendRawTransaction to downstream

**Forward Methods (default → ForwardHandler):**
- All non-sign methods forwarded transparently
- `eth_accounts` special case: returns empty array (non-KMS accounts)
- Batch requests: maintain order, responses match requests

---

## CONVENTIONS

**Handler Interface:**
```go
type Handler interface {
    Method() string           // Method name this handler supports
    Handle(ctx, request) (*Response, error)
}
```

**Thread-Safety:**
- Router uses sync.RWMutex for handler map operations
- RouteRequest is thread-safe for concurrent requests

**Response Validation:**
- Always set response.ID = request.ID after handling
- response.JSONRPC = "2.0" for all successful responses
- Log request/response at debug level (never keys/secrets)

---

## ANTI-PATTERNS

**Routing Errors:**
- Duplicate method registration: `Register()` returns error if method exists
- Empty method names: validation prevents handler.Method() == ""
- Nil default handler: SetDefaultHandler requires non-nil handler

**Handler Design:**
- Don't mix sign and forward logic in same handler
- Never bypass downstream client for non-sign methods
- Don't modify request.ID (must match response.ID)

---

## CODE MAP

| Symbol | Type | File | Role |
|--------|------|------|------|
| Router | struct | router.go | Main router with handlers map |
| SignHandler | struct | sign_handler.go | Handles sign methods + eth_sendTransaction |
| ForwardHandler | struct | forward_handler.go | Proxies all non-sign requests |
| IsSignMethod | func | sign_handler.go:375 | Routing decision helper |
| RouterFactory | struct | factory.go | Creates configured routers with handlers |