package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	Redis *redis.Client
}

func New(addr, password string, db int) *Client {
	return &Client{Redis: redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db, PoolSize: 100})}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.Redis.Ping(ctx).Err()
}

func (c *Client) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.Redis.Set(ctx, key, b, ttl).Err()
}

func (c *Client) GetJSON(ctx context.Context, key string, out any) (bool, error) {
	val, err := c.Redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(val), out); err != nil {
		return false, err
	}
	return true, nil
}
