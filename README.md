# onceo-core

> **one clean payment event, every rail, exactly once.**

A thin, auditable reconciliation layer for payment webhooks from African payment providers. It sits on top of rails you already use — it never becomes a payment processor itself.

[![Go Reference](https://pkg.go.dev/badge/github.com/HalxDocs/onceo-core.svg)](https://pkg.go.dev/github.com/HalxDocs/onceo-core)
[![Go Report Card](https://goreportcard.com/badge/github.com/HalxDocs/onceo-core)](https://goreportcard.com/report/github.com/HalxDocs/onceo-core)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

## Supported Providers

| Provider | Body Integrity | Status |
|----------|---------------|--------|
| [Paystack](https://paystack.com) | HMAC-SHA512 over body | Shipped |
| [Flutterwave](https://flutterwave.com) | **None** — see warning below | Shipped |
| [OPay](https://opayweb.com) | HMAC-SHA512 over body | Shipped |
| [M-Pesa (Daraja)](https://developer.safaricom.co.ke) | Pre-shared callback token (constant-time cmp) | Shipped |

## Installation

```
go get github.com/HalxDocs/onceo-core
```

## Quick Start

```go
package main

import (
    "context"
    "errors"
    "io"
    "net/http"
    "os"

    onceo "github.com/HalxDocs/onceo-core"
    "github.com/HalxDocs/onceo-core/providers/paystack"
)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    ctx := context.Background()
    body, _ := io.ReadAll(r.Body)

    provider, _ := paystack.New(os.Getenv("PAYSTACK_SECRET_KEY"))
    store := onceo.NewMemoryStore() // or store/redis

    event, err := onceo.Process(ctx, provider, store, r.Header, body)
    if errors.Is(err, onceo.ErrInvalidSignature) {
        http.Error(w, "invalid signature", http.StatusForbidden)
        return
    }
    if errors.Is(err, onceo.ErrDuplicateEvent) {
        w.WriteHeader(http.StatusOK) // idempotent: already processed
        return
    }
    if err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }

    // event is the canonical, normalized payment event
    switch event.Status {
    case onceo.StatusSuccess:
        fulfillOrder(event.Reference, event.AmountMinor, event.Currency)
    case onceo.StatusFailed:
        markOrderFailed(event.Reference)
    }
    w.WriteHeader(http.StatusOK)
}
```

## API

### `onceo.Process(ctx, provider, store, headers, body) (*Event, error)`

The single entry point. Runs the full pipeline:

1. **Body size check** — rejects payloads over `MaxBodySize` (1 MiB)
2. **Provider validation** — verifies provider name is non-empty and valid
3. **Event ID length check** — rejects `ProviderEventID` > `MaxProviderEventIDLength` (256)
4. **Signature verification** — delegates to `provider.VerifySignature`
5. **Parse** — decodes provider-specific payload
6. **Normalize** — maps to canonical `Event` schema
7. **Idempotency check** — `store.SaveIfNew` (atomic, exactly-once)
8. **Return** — the saved `Event` with `ProcessedAt` set

### `Event`

```go
type Event struct {
    ID              string    // unique event ID (generated)
    Provider        string    // provider name
    ProviderEventID string    // provider's event ID (unique key for idempotency)
    Type            string    // event type (e.g. "charge.success")
    Status          Status    // success, failed, pending, reversed
    AmountMinor     int64     // amount in minor currency unit (e.g. kobo, cents)
    Currency        string    // ISO 4217 currency code
    Reference       string    // merchant reference
    RawPayload      []byte    // original raw body
    ReceivedAt      time.Time // when the webhook was received
    ProcessedAt     time.Time // when Process returned successfully
}
```

Status enum: `StatusPending` (0), `StatusSuccess` (1), `StatusFailed` (2), `StatusReversed` (3).

Supports JSON marshal/unmarshal (string-based, case-insensitive).

### Sentinel Errors

Use `errors.Is` to check:

| Error | Meaning |
|---|---|
| `ErrInvalidSignature` | Signature does not match |
| `ErrDuplicateEvent` | Event already processed (idempotent hit) |
| `ErrUnknownProvider` | Provider name not recognized |
| `ErrMalformedPayload` | Body cannot be parsed for event ID |
| `ErrMissingHeader` | Required signature header missing |
| `ErrDuplicateHeader` | Header appears more than once |
| `ErrEventParseFailed` | Provider event ID is empty or unparseable |
| `ErrStoreFailed` | Store backend returned an error |

### `Store` Interface

```go
type Store interface {
    SaveIfNew(ctx context.Context, e Event) (created bool, err error)
}
```

**Contract:** atomic, exactly-one semantics. Two concurrent calls for the same event must never both return `created=true`.

#### Built-in: `onceo.NewMemoryStore(maxEvents ...int)`

In-memory FIFO store, safe for concurrent use. Max 100,000 events by default. **Not recommended for production** — events are lost on restart and evicted when capacity is reached.

#### Redis: `store/redis.New(addr, password, db)`

Production-grade store backed by Redis `SETNX`. Requires `github.com/redis/go-redis/v9`.

```go
import redisstore "github.com/HalxDocs/onceo-core/store/redis"

store, err := redisstore.New("localhost:6379", "", 0)
```

### `Provider[T]` Interface

```go
type Provider[T any] interface {
    Name() string
    BodyBound() bool
    VerifySignature(headers http.Header, body []byte) error
    ParsePayload(body []byte) (T, error)
    Normalize(payload T) (Event, error)
}
```

- `BodyBound()`: if true, the signature covers the request body (Paystack, OPay). If false (Flutterwave), callers should be aware there is no application-level body integrity.
- Each provider uses Go generics to define its own parsed payload type.

## Providers

### Paystack

```go
import "github.com/HalxDocs/onceo-core/providers/paystack"

provider, err := paystack.New("sk_test_...")
```

- **Signature:** HMAC-SHA512 of raw body, sent in `X-Paystack-Signature` header
- **Body integrity:** Yes (`BodyBound() = true`)
- **Parsed type:** `paystack.Payload`

### Flutterwave

```go
import "github.com/HalxDocs/onceo-core/providers/flutterwave"

provider, err := flutterwave.New("FLWSECK_TEST-...")
```

> **WARNING:** Flutterwave's `Verif-Hash` header is a static SHA-256 hash of the webhook secret hash (`secret_hash` in the dashboard). It does **not** include the request body. `BodyBound()` returns `false`. Any attacker who captures one valid `Verif-Hash` can forge arbitrary webhook payloads until the secret hash is rotated. There is **no application-level body integrity** — protection relies entirely on TLS.

- **Signature:** Static SHA-256 hash (`Verif-Hash` header)
- **Body integrity:** No (`BodyBound() = false`)
- **Parsed type:** `flutterwave.Payload`

### OPay

```go
import "github.com/HalxDocs/onceo-core/providers/opay"

provider, err := opay.New("OPAY_MERCHANT_ID", "OPAY_SECRET_KEY")
```

- **Signature:** HMAC-SHA512 of raw body, sent in `Authorization` header. Supports optional fallback `Opay-Signature` header for backward compatibility.
- **Body integrity:** Yes (`BodyBound() = true`)
- **Parsed type:** `opay.Payload`

### M-Pesa (Daraja)

```go
import "github.com/HalxDocs/onceo-core/providers/mpesa"

provider, err := mpesa.New("callback-token")
```

- **Signature:** Pre-shared callback token, verified via `subtle.ConstantTimeCompare`
- **Body integrity:** Yes (`BodyBound() = true`)
- **Parsed type:** `mpesa.STKCallbackPayload`
- **Only STK Push callbacks** are currently supported

## CLI Tool

Install the `onceo` CLI to verify webhook payloads locally:

```
go install github.com/HalxDocs/onceo-core/cmd/onceo@latest
```

```bash
# Verify a Paystack payload
onceo verify charge_success.json --provider=paystack --secret=sk_test_...

# Verify a Flutterwave payload
onceo verify transfer_event.json --provider=flutterwave --secret=FLWSECK_TEST-...
```

The CLI runs the full `Process` pipeline and prints the normalized `Event` as JSON. Use the `noop` secret (`--secret=noop`) to skip signature verification for inspection.

## Security

### Constant-time comparisons

All signature and token comparisons use `crypto/hmac.Equal`, `crypto/subtble.ConstantTimeCompare`, or `crypto/subtle.ConstantTimeEq`. No early-exit comparison paths.

### Body size enforcement

Payloads over `MaxBodySize` (1 MiB) are rejected early, before any signature or parsing work.

### Provider event ID constraints

`ProviderEventID` is limited to `MaxProviderEventIDLength` (256 characters) to prevent database indexing issues and memory exhaustion attacks.

### Idempotency key escaping

MemoryStore and RedisStore escape `:` and `\` characters in provider names and event IDs to prevent key delimiter collisions.

## Audit History

The library has been audited across 6 rounds of findings (critical → low):

- Body size enforcement added
- Provider BodyBound() flag for bodyless signatures
- Amount fields changed to `int64` for 32-bit safety
- Idempotency keys properly escaped
- Signature error wrapping fixed
- Duplicate/malformed header handling hardened
- Redis store input validation added
- Consistent timestamp snapshots

## License

MIT — see [LICENSE](LICENSE).
