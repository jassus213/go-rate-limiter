package store

import (
	"context"
	"sync"
	"time"

	ratelimiter "github.com/jassus213/go-rate-limitter"
)

// fixedWindowEntry stores the counter value and its expiration time.
type fixedWindowEntry struct {
	count     int64
	expiresAt time.Time
}

// tokenBucketEntry stores the state for a token bucket.
type tokenBucketEntry struct {
	tokens      float64
	lastUpdated time.Time
}

// MemoryStore implements the ratelimiter.Store interface by storing data in memory.
// Its lifecycle is managed by the context provided on creation.
type MemoryStore struct {
	mu                 sync.Mutex
	fixedWindowEntries map[string]fixedWindowEntry
	tokenBucketEntries map[string]tokenBucketEntry
}

// NewMemory creates a new instance of MemoryStore.
// It requires a parent context to manage the lifecycle of its background cleanup goroutine.
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

// Increment atomically increments the counter for a given key.
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

// TakeToken implements the token bucket logic atomically for the in-memory store.
// It returns true if a token was taken, the number of remaining tokens, and an error.
func (s *MemoryStore) TakeToken(ctx context.Context, key string, rate float64, burst int64) (bool, float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, found := s.tokenBucketEntries[key]
	now := time.Now()

	if !found {
		// First request, start with a full bucket minus one token.
		remaining := float64(burst) - 1
		entry = tokenBucketEntry{
			tokens:      remaining,
			lastUpdated: now,
		}
		s.tokenBucketEntries[key] = entry
		return true, remaining, nil
	}

	// Refill tokens based on elapsed time.
	elapsed := now.Sub(entry.lastUpdated).Seconds()
	if elapsed > 0 {
		newTokens := elapsed * rate
		entry.tokens += newTokens
	}

	if entry.tokens > float64(burst) {
		entry.tokens = float64(burst)
	}

	// Check if a token is available.
	if entry.tokens >= 1 {
		entry.tokens--
		entry.lastUpdated = now
		s.tokenBucketEntries[key] = entry
		return true, entry.tokens, nil
	}

	// No token available.
	entry.lastUpdated = now
	s.tokenBucketEntries[key] = entry
	return false, entry.tokens, nil
}

// runCleanup periodically removes expired entries for both algorithms.
func (s *MemoryStore) runCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// A stale entry is one that hasn't been touched for a long time.
	// We use a multiple of the cleanup interval as a threshold.
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
