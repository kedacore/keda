package log

import (
	"go.temporal.io/sdk/log"
)

var _ log.Logger = (*ReplayLogger)(nil)
var _ log.WithLogger = (*ReplayLogger)(nil)
var _ log.WithSkipCallers = (*ReplayLogger)(nil)

// ReplayLogger is Logger implementation that is aware of replay.
type ReplayLogger struct {
	logger                log.Logger
	isReplay              *bool // pointer to bool that indicate if it is in replay mode
	enableLoggingInReplay *bool // pointer to bool that indicate if logging is enabled in replay mode
}

// NewReplayLogger crates new instance of ReplayLogger.
func NewReplayLogger(logger log.Logger, isReplay *bool, enableLoggingInReplay *bool) log.Logger {
	return &ReplayLogger{
		logger:                logger,
		isReplay:              isReplay,
		enableLoggingInReplay: enableLoggingInReplay,
	}
}

func (l *ReplayLogger) check() bool {
	return !*l.isReplay || *l.enableLoggingInReplay
}

// Debug writes message to the log if it is not a replay.
func (l *ReplayLogger) Debug(msg string, keyvals ...interface{}) {
	if l.check() {
		l.logger.Debug(msg, keyvals...)
	}
}

// Info writes message to the log if it is not a replay.
func (l *ReplayLogger) Info(msg string, keyvals ...interface{}) {
	if l.check() {
		l.logger.Info(msg, keyvals...)
	}
}

// Warn writes message to the log if it is not a replay.
func (l *ReplayLogger) Warn(msg string, keyvals ...interface{}) {
	if l.check() {
		l.logger.Warn(msg, keyvals...)
	}
}

// Error writes message to the log if it is not a replay.
func (l *ReplayLogger) Error(msg string, keyvals ...interface{}) {
	if l.check() {
		l.logger.Error(msg, keyvals...)
	}
}

// With returns new logger that prepend every log entry with keyvals.
func (l *ReplayLogger) With(keyvals ...interface{}) log.Logger {
	return NewReplayLogger(log.With(l.logger, keyvals...), l.isReplay, l.enableLoggingInReplay)
}

func (l *ReplayLogger) WithCallerSkip(depth int) log.Logger {
	if sl, ok := l.logger.(log.WithSkipCallers); ok {
		return NewReplayLogger(sl.WithCallerSkip(depth), l.isReplay, l.enableLoggingInReplay)
	}
	return l
}
