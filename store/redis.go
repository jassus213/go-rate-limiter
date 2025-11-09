// Package store provides storage backends for github.com/jassus213/go-rate-limiter.
//
// Currently supported backends:
//   - MemoryStore: in-memory store for single-instance applications
//   - RedisStore: Redis-based store for distributed applications
//
// Stores implement the ratelimiter.Store interface, providing atomic operations
// for rate limiting algorithms such as fixed window and token bucket.
package store

import (
	"context"
	"strconv"
	"time"

	ratelimiter2 "github.com/jassus213/go-rate-limiter/ratelimiter"
	"github.com/redis/go-redis/v9"
)

// RedisStore implements the ratelimiter.Store interface using Redis as the backend.
//
// It is suitable for distributed systems where multiple application instances
// need to share a common rate-limiting state. RedisStore uses Lua scripts to
// ensure atomicity of incrementing counters and token bucket operations.
//
// Example usage:
//
//	client := redis.NewClient(&redis.Options{
//	    Addr: "localhost:6379",
//	})
//	store := store.NewRedis(client)
//	limiter := ratelimiter.NewFixedWindow(store, 100, time.Minute)
type RedisStore struct {
	client          *redis.Client
	incrementScript *redis.Script
	takeTokenScript *redis.Script
}

// NewRedis creates a new RedisStore instance.
//
// It pre-compiles Lua scripts for both fixed window and token bucket
// algorithms to maximize performance.
//
// Example:
//
//	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	store := store.NewRedis(client)
func NewRedis(client *redis.Client) ratelimiter2.Store {
	const incrementLua = `
		local current = redis.call("INCR", KEYS[1])
		if tonumber(current) == 1 then
			redis.call("PEXPIRE", KEYS[1], ARGV[1])
		end
		return current
	`

	const takeTokenLua = `
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local burst = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local cost = 1

		local entry = redis.call("HGETALL", key)
		local tokens
		local last_updated

		if #entry == 0 then
			tokens = burst
			last_updated = now
		else
			tokens = tonumber(entry[2])
			last_updated = tonumber(entry[4])
		end
		
		local elapsed = now - last_updated
		if elapsed > 0 then
			local new_tokens = elapsed * rate
			tokens = tokens + new_tokens
		end
		
		if tokens > burst then
			tokens = burst
		end
		
		local allowed = 0
		if tokens >= cost then
			tokens = tokens - cost
			allowed = 1
		end
		
		redis.call("HSET", key, "tokens", tokens, "last_updated", now)
		local ttl = math.ceil((burst / rate) * 2)
		if ttl < 10 then
			ttl = 10
		end
		redis.call("EXPIRE", key, ttl)
		
		return {allowed, tostring(tokens)}
	`

	return &RedisStore{
		client:          client,
		incrementScript: redis.NewScript(incrementLua),
		takeTokenScript: redis.NewScript(takeTokenLua),
	}
}

// Increment executes the pre-compiled Lua script for the Fixed Window algorithm.
//
// Returns the new counter value for the given key or an error if the Redis call fails.
//
// Example:
//
//	count, err := store.Increment(ctx, "user:123", time.Minute)
func (s *RedisStore) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	res, err := s.incrementScript.Run(ctx, s.client, []string{key}, window.Milliseconds()).Result()
	if err != nil {
		return 0, err
	}
	return res.(int64), nil
}

// TakeToken executes the token bucket Lua script for the given key.
//
// Returns:
//   - allowed: true if a token was successfully taken
//   - remaining: number of tokens remaining in the bucket
//   - error: any error occurred during the Redis operation
//
// rate: refill rate per second
// burst: maximum number of tokens
//
// Example:
//
//	allowed, remaining, err := store.TakeToken(ctx, "user:123", 1.0, 5)
func (s *RedisStore) TakeToken(ctx context.Context, key string, rate float64, burst int64) (bool, float64, error) {
	now := float64(time.Now().UnixNano()) / 1e9

	res, err := s.takeTokenScript.Run(ctx, s.client, []string{key}, rate, burst, now).Result()
	if err != nil {
		return false, 0, err
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) < 2 {
		return false, 0, ratelimiter2.ErrorExceeded
	}

	allowed := arr[0].(int64) == 1

	remainingTokensStr, _ := arr[1].(string)
	remainingTokens, _ := strconv.ParseFloat(remainingTokensStr, 64)

	return allowed, remainingTokens, nil
}
