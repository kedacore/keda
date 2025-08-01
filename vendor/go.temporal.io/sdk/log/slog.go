//go:build go1.21

package log

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

type slogLogger struct {
	logger *slog.Logger
	depth  int
}

// NewStructuredLogger creates an adapter around the given logger to be passed to Temporal.
func NewStructuredLogger(logger *slog.Logger) Logger {
	return &slogLogger{
		logger: logger,
		depth:  3,
	}
}

func (s *slogLogger) Debug(msg string, keyvals ...interface{}) {
	s.log(context.Background(), slog.LevelDebug, msg, keyvals...)
}

func (s *slogLogger) Info(msg string, keyvals ...interface{}) {
	s.log(context.Background(), slog.LevelInfo, msg, keyvals...)
}

func (s *slogLogger) Warn(msg string, keyvals ...interface{}) {
	s.log(context.Background(), slog.LevelWarn, msg, keyvals...)
}

func (s *slogLogger) Error(msg string, keyvals ...interface{}) {
	s.log(context.Background(), slog.LevelError, msg, keyvals...)
}

func (s *slogLogger) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	if !s.logger.Enabled(ctx, level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(s.depth, pcs[:])

	record := slog.NewRecord(time.Now(), level, msg, pcs[0])
	record.Add(args...)

	if ctx == nil {
		ctx = context.Background()
	}
	_ = s.logger.Handler().Handle(ctx, record)
}

func (s *slogLogger) With(keyvals ...interface{}) Logger {
	return &slogLogger{
		logger: s.logger.With(keyvals...),
		depth:  s.depth,
	}
}

func (s *slogLogger) WithCallerSkip(depth int) Logger {
	return &slogLogger{
		logger: s.logger,
		depth:  s.depth + depth,
	}
}
