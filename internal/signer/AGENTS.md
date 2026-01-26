# internal/signer - Agent Development Guide

**Generated:** Mon, Jan 26 2026
**Package:** Ethereum transaction signing with MPC-KMS delegation
**Stack:** ethgo.Key, fastrlp

---

## OVERVIEW
Implements ethgo.Key interface for MPC-KMS-backed signing with multi-key support.

---

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Main signer implementation | signer.go | MPCKMSSigner.SignTransaction, signHash, trimBytesZeros |
| Transaction parsing | transaction.go | JSONRPCTransaction, EIP-1559/2930/Legacy types via fastjson |
| Multi-key management | multikey_signer.go | Dynamic keyID selection, AddClient/RemoveClient |
| eth_sign parameter parsing | builder.go | ParseSignParams, parseHex helper |

---

## CONVENTIONS

**Package-Specific:**
- Transaction deep copy: SignTransaction creates full copy of tx to avoid mutation
- RLP encoding: Uses fastrlp ArenaPool for signHash calculation
- JSON parsing: fastjson.ParserPool for transaction parameter parsing
- Address validation: Strict 0x prefix + 42 chars + hex digits check

**Signature Encoding:**
- Legacy: v = signature_v + 35 + chainID * 2 (EIP-155)
- EIP-1559/2930: v = signature_v (no chainID adjustment)

---

## ANTI-PATTERNS (THIS PACKAGE)

**Technical Debt:**
- `trimBytesZeros` (signer.go:238-250) - should be at KMS client layer, not signer
- `SignSummary` TODO - extract transaction fields from data blob not implemented

**Implementation Notes:**
- Manual field copying in SignTransaction - consider ethgo.Transaction.Copy() or reflection
- big.Int for V value prevents overflow on large chainIDs (intentional)
- MultiKeySigner type asserts to *MPCKMSSigner for summary support (design decision)

**Testing Gaps:**
- MultiKeySigner concurrent access to clients map untested
- EIP-2930 (Type 1) transaction signing integration test missing
