package paystack

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	onceo "github.com/HalxDocs/onceo-core"
)

const ProviderName = "paystack"

type WebhookPayload struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type ChargeSuccessData struct {
	ID        int    `json:"id"`
	Domain    string `json:"domain"`
	Status    string `json:"status"`
	Reference string `json:"reference"`
	Amount    int    `json:"amount"`
	Currency  string `json:"currency"`
	PaidAt    string `json:"paid_at"`
	CreatedAt string `json:"created_at"`
}

type TransferSuccessData struct {
	ID          int    `json:"id"`
	Reference   string `json:"reference"`
	Amount      int    `json:"amount"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
	Recipient   string `json:"recipient"`
	CreatedAt   string `json:"created_at"`
}

type Provider struct {
	SecretKey string
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) VerifySignature(headers http.Header, body []byte) error {
	signature, err := onceo.ExtractHeader(headers, "X-Paystack-Signature")
	if err != nil {
		return err
	}
	return onceo.VerifyHMACSHA512([]byte(p.SecretKey), body, signature)
}

func (p *Provider) Parse(body []byte) (WebhookPayload, error) {
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return payload, onceo.ErrMalformedPayload
	}
	if payload.Event == "" {
		return payload, onceo.ErrMalformedPayload
	}
	return payload, nil
}

func (p *Provider) Normalize(raw WebhookPayload) (onceo.Event, error) {
	event := onceo.Event{
		Provider:   ProviderName,
		Type:       raw.Event,
		RawPayload: nil,
		ReceivedAt: time.Now().UTC(),
	}

	switch raw.Event {
	case "charge.success":
		return normalizeChargeSuccess(raw.Data, event)
	case "charge.failed":
		return normalizeChargeFailed(raw.Data, event)
	case "transfer.success":
		return normalizeTransferSuccess(raw.Data, event)
	case "transfer.failed":
		return normalizeTransferFailed(raw.Data, event)
	default:
		return onceo.Event{}, fmt.Errorf("%w: unknown event type: %s", onceo.ErrEventParseFailed, raw.Event)
	}
}

func normalizeChargeSuccess(data json.RawMessage, e onceo.Event) (onceo.Event, error) {
	var d ChargeSuccessData
	if err := json.Unmarshal(data, &d); err != nil {
		return e, onceo.ErrMalformedPayload
	}

	e.ProviderEventID = strconv.Itoa(d.ID)
	e.Status = mapStatus(d.Status)
	e.AmountMinor = int64(d.Amount)
	e.Currency = sanitiseCurrency(d.Currency)
	e.Reference = d.Reference

	return e, nil
}

func normalizeChargeFailed(data json.RawMessage, e onceo.Event) (onceo.Event, error) {
	var d ChargeSuccessData
	if err := json.Unmarshal(data, &d); err != nil {
		return e, onceo.ErrMalformedPayload
	}

	e.ProviderEventID = strconv.Itoa(d.ID)
	e.Status = onceo.StatusFailed
	e.AmountMinor = int64(d.Amount)
	e.Currency = sanitiseCurrency(d.Currency)
	e.Reference = d.Reference

	return e, nil
}

func normalizeTransferSuccess(data json.RawMessage, e onceo.Event) (onceo.Event, error) {
	var d TransferSuccessData
	if err := json.Unmarshal(data, &d); err != nil {
		return e, onceo.ErrMalformedPayload
	}

	e.ProviderEventID = strconv.Itoa(d.ID)
	e.Status = mapStatus(d.Status)
	e.AmountMinor = int64(d.Amount)
	e.Currency = sanitiseCurrency(d.Currency)
	e.Reference = d.Reference

	return e, nil
}

func normalizeTransferFailed(data json.RawMessage, e onceo.Event) (onceo.Event, error) {
	var d TransferSuccessData
	if err := json.Unmarshal(data, &d); err != nil {
		return e, onceo.ErrMalformedPayload
	}

	e.ProviderEventID = strconv.Itoa(d.ID)
	e.Status = onceo.StatusFailed
	e.AmountMinor = int64(d.Amount)
	e.Currency = sanitiseCurrency(d.Currency)
	e.Reference = d.Reference

	return e, nil
}

func mapStatus(status string) onceo.Status {
	switch status {
	case "success":
		return onceo.StatusSuccess
	case "failed":
		return onceo.StatusFailed
	case "reversed":
		return onceo.StatusReversed
	default:
		return onceo.StatusPending
	}
}

func sanitiseCurrency(currency string) string {
	if len(currency) == 3 {
		return currency
	}
	return ""
}
