package zerologadapter

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ZerologLogger implements ratelimiter.Logger using zerolog
type ZerologLogger struct {
	logger zerolog.Logger
}

// New creates a new ZerologLogger. If nil is passed, uses zerolog's global logger.
func New(l *zerolog.Logger) *ZerologLogger {
	if l == nil {
		l = &log.Logger
	}
	return &ZerologLogger{
		logger: *l,
	}
}

// Debugf logs a debug-level message
func (z *ZerologLogger) Debugf(format string, args ...interface{}) {
	z.logger.Debug().Msgf(format, args...)
}

// Errorf logs an error-level message
func (z *ZerologLogger) Errorf(format string, args ...interface{}) {
	z.logger.Error().Msgf(format, args...)
}
