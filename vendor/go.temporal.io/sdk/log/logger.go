package log

type (
	// Logger is an interface that can be passed to ClientOptions.Logger.
	Logger interface {
		Debug(msg string, keyvals ...interface{})
		Info(msg string, keyvals ...interface{})
		Warn(msg string, keyvals ...interface{})
		Error(msg string, keyvals ...interface{})
	}

	// WithSkipCallers is an optional interface that a Logger can implement that
	// may create a new child logger that skips the number of stack frames of the caller.
	// This call must not mutate the original logger.
	WithSkipCallers interface {
		WithCallerSkip(int) Logger
	}

	// WithLogger is an optional interface that prepend every log entry with keyvals.
	// This call must not mutate the original logger.
	WithLogger interface {
		With(keyvals ...interface{}) Logger
	}
)
