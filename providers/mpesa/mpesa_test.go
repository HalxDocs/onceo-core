package mpesa

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	onceo "github.com/HalxDocs/onceo-core"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return data
}

func TestVerifySignature(t *testing.T) {
	tests := []struct {
		name              string
		insecureSkipVerify bool
		wantErr           bool
	}{
		{
			name:              "insecure skip verify",
			insecureSkipVerify: true,
			wantErr:           false,
		},
		{
			name:              "default no verification",
			insecureSkipVerify: false,
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{InsecureSkipVerify: tt.insecureSkipVerify}
			err := p.VerifySignature(http.Header{}, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifySignature() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseSTK(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name    string
		body    []byte
		wantErr bool
		wantID  string
	}{
		{
			name:    "stk success",
			body:    loadFixture(t, "stk_success.json"),
			wantErr: false,
			wantID:  "ws_CO_07072026123000000",
		},
		{
			name:    "stk failed",
			body:    loadFixture(t, "stk_failed.json"),
			wantErr: false,
			wantID:  "ws_CO_07072026123100000",
		},
		{
			name:    "malformed json",
			body:    []byte(`not json`),
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    []byte(`{}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Parse(tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Body.StkCallback.CheckoutRequestID != tt.wantID {
				t.Errorf("Parse() CheckoutRequestID = %q, want %q", got.Body.StkCallback.CheckoutRequestID, tt.wantID)
			}
		})
	}
}

func TestNormalizeSTKSuccess(t *testing.T) {
	p := &Provider{}
	body := loadFixture(t, "stk_success.json")

	parsed, err := p.Parse(body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	event, err := p.Normalize(parsed)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	if event.Provider != "mpesa" {
		t.Errorf("Provider = %q, want %q", event.Provider, "mpesa")
	}
	if event.Status != onceo.StatusSuccess {
		t.Errorf("Status = %v, want %v", event.Status, onceo.StatusSuccess)
	}
	if event.Type != "stk.push" {
		t.Errorf("Type = %q, want %q", event.Type, "stk.push")
	}
	if event.AmountMinor != 150000 {
		t.Errorf("AmountMinor = %d, want %d", event.AmountMinor, 150000)
	}
	if event.Currency != "KES" {
		t.Errorf("Currency = %q, want %q", event.Currency, "KES")
	}
}

func TestNormalizeSTKFailed(t *testing.T) {
	p := &Provider{}
	body := loadFixture(t, "stk_failed.json")

	parsed, err := p.Parse(body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	event, err := p.Normalize(parsed)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	if event.Status != onceo.StatusFailed {
		t.Errorf("Status = %v, want %v", event.Status, onceo.StatusFailed)
	}
	if event.Type != "stk.push" {
		t.Errorf("Type = %q, want %q", event.Type, "stk.push")
	}
}

func TestProviderName(t *testing.T) {
	p := &Provider{}
	if got := p.Name(); got != "mpesa" {
		t.Errorf("Name() = %q, want %q", got, "mpesa")
	}
}

func TestExtractAmount(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int64
	}{
		{"float64", float64(1500.0), 150000},
		{"int", 1500, 150000},
		{"int64", int64(1500), 150000},
		{"string", "1500", 150000},
		{"string decimal", "12.50", 1250},
		{"nil", nil, 0},
		{"invalid string", "abc", 0},
		{"bool", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractAmount(tt.input); got != tt.want {
				t.Errorf("extractAmount(%v) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseC2B(t *testing.T) {
	p := &Provider{}
	body := loadFixture(t, "c2b_payment.json")

	payload, err := p.ParseC2B(body)
	if err != nil {
		t.Fatalf("ParseC2B failed: %v", err)
	}

	if payload.TransID != "LGR12G8H1" {
		t.Errorf("TransID = %q, want %q", payload.TransID, "LGR12G8H1")
	}
	if payload.BillRefNumber != "INV-001" {
		t.Errorf("BillRefNumber = %q, want %q", payload.BillRefNumber, "INV-001")
	}
	if payload.TransAmount != "2000" {
		t.Errorf("TransAmount = %q, want %q", payload.TransAmount, "2000")
	}
}

func TestParseC2BMalformed(t *testing.T) {
	p := &Provider{}
	_, err := p.ParseC2B([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestNormalizeC2B(t *testing.T) {
	p := &Provider{}
	body := loadFixture(t, "c2b_payment.json")

	parsed, err := p.ParseC2B(body)
	if err != nil {
		t.Fatalf("ParseC2B failed: %v", err)
	}

	event, err := p.NormalizeC2B(parsed)
	if err != nil {
		t.Fatalf("NormalizeC2B failed: %v", err)
	}

	if event.Provider != "mpesa" {
		t.Errorf("Provider = %q, want %q", event.Provider, "mpesa")
	}
	if event.Type != "c2b.payment" {
		t.Errorf("Type = %q, want %q", event.Type, "c2b.payment")
	}
	if event.Status != onceo.StatusSuccess {
		t.Errorf("Status = %v, want %v", event.Status, onceo.StatusSuccess)
	}
	if event.ProviderEventID != "LGR12G8H1" {
		t.Errorf("ProviderEventID = %q, want %q", event.ProviderEventID, "LGR12G8H1")
	}
	if event.AmountMinor != 200000 {
		t.Errorf("AmountMinor = %d, want %d", event.AmountMinor, 200000)
	}
	if event.Currency != "KES" {
		t.Errorf("Currency = %q, want %q", event.Currency, "KES")
	}
	if event.Reference != "INV-001" {
		t.Errorf("Reference = %q, want %q", event.Reference, "INV-001")
	}
}

func TestProcessFullFlow(t *testing.T) {
	body := loadFixture(t, "stk_success.json")

	p := &Provider{InsecureSkipVerify: true}
	store := onceo.NewMemoryStore()

	event, err := onceo.Process(t.Context(), p, store, http.Header{}, body)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if event.Provider != "mpesa" {
		t.Errorf("Provider = %q, want %q", event.Provider, "mpesa")
	}
	if event.Status != onceo.StatusSuccess {
		t.Errorf("Status = %v, want %v", event.Status, onceo.StatusSuccess)
	}
	if event.Currency != "KES" {
		t.Errorf("Currency = %q, want %q", event.Currency, "KES")
	}
}

func TestNormalizeSTKResultCode1032(t *testing.T) {
	p := &Provider{}
	payload := struct {
		Body struct {
			StkCallback struct {
				MerchantRequestID string `json:"MerchantRequestID"`
				CheckoutRequestID  string `json:"CheckoutRequestID"`
				ResultCode         int    `json:"ResultCode"`
				ResultDesc         string `json:"ResultDesc"`
			} `json:"stkCallback"`
		} `json:"Body"`
	}{}
	payload.Body.StkCallback.CheckoutRequestID = "ws_test_1032"
	payload.Body.StkCallback.ResultCode = 1032
	payload.Body.StkCallback.ResultDesc = "Request cancelled by user"

	body, _ := json.Marshal(payload)
	parsed, err := p.Parse(body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	event, err := p.Normalize(parsed)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}
	if event.Status != onceo.StatusFailed {
		t.Errorf("Status = %v, want %v", event.Status, onceo.StatusFailed)
	}
}

func TestNormalizeSTKDefaultResultCode(t *testing.T) {
	p := &Provider{}
	payload := struct {
		Body struct {
			StkCallback struct {
				MerchantRequestID string `json:"MerchantRequestID"`
				CheckoutRequestID  string `json:"CheckoutRequestID"`
				ResultCode         int    `json:"ResultCode"`
				ResultDesc         string `json:"ResultDesc"`
			} `json:"stkCallback"`
		} `json:"Body"`
	}{}
	payload.Body.StkCallback.CheckoutRequestID = "ws_test_9999"
	payload.Body.StkCallback.ResultCode = 9999

	body, _ := json.Marshal(payload)
	parsed, err := p.Parse(body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	event, err := p.Normalize(parsed)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}
	if event.Status != onceo.StatusFailed {
		t.Errorf("Status = %v, want %v", event.Status, onceo.StatusFailed)
	}
}

func FuzzParse(f *testing.F) {
	fixtures := []string{"stk_success.json", "stk_failed.json"}
	for _, name := range fixtures {
		data, err := os.ReadFile("testdata/" + name)
		if err == nil {
			f.Add(data)
		}
	}
	f.Add([]byte(`{}`))
	f.Add([]byte(`not json`))

	p := &Provider{}

	f.Fuzz(func(t *testing.T, body []byte) {
		_, err := p.Parse(body)
		if err != nil && err != onceo.ErrMalformedPayload {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
