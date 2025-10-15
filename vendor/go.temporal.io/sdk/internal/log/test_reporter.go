package log

import (
	"fmt"
	"os"

	"go.temporal.io/sdk/log"
)

// TestReporter is a log adapter for gomock.
type TestReporter struct {
	logger log.Logger
}

// NewTestReporter creates new instance of TestReporter.
func NewTestReporter(logger log.Logger) *TestReporter {
	return &TestReporter{logger: logger}
}

// Errorf writes error to the log.
func (t *TestReporter) Errorf(format string, args ...interface{}) {
	t.logger.Error(fmt.Sprintf(format, args...))
}

// Fatalf writes error to the log and exits.
func (t *TestReporter) Fatalf(format string, args ...interface{}) {
	t.logger.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}
