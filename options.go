package ratelimiter

import (
	"errors"
	"math"
	"net/http"
	"strconv"
)

// Logger is a simple interface for logging.
// Users can provide their own logger that implements this interface.
type Logger interface {
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// noopLogger is a default logger that does nothing.
// It is used when no logger is provided by the user to avoid nil panics.
type noopLogger struct{}

func (l *noopLogger) Debugf(format string, args ...interface{}) {}
func (l *noopLogger) Errorf(format string, args ...interface{}) {}

// ErrorExceeded is a sentinel error returned when the rate limit is surpassed.
// It can be used by custom error handlers to check for this specific condition.
var ErrorExceeded = errors.New("rate limit exceeded")

// KeyFunc is a function type used to extract a unique client identifier from an
// incoming HTTP request. The returned string is used as the key for the rate limiter.
// Common implementations use the client's IP address or an API key from a header.
type KeyFunc func(r *http.Request) (string, error)

// ErrorHandler is a function type that defines how to respond to a client when
// a rate limit is exceeded. This gives the user full control over the status code,
// headers, and body of the error response.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error, result Result)

// Config holds all configurable parameters for the middleware.
// It is an internal struct that users interact with via functional options.
type Config struct {
	KeyFunc      KeyFunc
	ErrorHandler ErrorHandler
	Logger       Logger
}

// Option is a function type that applies a configuration setting to a Config struct.
// It's the core of the Functional Options Pattern.
type Option func(*Config)

// NewConfig creates a Config instance with default settings and then applies
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

// WithKeyFunc returns an Option that sets a custom function for client identification.
// This allows users to rate-limit based on criteria like API keys, user IDs, etc.
func WithKeyFunc(f KeyFunc) Option {
	return func(c *Config) {
		if f != nil {
			c.KeyFunc = f
		}
	}
}

// WithErrorHandler returns an Option that sets a custom handler for rate limit errors.
// This is useful for sending structured JSON error responses or logging detailed information.
func WithErrorHandler(f ErrorHandler) Option {
	return func(c *Config) {
		if f != nil {
			c.ErrorHandler = f
		}
	}
}

// WithLogger returns an Option that sets a custom logger.
func WithLogger(l Logger) Option {
	return func(c *Config) {
		if l != nil {
			c.Logger = l
		}
	}
}
