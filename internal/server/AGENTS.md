# internal/server - Agent Development Guide

**Generated:** Mon, Jan 26 2026
**Package:** HTTP server with Gin framework for web3signer service

---

## OVERVIEW
Gin-based HTTP server serving JSON-RPC requests with TLS, CORS, authentication, and health checks.

---

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Server lifecycle | `server.go` | Start/Stop, HTTP server configuration |
| Router setup | `builder.go` | Gin middleware, endpoints, logging |
| Authentication | `middleware.go` | Bearer/API-Key, constant-time comparison |
| CORS config | `builder.go:280-293` | Allows all origins, POST/GET/OPTIONS |
| Health endpoints | `builder.go:239-257` | `/health`, `/ready` (bypass auth) |
| TLS setup | `server.go:50-54` | ListenAndServeTLS with cert/key files |

---

## CONVENTIONS

**Middleware Chain Order (critical):**
1. Request ID middleware (`requestIDMiddleware`)
2. Logrus logger (`ginlogrus.Logger`)
3. Gin recovery (`gin.Recovery`)
4. CORS middleware (`corsMiddleware`)
5. Auth middleware (`AuthMiddleware`)
6. TLS redirect (if enabled)

**Security:**
- Constant-time comparison for auth tokens (`crypto/subtle.ConstantTimeCompare`)
- Generic error messages: "authentication failed" (no secrets leaked)
- Whitelisted paths bypass authentication (health/ready)

**HTTP Configuration:**
- ReadHeaderTimeout: 5 seconds (prevents slowloris)
- Max request size: configurable (default 10MB)
- Gin mode: Debug for log level debug, Release otherwise

---

## ANTI-PATTERNS

**Deprecated:**
- Missing connection pool config in server (handled by downstream client, not server)

**Security Issues:**
- CORS allows all origins (`*`) - should be configurable for production
- Generic auth errors may frustrate debugging (trade-off for security)

**Technical Debt:**
- Request ID generation uses crypto/rand but no collision handling
- Health endpoints return static status without actual dependency checks
