/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// logging.go contains the logic to configure and interact with the
// logging and metrics libraries.

package logging

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/knative/pkg/logging"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

const (
	// VerboseLogLevel defines verbose log level as 10
	VerboseLogLevel glog.Level = 10

	// 1 second was chosen arbitrarily
	metricViewReportingPeriod = 1 * time.Second

	// prefix attached to metric name that indicates to the
	// ExportSpan method that span needs to be emitted.
	emitableSpanNamePrefix = "emitspan-"
)

// FormatLogger is a printf style function for logging in tests.
type FormatLogger func(template string, args ...interface{})

var logger *zap.SugaredLogger

var exporter *zapMetricExporter

// zapMetricExporter is a stats and trace exporter that logs the
// exported data to the provided (probably test specific) zap logger.
// It conforms to the view.Exporter and trace.Exporter interfaces.
type zapMetricExporter struct {
	logger *zap.SugaredLogger
}

// ExportView will emit the view data vd (i.e. the stats that have been
// recorded) to the zap logger.
func (e *zapMetricExporter) ExportView(vd *view.Data) {
	// We are not currently consuming these metrics, so for now we'll juse
	// dump the view.Data object as is.
	e.logger.Debug(spew.Sprint(vd))
}

// GetEmitableSpan starts and returns a trace.Span with a name that
// is used by the ExportSpan method to emit the span.
func GetEmitableSpan(ctx context.Context, metricName string) *trace.Span {
	_, span := trace.StartSpan(ctx, emitableSpanNamePrefix+metricName)
	return span
}

// ExportSpan will emit the trace data to the zap logger. The span is emitted
// only if the metric name is prefix with emitableSpanNamePrefix constant.
func (e *zapMetricExporter) ExportSpan(vd *trace.SpanData) {
	if strings.HasPrefix(vd.Name, emitableSpanNamePrefix) {
		duration := vd.EndTime.Sub(vd.StartTime)
		// We will start the log entry with `metric` to identify it as a metric for parsing
		e.logger.Infof("metric %s %d %d %s", vd.Name[len(emitableSpanNamePrefix):], vd.StartTime.UnixNano(), vd.EndTime.UnixNano(), duration)
	}
}

func newLogger(logLevel string) *zap.SugaredLogger {
	configJSONTemplate := `{
	  "level": "%s",
	  "encoding": "console",
	  "outputPaths": ["stdout"],
	  "errorOutputPaths": ["stderr"],
	  "encoderConfig": {
	    "timeKey": "ts",
	    "messageKey": "message",
	    "levelKey": "level",
	    "nameKey": "logger",
	    "callerKey": "caller",
	    "messageKey": "msg",
	    "stacktraceKey": "stacktrace",
	    "lineEnding": "",
	    "levelEncoder": "",
	    "timeEncoder": "iso8601",
	    "durationEncoder": "",
	    "callerEncoder": ""
	  }
	}`
	configJSON := fmt.Sprintf(configJSONTemplate, logLevel)
	l, _ := logging.NewLogger(string(configJSON), logLevel, zap.AddCallerSkip(1))
	return l
}

// InitializeMetricExporter initializes the metric exporter logger
func InitializeMetricExporter(context string) {
	// If there was a previously registered exporter, unregister it so we only emit
	// the metrics in the current context.
	if exporter != nil {
		view.UnregisterExporter(exporter)
		trace.UnregisterExporter(exporter)
	}

	logger := logger.Named(context)

	exporter = &zapMetricExporter{logger: logger}
	view.RegisterExporter(exporter)
	trace.RegisterExporter(exporter)

	view.SetReportingPeriod(metricViewReportingPeriod)
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
}

// InitializeLogger initializes the base logger
func InitializeLogger(logVerbose bool) {
	logLevel := "info"
	if logVerbose {
		// Both gLog and "go test" use -v flag. The code below is a work around so that we can still set v value for gLog
		flag.StringVar(&logLevel, "logLevel", fmt.Sprint(VerboseLogLevel), "verbose log level")
		flag.Lookup("v").Value.Set(logLevel)
		glog.Infof("Logging set to verbose mode with logLevel %d", VerboseLogLevel)

		logLevel = "debug"
	}

	logger = newLogger(logLevel)
}
