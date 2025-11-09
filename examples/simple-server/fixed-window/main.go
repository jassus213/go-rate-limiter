package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	ginMiddleware "github.com/jassus213/go-rate-limiter/middleware/gin"
	"github.com/jassus213/go-rate-limiter/ratelimiter"
	"github.com/jassus213/go-rate-limiter/store"

	"github.com/gin-gonic/gin"
)

func main() {
	// --- Step 1: Create a cancellable context for the application lifecycle ---
	// This is the standard Go pattern for graceful shutdown.
	// `signal.NotifyContext` creates a context that is canceled when the application
	// receives an interrupt signal (like Ctrl+C).
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Step 2: Initialize the store, passing the application context ---
	// The memory store's background cleanup goroutine will now automatically stop
	// when the application context is canceled. No more manual .Close() calls!
	limiterStore := store.NewMemory(ctx, 10*time.Minute)

	// --- Step 3: Create a limiter instance ---
	limiter := ratelimiter.NewFixedWindow(limiterStore, 5, time.Minute)

	// --- Step 4: Set up and run the Gin server ---
	router := gin.Default()
	router.Use(ginMiddleware.RateLimiter(limiter))
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	log.Println("Starting server on http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
