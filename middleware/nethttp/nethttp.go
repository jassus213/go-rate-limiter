package nethttp

import (
	"net/http"
	"strconv"
	"time"

	ratelimiter "github.com/jassus213/go-rate-limitter"
)

// Middleware creates a new middleware handler for the standard `net/http` library.
//
// It wraps an existing `http.Handler` and checks incoming requests against the provided
// Limiter instance. On every request, it adds the standard `X-RateLimit-*` headers
// to the response. The behavior can be customized using functional options.
//
// Example:
//
//	limiter := ratelimiter.NewFixedWindow(store, 100, time.Minute)
//	mux := http.NewServeMux()
//	mux.HandleFunc("/", myHandler)
//
//	rateLimitMiddleware := nethttp.Middleware(limiter)
//	http.ListenAndServe(":8080", rateLimitMiddleware(mux))
func Middleware(limiter ratelimiter.Limiter, options ...ratelimiter.Option) func(http.Handler) http.Handler {
	cfg := ratelimiter.NewConfig(options...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, err := cfg.KeyFunc(r)
			if err != nil {
				cfg.Logger.Errorf("Failed to extract key: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			result, err := limiter.Allow(r.Context(), key)
			if err != nil {
				cfg.Logger.Errorf("Limiter failed for key '%s': %v", key, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))
			resetTimestamp := time.Now().Add(result.ResetAfter).Unix()
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTimestamp, 10))

			if !result.Allowed {
				cfg.Logger.Debugf(
					"Request denied for key '%s'. Remaining: %d, Limit: %d",
					key, result.Remaining, result.Limit,
				)
				cfg.ErrorHandler(w, r, ratelimiter.ErrorExceeded, result)
				return
			}

			cfg.Logger.Debugf(
				"Request allowed for key '%s'. Remaining: %d, Limit: %d",
				key, result.Remaining, result.Limit,
			)
			next.ServeHTTP(w, r)
		})
	}
}
