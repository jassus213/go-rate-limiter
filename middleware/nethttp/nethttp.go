// Package nethttp provides middleware for the standard net/http library
// that enforces rate limiting using github.com/jassus213/go-rate-limiter.
//
// This package allows you to wrap any http.Handler and automatically apply
// rate limiting based on a Limiter instance (fixed window, token bucket, etc.).
// The middleware sets standard `X-RateLimit-*` headers and supports
// custom logging and error handling via functional options.
//
// Example usage:
//
//	import (
//	    "net/http"
//	    "time"
//	    ratelimiter "github.com/jassus213/go-rate-limiter"
//	    "github.com/jassus213/go-rate-limiter/middleware/nethttp"
//	)
//
//	func main() {
//	    store := ratelimiter.NewMemoryStore()
//	    limiter := ratelimiter.NewFixedWindow(store, 100, time.Minute)
//
//	    mux := http.NewServeMux()
//	    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//	        w.Write([]byte("Hello, world!"))
//	    })
//
//	    // Wrap the mux with rate limiting middleware
//	    rateLimitMiddleware := nethttp.Middleware(limiter)
//
//	    http.ListenAndServe(":8080", rateLimitMiddleware(mux))
//	}
package nethttp

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jassus213/go-rate-limiter/ratelimiter"
)

// Middleware returns a middleware handler for the standard net/http library.
//
// It wraps an existing http.Handler and checks incoming requests against
// the provided Limiter instance. The middleware adds standard headers:
//
//   - X-RateLimit-Limit: the maximum number of requests allowed
//   - X-RateLimit-Remaining: the number of requests remaining in the current window
//   - X-RateLimit-Reset: Unix timestamp when the limit will reset
//
// Behavior can be customized using functional options such as WithKeyFunc,
// WithErrorHandler, or WithLogger.
func Middleware(limiter ratelimiter.Limiter, options ...ratelimiter.Option) func(http.Handler) http.Handler {
	cfg := ratelimiter.NewConfig(options...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, err := cfg.KeyFunc(r)
			if err != nil {
				cfg.Logger.Errorf("[RateLimiter] Failed to extract key: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			result, err := limiter.Allow(r.Context(), key)
			if err != nil {
				cfg.Logger.Errorf("[RateLimiter]Limiter failed for key '%s': %v", key, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))
			resetTimestamp := time.Now().Add(result.ResetAfter).Unix()
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTimestamp, 10))

			if !result.Allowed {
				cfg.Logger.Debugf(
					"[RateLimiter] Request denied for key '%s'. Remaining: %d, Limit: %d",
					key, result.Remaining, result.Limit,
				)
				cfg.ErrorHandler(w, r, ratelimiter.ErrorExceeded, result)
				return
			}

			cfg.Logger.Debugf(
				"[RateLimiter] Request allowed for key '%s'. Remaining: %d, Limit: %d",
				key, result.Remaining, result.Limit,
			)
			next.ServeHTTP(w, r)
		})
	}
}
