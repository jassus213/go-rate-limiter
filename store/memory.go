// Package store provides storage backends for github.com/jassus213/go-rate-limiter.
//
// Currently supported backends:
//   - MemoryStore: in-memory store for single-instance applications
//   - RedisStore: Redis-based store for distributed applications (not shown here)
//
// Stores implement the ratelimiter.Store interface, providing atomic operations
// for rate limiting algorithms such as fixed window and token bucket.
//
// Example usage:
//
//	ctx := context.Background()
//	store := store.NewMemory(ctx, time.Minute) // cleanup interval = 1 minute
//	limiter := ratelimiter.NewFixedWindow(store, 100, time.Minute)
package store

import (
	"context"
	"sync"
	"time"

	"github.com/jassus213/go-rate-limiter/ratelimiter"
)

// fixedWindowEntry stores the counter and expiration time for a fixed window key.
type fixedWindowEntry struct {
	count     int64
	expiresAt time.Time
}

// tokenBucketEntry stores the state of a token bucket key.
type tokenBucketEntry struct {
	tokens      float64
	lastUpdated time.Time
}

// MemoryStore is an in-memory implementation of ratelimiter.Store.
//
// It supports both fixed window and token bucket algorithms, and optionally
// runs a background cleanup goroutine to remove stale entries.
//
// Note: MemoryStore is suitable for single-instance applications.
type MemoryStore struct {
	mu                 sync.Mutex
	fixedWindowEntries map[string]fixedWindowEntry
	tokenBucketEntries map[string]tokenBucketEntry
}

// NewMemory creates a new MemoryStore instance.
//
// ctx: a parent context used to manage the lifecycle of the background cleanup goroutine.
// cleanupInterval: interval at which expired entries are removed. Pass 0 to disable cleanup.
//
// Example:
//
//	ctx := context.Background()
//	store := store.NewMemory(ctx, time.Minute)
func NewMemory(ctx context.Context, cleanupInterval time.Duration) ratelimiter.Store {
	store := &MemoryStore{
		fixedWindowEntries: make(map[string]fixedWindowEntry),
		tokenBucketEntries: make(map[string]tokenBucketEntry),
	}

	if cleanupInterval > 0 {
		go store.runCleanup(ctx, cleanupInterval)
	}

	return store
}

// Increment atomically increases the counter for a given key in the fixed window.
//
// Returns the new counter value or an error.
//
// Example:
//
//	count, err := store.Increment(ctx, "user:123", time.Minute)
func (s *MemoryStore) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, found := s.fixedWindowEntries[key]
	if found && time.Now().After(e.expiresAt) {
		found = false
	}

	if !found {
		e = fixedWindowEntry{
			count:     1,
			expiresAt: time.Now().Add(window),
		}
	} else {
		e.count++
	}

	s.fixedWindowEntries[key] = e
	return e.count, nil
}

// TakeToken atomically consumes a token from the token bucket for the given key.
//
// Returns:
//   - allowed: true if a token was successfully taken
//   - remaining: number of tokens remaining in the bucket
//   - error: always nil for MemoryStore
//
// rate: refill rate per second
// burst: maximum number of tokens
//
// Example:
//
//	allowed, remaining, _ := store.TakeToken(ctx, "user:123", 1.0, 5)
func (s *MemoryStore) TakeToken(ctx context.Context, key string, rate float64, burst int64) (bool, float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, found := s.tokenBucketEntries[key]
	now := time.Now()

	if !found {
		remaining := float64(burst) - 1
		entry = tokenBucketEntry{
			tokens:      remaining,
			lastUpdated: now,
		}
		s.tokenBucketEntries[key] = entry
		return true, remaining, nil
	}

	elapsed := now.Sub(entry.lastUpdated).Seconds()
	if elapsed > 0 {
		entry.tokens += elapsed * rate
	}

	if entry.tokens > float64(burst) {
		entry.tokens = float64(burst)
	}

	if entry.tokens >= 1 {
		entry.tokens--
		entry.lastUpdated = now
		s.tokenBucketEntries[key] = entry
		return true, entry.tokens, nil
	}

	entry.lastUpdated = now
	s.tokenBucketEntries[key] = entry
	return false, entry.tokens, nil
}

// runCleanup periodically removes expired or stale entries for both fixed window and token bucket.
//
// Entries are considered stale if they haven't been updated for 10 times the cleanup interval.
func (s *MemoryStore) runCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	staleThreshold := interval * 10

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()

			for key, e := range s.fixedWindowEntries {
				if now.After(e.expiresAt) {
					delete(s.fixedWindowEntries, key)
				}
			}

			for key, e := range s.tokenBucketEntries {
				if now.Sub(e.lastUpdated) > staleThreshold {
					delete(s.tokenBucketEntries, key)
				}
			}
			s.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
