/*
Package onceo provides a thin, auditable reconciliation layer for payment webhooks.

It sits on top of payment providers (Paystack, Flutterwave, OPay, M-Pesa) and
answers one question reliably: exactly what happened with a payment, exactly once,
with a provable audit trail.

The core flow is:
  1. VerifySignature — per-provider HMAC scheme
  2. Parse — decode the provider-specific payload
  3. Normalize — map to the canonical Event schema
  4. Idempotency check — deduplicate by provider event ID
  5. Persist — save to a Store implementation

Usage:

	import onceo "github.com/HalxDocs/onceo-core"

	// Use a provider adapter directly:
	headers := r.Header
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// handle body read failure (log, return 500)
	}

	adapter, err := paystack.New(os.Getenv("PAYSTACK_SECRET_KEY"))
	if err != nil {
		// handle configuration error
	}
	event, err := onceo.Process(ctx, adapter, store, headers, body)
	if errors.Is(err, onceo.ErrInvalidSignature) {
		// handle bad signature
	}
*/
package onceo
