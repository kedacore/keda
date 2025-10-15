package log

import (
	"fmt"
	golog "log"
	"os"
	"strings"

	"go.temporal.io/sdk/log"
)

// DefaultLogger is Logger implementation on top of standard log.Logger. It is used if logger is not specified.
type DefaultLogger struct {
	logger        *golog.Logger
	globalKeyvals string
}

// NewDefaultLogger creates new instance of DefaultLogger.
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{logger: golog.New(os.Stdout, "", golog.LstdFlags)}
}

func (l *DefaultLogger) println(level, msg string, keyvals []interface{}) {
	// To avoid extra space when globalKeyvals is not specified.
	if l.globalKeyvals == "" {
		l.logger.Println(append([]interface{}{level, msg}, keyvals...)...)
	} else {
		l.logger.Println(append([]interface{}{level, msg, l.globalKeyvals}, keyvals...)...)
	}
}

// Debug writes message to the log.
func (l *DefaultLogger) Debug(msg string, keyvals ...interface{}) {
	l.println("DEBUG", msg, keyvals)
}

// Info writes message to the log.
func (l *DefaultLogger) Info(msg string, keyvals ...interface{}) {
	l.println("INFO ", msg, keyvals)
}

// Warn writes message to the log.
func (l *DefaultLogger) Warn(msg string, keyvals ...interface{}) {
	l.println("WARN ", msg, keyvals)
}

// Error writes message to the log.
func (l *DefaultLogger) Error(msg string, keyvals ...interface{}) {
	l.println("ERROR", msg, keyvals)
}

// With returns new logger the prepend every log entry with keyvals.
func (l *DefaultLogger) With(keyvals ...interface{}) log.Logger {
	logger := &DefaultLogger{
		logger: l.logger,
	}

	if l.globalKeyvals != "" {
		logger.globalKeyvals = l.globalKeyvals + " "
	}

	logger.globalKeyvals += strings.TrimSuffix(fmt.Sprintln(keyvals...), "\n")

	return logger
}
