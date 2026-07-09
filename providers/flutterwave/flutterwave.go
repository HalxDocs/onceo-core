package flutterwave

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	onceo "github.com/HalxDocs/onceo-core"
)

const ProviderName = "flutterwave"

type WebhookPayload struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type ChargeData struct {
	ID           int    `json:"id"`
	TxRef        string `json:"tx_ref"`
	FlwRef       string `json:"flw_ref"`
	Status       string `json:"status"`
	Amount       int    `json:"amount"`
	ChargedAmount int   `json:"charged_amount"`
	Currency     string `json:"currency"`
	CreatedAt    string `json:"created_at"`
}

type TransferData struct {
	ID       int    `json:"id"`
	Ref      string `json:"reference"`
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
	Status   string `json:"status"`
}

type Provider struct {
	SecretHash string
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) VerifySignature(headers http.Header, body []byte) error {
	if p.SecretHash == "" {
		return onceo.ErrInvalidSignature
	}
	signature, err := onceo.ExtractHeader(headers, "Verif-Hash")
	if err != nil {
		return err
	}
	return onceo.VerifyHMACSHA256([]byte(p.SecretHash), body, signature)
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
		ReceivedAt: time.Now().UTC(),
	}

	switch raw.Event {
	case "charge.completed", "charge.success":
		return normalizeCharge(raw.Data, event, onceo.StatusSuccess)
	case "charge.failed":
		return normalizeCharge(raw.Data, event, onceo.StatusFailed)
	case "transfer.completed":
		return normalizeTransfer(raw.Data, event, onceo.StatusSuccess)
	case "transfer.failed":
		return normalizeTransfer(raw.Data, event, onceo.StatusFailed)
	default:
		return onceo.Event{}, fmt.Errorf("%w: unknown event type: %s", onceo.ErrEventParseFailed, raw.Event)
	}
}

func normalizeCharge(data json.RawMessage, e onceo.Event, status onceo.Status) (onceo.Event, error) {
	var d ChargeData
	if err := json.Unmarshal(data, &d); err != nil {
		return e, onceo.ErrMalformedPayload
	}

	e.ProviderEventID = fmt.Sprintf("%d", d.ID)
	e.Status = mapStatus(d.Status, status)
	e.AmountMinor = int64(d.Amount)
	e.Currency = d.Currency
	e.Reference = d.TxRef

	return e, nil
}

func normalizeTransfer(data json.RawMessage, e onceo.Event, status onceo.Status) (onceo.Event, error) {
	var d TransferData
	if err := json.Unmarshal(data, &d); err != nil {
		return e, onceo.ErrMalformedPayload
	}

	e.ProviderEventID = fmt.Sprintf("%d", d.ID)
	e.Status = mapStatus(d.Status, status)
	e.AmountMinor = int64(d.Amount)
	e.Currency = d.Currency
	e.Reference = d.Ref

	return e, nil
}

func mapStatus(providerStatus string, defaultStatus onceo.Status) onceo.Status {
	switch providerStatus {
	case "successful", "success":
		return onceo.StatusSuccess
	case "failed":
		return onceo.StatusFailed
	default:
		return defaultStatus
	}
}
