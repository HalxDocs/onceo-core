package redisstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	onceo "github.com/HalxDocs/onceo-core"
	"github.com/redis/go-redis/v9"
)

const (
	idempotencyPrefix = "onceo:idempotency:"
	eventPrefix       = "onceo:event:"
	defaultTTL        = 24 * time.Hour
)

type Store struct {
	client *redis.Client
	ttl    time.Duration
}

func New(client *redis.Client) *Store {
	return &Store{
		client: client,
		ttl:    defaultTTL,
	}
}

func NewWithTTL(client *redis.Client, ttl time.Duration) *Store {
	return &Store{
		client: client,
		ttl:    ttl,
	}
}

func idempotencyKey(provider, providerEventID string) string {
	return idempotencyPrefix + fmt.Sprintf("%s:%s", provider, providerEventID)
}

func eventKey(provider, providerEventID string) string {
	return eventPrefix + fmt.Sprintf("%s:%s", provider, providerEventID)
}

func (s *Store) SeenBefore(ctx context.Context, provider, providerEventID string) (bool, error) {
	exists, err := s.client.Exists(ctx, idempotencyKey(provider, providerEventID)).Result()
	if err != nil {
		return false, fmt.Errorf("%w: redis exists: %w", onceo.ErrStoreFailed, err)
	}
	return exists == 1, nil
}

func (s *Store) Save(ctx context.Context, e onceo.Event) error {
	key := idempotencyKey(e.Provider, e.ProviderEventID)

	ok, err := s.client.SetNX(ctx, key, "1", s.ttl).Result()
	if err != nil {
		return fmt.Errorf("%w: redis setnx: %w", onceo.ErrStoreFailed, err)
	}
	if !ok {
		return onceo.ErrDuplicateEvent
	}

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("%w: marshal event: %w", onceo.ErrStoreFailed, err)
	}

	if err := s.client.Set(ctx, eventKey(e.Provider, e.ProviderEventID), data, s.ttl).Err(); err != nil {
		return fmt.Errorf("%w: redis set event: %w", onceo.ErrStoreFailed, err)
	}

	return nil
}
