// Package ratelimiter provides middleware and utilities to enforce
// rate limiting in Go applications.
//
// It supports different algorithms such as token bucket and fixed window,
// pluggable storage backends (memory, Redis), and customizable logging.
//
// Users can configure the limiter via functional options, supplying
// custom key extraction, error handling, and logging.
package ratelimiter

import (
	"errors"
	"math"
	"net/http"
	"strconv"
)

// Logger is the interface used for logging inside the rate limiter.
//
// Implement this interface to provide your own logging backend. For example,
// you can wrap logrus, zap, or the standard log package.
//
// Example:
//
//	type MyLogger struct{}
//	func (l *MyLogger) Debugf(format string, args ...interface{}) { ... }
//	func (l *MyLogger) Errorf(format string, args ...interface{}) { ... }
type Logger interface {
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// ErrorExceeded is returned when a client exceeds the rate limit.
//
// Users can use errors.Is(err, ratelimiter.ErrorExceeded) to detect
// this specific condition.
var ErrorExceeded = errors.New("rate limit exceeded")

// KeyFunc defines a function type that extracts a unique identifier
// from an HTTP request.
//
// The identifier is used to track individual clients for rate limiting.
//
// Example: use the client's IP address or an API key header.
type KeyFunc func(r *http.Request) (string, error)

// ErrorHandler defines a function type that handles a client request
// after a rate limit is exceeded.
//
// This allows custom responses, e.g., JSON, headers, or logging.
//
// Example:
//
//	func myHandler(w http.ResponseWriter, r *http.Request, err error, result Result) {
//	    w.Header().Set("Retry-After", strconv.Itoa(int(result.ResetAfter.Seconds())))
//	    w.WriteHeader(http.StatusTooManyRequests)
//	}
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error, result Result)

// Config holds all configurable options for the rate limiter middleware.
//
// Users typically create a Config via NewConfig and provide functional options.
type Config struct {
	KeyFunc      KeyFunc
	ErrorHandler ErrorHandler
	Logger       Logger
}

// Option defines a functional option type for configuring the rate limiter.
//
// Example:
//
//	cfg := NewConfig(
//	    WithLogger(myLogger),
//	    WithKeyFunc(myKeyFunc),
//	)
type Option func(*Config)

// NewConfig creates a Config with default settings, then applies
// any provided functional options.
func NewConfig(opts ...Option) *Config {
	cfg := &Config{
		KeyFunc: func(r *http.Request) (string, error) {
			return r.RemoteAddr, nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error, result Result) {
			retryAfter := int(math.Ceil(result.ResetAfter.Seconds()))
			if retryAfter <= 0 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		},
		Logger: &noopLogger{},
	}

	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// WithKeyFunc returns an Option to set a custom KeyFunc.
//
// Example:
//
//	cfg := NewConfig(WithKeyFunc(myKeyFunc))
func WithKeyFunc(f KeyFunc) Option {
	return func(c *Config) {
		if f != nil {
			c.KeyFunc = f
		}
	}
}

// WithErrorHandler returns an Option to set a custom ErrorHandler.
//
// Example:
//
//	cfg := NewConfig(WithErrorHandler(myHandler))
func WithErrorHandler(f ErrorHandler) Option {
	return func(c *Config) {
		if f != nil {
			c.ErrorHandler = f
		}
	}
}

// WithLogger returns an Option to set a custom Logger.
//
// Example:
//
//	cfg := NewConfig(WithLogger(myLogger))
func WithLogger(l Logger) Option {
	return func(c *Config) {
		if l != nil {
			c.Logger = l
		}
	}
}

// noopLogger is a private default logger that does nothing.
type noopLogger struct{}

func (l *noopLogger) Debugf(format string, args ...interface{}) {}
func (l *noopLogger) Errorf(format string, args ...interface{}) {}
