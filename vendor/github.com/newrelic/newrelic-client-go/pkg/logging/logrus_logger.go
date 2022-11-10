package logging

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

var (
	defaultFields   map[string]string
	defaultLogLevel = "info"
)

// LogrusLogger is a logger based on logrus.
type LogrusLogger struct {
	logger *log.Logger
}

type ConfigOption func(*LogrusLogger)

func ConfigLoggerInstance(logger *log.Logger) ConfigOption {
	return func(l *LogrusLogger) {
		l.logger = logger
	}
}

// NewLogrusLogger creates a new structured logger.
func NewLogrusLogger(opts ...ConfigOption) *LogrusLogger {
	l := &LogrusLogger{
		logger: log.New(),
	}

	// Loop through config options
	for _, fn := range opts {
		if nil != fn {
			fn(l)
		}
	}

	return l
}

// SetLevel allows the log level to be set.
func (l LogrusLogger) SetLevel(levelName string) {
	if levelName == "" {
		levelName = defaultLogLevel
	}

	level, err := log.ParseLevel(levelName)
	if err != nil {
		l.logger.Warn(fmt.Sprintf("could not parse log level '%s', logging will proceed at %s level", levelName, defaultLogLevel))
		level, _ = log.ParseLevel(defaultLogLevel)
	}

	l.logger.SetLevel(level)
}

// LogJSON determines whether or not to format the logs as JSON.
func (l LogrusLogger) SetLogJSON(value bool) {
	if value {
		l.logger.SetFormatter(&log.JSONFormatter{})
	}
}

// SetDefaultFields sets fields to be logged on every use of the logger.
func (l LogrusLogger) SetDefaultFields(defaultFields map[string]string) {
	l.logger.AddHook(&defaultFieldHook{})
}

// Error logs an error message.
func (l LogrusLogger) Error(msg string, fields ...interface{}) {
	l.logger.WithFields(createFieldMap(fields)).Error(msg)
}

// Warn logs an warning message.
func (l LogrusLogger) Warn(msg string, fields ...interface{}) {
	l.logger.WithFields(createFieldMap(fields)).Warn(msg)
}

// Info logs an info message.
func (l LogrusLogger) Info(msg string, fields ...interface{}) {
	l.logger.WithFields(createFieldMap(fields)).Info(msg)
}

// Debug logs a debug message.
func (l LogrusLogger) Debug(msg string, fields ...interface{}) {
	l.logger.WithFields(createFieldMap(fields)).Debug(msg)
}

// Trace logs a trace message.
func (l LogrusLogger) Trace(msg string, fields ...interface{}) {
	l.logger.WithFields(createFieldMap(fields)).Trace(msg)
}

func createFieldMap(fields ...interface{}) map[string]interface{} {
	m := map[string]interface{}{}

	fields = fields[0].([]interface{})

	for i := 0; i < len(fields); i += 2 {
		m[fields[i].(string)] = fields[i+1]
	}

	return m
}

type defaultFieldHook struct{}

func (h *defaultFieldHook) Levels() []log.Level {
	return log.AllLevels
}

func (h *defaultFieldHook) Fire(e *log.Entry) error {
	for k, v := range defaultFields {
		e.Data[k] = v
	}
	return nil
}
