package log

// With creates a child Logger that includes the supplied key-value pairs in each log entry. It does this by
// using the supplied logger if it implements WithLogger; otherwise, it does so by intercepting every log call.
func With(logger Logger, keyvals ...interface{}) Logger {
	if wl, ok := logger.(WithLogger); ok {
		return wl.With(keyvals...)
	}

	return newWithLogger(logger, keyvals...)
}

// Skip creates a child Logger that increase increases its' caller skip depth if it
// implements [WithSkipCallers]. Otherwise returns the original logger.
func Skip(logger Logger, depth int) Logger {
	if sl, ok := logger.(WithSkipCallers); ok {
		return sl.WithCallerSkip(depth)
	}
	return logger
}

var _ Logger = (*withLogger)(nil)
var _ WithSkipCallers = (*withLogger)(nil)

type withLogger struct {
	logger  Logger
	keyvals []interface{}
}

func newWithLogger(logger Logger, keyvals ...interface{}) *withLogger {
	return &withLogger{logger: Skip(logger, 1), keyvals: keyvals}
}

func (l *withLogger) prependKeyvals(keyvals []interface{}) []interface{} {
	return append(l.keyvals, keyvals...)
}

// Debug writes message to the log.
func (l *withLogger) Debug(msg string, keyvals ...interface{}) {
	l.logger.Debug(msg, l.prependKeyvals(keyvals)...)
}

// Info writes message to the log.
func (l *withLogger) Info(msg string, keyvals ...interface{}) {
	l.logger.Info(msg, l.prependKeyvals(keyvals)...)
}

// Warn writes message to the log.
func (l *withLogger) Warn(msg string, keyvals ...interface{}) {
	l.logger.Warn(msg, l.prependKeyvals(keyvals)...)
}

// Error writes message to the log.
func (l *withLogger) Error(msg string, keyvals ...interface{}) {
	l.logger.Error(msg, l.prependKeyvals(keyvals)...)
}

func (l *withLogger) WithCallerSkip(depth int) Logger {
	if sl, ok := l.logger.(WithSkipCallers); ok {
		return newWithLogger(sl.WithCallerSkip(depth), l.keyvals...)
	}
	return l
}
