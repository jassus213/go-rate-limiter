package stdlogadapter

import (
	"log"
)

// StdLogger implements ratelimiter.Logger using Go standard library log
type StdLogger struct {
	logger *log.Logger
}

// New creates a new StdLogger. If nil is passed, uses the default logger.
func New(l *log.Logger) *StdLogger {
	if l == nil {
		l = log.Default()
	}
	return &StdLogger{
		logger: l,
	}
}

// Debugf logs a debug-level message (same as Printf in std log)
func (s *StdLogger) Debugf(format string, args ...interface{}) {
	s.logger.Printf("[DEBUG] "+format, args...)
}

// Errorf logs an error-level message
func (s *StdLogger) Errorf(format string, args ...interface{}) {
	s.logger.Printf("[ERROR] "+format, args...)
}
