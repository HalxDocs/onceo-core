package mpesa

import (
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
		name    string
		headers http.Header
		allowed []string
		wantErr bool
	}{
		{
			name:    "no IP restriction",
			headers: http.Header{},
			allowed: nil,
			wantErr: false,
		},
		{
			name:    "allowed IP",
			headers: http.Header{"X-Forwarded-For": {"196.1.2.3"}},
			allowed: []string{"196.1.2.3"},
			wantErr: false,
		},
		{
			name:    "disallowed IP",
			headers: http.Header{"X-Forwarded-For": {"1.2.3.4"}},
			allowed: []string{"196.1.2.3"},
			wantErr: true,
		},
		{
			name:    "allowed via X-Real-IP",
			headers: http.Header{"X-Real-IP": {"196.1.2.3"}},
			allowed: []string{"196.1.2.3"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{AllowedIPs: tt.allowed}
			err := p.VerifySignature(tt.headers, nil)
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

func TestProcessFullFlow(t *testing.T) {
	body := loadFixture(t, "stk_success.json")

	p := &Provider{}
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
