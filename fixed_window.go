package ratelimiter

import (
	"context"
	"math"
	"time"
)

// FixedWindowLimiter implements the "Fixed Window" rate-limiting algorithm.
// This algorithm limits the number of requests (Limit) within a specific time frame (Window).
// It's simple and memory-efficient but can allow bursts of traffic at the edges of a window.
type FixedWindowLimiter struct {
	store  Store
	limit  int64
	window time.Duration
}

// NewFixedWindow creates a new limiter based on the Fixed Window algorithm.
// It requires a Store to persist the counts, a limit for the number of requests,
// and a window duration. It returns a Limiter interface for flexible usage.
func NewFixedWindow(store Store, limit int64, window time.Duration) Limiter {
	return &FixedWindowLimiter{
		store:  store,
		limit:  limit,
		window: window,
	}
}

// Allow checks if the request count for the given key is within the defined limit.
// It now returns a rich Result struct with details for HTTP headers.
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
