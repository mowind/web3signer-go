# internal/errors - Agent Development Guide

**Generated:** Mon, Jan 26 2026
**Package:** Error definitions and wrappers with JSON-RPC error codes

---

## OVERVIEW
Centralized error handling with JSON-RPC error code mapping, structured error wrapping, and type-specific error helpers.

---

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Error wrapping | `errors.go` | WrapError(fmt.Sprintf, err, code) |
| Error codes | `errors.go` | Error code constants (-32000 to -32767) |
| Type helpers | `errors.go` | IsConnectionError, IsInvalidResponseError |
| Error types | `errors.go` | DownstreamError, APIError, InvalidRequestError |

---

## CONVENTIONS

**Error Wrapping:**
- Use `WrapError(fmt.Sprintf("failed to do X: %v", err), err, code)` pattern
- Error messages: lowercase, no trailing punctuation
- Fail fast: validate inputs before processing

**Error Codes:**
- JSON-RPC standard range: -32700 to -32603 (parse, invalid request, method not found, etc.)
- Application errors: -32000 to -32099 (KMS errors, signing errors, etc.)
- Downstream errors: -32100 to -32199 (Ethereum node errors)

**Error Type Checks:**
- Use helper functions instead of type assertions
- `IsConnectionError(err)`, `IsInvalidResponseError(err)`, etc.
- Never check errors with `err.(*DownstreamError)` - use helpers

---

## ANTI-PATTERNS (THIS PACKAGE)

**Error Formatting:**
- Never use uppercase error messages - use lowercase
- Never add trailing punctuation to error messages
- Never wrap nil errors - check before wrapping

**Error Code Usage:**
- Never use error codes outside JSON-RPC standard ranges
- Never return generic errors without error codes
- Never swallow errors - always wrap with context

**Anti-Patterns in Code:**
- Never use `errors.New()` without wrapping context
- Never return raw errors from downstream - wrap with DownstreamError
- Never check error type with type assertions - use helper functions
- Never ignore error codes when wrapping errors
- Never create new error types without adding helper functions

---

## CODE MAP

| Symbol | Type | File | Role |
|--------|------|------|------|
| WrapError | func | errors.go | Central error wrapper with code |
| DownstreamError | struct | errors.go | Ethereum node error |
| APIError | struct | errors.go | JSON-RPC API error |
| IsConnectionError | func | errors.go | Type helper for connection errors |
| IsInvalidResponseError | func | errors.go | Type helper for response errors |
