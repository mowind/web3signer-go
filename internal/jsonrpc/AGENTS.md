# internal/jsonrpc - Agent Development Guide

**Generated:** Mon, Jan 26 2026
**Complexity:** ~15-20 (JSON-RPC types/errors, 4 Go files)

---

## OVERVIEW
JSON-RPC 2.0 protocol types, error definitions, and request/response utilities for Ethereum JSON-RPC communication.

---

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Request/Response types | `types.go` | Request, Response, Error structs with json.RawMessage for Params/Result |
| Error codes | `errors.go` | Standard JSON-RPC error codes and predefined error instances |
| Batch parsing | `types.go:32` | ParseRequest: auto-detects single vs batch, validates all requests |
| Response creation | `types.go:87` | NewResponse/NewErrorResponse: marshal results, set JSONRPC version |
| Server errors | `errors.go:61` | NewServerError: validates range (-32000 to -32099), fallback to InternalError |

---

## CONVENTIONS

**ID Handling:** Request.ID: interface{} (string, float64, int, int64, or null). Response.ID must exactly match request.ID. Validate ID type in validateRequest.

**Request/Response Structure:** JSONRPC always "2.0". Params/Result use json.RawMessage (preserve raw JSON, lazy unmarshal). Batch: maintain order, validate all.

**Error Codes:** Use predefined errors (ParseError, InvalidRequestError, etc.). Server errors: -32000 to -32099 (NewServerError validates). Custom: NewCustomError or Errorf.

---

## ANTI-PATTERNS

**ID Mistakes:** Don't change request.ID. Never accept unsupported ID types (bool, object, array). Don't set JSONRPC version to anything other than "2.0".

**Error Handling:** Don't use error codes outside reserved ranges. Never ignore batch validation failures. Don't return non-JSON-RPC errors (must use *Error struct).

**Batch Processing:** Don't reorder batch responses. Never accept empty batch requests. Don't mix single/batch handling (ParseRequest auto-detects).

---

## CODE MAP

| Symbol | Type | File | Role |
|--------|------|------|------|
| Request | struct | types.go | JSON-RPC 2.0 request |
| Response | struct | types.go | JSON-RPC 2.0 response |
| Error | struct | types.go | JSON-RPC 2.0 error with code/message/data |
| ParseRequest | func | types.go | Parse single/batch, validate all |
| NewServerError | func | errors.go | Create server error with range validation |
