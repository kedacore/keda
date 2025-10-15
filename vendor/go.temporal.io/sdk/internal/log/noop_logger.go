package log

import (
	"go.temporal.io/sdk/log"
)

// NoopLogger is Logger implementation that doesn't produce any logs.
type NoopLogger struct {
}

// NewNopLogger creates new instance of NoopLogger.
func NewNopLogger() *NoopLogger {
	return &NoopLogger{}
}

// Debug does nothing.
func (l *NoopLogger) Debug(string, ...interface{}) {}

// Info does nothing.
func (l *NoopLogger) Info(string, ...interface{}) {}

// Warn does nothing.
func (l *NoopLogger) Warn(string, ...interface{}) {}

// Error does nothing.
func (l *NoopLogger) Error(string, ...interface{}) {}

// With returns new NoopLogger.
func (l *NoopLogger) With(...interface{}) log.Logger {
	return NewNopLogger()
}
