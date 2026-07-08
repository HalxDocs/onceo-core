# onceo-core

**one clean payment event, every rail, exactly once.**

A thin, boring, auditable reconciliation layer for payment webhooks from
African payment providers. Sits on top of rails you already use — never
becomes a payment processor.

## Supported Providers

| Provider | Status |
|----------|--------|
| Paystack | Shipped |
| Flutterwave | Shipped |
| OPay | Shipped |
| M-Pesa (Daraja) | Shipped |

## Quick Start

```go
import onceo "github.com/HalxDocs/onceo-core"
import "github.com/HalxDocs/onceo-core/providers/paystack"

provider := &paystack.Provider{SecretKey: "sk_test_..."}
store := onceo.NewMemoryStore()

event, err := onceo.Process(ctx, provider, store, headers, body)
if err != nil {
    // handle error
}
```

## Pipeline

```
VerifySignature → Parse → Normalize → Idempotency check → Persist
```

## Design

- **Zero required external dependencies** — only Go stdlib for security-critical code
- **Generics** — each provider defines its own parsed payload type
- **Interfaces** — `Store` and `Provider[T]` are interfaces; nothing is hardcoded
- **No global state** — everything is explicitly constructed
- **Constant-time comparison** — every signature check uses `hmac.Equal`
- **Sentinel errors** — `errors.Is`-friendly: `ErrInvalidSignature`, `ErrDuplicateEvent`, etc.

## License

MIT — see [LICENSE](LICENSE).
