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
	"fmt"
	"strings"

	"go.temporal.io/sdk/log"
)

// MemoryLogger is Logger implementation that stores logs in memory (useful for testing). Use Lines() to get log lines.
type MemoryLogger struct {
	lines         *[]string
	globalKeyvals string
}

// NewMemoryLogger creates new instance of MemoryLogger.
func NewMemoryLogger() *MemoryLogger {
	var lines []string
	return &MemoryLogger{
		lines: &lines,
	}
}

func (l *MemoryLogger) println(level, msg string, keyvals []interface{}) {
	// To avoid extra space when globalKeyvals is not specified.
	if l.globalKeyvals == "" {
		*l.lines = append(*l.lines, fmt.Sprintln(append([]interface{}{level, msg}, keyvals...)...))
	} else {
		*l.lines = append(*l.lines, fmt.Sprintln(append([]interface{}{level, msg, l.globalKeyvals}, keyvals...)...))
	}
}

// Debug appends message to the log.
func (l *MemoryLogger) Debug(msg string, keyvals ...interface{}) {
	l.println("DEBUG", msg, keyvals)
}

// Info appends message to the log.
func (l *MemoryLogger) Info(msg string, keyvals ...interface{}) {
	l.println("INFO ", msg, keyvals)
}

// Warn appends message to the log.
func (l *MemoryLogger) Warn(msg string, keyvals ...interface{}) {
	l.println("WARN ", msg, keyvals)
}

// Error appends message to the log.
func (l *MemoryLogger) Error(msg string, keyvals ...interface{}) {
	l.println("ERROR", msg, keyvals)
}

// With returns new logger the prepend every log entry with keyvals.
func (l *MemoryLogger) With(keyvals ...interface{}) log.Logger {
	logger := &MemoryLogger{
		lines: l.lines,
	}

	if l.globalKeyvals != "" {
		logger.globalKeyvals = l.globalKeyvals + " "
	}

	logger.globalKeyvals += strings.TrimSuffix(fmt.Sprintln(keyvals...), "\n")

	return logger
}

// Lines returns written log lines.
func (l *MemoryLogger) Lines() []string {
	return *l.lines
}
