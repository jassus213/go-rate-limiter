package ratelimiter

import (
	"context"
	"time"
)

// Result contains the outcome of a rate limit check.
// It provides the necessary data to populate standard rate-limiting HTTP headers.
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
// It is the primary interface that middleware and users will interact with.
type Limiter interface {
	// Allow checks if a request is permitted for a given key.
	// It returns true if the request is allowed, and false otherwise.
	Allow(ctx context.Context, key string) (Result, error)
}

// Store defines the interface for storing rate-limiting data.
// This abstraction allows for interchangeable backend implementations (e.g., in-memory, Redis).
type Store interface {
	// Increment atomically increments the counter for a given key and returns the new value.
	// If the key does not exist, it should be created with a value of 1 and an expiration equal to the window.
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)

	// TakeToken is the primitive for token-based algorithms like Token Bucket.
	// It should atomically refill tokens based on the rate and burst capacity,
	// then consume one token if available. It returns true if a token was taken.
	TakeToken(ctx context.Context, key string, rate float64, burst int64) (bool, float64, error)
}
