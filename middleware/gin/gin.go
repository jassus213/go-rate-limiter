// Package gin provides a Gin middleware adapter for
// github.com/jassus213/go-rate-limiter.
//
// This package allows you to easily integrate rate limiting
// into your Gin HTTP server using any Limiter implementation
// (e.g., fixed window, token bucket) and custom configurations.
//
// Example usage:
//
//	import (
//	    "time"
//	    "github.com/gin-gonic/gin"
//	    ratelimiter "github.com/jassus213/go-rate-limiter"
//	    "github.com/jassus213/go-rate-limiter/middleware/gin"
//	)
//
//	func main() {
//	    // Create a rate limiter instance (fixed window example)
//	    store := ratelimiter.NewMemoryStore()
//	    limiter := ratelimiter.NewFixedWindow(store, 100, time.Minute)
//
//	    router := gin.Default()
//
//	    // Apply the middleware globally
//	    router.Use(gin.RateLimiter(limiter))
//
//	    router.GET("/ping", func(c *gin.Context) {
//	        c.String(200, "pong")
//	    })
//
//	    router.Run(":8080")
//	}
package gin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	ratelimiter "github.com/jassus213/go-rate-limiter"
)

// RateLimiter creates a Gin middleware handler that enforces rate limiting.
//
// It uses the provided Limiter instance (the core rate-limiting logic) to check
// if a request should be allowed or denied. Users can customize the behavior
// by passing functional options, such as WithKeyFunc, WithErrorHandler, or WithLogger.
//
// Headers set by the middleware:
//
//   - X-RateLimit-Limit: the maximum number of requests allowed
//   - X-RateLimit-Remaining: the number of requests remaining in the current window
//   - X-RateLimit-Reset: Unix timestamp when the limit will reset
//
// Logging: the middleware logs debug and error information using the provided Logger
// (or the default noop logger if none is provided).
//
// Example usage:
//
//	router := gin.Default()
//	router.Use(gin.RateLimiter(limiter))
func RateLimiter(limiter ratelimiter.Limiter, options ...ratelimiter.Option) gin.HandlerFunc {
	cfg := ratelimiter.NewConfig(options...)

	return func(c *gin.Context) {
		key, err := cfg.KeyFunc(c.Request)
		if err != nil {
			cfg.Logger.Errorf("[RateLimiter] Failed to extract key: %v", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		result, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			cfg.Logger.Errorf("[RateLimiter] Limiter failed for key '%s': %v", key, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.Header("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))

		resetTimestamp := time.Now().Add(result.ResetAfter).Unix()
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTimestamp, 10))

		if !result.Allowed {
			cfg.Logger.Debugf(
				"[RateLimiter]Request denied for key '%s'. Remaining: %d, Limit: %d",
				key, result.Remaining, result.Limit,
			)
			cfg.ErrorHandler(c.Writer, c.Request, ratelimiter.ErrorExceeded, result)
			c.Abort()
			return
		}

		cfg.Logger.Debugf(
			"[RateLimiter]Request allowed for key '%s'. Remaining: %d, Limit: %d",
			key, result.Remaining, result.Limit,
		)

		c.Next()
	}
}
