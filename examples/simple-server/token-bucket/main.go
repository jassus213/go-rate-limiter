package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	ginMiddleware "github.com/jassus213/go-rate-limiter/middleware/gin"
	"github.com/jassus213/go-rate-limiter/ratelimiter"
	"github.com/jassus213/go-rate-limiter/store"
	"github.com/redis/go-redis/v9"
)

func main() {
	// --- Step 1: Set up a context for graceful shutdown ---
	// This context will be canceled when the application receives an interrupt signal (Ctrl+C).
	// It's used to manage the lifecycle of background processes.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Step 2: Initialize Redis client and the store ---
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // Default Redis address
	})

	// Check the connection to Redis
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis.")

	// Create a new Redis-backed store.
	limiterStore := store.NewRedis(redisClient)

	// --- Step 3: Configure and create the Token Bucket limiter ---
	// Configuration:
	// - rate: 5.0 tokens are added to the bucket per second.
	// - burst: The bucket can hold a maximum of 20 tokens.
	// This allows for an initial burst of 20 requests, after which the rate
	// is limited to a sustained 5 requests per second.
	const rate = 5.0
	const burst = 20
	limiter := ratelimiter.NewTokenBucket(limiterStore, rate, burst)
	log.Printf("Token Bucket Limiter configured: rate=%.2f/s, burst=%d", rate, burst)

	// --- Step 4: Set up Gin server and apply the middleware ---
	router := gin.Default()

	// Apply the rate limiter middleware to all routes.
	router.Use(ginMiddleware.RateLimiter(limiter))

	// --- Step 5: Define a protected route ---
	router.GET("/api/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Here is your data!",
		})
	})

	// --- Step 6: Start the server ---
	log.Println("Server is running on http://localhost:8080")
	log.Println("Protected endpoint is available at GET /api/data")
	log.Println("Press Ctrl+C to shut down.")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

/*
--- HOW TO TEST THIS EXAMPLE ---

1. Make sure Redis is running on localhost:6379.
2. Run this program: `go run main.go`
3. Open a new terminal and use `curl` to send requests.

4. Test the burst capacity:
   Run this command to send 25 requests in quick succession:
   for i in {1..25}; do curl -w " | Status: %{http_code}\n" -s http://localhost:8080/api/data; done

   Expected output:
   - The first 20 requests will succeed with "Status: 200".
   - The next 5 requests will fail with "Status: 429" (Too Many Requests).

5. Test the sustained rate:
   - Wait for a second. The bucket will refill with 5 tokens.
   - Send another burst of requests. You will see that about 5 requests succeed,
     and the rest are blocked.

*/
