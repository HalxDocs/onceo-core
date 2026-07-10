package flutterwave

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	secretHash string
}

func New(secretHash string) (*Provider, error) {
	if len(secretHash) == 0 {
		return nil, fmt.Errorf("flutterwave: secret hash must not be empty")
	}
	return &Provider{secretHash: secretHash}, nil
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) VerifySignature(headers http.Header, body []byte) error {
	signature, err := onceo.SingleHeader(headers, "Verif-Hash")
	if err != nil {
		return err
	}
	h := sha256.Sum256([]byte(p.secretHash))
	expected := hex.EncodeToString(h[:])
	if subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) != 1 {
		return onceo.ErrInvalidSignature
	}
	return nil
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

var validCurrencies = map[string]bool{
	"NGN": true, "USD": true, "GBP": true, "EUR": true, "ZAR": true,
	"GHS": true, "KES": true, "UGX": true, "TZS": true, "RWF": true,
	"XAF": true, "XOF": true, "ZMW": true, "MZN": true, "AOA": true,
	"BWP": true, "MAD": true, "EGP": true, "GNF": true, "ZWL": true,
}

func sanitiseCurrency(raw string) (string, error) {
	c := strings.ToUpper(strings.TrimSpace(raw))
	if !validCurrencies[c] {
		return "", fmt.Errorf("flutterwave: invalid or unsupported currency code %q", raw)
	}
	return c, nil
}

func normalizeCharge(data json.RawMessage, e onceo.Event, status onceo.Status) (onceo.Event, error) {
	var d ChargeData
	if err := json.Unmarshal(data, &d); err != nil {
		return e, onceo.ErrMalformedPayload
	}

	if d.ID == 0 {
		return e, fmt.Errorf("%w: charge event missing id", onceo.ErrEventParseFailed)
	}
	e.ProviderEventID = fmt.Sprintf("%d", d.ID)
	e.Status = mapStatus(d.Status, status)
	e.AmountMinor = int64(d.Amount)
	cur, err := sanitiseCurrency(d.Currency)
	if err != nil {
		return e, err
	}
	e.Currency = cur
	e.Reference = d.TxRef

	return e, nil
}

func normalizeTransfer(data json.RawMessage, e onceo.Event, status onceo.Status) (onceo.Event, error) {
	var d TransferData
	if err := json.Unmarshal(data, &d); err != nil {
		return e, onceo.ErrMalformedPayload
	}

	if d.ID == 0 {
		return e, fmt.Errorf("%w: transfer event missing id", onceo.ErrEventParseFailed)
	}
	e.ProviderEventID = fmt.Sprintf("%d", d.ID)
	e.Status = mapStatus(d.Status, status)
	e.AmountMinor = int64(d.Amount)
	cur, err := sanitiseCurrency(d.Currency)
	if err != nil {
		return e, err
	}
	e.Currency = cur
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
