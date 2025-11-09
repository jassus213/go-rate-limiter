// Package ratelimiter provides flexible rate-limiting algorithms and interfaces.
//
// It includes support for Fixed Window, Token Bucket, and pluggable storage backends.
// Users can integrate it with standard net/http, Gin, Echo, Chi, or custom frameworks.
//
// The package defines three core abstractions:
//   - Limiter: the rate-limiting algorithm interface (e.g., FixedWindowLimiter, TokenBucketLimiter)
//   - Store: backend interface for storing rate-limiting state (e.g., MemoryStore, RedisStore)
//   - Result: struct containing the outcome of a rate limit check, useful for HTTP headers
package ratelimiter

import (
	"context"
	"time"
)

// Result contains the outcome of a rate limit check.
//
// It provides the necessary data to populate standard rate-limiting HTTP headers
// such as `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset`.
type Result struct {
	// Allowed indicates whether the request is permitted.
	Allowed bool
	// Limit is the total number of requests allowed in the current window.
	Limit int64
	// Remaining is the number of requests left in the current window.
	Remaining int64
	// ResetAfter is the duration after which the rate limit will be reset.
	ResetAfter time.Duration
}

// Limiter defines the interface for rate-limiting algorithms.
//
// Middleware and users interact with Limiter to enforce limits on requests.
// Implementations can include Fixed Window, Token Bucket, or custom strategies.
type Limiter interface {
	// Allow checks if a request is permitted for a given key.
	//
	// Parameters:
	//   - ctx: context for cancellation and timeouts
	//   - key: unique identifier for the client (e.g., IP address, API key)
	//
	// Returns:
	//   - Result: contains the outcome and headers-related info
	//   - error: any error occurred while checking the limit
	Allow(ctx context.Context, key string) (Result, error)
}

// Store defines the interface for storing rate-limiting data.
//
// This abstraction allows interchangeable backends such as in-memory stores
// or Redis for distributed rate limiting.
type Store interface {
	// Increment atomically increments the counter for a given key and returns the new value.
	//
	// If the key does not exist, it should be created with a value of 1 and an expiration equal to the window.
	//
	// Parameters:
	//   - ctx: context for cancellation
	//   - key: unique client identifier
	//   - window: duration of the fixed window
	//
	// Returns:
	//   - new counter value
	//   - error if the operation fails
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)

	// TakeToken atomically refills and consumes tokens for token-based algorithms like Token Bucket.
	//
	// Parameters:
	//   - ctx: context for cancellation
	//   - key: unique client identifier
	//   - rate: refill rate per second
	//   - burst: maximum tokens in the bucket
	//
	// Returns:
	//   - allowed: true if a token was successfully taken
	//   - remaining: number of tokens left
	//   - error if the operation fails
	TakeToken(ctx context.Context, key string, rate float64, burst int64) (bool, float64, error)
}
