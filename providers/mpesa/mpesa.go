package mpesa

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	onceo "github.com/HalxDocs/onceo-core"
)

var ErrMpesaNoSignature = errors.New("mpesa: no cryptographic signature scheme; set InsecureSkipVerify=true and configure network-level IP allowlisting")

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
	InsecureSkipVerify bool
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) VerifySignature(headers http.Header, body []byte) error {
	if p.InsecureSkipVerify {
		return nil
	}
	return ErrMpesaNoSignature
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
					event.AmountMinor = extractAmount(item.Value)
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
	var amount int64
	fmt.Sscanf(raw.TransAmount, "%d", &amount)

	event := onceo.Event{
		Provider:        ProviderName,
		ProviderEventID: raw.TransID,
		Type:            "c2b.payment",
		Status:          onceo.StatusSuccess,
		AmountMinor:     amount * 100,
		Currency:        "KES",
		Reference:       raw.BillRefNumber,
		ReceivedAt:      time.Now().UTC(),
	}

	return event, nil
}

func extractAmount(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val * 100)
	case int:
		return int64(val * 100)
	case int64:
		return val * 100
	case string:
		amount, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return int64(amount * 100)
		}
	}
	return 0
}
