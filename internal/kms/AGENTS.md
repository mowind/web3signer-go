# internal/kms - Agent Development Guide

**Generated:** Mon, Jan 26 2026
**Module:** MPC-KMS HTTP client with HMAC-SHA256 authentication

---

## OVERVIEW
MPC-KMS HTTP client implementing HMAC-SHA256 authentication and async approval workflow with task polling.

---

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| HMAC-SHA256 auth flow | http_client.go:76-104 | Generate GMT timestamp → Content-SHA256 (base64) → signing string → signature → "MPC-KMS AK:Signature" header |
| SignRequest structure | types.go:10-15 | data, data_encoding (PLAIN/BASE64/HEX), summary (type/from/to/amount/token/remark), callback_url |
| WaitForTaskCompletion | client.go:382-457 | Polls every 5s, max 5min timeout. States: PENDING_APPROVAL/APPROVED → continue, DONE → return, FAILED/REJECTED → error |
| Sign/SignWithOptions | client.go:125-308 | HTTP 200 = direct signature, HTTP 201 = task_id (poll workflow) |
| HTTP client config | http_client.go:38-59 | 30s timeout, MaxIdleConns=100, IdleConnTimeout=90s |

---

## CODE MAP

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| Client | struct | client.go:21-31 | Main MPC-KMS client with URL caching |
| HTTPClient | struct | http_client.go:18-22 | HTTP client with auto-signing |
| SignRequest | struct | types.go:10-15 | Request payload for signing |
| SignResponse | struct | types.go:28-30 | Direct signature response (HTTP 200) |
| TaskResponse | struct | types.go:33-35 | Task ID response (HTTP 201) |
| TaskResult | struct | types.go:49-53 | Async task result with status |
| ClientInterface | interface | interface.go:9-21 | Mockable client interface |
| Signer | interface | interface.go:24-30 | SignMessage/SignTransaction abstraction |
| MPCKMSSigner | struct | interface.go:33-36 | Signer implementation using ClientInterface |

---

## ANTI-PATTERNS (THIS MODULE)

**Deprecated:**
- `NewClientWithLogger` (client.go:79) - Use `NewClient` or `NewClientWithHTTPClient` instead

**Technical Debt:**
- `SignSummary` in interface.go:53-61 has placeholder values (0x000... addresses) - TODO in code to extract from transaction data blob
- Task status logging at debug level might be insufficient for production debugging of stuck approvals

**Anti-Patterns in Code:**
- Empty catch blocks prohibited
- Type error suppression forbidden
- Shotgun debugging prohibited

---

## NOTES

**HMAC-SHA256 Authentication:**
1. Generate GMT timestamp: `time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")`
2. Calculate Content-SHA256: `sha256.Sum256(body) → base64.EncodeToString`
3. Build signing string: `VERB\nContent-SHA256\nContent-Type\nDate`
4. Calculate signature: `hmac.New(sha256.New, secretKey).Write(signingString) → base64.EncodeToString`
5. Set Authorization header: `"MPC-KMS {accessKeyID}:{signature}"`

**Async Approval Flow:**
- HTTP 200: Direct signature returned in SignResponse
- HTTP 201: Returns TaskResponse with task_id → triggers WaitForTaskCompletion
- Task states: PENDING_APPROVAL/APPROVED (continue polling), DONE (success with signature), FAILED/REJECTED (error)
- Default polling: 5s interval, 5min timeout (60 attempts max)

**URL Caching:**
- `getSignURL(keyID)` and `getTaskURL(taskID)` use sync.Once for thread-safe lazy initialization
- Pre-computes base endpoint URL once, then appends keyID/taskID

**Data Encoding:**
- PLAIN: Direct string conversion
- BASE64: `base64.StdEncoding.EncodeToString`
- HEX: `hex.EncodeToString`
