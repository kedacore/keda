package log

import (
	"fmt"
	"strings"
	"sync"

	"go.temporal.io/sdk/log"
)

// MemoryLoggerWithoutWith is a Logger implementation that stores logs in memory (useful for testing). Use Lines() to get log lines.
type MemoryLoggerWithoutWith struct {
	lock          sync.RWMutex
	lines         *[]string
	globalKeyvals string
}

// NewMemoryLoggerWithoutWith creates new instance of MemoryLoggerWithoutWith.
func NewMemoryLoggerWithoutWith() *MemoryLoggerWithoutWith {
	var lines []string
	return &MemoryLoggerWithoutWith{
		lines: &lines,
	}
}

func (l *MemoryLoggerWithoutWith) println(level, msg string, keyvals []interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()
	// To avoid extra space when globalKeyvals is not specified.
	if l.globalKeyvals == "" {
		*l.lines = append(*l.lines, fmt.Sprintln(append([]interface{}{level, msg}, keyvals...)...))
	} else {
		*l.lines = append(*l.lines, fmt.Sprintln(append([]interface{}{level, msg, l.globalKeyvals}, keyvals...)...))
	}
}

// Debug appends message to the log.
func (l *MemoryLoggerWithoutWith) Debug(msg string, keyvals ...interface{}) {
	l.println("DEBUG", msg, keyvals)
}

// Info appends message to the log.
func (l *MemoryLoggerWithoutWith) Info(msg string, keyvals ...interface{}) {
	l.println("INFO ", msg, keyvals)
}

// Warn appends message to the log.
func (l *MemoryLoggerWithoutWith) Warn(msg string, keyvals ...interface{}) {
	l.println("WARN ", msg, keyvals)
}

// Error appends message to the log.
func (l *MemoryLoggerWithoutWith) Error(msg string, keyvals ...interface{}) {
	l.println("ERROR", msg, keyvals)
}

// Lines returns written log lines.
func (l *MemoryLoggerWithoutWith) Lines() []string {
	l.lock.RLock()
	defer l.lock.RUnlock()
	ret := make([]string, len(*l.lines))
	copy(ret, *l.lines)
	return ret
}

type MemoryLogger struct {
	*MemoryLoggerWithoutWith
}

// NewMemoryLogger creates new instance of MemoryLogger.
func NewMemoryLogger() *MemoryLogger {
	return &MemoryLogger{
		NewMemoryLoggerWithoutWith(),
	}
}

// With returns new logger that prepend every log entry with keyvals.
func (l *MemoryLogger) With(keyvals ...interface{}) log.Logger {
	l.lock.RLock()
	defer l.lock.RUnlock()

	logger := &MemoryLoggerWithoutWith{
		lines: l.lines,
	}

	if l.globalKeyvals != "" {
		logger.globalKeyvals = l.globalKeyvals + " "
	}

	logger.globalKeyvals += strings.TrimSuffix(fmt.Sprintln(keyvals...), "\n")

	return logger
}
