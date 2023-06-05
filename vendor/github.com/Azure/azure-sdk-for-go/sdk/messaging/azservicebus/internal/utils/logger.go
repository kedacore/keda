// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package utils

import (
	"fmt"
	"sync/atomic"

	azlog "github.com/Azure/azure-sdk-for-go/sdk/internal/log"
)

type Logger struct {
	prefix *atomic.Value
}

func NewLogger() Logger {
	value := &atomic.Value{}
	value.Store("")

	return Logger{
		prefix: value,
	}
}

func (l *Logger) SetPrefix(format string, args ...any) {
	l.prefix.Store(fmt.Sprintf("["+format+"] ", args...))
}

func (l *Logger) Prefix() string {
	return l.prefix.Load().(string)
}

func (l *Logger) Writef(evt azlog.Event, format string, args ...any) {
	prefix := l.prefix.Load().(string)
	azlog.Writef(evt, prefix+format, args...)
}
