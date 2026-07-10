package opay

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
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
	secretKey string
}

func New(secretKey string) (*Provider, error) {
	if len(secretKey) == 0 {
		return nil, fmt.Errorf("opay: secret key must not be empty")
	}
	return &Provider{secretKey: secretKey}, nil
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) VerifySignature(headers http.Header, body []byte) error {
	sig, err := onceo.SingleHeader(headers, "Authorization")
	if errors.Is(err, onceo.ErrMissingHeader) {
		var sig2 string
		sig2, err2 := onceo.SingleHeader(headers, "Opay-Signature")
		if err2 != nil {
			return fmt.Errorf("opay: neither Authorization nor Opay-Signature present (%v, %v)", err, err2)
		}
		sig = sig2
	} else if err != nil {
		return err
	}
	return onceo.VerifyHMACSHA512([]byte(p.secretKey), body, sig)
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

var validCurrencies = map[string]bool{
	"NGN": true, "USD": true, "GBP": true, "EUR": true, "ZAR": true,
	"GHS": true, "KES": true, "UGX": true, "TZS": true, "RWF": true,
	"XAF": true, "XOF": true, "ZMW": true, "MZN": true, "AOA": true,
	"BWP": true, "MAD": true, "EGP": true, "GNF": true, "ZWL": true,
}

func sanitiseCurrency(raw string) (string, error) {
	c := strings.ToUpper(strings.TrimSpace(raw))
	if !validCurrencies[c] {
		return "", fmt.Errorf("opay: invalid or unsupported currency code %q", raw)
	}
	return c, nil
}

func normalizePayment(data json.RawMessage, e onceo.Event, status onceo.Status) (onceo.Event, error) {
	var d PaymentData
	if err := json.Unmarshal(data, &d); err != nil {
		return e, onceo.ErrMalformedPayload
	}

	if d.OrderNo == "" {
		return e, fmt.Errorf("%w: payment event missing orderNo", onceo.ErrEventParseFailed)
	}
	e.ProviderEventID = d.OrderNo
	e.Status = status
	e.Reference = d.Reference
	cur, err := sanitiseCurrency(d.Currency)
	if err != nil {
		return e, err
	}
	e.Currency = cur

	amount, err := parseOpayAmount(d.Amount)
	if err != nil {
		return e, fmt.Errorf("opay: normalize: %w", err)
	}
	e.AmountMinor = amount

	return e, nil
}

func parseOpayAmount(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q: %w", raw, err)
	}
	if f < 0 {
		return 0, fmt.Errorf("negative amount %q", raw)
	}
	return int64(math.Round(f * 100)), nil
}
