package redisstore

import (
	"context"
	"errors"
	"testing"
	"time"

	onceo "github.com/HalxDocs/onceo-core"
	"github.com/redis/go-redis/v9"
)

func newTestClient(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		DB:           1,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("redis not available:", err)
	}
	return client
}

func cleanup(t *testing.T, client *redis.Client) {
	t.Helper()
	if err := client.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

func TestRedisStoreSeenBefore(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	defer cleanup(t, client)

	ctx := context.Background()
	store := New(client)

	seen, err := store.SeenBefore(ctx, "paystack", "evt_001")
	if err != nil {
		t.Fatalf("SeenBefore failed: %v", err)
	}
	if seen {
		t.Error("expected event not to be seen yet")
	}
}

func TestRedisStoreSaveAndCheck(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	defer cleanup(t, client)

	ctx := context.Background()
	store := New(client)

	event := onceo.Event{
		ID:              "evt_001",
		Provider:        "paystack",
		ProviderEventID: "evt_001",
		Type:            "charge.success",
		Status:          onceo.StatusSuccess,
		AmountMinor:     50000,
		Currency:        "NGN",
		Reference:       "ref_001",
		ReceivedAt:      time.Now().UTC(),
	}

	if err := store.Save(ctx, event); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	seen, err := store.SeenBefore(ctx, "paystack", "evt_001")
	if err != nil {
		t.Fatalf("SeenBefore failed: %v", err)
	}
	if !seen {
		t.Error("expected event to be seen after save")
	}
}

func TestRedisStoreDuplicateRejection(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	defer cleanup(t, client)

	ctx := context.Background()
	store := New(client)

	event := onceo.Event{
		ID:              "evt_001",
		Provider:        "paystack",
		ProviderEventID: "evt_001",
		Type:            "charge.success",
		Status:          onceo.StatusSuccess,
		AmountMinor:     50000,
		Currency:        "NGN",
		Reference:       "ref_001",
		ReceivedAt:      time.Now().UTC(),
	}

	if err := store.Save(ctx, event); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}

	err := store.Save(ctx, event)
	if !errors.Is(err, onceo.ErrDuplicateEvent) {
		t.Fatalf("expected ErrDuplicateEvent, got %v", err)
	}
}

func TestRedisStoreContextCancel(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	defer cleanup(t, client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	store := New(client)

	_, err := store.SeenBefore(ctx, "paystack", "evt_001")
	if err == nil {
		t.Error("expected error from cancelled context")
	}

	err = store.Save(ctx, onceo.Event{})
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestRedisStoreDifferentProviders(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	defer cleanup(t, client)

	ctx := context.Background()
	store := New(client)

	e1 := onceo.Event{
		ID: "evt_001", Provider: "paystack", ProviderEventID: "evt_001",
		Type: "charge.success", Status: onceo.StatusSuccess,
		AmountMinor: 1000, Currency: "NGN", Reference: "ref_001",
		ReceivedAt: time.Now().UTC(),
	}
	e2 := onceo.Event{
		ID: "evt_001", Provider: "flutterwave", ProviderEventID: "evt_001",
		Type: "charge.success", Status: onceo.StatusSuccess,
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

func TestRedisStoreTTL(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	defer cleanup(t, client)

	ctx := context.Background()
	store := NewWithTTL(client, 1*time.Second)

	event := onceo.Event{
		ID: "evt_001", Provider: "paystack", ProviderEventID: "evt_001",
		Type: "charge.success", Status: onceo.StatusSuccess,
		AmountMinor: 50000, Currency: "NGN", Reference: "ref_001",
		ReceivedAt: time.Now().UTC(),
	}

	if err := store.Save(ctx, event); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	seen, err := store.SeenBefore(ctx, "paystack", "evt_001")
	if err != nil {
		t.Fatalf("SeenBefore failed: %v", err)
	}
	if seen {
		t.Error("expected event to expire after TTL")
	}
}
