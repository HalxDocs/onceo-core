package onceo

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

type testProviderData struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type testProvider struct {
	name             string
	verifyErr        error
	parseResult      testProviderData
	parseErr         error
	normalizeResult  Event
	normalizeErr     error
	providerEventID  string
}

func (p *testProvider) Name() string { return p.name }

func (p *testProvider) VerifySignature(_ http.Header, _ []byte) error {
	return p.verifyErr
}

func (p *testProvider) Parse(_ []byte) (testProviderData, error) {
	return p.parseResult, p.parseErr
}

func (p *testProvider) Normalize(_ testProviderData) (Event, error) {
	return p.normalizeResult, p.normalizeErr
}

func TestProcessSuccess(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	provider := &testProvider{
		name:            "test",
		providerEventID: "evt_001",
		normalizeResult: Event{
			ID:              "evt_001",
			Provider:        "test",
			ProviderEventID: "evt_001",
			Type:            "charge.success",
			Status:          StatusSuccess,
			AmountMinor:     50000,
			Currency:        "NGN",
			Reference:       "ref_001",
			ReceivedAt:      time.Now().UTC(),
		},
	}

	event, err := Process(ctx, provider, store, http.Header{}, []byte(`{}`))
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if event.ProviderEventID != "evt_001" {
		t.Errorf("expected ProviderEventID evt_001, got %s", event.ProviderEventID)
	}

	seen, err := store.SeenBefore(ctx, "test", "evt_001")
	if err != nil {
		t.Fatalf("SeenBefore failed: %v", err)
	}
	if !seen {
		t.Error("expected event to be stored")
	}
}

func TestProcessBadSignature(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	provider := &testProvider{
		name:      "test",
		verifyErr: ErrInvalidSignature,
	}

	_, err := Process(ctx, provider, store, http.Header{}, []byte(`{}`))
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestProcessDuplicate(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	provider := &testProvider{
		name:            "test",
		providerEventID: "evt_001",
		normalizeResult: Event{
			ID:              "evt_001",
			Provider:        "test",
			ProviderEventID: "evt_001",
			Type:            "charge.success",
			Status:          StatusSuccess,
			AmountMinor:     50000,
			Currency:        "NGN",
			Reference:       "ref_001",
			ReceivedAt:      time.Now().UTC(),
		},
	}

	_, err := Process(ctx, provider, store, http.Header{}, []byte(`{}`))
	if err != nil {
		t.Fatalf("first Process failed: %v", err)
	}

	_, err = Process(ctx, provider, store, http.Header{}, []byte(`{}`))
	if !errors.Is(err, ErrDuplicateEvent) {
		t.Errorf("expected ErrDuplicateEvent, got %v", err)
	}
}

func TestProcessContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	store := NewMemoryStore()
	provider := &testProvider{name: "test"}

	_, err := Process(ctx, provider, store, http.Header{}, []byte(`{}`))
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestProcessProviderErrorWrapping(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	provider := &testProvider{
		name:      "testrail",
		verifyErr: ErrInvalidSignature,
	}

	_, err := Process(ctx, provider, store, http.Header{}, []byte(`{}`))

	var pErr *ProviderError
	if !errors.As(err, &pErr) {
		t.Fatal("expected *ProviderError")
	}
	if pErr.Provider != "testrail" {
		t.Errorf("expected provider testrail, got %s", pErr.Provider)
	}
}
