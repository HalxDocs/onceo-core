package onceo

import (
	"container/list"
	"context"
	"fmt"
	"strings"
	"sync"
)

const DefaultMaxMemoryEvents = 100000

// Store provides atomic idempotency — exactly-one delivery semantics
// for webhook events. Implementations MUST make SaveIfNew atomic: no
// caller-visible window where two concurrent calls for the same key can
// both return created=true.
//
// WARNING: Implementations that cap storage without a TTL (e.g., a
// fixed-size ring buffer) violate the exactly-one contract. Once an
// event is evicted, a subsequent SaveIfNew for the same key returns
// created=true again, producing a duplicate. In-memory stores are
// suitable only for tests and local development.
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
	mu       sync.Mutex
	events   map[string]Event
	order    *list.List
	maxItems int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		events:   make(map[string]Event),
		order:    list.New(),
		maxItems: DefaultMaxMemoryEvents,
	}
}

func NewMemoryStoreWithMax(max int) *MemoryStore {
	if max <= 0 {
		max = DefaultMaxMemoryEvents
	}
	return &MemoryStore{
		events:   make(map[string]Event),
		order:    list.New(),
		maxItems: max,
	}
}

func (s *MemoryStore) eventKey(provider, providerEventID string) string {
	escapedProvider := strings.ReplaceAll(provider, "\\", "\\\\")
	escapedProvider = strings.ReplaceAll(escapedProvider, ":", "\\:")
	escapedID := strings.ReplaceAll(providerEventID, "\\", "\\\\")
	escapedID = strings.ReplaceAll(escapedID, ":", "\\:")
	return escapedProvider + ":" + escapedID
}

func (s *MemoryStore) SaveIfNew(ctx context.Context, e Event) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	if e.Provider == "" || e.ProviderEventID == "" {
		return false, fmt.Errorf("%w: provider and provider event id must not be empty", ErrStoreFailed)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := s.eventKey(e.Provider, e.ProviderEventID)
	if _, exists := s.events[key]; exists {
		return false, nil
	}

	if s.maxItems > 0 && len(s.events) >= s.maxItems {
		for s.order.Len() > 0 {
			oldest := s.order.Front()
			oldKey, ok := oldest.Value.(string)
			if !ok {
				s.order.Remove(oldest)
				continue
			}
			delete(s.events, oldKey)
			s.order.Remove(oldest)
			break
		}
	}

	s.events[key] = e
	s.order.PushBack(key)
	return true, nil
}
