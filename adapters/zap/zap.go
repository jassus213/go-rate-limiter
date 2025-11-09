package zapadapter

import (
	"go.uber.org/zap"
)

// ZapLogger is an adapter that implements the ratelimiter.Logger interface
// using a zap.SugaredLogger internally.
type ZapLogger struct {
	logger *zap.SugaredLogger
}

// New creates a new ZapLogger from a zap.Logger.
//
// If a nil logger is provided, it uses zap.NewNop() internally, which
// is a no-op logger that discards all messages.
//
// Example:
//
//	zapLogger := zapadapter.New(logger)
func New(l *zap.Logger) *ZapLogger {
	if l == nil {
		l = zap.NewNop()
	}
	return &ZapLogger{logger: l.Sugar()}
}

// Debugf logs a debug-level message with formatting, compatible with
// ratelimiter.Logger interface.
//
// Example:
//
//	zapLogger.Debugf("Rate limit key: %s", key)
func (z *ZapLogger) Debugf(format string, args ...interface{}) {
	z.logger.Debugf(format, args...)
}

// Errorf logs an error-level message with formatting, compatible with
// ratelimiter.Logger interface.
//
// Example:
//
//	zapLogger.Errorf("Rate limit exceeded for key: %s", key)
func (z *ZapLogger) Errorf(format string, args ...interface{}) {
	z.logger.Errorf(format, args...)
}
