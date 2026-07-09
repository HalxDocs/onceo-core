package onceo

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestMemoryStoreSeenBefore(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	seen, err := store.SeenBefore(ctx, "paystack", "evt_001")
	if err != nil {
		t.Fatalf("SeenBefore failed: %v", err)
	}
	if seen {
		t.Error("expected event not to be seen yet")
	}

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

	if err := store.Save(ctx, event); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	seen, err = store.SeenBefore(ctx, "paystack", "evt_001")
	if err != nil {
		t.Fatalf("SeenBefore failed: %v", err)
	}
	if !seen {
		t.Error("expected event to be seen after save")
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

	if err := store.Save(ctx, event); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}

	if err := store.Save(ctx, event); err != ErrDuplicateEvent {
		t.Fatalf("expected ErrDuplicateEvent, got %v", err)
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
			_ = store.Save(ctx, event)
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

	_, err := store.SeenBefore(ctx, "paystack", "evt_001")
	if err == nil {
		t.Error("expected error from cancelled context")
	}

	err = store.Save(ctx, Event{})
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

	if err := store.Save(ctx, e1); err != nil {
		t.Fatalf("Save e1 failed: %v", err)
	}
	if err := store.Save(ctx, e2); err != nil {
		t.Fatalf("Save e2 failed: %v", err)
	}
	if err := store.Save(ctx, e3); err != nil {
		t.Fatalf("Save e3 failed: %v", err)
	}

	seen1, _ := store.SeenBefore(ctx, "p", "1")
	if seen1 {
		t.Error("e1 should have been evicted")
	}
	seen3, _ := store.SeenBefore(ctx, "p", "3")
	if !seen3 {
		t.Error("e3 should be present")
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

	if err := store.Save(ctx, e1); err != nil {
		t.Fatalf("Save e1 failed: %v", err)
	}
	if err := store.Save(ctx, e2); err != nil {
		t.Fatalf("Save e2 failed: %v", err)
	}

	seen1, _ := store.SeenBefore(ctx, "paystack", "evt_001")
	seen2, _ := store.SeenBefore(ctx, "flutterwave", "evt_001")
	if !seen1 || !seen2 {
		t.Error("both events should be seen")
	}
}
