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
	prefix     = "onceo:event:"
	defaultTTL = 24 * time.Hour
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

func key(provider, providerEventID string) string {
	return prefix + provider + ":" + providerEventID
}

func (s *Store) SaveIfNew(ctx context.Context, e onceo.Event) (bool, error) {
	if e.Provider == "" || e.ProviderEventID == "" {
		return false, fmt.Errorf("%w: provider and provider event id must not be empty", onceo.ErrStoreFailed)
	}

	data, err := json.Marshal(e)
	if err != nil {
		return false, fmt.Errorf("%w: marshal event: %w", onceo.ErrStoreFailed, err)
	}

	k := key(e.Provider, e.ProviderEventID)
	ok, err := s.client.SetNX(ctx, k, data, s.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("%w: redis setnx: %w", onceo.ErrStoreFailed, err)
	}
	return ok, nil
}
