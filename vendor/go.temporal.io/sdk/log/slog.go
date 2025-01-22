// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

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
