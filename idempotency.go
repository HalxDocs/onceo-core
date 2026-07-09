package onceo

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const DefaultMaxMemoryEvents = 100000

// Store provides atomic idempotency — exactly-one delivery semantics
// for webhook events. Implementations MUST make SaveIfNew atomic: no
// caller-visible window where two concurrent calls for the same key can
// both return created=true.
type Store interface {
	SaveIfNew(ctx context.Context, e Event) (created bool, err error)
}

// MemoryStore is an in-memory Store, safe for concurrent use within a
// single process. When the maximum number of events is reached, the
// oldest entries are evicted (FIFO). MemoryStore is intended for tests
// and local development. For production, use store/redis or another
// durable Store implementation with a TTL matching your reconciliation
// window.
type MemoryStore struct {
	mu        sync.Mutex
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

func (s *MemoryStore) SaveIfNew(ctx context.Context, e Event) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.eventKey(e.Provider, e.ProviderEventID)
	if _, exists := s.events[key]; exists {
		return false, nil
	}

	if len(s.events) >= s.maxEvents {
		oldest := s.order[0]
		delete(s.events, oldest)
		s.order = s.order[1:]
	}

	e.ProcessedAt = time.Now().UTC()
	s.events[key] = e
	s.order = append(s.order, key)
	return true, nil
}
