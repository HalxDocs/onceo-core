package onceo

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const DefaultMaxMemoryEvents = 100000

type Store interface {
	SeenBefore(ctx context.Context, provider, providerEventID string) (bool, error)
	Save(ctx context.Context, e Event) error
}

type MemoryStore struct {
	mu        sync.RWMutex
	events    map[string]Event
	order     []string
	maxEvents int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		events:    make(map[string]Event),
		maxEvents: DefaultMaxMemoryEvents,
	}
}

func NewMemoryStoreWithMax(max int) *MemoryStore {
	if max <= 0 {
		max = DefaultMaxMemoryEvents
	}
	return &MemoryStore{
		events:    make(map[string]Event),
		maxEvents: max,
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

	if len(s.events) >= s.maxEvents {
		oldest := s.order[0]
		delete(s.events, oldest)
		s.order = s.order[1:]
	}

	e.ProcessedAt = time.Now().UTC()
	s.events[key] = e
	s.order = append(s.order, key)
	return nil
}
