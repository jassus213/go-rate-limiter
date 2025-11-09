package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	ratelimiter "github.com/jassus213/go-rate-limiter"
	stdlogadapter "github.com/jassus213/go-rate-limiter/adapters/log"
	ginMiddleware "github.com/jassus213/go-rate-limiter/middleware/gin"
	"github.com/jassus213/go-rate-limiter/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stdLogger := stdlogadapter.New(log.Default())

	limiterStore := store.NewMemory(ctx, 10*time.Minute)
	limiter := ratelimiter.NewTokenBucket(limiterStore, 1.0, 5)

	config := []ratelimiter.Option{
		ratelimiter.WithLogger(stdLogger),
		ratelimiter.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error, result ratelimiter.Result) {
			stdLogger.Errorf(
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

	log.Println("Starting server on http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
