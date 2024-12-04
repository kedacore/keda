package logging

import "testing"

type MockLogger struct {
	t *testing.T
}

func NewMockLogger(t *testing.T) *MockLogger {
	return &MockLogger{t: t}
}

// Error logs an error message.
func (l MockLogger) Error(msg string, fields ...interface{}) {
	l.t.Errorf(msg, fields...)
}

// Warn logs an warning message.
func (l MockLogger) Warn(msg string, fields ...interface{}) {
	l.t.Logf(msg, fields...)
}

// Info logs an info message.
func (l MockLogger) Info(msg string, fields ...interface{}) {
	l.t.Logf(msg, fields...)
}

// Debug logs a debug message.
func (l MockLogger) Debug(msg string, fields ...interface{}) {
}

// Trace logs a trace message.
func (l MockLogger) Trace(msg string, fields ...interface{}) {
}

func (l MockLogger) SetLevel(logLevel string) {
}
