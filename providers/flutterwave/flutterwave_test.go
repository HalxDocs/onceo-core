package flutterwave

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

func signedHeaders(secretHash string, body []byte) http.Header {
	mac := hmac.New(sha256.New, []byte(secretHash))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	headers := http.Header{}
	headers.Set("Verif-Hash", sig)
	return headers
}

func TestVerifySignature(t *testing.T) {
	secretHash := "flw_secret_hash"
	body := loadFixture(t, "charge_completed.json")

	p := &Provider{SecretHash: secretHash}

	tests := []struct {
		name    string
		headers http.Header
		body    []byte
		wantErr bool
	}{
		{
			name:    "valid signature",
			headers: signedHeaders(secretHash, body),
			body:    body,
			wantErr: false,
		},
		{
			name:    "missing header",
			headers: http.Header{},
			body:    body,
			wantErr: true,
		},
		{
			name:    "wrong secret",
			headers: signedHeaders("wrong_secret", body),
			body:    body,
			wantErr: true,
		},
		{
			name:    "tampered body",
			headers: signedHeaders(secretHash, body),
			body:    []byte(`{"event":"charge.failed"}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := p.VerifySignature(tt.headers, tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifySignature() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParse(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name    string
		body    []byte
		wantErr bool
		wantEvt string
	}{
		{
			name:    "charge completed",
			body:    loadFixture(t, "charge_completed.json"),
			wantErr: false,
			wantEvt: "charge.completed",
		},
		{
			name:    "charge failed",
			body:    loadFixture(t, "charge_failed.json"),
			wantErr: false,
			wantEvt: "charge.failed",
		},
		{
			name:    "malformed json",
			body:    []byte(`not json`),
			wantErr: true,
		},
		{
			name:    "empty json",
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
			if !tt.wantErr && got.Event != tt.wantEvt {
				t.Errorf("Parse() event = %q, want %q", got.Event, tt.wantEvt)
			}
		})
	}
}

func TestNormalizeChargeCompleted(t *testing.T) {
	p := &Provider{}
	body := loadFixture(t, "charge_completed.json")

	parsed, err := p.Parse(body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	event, err := p.Normalize(parsed)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	if event.Provider != "flutterwave" {
		t.Errorf("Provider = %q, want %q", event.Provider, "flutterwave")
	}
	if event.ProviderEventID != "9876543" {
		t.Errorf("ProviderEventID = %q, want %q", event.ProviderEventID, "9876543")
	}
	if event.Type != "charge.completed" {
		t.Errorf("Type = %q, want %q", event.Type, "charge.completed")
	}
	if event.Status != onceo.StatusSuccess {
		t.Errorf("Status = %v, want %v", event.Status, onceo.StatusSuccess)
	}
	if event.AmountMinor != 15000 {
		t.Errorf("AmountMinor = %d, want %d", event.AmountMinor, 15000)
	}
	if event.Currency != "NGN" {
		t.Errorf("Currency = %q, want %q", event.Currency, "NGN")
	}
	if event.Reference != "flw_ref_abc123" {
		t.Errorf("Reference = %q, want %q", event.Reference, "flw_ref_abc123")
	}
}

func TestNormalizeChargeFailed(t *testing.T) {
	p := &Provider{}
	body := loadFixture(t, "charge_failed.json")

	parsed, err := p.Parse(body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	event, err := p.Normalize(parsed)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	if event.Type != "charge.failed" {
		t.Errorf("Type = %q, want %q", event.Type, "charge.failed")
	}
	if event.Status != onceo.StatusFailed {
		t.Errorf("Status = %v, want %v", event.Status, onceo.StatusFailed)
	}
	if event.AmountMinor != 30000 {
		t.Errorf("AmountMinor = %d, want %d", event.AmountMinor, 30000)
	}
}

func TestNormalizeUnknownEvent(t *testing.T) {
	raw := struct {
		Event string          `json:"event"`
		Data  json.RawMessage `json:"data"`
	}{
		Event: "unknown.event",
		Data:  json.RawMessage(`{}`),
	}

	body, _ := json.Marshal(raw)
	p := &Provider{}
	parsed, err := p.Parse(body)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	_, err = p.Normalize(parsed)
	if err == nil {
		t.Fatal("expected error for unknown event type")
	}
}

func TestProcessFullFlow(t *testing.T) {
	secretHash := "flw_secret_hash"
	body := loadFixture(t, "charge_completed.json")
	headers := signedHeaders(secretHash, body)

	p := &Provider{SecretHash: secretHash}
	store := onceo.NewMemoryStore()

	event, err := onceo.Process(t.Context(), p, store, headers, body)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if event.Provider != "flutterwave" {
		t.Errorf("Provider = %q, want %q", event.Provider, "flutterwave")
	}
	if event.Reference != "flw_ref_abc123" {
		t.Errorf("Reference = %q, want %q", event.Reference, "flw_ref_abc123")
	}
}

func TestProcessDuplicate(t *testing.T) {
	secretHash := "flw_secret_hash"
	body := loadFixture(t, "charge_completed.json")
	headers := signedHeaders(secretHash, body)

	p := &Provider{SecretHash: secretHash}
	store := onceo.NewMemoryStore()

	_, err := onceo.Process(t.Context(), p, store, headers, body)
	if err != nil {
		t.Fatalf("first Process failed: %v", err)
	}

	_, err = onceo.Process(t.Context(), p, store, headers, body)
	if !errors.Is(err, onceo.ErrDuplicateEvent) {
		t.Errorf("expected ErrDuplicateEvent, got %v", err)
	}
}

func FuzzParse(f *testing.F) {
	fixtures := []string{"charge_completed.json", "charge_failed.json"}
	for _, name := range fixtures {
		data, err := os.ReadFile("testdata/" + name)
		if err == nil {
			f.Add(data)
		}
	}
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"event":""}`))
	f.Add([]byte(`not json`))

	p := &Provider{}

	f.Fuzz(func(t *testing.T, body []byte) {
		_, err := p.Parse(body)
		if err != nil && err != onceo.ErrMalformedPayload {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
