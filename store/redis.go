package store

import (
	"context"
	"strconv"
	"time"

	ratelimiter "github.com/jassus213/go-rate-limitter"
	"github.com/redis/go-redis/v9"
)

// RedisStore implements the ratelimiter.Store interface using Redis as the backend.
// It is suitable for distributed systems where multiple application instances need to share
// a common rate-limiting state. It uses Lua scripts to ensure atomicity.
type RedisStore struct {
	client          *redis.Client
	incrementScript *redis.Script
	takeTokenScript *redis.Script
}

// NewRedis creates a new instance of RedisStore.
// It pre-compiles Lua scripts for both algorithms for maximum performance.
func NewRedis(client *redis.Client) ratelimiter.Store {
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
func (s *RedisStore) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	res, err := s.incrementScript.Run(ctx, s.client, []string{key}, window.Milliseconds()).Result()
	if err != nil {
		return 0, err
	}
	return res.(int64), nil
}

// TakeToken executes the token bucket Lua script and parses its multi-value response.
// It returns true if the request is allowed, the number of remaining tokens, and an error.
func (s *RedisStore) TakeToken(ctx context.Context, key string, rate float64, burst int64) (bool, float64, error) {
	now := float64(time.Now().UnixNano()) / 1e9

	res, err := s.takeTokenScript.Run(ctx, s.client, []string{key}, rate, burst, now).Result()
	if err != nil {
		return false, 0, err
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) < 2 {
		return false, 0, ratelimiter.ErrorExceeded
	}

	allowed := arr[0].(int64) == 1

	remainingTokensStr, _ := arr[1].(string)
	remainingTokens, _ := strconv.ParseFloat(remainingTokensStr, 64)

	return allowed, remainingTokens, nil
}
