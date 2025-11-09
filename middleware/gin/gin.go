package gin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	ratelimiter "github.com/jassus213/go-rate-limitter"
)

// RateLimiter creates a new Gin middleware handler.
//
// It uses the provided Limiter instance (the core rate-limiting logic) to check
// if a request should be allowed or denied. The behavior of the middleware can be
// customized by passing functional options, such as changing how a client is
// identified (WithKeyFunc) or how rate limit errors are handled (WithErrorHandler).
//
// Example:
//
//	limiter := ratelimiter.NewFixedWindow(store, 100, time.Minute)
//	router := gin.Default()
//	// Apply middleware globally
//	router.Use(gin.RateLimiter(limiter))
func RateLimiter(limiter ratelimiter.Limiter, options ...ratelimiter.Option) gin.HandlerFunc {
	cfg := ratelimiter.NewConfig(options...)

	return func(c *gin.Context) {
		key, err := cfg.KeyFunc(c.Request)
		if err != nil {
			cfg.Logger.Errorf("Failed to extract key: %v", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		result, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			cfg.Logger.Errorf("Limiter failed for key '%s': %v", key, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.Header("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))

		resetTimestamp := time.Now().Add(result.ResetAfter).Unix()
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTimestamp, 10))

		if !result.Allowed {
			cfg.Logger.Debugf(
				"Request denied for key '%s'. Remaining: %d, Limit: %d",
				key, result.Remaining, result.Limit,
			)
			cfg.ErrorHandler(c.Writer, c.Request, ratelimiter.ErrorExceeded, result)
			c.Abort()
			return
		}

		cfg.Logger.Debugf(
			"Request allowed for key '%s'. Remaining: %d, Limit: %d",
			key, result.Remaining, result.Limit,
		)

		c.Next()
	}
}
