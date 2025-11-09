package ratelimiter

import (
	"context"
	"math"
	"time"
)

// TokenBucketLimiter implements the "Token Bucket" algorithm.
// This algorithm allows for bursts of requests up to the 'burst' size,
// while sustaining a steady rate of requests defined by 'rate'.
type TokenBucketLimiter struct {
	store Store
	rate  float64 // Tokens generated per second
	burst int64   // Maximum number of tokens in the bucket (the "burst" capacity)
}

// NewTokenBucket creates a new limiter based on the Token Bucket algorithm.
// - store: The storage backend.
// - rate: The number of tokens to add to the bucket per second.
// - burst: The maximum capacity of the bucket.
func NewTokenBucket(store Store, rate float64, burst int64) Limiter {
	return &TokenBucketLimiter{
		store: store,
		rate:  rate,
		burst: burst,
	}
}

// Allow checks if a token can be taken from the bucket for the given key.
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
