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
	saveScript        = `
local ok = redis.call("SETNX", KEYS[1], "1")
if ok == 1 then
    redis.call("SET", KEYS[2], ARGV[1])
    redis.call("EXPIRE", KEYS[1], ARGV[2])
    redis.call("EXPIRE", KEYS[2], ARGV[2])
    return 1
end
return 0
`
)

type Store struct {
	client *redis.Client
	ttl    time.Duration
	cmd    *redis.Script
}

func New(client *redis.Client) *Store {
	return &Store{
		client: client,
		ttl:    defaultTTL,
		cmd:    redis.NewScript(saveScript),
	}
}

func NewWithTTL(client *redis.Client, ttl time.Duration) *Store {
	return &Store{
		client: client,
		ttl:    ttl,
		cmd:    redis.NewScript(saveScript),
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
	idKey := idempotencyKey(e.Provider, e.ProviderEventID)
	evKey := eventKey(e.Provider, e.ProviderEventID)

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("%w: marshal event: %w", onceo.ErrStoreFailed, err)
	}

	ttl := int64(s.ttl.Seconds())
	result, err := s.cmd.Run(ctx, s.client, []string{idKey, evKey}, string(data), ttl).Int()
	if err != nil {
		return fmt.Errorf("%w: redis save: %w", onceo.ErrStoreFailed, err)
	}
	if result == 0 {
		return onceo.ErrDuplicateEvent
	}
	return nil
}
