package clickhouse

import (
	"context"
	"maps"
	"slices"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/ClickHouse/clickhouse-go/v2/ext"
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

// ColumnNameAndType represents a column name and type
type ColumnNameAndType struct {
	Name string
	Type string
}

type Parameters map[string]string
type (
	QueryOption  func(*QueryOptions) error
	AsyncOptions struct {
		ok   bool
		wait bool
	}
	QueryOptions struct {
		span     trace.SpanContext
		async    AsyncOptions
		queryID  string
		quotaKey string
		jwt      string
		events   struct {
			logs          func(*Log)
			progress      func(*Progress)
			profileInfo   func(*ProfileInfo)
			profileEvents func([]ProfileEvent)
		}
		settings            Settings
		parameters          Parameters
		external            []*ext.Table
		blockBufferSize     uint8
		userLocation        *time.Location
		columnNamesAndTypes []ColumnNameAndType
		clientInfo          ClientInfo
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

// WithJWT overrides the existing authentication with the given JWT.
// This only applies for clients connected with HTTPS to ClickHouse Cloud.
func WithJWT(jwt string) QueryOption {
	return func(o *QueryOptions) error {
		o.jwt = jwt
		return nil
	}
}

// WithColumnNamesAndTypes is used to provide a predetermined list of
// column names and types for HTTP inserts.
// Without this, the HTTP implementation will parse the query and run a
// DESCRIBE TABLE request to fetch and validate column names.
func WithColumnNamesAndTypes(columnNamesAndTypes []ColumnNameAndType) QueryOption {
	return func(o *QueryOptions) error {
		o.columnNamesAndTypes = columnNamesAndTypes
		return nil
	}
}

// WithClientInfo appends client info data to the query, visible in the system.query_log table.
// This does not replace the client info provided in the connection options, it appends to it.
// Can be called multiple times to append more info.
func WithClientInfo(ci ClientInfo) QueryOption {
	return func(o *QueryOptions) error {
		o.clientInfo = o.clientInfo.Append(ci)
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

func WithAsync(wait bool) QueryOption {
	return func(o *QueryOptions) error {
		o.async.ok, o.async.wait = true, wait
		return nil
	}
}

// Deprecated: use `WithAsync` instead.
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

// Context returns a derived context with the given ClickHouse QueryOptions.
// Existing QueryOptions will be overwritten per option if present.
// The QueryOptions Settings map will be initialized if nil.
func Context(parent context.Context, options ...QueryOption) context.Context {
	var opt QueryOptions
	if ctxOpt, ok := parent.Value(_contextOptionKey).(QueryOptions); ok {
		opt = ctxOpt
	}

	for _, f := range options {
		f(&opt)
	}

	if opt.settings == nil {
		opt.settings = make(Settings)
	}

	return context.WithValue(parent, _contextOptionKey, opt)
}

// queryOptions returns a mutable copy of the QueryOptions struct within the given context.
// If ClickHouse context was not provided, an empty struct with a valid Settings map is returned.
// If the context has a deadline greater than 1s then max_execution_time setting is appended.
func queryOptions(ctx context.Context) QueryOptions {
	var opt QueryOptions

	if ctxOpt, ok := ctx.Value(_contextOptionKey).(QueryOptions); ok {
		opt = ctxOpt.clone()
	} else {
		opt = QueryOptions{
			settings: make(Settings),
		}
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		return opt
	}

	if sec := time.Until(deadline).Seconds(); sec > 1 {
		opt.settings["max_execution_time"] = int(sec + 5)
	}

	return opt
}

// queryOptionsJWT returns the JWT within the given context's QueryOptions.
// Empty string if not present.
func queryOptionsJWT(ctx context.Context) string {
	if opt, ok := ctx.Value(_contextOptionKey).(QueryOptions); ok {
		return opt.jwt
	}

	return ""
}

// queryOptionsAsync returns the AsyncOptions struct within the given context's QueryOptions.
func queryOptionsAsync(ctx context.Context) AsyncOptions {
	if opt, ok := ctx.Value(_contextOptionKey).(QueryOptions); ok {
		return opt.async
	}

	return AsyncOptions{}
}

// queryOptionsUserLocation returns the *time.Location within the given context's QueryOptions.
func queryOptionsUserLocation(ctx context.Context) *time.Location {
	if opt, ok := ctx.Value(_contextOptionKey).(QueryOptions); ok {
		return opt.userLocation
	}

	return nil
}

// WithoutProfileEvents instructs the server not to send profile events for this query.
// This is a performance optimization for servers >= 25.11 that support the send_profile_events setting.
// On older servers, the setting is unknown and the server will return an error.
func WithoutProfileEvents() QueryOption {
	return func(o *QueryOptions) error {
		if o.settings == nil {
			o.settings = make(Settings)
		}
		o.settings["send_profile_events"] = 0
		return nil
	}
}

func (q *QueryOptions) onProcess() *onProcess {
	onProcess := &onProcess{
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
	}

	profileEventsHandler := q.events.profileEvents
	if profileEventsHandler != nil {
		onProcess.profileEvents = func(events []ProfileEvent) {
			profileEventsHandler(events)
		}
	}

	return onProcess
}

// clone returns a copy of QueryOptions where Settings and Parameters are safely mutable.
func (q *QueryOptions) clone() QueryOptions {
	c := QueryOptions{
		span:                q.span,
		async:               q.async,
		queryID:             q.queryID,
		quotaKey:            q.quotaKey,
		events:              q.events,
		settings:            nil,
		parameters:          nil,
		external:            q.external,
		blockBufferSize:     q.blockBufferSize,
		userLocation:        q.userLocation,
		columnNamesAndTypes: nil,
	}

	if q.settings != nil {
		c.settings = maps.Clone(q.settings)
	}

	if q.parameters != nil {
		c.parameters = maps.Clone(q.parameters)
	}

	if q.columnNamesAndTypes != nil {
		c.columnNamesAndTypes = slices.Clone(q.columnNamesAndTypes)
	}

	if q.clientInfo.Products != nil || q.clientInfo.Comment != nil {
		c.clientInfo = q.clientInfo.Append(ClientInfo{})
	}

	return c
}
