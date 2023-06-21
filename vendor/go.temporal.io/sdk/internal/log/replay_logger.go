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

package log

import (
	"go.temporal.io/sdk/log"
)

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

// With returns new logger the prepend every log entry with keyvals.
func (l *ReplayLogger) With(keyvals ...interface{}) log.Logger {
	return NewReplayLogger(log.With(l.logger, keyvals...), l.isReplay, l.enableLoggingInReplay)
}
