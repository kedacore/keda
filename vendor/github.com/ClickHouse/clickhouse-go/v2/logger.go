package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
)

// debugfHandler is a slog.Handler that wraps the legacy Debugf function
// for backward compatibility. It converts structured log records back to
// format string calls.
type debugfHandler struct {
	debugf func(format string, v ...any)
	attrs  []slog.Attr
	groups []string
}

func (h *debugfHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Legacy Debugf has no level filtering - all logs are enabled
	return true
}

func (h *debugfHandler) Handle(ctx context.Context, record slog.Record) error {
	// Build message with attributes
	msg := record.Message

	// Collect all attributes
	attrs := make([]any, 0, len(h.attrs)*2+record.NumAttrs()*2)

	// Add pre-existing attributes (from With)
	for _, a := range h.attrs {
		attrs = append(attrs, a.Key, a.Value)
	}

	// Add record attributes
	record.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a.Key, a.Value)
		return true
	})

	// Format message with attributes if present
	if len(attrs) > 0 {
		h.debugf("%s %v", msg, attrs)
	} else {
		h.debugf("%s", msg)
	}

	return nil
}

func (h *debugfHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Accumulate attributes for later formatting
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &debugfHandler{
		debugf: h.debugf,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

func (h *debugfHandler) WithGroup(name string) slog.Handler {
	// Accumulate groups (though legacy Debugf won't use them meaningfully)
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &debugfHandler{
		debugf: h.debugf,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

// noopHandler is a slog.Handler that discards all logs.
// Used when no logger is configured.
type noopHandler struct{}

func (h *noopHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Disable all log levels
	return false
}

func (h *noopHandler) Handle(ctx context.Context, record slog.Record) error {
	// Discard the log
	return nil
}

func (h *noopHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *noopHandler) WithGroup(name string) slog.Handler {
	return h
}

// newDebugfLogger creates a slog.Logger that wraps the legacy Debugf function.
// This is used for backward compatibility when Debug=true and Debugf is provided.
func newDebugfLogger(debugf func(format string, v ...any)) *slog.Logger {
	return slog.New(&debugfHandler{debugf: debugf})
}

// newNoopLogger creates a slog.Logger that discards all logs.
// This is used when no logger is configured (default behavior).
func newNoopLogger() *slog.Logger {
	return slog.New(&noopHandler{})
}

// prepareConnLogger enriches a base logger with connection-specific attributes.
// This adds context like connection ID, remote address, and protocol type.
func prepareConnLogger(base *slog.Logger, connID int, remoteAddr, protocol string) *slog.Logger {
	return base.With(
		slog.Int("conn_id", connID),
		slog.String("remote_addr", remoteAddr),
		slog.String("protocol", protocol),
	)
}

// formatForDebugf is a helper that formats a message for the legacy debugf wrapper.
// It's used by the debugf() methods to maintain compatibility with existing call sites.
func formatForDebugf(format string, v ...any) string {
	return fmt.Sprintf(format, v...)
}
