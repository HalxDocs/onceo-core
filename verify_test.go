package onceo

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"net/http"
	"testing"
)

func TestVerifyHMACSHA512(t *testing.T) {
	secret := []byte("test-secret-key")
	body := []byte(`{"event":"charge.success","data":{"id":"123"}}`)

	mac := hmac.New(sha512.New, secret)
	mac.Write(body)
	validSig := hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		secret    []byte
		body      []byte
		signature string
		wantErr   bool
	}{
		{"valid signature", secret, body, validSig, false},
		{"wrong body", secret, []byte(`{"event":"charge.failed"}`), validSig, true},
		{"wrong secret", []byte("wrong-secret"), body, validSig, true},
		{"empty signature", secret, body, "", true},
		{"tampered signature", secret, body, "abc123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyHMACSHA512(tt.secret, tt.body, tt.signature)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyHMACSHA512() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestVerifyHMACSHA512ConstantTime(t *testing.T) {
	secret := []byte("test-secret-key")
	body := []byte(`{"event":"charge.success"}`)

	mac := hmac.New(sha512.New, secret)
	mac.Write(body)
	validSig := hex.EncodeToString(mac.Sum(nil))

	err := VerifyHMACSHA512(secret, body, validSig)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestExtractHeader(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Paystack-Signature", "test_sig")

	tests := []struct {
		name      string
		key       string
		want      string
		wantErr   bool
	}{
		{"existing header", "X-Paystack-Signature", "test_sig", false},
		{"missing header", "X-Missing", "", true},
		{"case insensitive", "x-paystack-signature", "test_sig", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractHeader(headers, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractHeader() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVerifyWithExtractHeader(t *testing.T) {
	secret := []byte("sk_test_key")
	body := []byte(`{"event":"charge.success"}`)

	mac := hmac.New(sha512.New, secret)
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	headers := http.Header{}
	headers.Set("X-Paystack-Signature", sig)

	signature, err := ExtractHeader(headers, "X-Paystack-Signature")
	if err != nil {
		t.Fatalf("ExtractHeader failed: %v", err)
	}

	if err := VerifyHMACSHA512(secret, body, signature); err != nil {
		t.Errorf("VerifyHMACSHA512 failed: %v", err)
	}
}
