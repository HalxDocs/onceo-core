package onceo

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryStoreSaveIfNew(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	event := Event{
		ID:              "evt_001",
		Provider:        "paystack",
		ProviderEventID: "evt_001",
		Type:            "charge.success",
		Status:          StatusSuccess,
		AmountMinor:     50000,
		Currency:        "NGN",
		Reference:       "ref_001",
		ReceivedAt:      time.Now().UTC(),
	}

	created, err := store.SaveIfNew(ctx, event)
	if err != nil {
		t.Fatalf("SaveIfNew failed: %v", err)
	}
	if !created {
		t.Error("expected event to be created")
	}

	created, err = store.SaveIfNew(ctx, event)
	if err != nil {
		t.Fatalf("SaveIfNew failed: %v", err)
	}
	if created {
		t.Error("expected event NOT to be created on duplicate")
	}
}

func TestMemoryStoreDuplicateRejection(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	event := Event{
		ID:              "evt_001",
		Provider:        "paystack",
		ProviderEventID: "evt_001",
		Type:            "charge.success",
		Status:          StatusSuccess,
		AmountMinor:     50000,
		Currency:        "NGN",
		Reference:       "ref_001",
		ReceivedAt:      time.Now().UTC(),
	}

	created, err := store.SaveIfNew(ctx, event)
	if err != nil {
		t.Fatalf("first SaveIfNew failed: %v", err)
	}
	if !created {
		t.Fatal("expected first to be created")
	}

	created, err = store.SaveIfNew(ctx, event)
	if err != nil {
		t.Fatalf("second SaveIfNew failed: %v", err)
	}
	if created {
		t.Fatal("expected duplicate to return created=false")
	}
}

func TestMemoryStoreConcurrent(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			event := Event{
				ID:              "evt_001",
				Provider:        "paystack",
				ProviderEventID: "evt_001",
				Type:            "charge.success",
				Status:          StatusSuccess,
				AmountMinor:     int64(i),
				Currency:        "NGN",
				Reference:       "ref_001",
				ReceivedAt:      time.Now().UTC(),
			}
			_, _ = store.SaveIfNew(ctx, event)
		}(i)
	}
	wg.Wait()

	events := store.events
	if len(events) != 1 {
		t.Errorf("expected exactly 1 event saved, got %d", len(events))
	}
}

func TestMemoryStoreContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	store := NewMemoryStore()

	_, err := store.SaveIfNew(ctx, Event{Provider: "t", ProviderEventID: "1"})
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestMemoryStoreWithMax(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStoreWithMax(2)

	e1 := Event{ID: "1", Provider: "p", ProviderEventID: "1", ReceivedAt: time.Now().UTC()}
	e2 := Event{ID: "2", Provider: "p", ProviderEventID: "2", ReceivedAt: time.Now().UTC()}
	e3 := Event{ID: "3", Provider: "p", ProviderEventID: "3", ReceivedAt: time.Now().UTC()}

	if _, err := store.SaveIfNew(ctx, e1); err != nil {
		t.Fatalf("SaveIfNew e1 failed: %v", err)
	}
	if _, err := store.SaveIfNew(ctx, e2); err != nil {
		t.Fatalf("SaveIfNew e2 failed: %v", err)
	}
	if _, err := store.SaveIfNew(ctx, e3); err != nil {
		t.Fatalf("SaveIfNew e3 failed: %v", err)
	}

	created, err := store.SaveIfNew(ctx, e1)
	if err != nil {
		t.Fatalf("SaveIfNew e1 check failed: %v", err)
	}
	if !created {
		t.Error("e1 should have been evicted and re-created")
	}

	created, err = store.SaveIfNew(ctx, e3)
	if err != nil {
		t.Fatalf("SaveIfNew e3 check failed: %v", err)
	}
	if created {
		t.Error("e3 should already be present")
	}
}

func TestMemoryStoreWithMaxZero(t *testing.T) {
	store := NewMemoryStoreWithMax(0)
	if store.maxEvents != DefaultMaxMemoryEvents {
		t.Errorf("maxEvents = %d, want %d", store.maxEvents, DefaultMaxMemoryEvents)
	}
}

func TestMemoryStoreDifferentProviders(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	e1 := Event{
		ID: "evt_001", Provider: "paystack", ProviderEventID: "evt_001",
		Type: "charge.success", Status: StatusSuccess,
		AmountMinor: 1000, Currency: "NGN", Reference: "ref_001",
		ReceivedAt: time.Now().UTC(),
	}
	e2 := Event{
		ID: "evt_001", Provider: "flutterwave", ProviderEventID: "evt_001",
		Type: "charge.success", Status: StatusSuccess,
		AmountMinor: 2000, Currency: "NGN", Reference: "ref_002",
		ReceivedAt: time.Now().UTC(),
	}

	created1, err := store.SaveIfNew(ctx, e1)
	if err != nil {
		t.Fatalf("SaveIfNew e1 failed: %v", err)
	}
	if !created1 {
		t.Error("expected e1 to be created")
	}

	created2, err := store.SaveIfNew(ctx, e2)
	if err != nil {
		t.Fatalf("SaveIfNew e2 failed: %v", err)
	}
	if !created2 {
		t.Error("expected e2 to be created")
	}
}
