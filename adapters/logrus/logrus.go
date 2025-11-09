package logrusadapter

import (
	"github.com/sirupsen/logrus"
)

// LogrusLogger implements ratelimiter.Logger using logrus
type LogrusLogger struct {
	logger *logrus.Entry
}

// New creates a new LogrusLogger. If nil is passed, uses the standard logger.
func New(l *logrus.Logger) *LogrusLogger {
	if l == nil {
		l = logrus.New()
	}
	return &LogrusLogger{
		logger: logrus.NewEntry(l),
	}
}

// Debugf logs a debug-level message
func (l *LogrusLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

// Errorf logs an error-level message
func (l *LogrusLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}
