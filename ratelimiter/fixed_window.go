// Package ratelimiter provides flexible rate limiting algorithms and interfaces.
//
// It includes support for Fixed Window, Token Bucket, and pluggable storage backends.
// Users can integrate it with standard net/http, Gin, Echo, Chi, or custom frameworks.
package ratelimiter

import (
	"context"
	"math"
	"time"
)

// FixedWindowLimiter implements the "Fixed Window" rate-limiting algorithm.
//
// The Fixed Window algorithm limits the number of requests (Limit) within a specific
// time frame (Window). It is simple and memory-efficient but may allow bursts of
// traffic at the edges of windows.
//
// Example usage:
//
//	store := store.NewMemory(ctx, time.Minute)
//	limiter := ratelimiter.NewFixedWindow(store, 100, time.Minute)
//	result, err := limiter.Allow(ctx, "user:123")
//	if result.Allowed {
//	    // process request
//	} else {
//	    // reject request
//	}
type FixedWindowLimiter struct {
	store  Store
	limit  int64
	window time.Duration
}

// NewFixedWindow creates a new FixedWindowLimiter instance.
//
// Parameters:
//   - store: a ratelimiter.Store implementation to persist request counts
//   - limit: maximum number of requests allowed per window
//   - window: duration of each fixed window
//
// Returns a Limiter interface that can be used with any middleware or custom logic.
func NewFixedWindow(store Store, limit int64, window time.Duration) Limiter {
	return &FixedWindowLimiter{
		store:  store,
		limit:  limit,
		window: window,
	}
}

// Allow checks whether a request with the given key is allowed under the fixed window.
//
// It returns a Result struct containing details that can be used for HTTP headers:
//
//   - Allowed: true if the request is within the limit
//   - Limit: maximum number of requests in the window
//   - Remaining: requests left in the current window
//   - ResetAfter: duration until the window resets
//
// Example:
//
//	result, err := limiter.Allow(ctx, "user:123")
//	if result.Allowed {
//	    // process request
//	} else {
//	    // reject request
//	}
func (l *FixedWindowLimiter) Allow(ctx context.Context, key string) (Result, error) {
	currentCount, err := l.store.Increment(ctx, key, l.window)
	if err != nil {
		return Result{Allowed: false}, err
	}

	allowed := currentCount <= l.limit
	remaining := int64(math.Max(0, float64(l.limit-currentCount)))

	now := time.Now()
	endOfWindow := now.Truncate(l.window).Add(l.window)
	resetAfter := time.Until(endOfWindow)

	result := Result{
		Allowed:    allowed,
		Limit:      l.limit,
		Remaining:  remaining,
		ResetAfter: resetAfter,
	}

	return result, nil
}
