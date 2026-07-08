package onceo

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Store interface {
	SeenBefore(ctx context.Context, provider, providerEventID string) (bool, error)
	Save(ctx context.Context, e Event) error
}

type MemoryStore struct {
	mu     sync.RWMutex
	events map[string]Event
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		events: make(map[string]Event),
	}
}

func (s *MemoryStore) eventKey(provider, providerEventID string) string {
	return fmt.Sprintf("%s:%s", provider, providerEventID)
}

func (s *MemoryStore) SeenBefore(ctx context.Context, provider, providerEventID string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.events[s.eventKey(provider, providerEventID)]
	return ok, nil
}

func (s *MemoryStore) Save(ctx context.Context, e Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.eventKey(e.Provider, e.ProviderEventID)
	if _, exists := s.events[key]; exists {
		return ErrDuplicateEvent
	}

	e.ProcessedAt = time.Now().UTC()
	s.events[key] = e
	return nil
}
