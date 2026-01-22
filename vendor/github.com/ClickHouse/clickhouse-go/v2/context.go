// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package clickhouse

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/ext"
	"go.opentelemetry.io/otel/trace"
)

var _contextOptionKey = &QueryOptions{
	settings: Settings{
		"_contextOption": struct{}{},
	},
}

type Settings map[string]any

// CustomSetting is a helper struct to distinguish custom settings from important ones.
// For native protocol, is_important flag is set to value 0x02 (see https://github.com/ClickHouse/ClickHouse/blob/c873560fe7185f45eed56520ec7d033a7beb1551/src/Core/BaseSettings.h#L516-L521)
// Only string value is supported until formatting logic that exists in ClickHouse is implemented in clickhouse-go. (https://github.com/ClickHouse/ClickHouse/blob/master/src/Core/Field.cpp#L312 and https://github.com/ClickHouse/clickhouse-go/issues/992)
type CustomSetting struct {
	Value string
}

type Parameters map[string]string
type (
	QueryOption  func(*QueryOptions) error
	QueryOptions struct {
		span  trace.SpanContext
		async struct {
			ok   bool
			wait bool
		}
		queryID  string
		quotaKey string
		events   struct {
			logs          func(*Log)
			progress      func(*Progress)
			profileInfo   func(*ProfileInfo)
			profileEvents func([]ProfileEvent)
		}
		settings        Settings
		parameters      Parameters
		external        []*ext.Table
		blockBufferSize uint8
		userLocation    *time.Location
	}
)

func WithSpan(span trace.SpanContext) QueryOption {
	return func(o *QueryOptions) error {
		o.span = span
		return nil
	}
}

func WithQueryID(queryID string) QueryOption {
	return func(o *QueryOptions) error {
		o.queryID = queryID
		return nil
	}
}

func WithBlockBufferSize(size uint8) QueryOption {
	return func(o *QueryOptions) error {
		o.blockBufferSize = size
		return nil
	}
}

func WithQuotaKey(quotaKey string) QueryOption {
	return func(o *QueryOptions) error {
		o.quotaKey = quotaKey
		return nil
	}
}

func WithSettings(settings Settings) QueryOption {
	return func(o *QueryOptions) error {
		o.settings = settings
		return nil
	}
}

func WithParameters(params Parameters) QueryOption {
	return func(o *QueryOptions) error {
		o.parameters = params
		return nil
	}
}

func WithLogs(fn func(*Log)) QueryOption {
	return func(o *QueryOptions) error {
		o.events.logs = fn
		return nil
	}
}

func WithProgress(fn func(*Progress)) QueryOption {
	return func(o *QueryOptions) error {
		o.events.progress = fn
		return nil
	}
}

func WithProfileInfo(fn func(*ProfileInfo)) QueryOption {
	return func(o *QueryOptions) error {
		o.events.profileInfo = fn
		return nil
	}
}

func WithProfileEvents(fn func([]ProfileEvent)) QueryOption {
	return func(o *QueryOptions) error {
		o.events.profileEvents = fn
		return nil
	}
}

func WithExternalTable(t ...*ext.Table) QueryOption {
	return func(o *QueryOptions) error {
		o.external = append(o.external, t...)
		return nil
	}
}

func WithStdAsync(wait bool) QueryOption {
	return func(o *QueryOptions) error {
		o.async.ok, o.async.wait = true, wait
		return nil
	}
}

func WithUserLocation(location *time.Location) QueryOption {
	return func(o *QueryOptions) error {
		o.userLocation = location
		return nil
	}
}

func ignoreExternalTables() QueryOption {
	return func(o *QueryOptions) error {
		o.external = nil
		return nil
	}
}

func Context(parent context.Context, options ...QueryOption) context.Context {
	opt := queryOptions(parent)
	for _, f := range options {
		f(&opt)
	}
	return context.WithValue(parent, _contextOptionKey, opt)
}

func queryOptions(ctx context.Context) QueryOptions {
	if o, ok := ctx.Value(_contextOptionKey).(QueryOptions); ok {
		if deadline, ok := ctx.Deadline(); ok {
			if sec := time.Until(deadline).Seconds(); sec > 1 {
				o.settings["max_execution_time"] = int(sec + 5)
			}
		}
		return o
	}
	return QueryOptions{
		settings: make(Settings),
	}
}

func (q *QueryOptions) onProcess() *onProcess {
	return &onProcess{
		logs: func(logs []Log) {
			if q.events.logs != nil {
				for _, l := range logs {
					q.events.logs(&l)
				}
			}
		},
		progress: func(p *Progress) {
			if q.events.progress != nil {
				q.events.progress(p)
			}
		},
		profileInfo: func(p *ProfileInfo) {
			if q.events.profileInfo != nil {
				q.events.profileInfo(p)
			}
		},
		profileEvents: func(events []ProfileEvent) {
			if q.events.profileEvents != nil {
				q.events.profileEvents(events)
			}
		},
	}
}
