# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-07-08

### Added

- Core `Event` struct with `Status` enum and canonical schema
- `Provider[T]` generic interface with `Process()` pipeline
- Paystack adapter with HMAC-SHA512 signature verification
- Flutterwave adapter with Verif-Hash signature verification
- OPay adapter with HMAC-SHA512 signature verification
- M-Pesa (Daraja) adapter with STK/C2B callback parsing
- `Store` interface with `MemoryStore` implementation
- Sentinel errors: `ErrInvalidSignature`, `ErrDuplicateEvent`, `ErrUnknownProvider`, `ErrMalformedPayload`
- HMAC verification helpers in `verify.go` with constant-time comparison
- CLI: `onceo verify` and `onceo replay` commands
- Full test coverage with golden fixture payloads
- Fuzz tests for all provider parsers
- CI/CD: GitHub Actions workflows for CI, CodeQL, and release
- Documentation: architecture, adding-a-provider, security
