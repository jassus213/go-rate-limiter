package main

import (
	"context"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	ratelimiter "github.com/jassus213/go-rate-limiter"
	zerologadapter "github.com/jassus213/go-rate-limiter/adapters/zerolog"
	ginMiddleware "github.com/jassus213/go-rate-limiter/middleware/gin"
	"github.com/jassus213/go-rate-limiter/store"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// Адаптер Zerolog
	zeroLogger := zerologadapter.New(&log.Logger)

	limiterStore := store.NewMemory(ctx, 10*time.Minute)
	limiter := ratelimiter.NewTokenBucket(limiterStore, 1.0, 5)

	config := []ratelimiter.Option{
		ratelimiter.WithLogger(zeroLogger),
		ratelimiter.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error, result ratelimiter.Result) {
			zeroLogger.Errorf(
				"Rate limit exceeded for key: %s | Remaining: %d | Limit: %d",
				r.RemoteAddr, result.Remaining, result.Limit,
			)
			retryAfter := int(result.ResetAfter.Seconds())
			if retryAfter <= 0 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		}),
	}

	router := gin.Default()
	router.Use(ginMiddleware.RateLimiter(limiter, config...))
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	log.Info().Msg("Starting server on http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal().Err(err).Msg("Failed to run server")
	}
}
