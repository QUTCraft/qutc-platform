package cache

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

func New(addr, password string, db int, ttl time.Duration) *Cache {
	return &Cache{
		client: redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db}),
		ttl:    ttl,
	}
}

func (c *Cache) Get(ctx context.Context, key string, destination any) bool {
	if c == nil || c.client == nil {
		return false
	}
	value, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	return json.Unmarshal([]byte(value), destination) == nil
}

func (c *Cache) Set(ctx context.Context, key string, value any) {
	if c == nil || c.client == nil || c.ttl <= 0 {
		return
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = c.client.Set(ctx, key, encoded, c.ttl).Err()
}

func (c *Cache) DeletePrefix(ctx context.Context, prefix string) {
	if c == nil || c.client == nil {
		return
	}
	var cursor uint64
	for {
		keys, next, err := c.client.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			_ = c.client.Del(ctx, keys...).Err()
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}

func NormalizeQuery(query string) string {
	if strings.TrimSpace(query) == "" {
		return "default"
	}
	values, err := url.ParseQuery(query)
	if err != nil || values.Encode() == "" {
		return "default"
	}
	return values.Encode()
}
