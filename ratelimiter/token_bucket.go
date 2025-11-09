// Package ratelimiter provides flexible rate-limiting algorithms and interfaces.
//
// It includes support for Fixed Window, Token Bucket, and pluggable storage backends.
// Users can integrate it with standard net/http, Gin, Echo, Chi, or custom frameworks.
package ratelimiter

import (
	"context"
	"math"
	"time"
)

// TokenBucketLimiter implements the "Token Bucket" rate-limiting algorithm.
//
// The Token Bucket algorithm allows for bursts of requests up to the 'burst' size,
// while maintaining a steady request rate defined by 'rate' tokens per second.
//
// Example usage:
//
//	store := store.NewMemory(ctx, time.Minute)
//	limiter := ratelimiter.NewTokenBucket(store, 1.0, 5) // 1 token/sec, burst of 5
//	result, err := limiter.Allow(ctx, "user:123")
//	if result.Allowed {
//	    // process request
//	} else {
//	    // reject request
//	}
type TokenBucketLimiter struct {
	store Store
	rate  float64 // Tokens generated per second
	burst int64   // Maximum number of tokens in the bucket
}

// NewTokenBucket creates a new TokenBucketLimiter instance.
//
// Parameters:
//   - store: a ratelimiter.Store implementation for persisting token state
//   - rate: number of tokens added to the bucket per second
//   - burst: maximum number of tokens in the bucket (burst capacity)
//
// Returns a Limiter interface that can be used with any middleware or custom logic.
//
// Example:
//
//	store := store.NewMemory(ctx, time.Minute)
//	limiter := ratelimiter.NewTokenBucket(store, 1.0, 5)
func NewTokenBucket(store Store, rate float64, burst int64) Limiter {
	return &TokenBucketLimiter{
		store: store,
		rate:  rate,
		burst: burst,
	}
}

// Allow checks whether a request is allowed under the token bucket algorithm.
//
// It returns a Result struct containing details that can be used for HTTP headers:
//
//   - Allowed: true if a token was successfully consumed
//   - Limit: maximum number of tokens (burst)
//   - Remaining: number of tokens remaining in the bucket
//   - ResetAfter: estimated duration until the next token is available if request is denied
//
// Example:
//
//	result, err := limiter.Allow(ctx, "user:123")
//	if result.Allowed {
//	    // process request
//	} else {
//	    // reject request
//	}
func (l *TokenBucketLimiter) Allow(ctx context.Context, key string) (Result, error) {
	allowed, remaining, err := l.store.TakeToken(ctx, key, l.rate, l.burst)
	if err != nil {
		return Result{Allowed: false}, err
	}

	remainingInt := int64(math.Floor(remaining))
	if remainingInt < 0 {
		remainingInt = 0
	}

	var resetAfter time.Duration
	if allowed {
		resetAfter = 0
	} else {
		secondsToWait := (1.0 - remaining) / l.rate
		resetAfter = time.Duration(secondsToWait * float64(time.Second))
	}

	result := Result{
		Allowed:    allowed,
		Limit:      l.burst,
		Remaining:  remainingInt,
		ResetAfter: resetAfter,
	}

	return result, nil
}
