package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/yourusername/signalfence/core"
)

// RedisStore provides Redis-backed storage for bucket states
type RedisStore struct {
	client *redis.Client
	ctx    context.Context
	ttl    time.Duration // How long to keep bucket state in Redis
}

// Ensure RedisStore implements Store interface
var _ Store = (*RedisStore)(nil)

// RedisConfig for creating a Redis store
type RedisConfig struct {
	Addr     string        // Redis address (e.g., "localhost:6379")
	Password string        // Redis password (empty for no auth)
	DB       int           // Redis database number
	TTL      time.Duration // TTL for bucket states (default: 1 hour)
}

// NewRedisStore creates a new Redis-backed store
func NewRedisStore(config RedisConfig) *RedisStore {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	ttl := config.TTL
	if ttl == 0 {
		ttl = 1 * time.Hour // Default TTL
	}

	return &RedisStore{
		client: client,
		ctx:    context.Background(),
		ttl:    ttl,
	}
}

// Get retrieves the bucket state for a given key
func (s *RedisStore) Get(key string) *core.BucketState {
	redisKey := "signalfence:" + key
	
	val, err := s.client.Get(s.ctx, redisKey).Result()
	if err != nil {
		// Key doesn't exist or error occurred
		return nil
	}

	var state core.BucketState
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		return nil
	}

	return &state
}

// Set stores the bucket state for a given key
func (s *RedisStore) Set(key string, state *core.BucketState) {
	redisKey := "signalfence:" + key
	
	data, err := json.Marshal(state)
	if err != nil {
		return
	}

	s.client.Set(s.ctx, redisKey, data, s.ttl)
}

// Delete removes the bucket state for a given key
func (s *RedisStore) Delete(key string) {
	redisKey := "signalfence:" + key
	s.client.Del(s.ctx, redisKey)
}

// Clear removes all SignalFence keys from Redis
func (s *RedisStore) Clear() {
	// Scan for all signalfence: keys
	iter := s.client.Scan(s.ctx, 0, "signalfence:*", 0).Iterator()
	for iter.Next(s.ctx) {
		s.client.Del(s.ctx, iter.Val())
	}
}

// Ping checks if Redis connection is alive
func (s *RedisStore) Ping() error {
	return s.client.Ping(s.ctx).Err()
}

// Close closes the Redis connection
func (s *RedisStore) Close() error {
	return s.client.Close()
}
