package opay

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	onceo "github.com/HalxDocs/onceo-core"
)

const ProviderName = "opay"

type WebhookPayload struct {
	EventType string          `json:"eventType"`
	Data      json.RawMessage `json:"data"`
}

type PaymentData struct {
	Reference    string `json:"reference"`
	Amount       string `json:"amount"`
	Currency     string `json:"currency"`
	Status       string `json:"status"`
	OrderNo      string `json:"orderNo"`
	CompleteTime string `json:"completeTime"`
}

type Provider struct {
	MerchantID string
	SecretKey  string
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) VerifySignature(headers http.Header, body []byte) error {
	signature, err := onceo.ExtractHeader(headers, "Authorization")
	if err != nil {
		signature, err = onceo.ExtractHeader(headers, "Opay-Signature")
		if err != nil {
			return err
		}
	}

	mac := hmac.New(sha512.New, []byte(p.SecretKey))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return onceo.ErrInvalidSignature
	}
	return nil
}

func (p *Provider) Parse(body []byte) (WebhookPayload, error) {
	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return payload, onceo.ErrMalformedPayload
	}
	if payload.EventType == "" {
		return payload, onceo.ErrMalformedPayload
	}
	return payload, nil
}

func (p *Provider) Normalize(raw WebhookPayload) (onceo.Event, error) {
	event := onceo.Event{
		Provider:   ProviderName,
		Type:       raw.EventType,
		ReceivedAt: time.Now().UTC(),
	}

	switch raw.EventType {
	case "PAYMENT_SUCCESSFUL":
		return normalizePayment(raw.Data, event, onceo.StatusSuccess)
	case "PAYMENT_FAILED":
		return normalizePayment(raw.Data, event, onceo.StatusFailed)
	case "REFUND_SUCCESSFUL":
		return normalizePayment(raw.Data, event, onceo.StatusReversed)
	default:
		return onceo.Event{}, fmt.Errorf("%w: unknown event type: %s", onceo.ErrEventParseFailed, raw.EventType)
	}
}

func normalizePayment(data json.RawMessage, e onceo.Event, status onceo.Status) (onceo.Event, error) {
	var d PaymentData
	if err := json.Unmarshal(data, &d); err != nil {
		return e, onceo.ErrMalformedPayload
	}

	e.ProviderEventID = d.OrderNo
	e.Status = status
	e.Reference = d.Reference
	e.Currency = d.Currency

	amount := int64(0)
	fmt.Sscanf(d.Amount, "%d", &amount)
	e.AmountMinor = amount

	return e, nil
}
