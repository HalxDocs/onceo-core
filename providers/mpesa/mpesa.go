package mpesa

import (
	"crypto/hmac"
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

var ErrMpesaNoSignature = errors.New("mpesa: no cryptographic signature scheme; use callback URL path token verification instead")

const ProviderName = "mpesa"

type STKCallbackPayload struct {
	Body struct {
		StkCallback struct {
			MerchantRequestID string `json:"MerchantRequestID"`
			CheckoutRequestID  string `json:"CheckoutRequestID"`
			ResultCode         int    `json:"ResultCode"`
			ResultDesc         string `json:"ResultDesc"`
			CallbackMetadata   *struct {
				Item []struct {
					Name  string      `json:"Name"`
					Value interface{} `json:"Value"`
				} `json:"Item"`
			} `json:"CallbackMetadata"`
		} `json:"stkCallback"`
	} `json:"Body"`
}

type C2BPayload struct {
	TransactionType   string `json:"TransactionType"`
	TransID           string `json:"TransID"`
	TransTime         string `json:"TransTime"`
	TransAmount       string `json:"TransAmount"`
	BusinessShortCode string `json:"BusinessShortCode"`
	BillRefNumber     string `json:"BillRefNumber"`
	InvoiceNumber     string `json:"InvoiceNumber"`
	OrgAccountBalance string `json:"OrgAccountBalance"`
	ThirdPartyTransID string `json:"ThirdPartyTransID"`
	MSISDN            string `json:"MSISDN"`
	FirstName         string `json:"FirstName"`
	MiddleName        string `json:"MiddleName"`
	LastName          string `json:"LastName"`
}

type Provider struct {
	callbackToken string
}

func New(callbackToken string) (*Provider, error) {
	if len(callbackToken) < 32 {
		return nil, fmt.Errorf("mpesa: callback token must be at least 32 bytes of entropy")
	}
	return &Provider{callbackToken: callbackToken}, nil
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) VerifySignature(headers http.Header, body []byte) error {
	token, err := onceo.SingleHeader(headers, "X-Onceo-Mpesa-Callback-Token")
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(token), []byte(p.callbackToken)) {
		return onceo.ErrInvalidSignature
	}
	return nil
}

func (p *Provider) Parse(body []byte) (STKCallbackPayload, error) {
	var payload STKCallbackPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return payload, onceo.ErrMalformedPayload
	}
	if payload.Body.StkCallback.CheckoutRequestID == "" {
		return payload, onceo.ErrMalformedPayload
	}
	return payload, nil
}

func (p *Provider) Normalize(raw STKCallbackPayload) (onceo.Event, error) {
	cb := raw.Body.StkCallback

	if cb.CheckoutRequestID == "" {
		return onceo.Event{}, fmt.Errorf("mpesa: missing CheckoutRequestID, cannot deduplicate safely")
	}

	event := onceo.Event{
		Provider:        ProviderName,
		ProviderEventID: cb.CheckoutRequestID,
		Type:            "stk.push",
		Reference:       cb.MerchantRequestID,
		ReceivedAt:      time.Now().UTC(),
	}

	switch cb.ResultCode {
	case 0:
		event.Status = onceo.StatusSuccess
		if cb.CallbackMetadata != nil {
			for _, item := range cb.CallbackMetadata.Item {
				switch item.Name {
				case "Amount":
					var err error
					event.AmountMinor, err = parseMpesaAmount(item.Value)
					if err != nil {
						return onceo.Event{}, fmt.Errorf("mpesa: normalize: %w", err)
					}
				}
			}
		}
		event.Currency = "KES"
	case 1037:
		event.Status = onceo.StatusFailed
	case 1032:
		event.Status = onceo.StatusFailed
	default:
		event.Status = onceo.StatusFailed
	}

	return event, nil
}

func (p *Provider) ParseC2B(body []byte) (C2BPayload, error) {
	var payload C2BPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return payload, onceo.ErrMalformedPayload
	}
	return payload, nil
}

func (p *Provider) NormalizeC2B(raw C2BPayload) (onceo.Event, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(raw.TransAmount), 64)
	if err != nil {
		return onceo.Event{}, fmt.Errorf("mpesa: invalid c2b amount %q: %w", raw.TransAmount, err)
	}
	amount := int64(math.Round(f * 100))

	event := onceo.Event{
		Provider:        ProviderName,
		ProviderEventID: raw.TransID,
		Type:            "c2b.payment",
		Status:          onceo.StatusSuccess,
		AmountMinor:     amount,
		Currency:        "KES",
		Reference:       raw.BillRefNumber,
		ReceivedAt:      time.Now().UTC(),
	}

	return event, nil
}

func parseMpesaAmount(v interface{}) (int64, error) {
	switch val := v.(type) {
	case float64:
		return int64(math.Round(val * 100)), nil
	case int:
		return int64(val * 100), nil
	case int64:
		return val * 100, nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			return 0, fmt.Errorf("unparseable amount string %q: %w", val, err)
		}
		return int64(math.Round(f * 100)), nil
	case json.Number:
		f, err := val.Float64()
		if err != nil {
			return 0, fmt.Errorf("unparseable json.Number %q: %w", val, err)
		}
		return int64(math.Round(f * 100)), nil
	default:
		return 0, fmt.Errorf("unexpected amount type %T", v)
	}
}
