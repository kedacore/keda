package log

import (
	"fmt"
	"os"
	"time"
)

type BoltLogger interface {
	LogClientMessage(context string, msg string, args ...any)
	LogServerMessage(context string, msg string, args ...any)
}

type ConsoleBoltLogger struct {
}

func (cbl *ConsoleBoltLogger) LogClientMessage(id, msg string, args ...any) {
	cbl.logBoltMessage("C", id, msg, args)
}

func (cbl *ConsoleBoltLogger) LogServerMessage(id, msg string, args ...any) {
	cbl.logBoltMessage("S", id, msg, args)
}

func (cbl *ConsoleBoltLogger) logBoltMessage(src, id string, msg string, args []any) {
	_, _ = fmt.Fprintf(os.Stdout, "%s   BOLT  %s%s: %s\n", time.Now().Format(timeFormat), formatId(id), src, fmt.Sprintf(msg, args...))
}

func formatId(id string) string {
	if id == "" {
		return ""
	}
	return fmt.Sprintf("[%s] ", id)
}
