/*
Copyright 2017 Google LLC

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

package spanner

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/internal/trace"
	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/GoogleCloudPlatform/grpc-gcp-go/grpcgcp"
	grpcgcppb "github.com/GoogleCloudPlatform/grpc-gcp-go/grpcgcp/grpc_gcp"
	"github.com/googleapis/gax-go/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	otrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/api/option/internaloption"
	gtransport "google.golang.org/api/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	vkit "cloud.google.com/go/spanner/apiv1"
	"cloud.google.com/go/spanner/internal"
)

const (
	// resourcePrefixHeader is the name of the metadata header used to indicate
	// the resource being operated on.
	resourcePrefixHeader = "google-cloud-resource-prefix"

	// routeToLeaderHeader is the name of the metadata header if RW/PDML
	// requests need  to route to leader.
	routeToLeaderHeader = "x-goog-spanner-route-to-leader"

	requestsCompressionHeader = "x-response-encoding"

	// endToEndTracingHeader is the name of the metadata header if client
	// has opted-in for the creation of trace spans on the Spanner layer.
	endToEndTracingHeader = "x-goog-spanner-end-to-end-tracing"

	// afeMetricHeader is the name of the metadata header if client
	// has opted-in for the receiving Spanner API Frontend server timing metrics.
	afeMetricHeader = "x-goog-spanner-enable-afe-server-timing"

	// numChannels is the default value for NumChannels of client.
	numChannels = 4

	// Server timing header constants
	serverTimingHeaderKey = "server-timing"
	gfeTimingHeader       = "gfet4t7"
	afeTimingHeader       = "afe"

	// MinSessions for Experimental Host connection
	experimentalHostMinSessions = 0
)

const (
	// Scope is the scope for Cloud Spanner Data API.
	Scope = "https://www.googleapis.com/auth/spanner.data"

	// AdminScope is the scope for Cloud Spanner Admin APIs.
	AdminScope = "https://www.googleapis.com/auth/spanner.admin"
)

var (
	validDBPattern      = regexp.MustCompile("^projects/(?P<project>[^/]+)/instances/(?P<instance>[^/]+)/databases/(?P<database>[^/]+)$")
	serverTimingPattern = regexp.MustCompile(`([a-zA-Z0-9_-]+);\s*dur=(\d*\.?\d+)`)
)

func validDatabaseName(db string) error {
	if matched := validDBPattern.MatchString(db); !matched {
		return fmt.Errorf("database name %q should conform to pattern %q",
			db, validDBPattern.String())
	}
	return nil
}

func parseDatabaseName(db string) (project, instance, database string, err error) {
	matches := validDBPattern.FindStringSubmatch(db)
	if len(matches) == 0 {
		return "", "", "", fmt.Errorf("failed to parse database name from %q according to pattern %q",
			db, validDBPattern.String())
	}
	return matches[1], matches[2], matches[3], nil
}

// Client is a client for reading and writing data to a Cloud Spanner database.
// A client is safe to use concurrently, except for its Close method.
type Client struct {
	sc                   *sessionClient
	sm                   *sessionManager
	logger               *log.Logger
	qo                   QueryOptions
	ro                   ReadOptions
	ao                   []ApplyOption
	txo                  TransactionOptions
	bwo                  BatchWriteOptions
	ct                   *commonTags
	disableRouteToLeader bool
	dro                  *sppb.DirectedReadOptions
	otConfig             *openTelemetryConfig
	metricsTracerFactory *builtinMetricsTracerFactory
	clientContext        *sppb.RequestOptions_ClientContext
	locationRouter       *locationRouter
}

// DatabaseName returns the full name of a database, e.g.,
// "projects/spanner-cloud-test/instances/foo/databases/foodb".
func (c *Client) DatabaseName() string {
	return c.sc.database
}

// ClientID returns the id of the Client. This is not recommended for customer applications and used internally for testing.
func (c *Client) ClientID() string {
	return c.sc.id
}

func createGCPMultiEndpoint(cfg *grpcgcp.GCPMultiEndpointOptions, config ClientConfig, opts ...option.ClientOption) (*grpcgcp.GCPMultiEndpoint, error) {
	if cfg.GRPCgcpConfig == nil {
		cfg.GRPCgcpConfig = &grpcgcppb.ApiConfig{}
	}
	if len(cfg.GRPCgcpConfig.Method) == 0 {
		cfg.GRPCgcpConfig.Method = []*grpcgcppb.MethodConfig{
			{
				Name: []string{"/google.spanner.v1.Spanner/CreateSession"},
				Affinity: &grpcgcppb.AffinityConfig{
					Command:     grpcgcppb.AffinityConfig_BIND,
					AffinityKey: "name",
				},
			},
			{
				Name: []string{"/google.spanner.v1.Spanner/BatchCreateSessions"},
				Affinity: &grpcgcppb.AffinityConfig{
					Command:     grpcgcppb.AffinityConfig_BIND,
					AffinityKey: "session.name",
				},
			},
			{
				Name: []string{"/google.spanner.v1.Spanner/DeleteSession"},
				Affinity: &grpcgcppb.AffinityConfig{
					Command:     grpcgcppb.AffinityConfig_UNBIND,
					AffinityKey: "name",
				},
			},
			{
				Name: []string{"/google.spanner.v1.Spanner/GetSession"},
				Affinity: &grpcgcppb.AffinityConfig{
					Command:     grpcgcppb.AffinityConfig_BOUND,
					AffinityKey: "name",
				},
			},
			{
				Name: []string{
					"/google.spanner.v1.Spanner/BeginTransaction",
					"/google.spanner.v1.Spanner/Commit",
					"/google.spanner.v1.Spanner/ExecuteBatchDml",
					"/google.spanner.v1.Spanner/ExecuteSql",
					"/google.spanner.v1.Spanner/ExecuteStreamingSql",
					"/google.spanner.v1.Spanner/PartitionQuery",
					"/google.spanner.v1.Spanner/PartitionRead",
					"/google.spanner.v1.Spanner/Read",
					"/google.spanner.v1.Spanner/Rollback",
					"/google.spanner.v1.Spanner/StreamingRead",
				},
				Affinity: &grpcgcppb.AffinityConfig{
					Command:     grpcgcppb.AffinityConfig_BOUND,
					AffinityKey: "session",
				},
			},
		}
	}
	// Append emulator options if SPANNER_EMULATOR_HOST has been set.
	if emulatorAddr := os.Getenv("SPANNER_EMULATOR_HOST"); emulatorAddr != "" {
		schemeRemoved := regexp.MustCompile("^(http://|https://|passthrough:///)").ReplaceAllString(emulatorAddr, "")
		emulatorOpts := []option.ClientOption{
			option.WithEndpoint("passthrough:///" + schemeRemoved),
			option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
			option.WithoutAuthentication(),
			internaloption.SkipDialSettingsValidation(),
		}
		opts = append(opts, emulatorOpts...)
		// Replace all endpoints with emulator target.
		for _, meo := range cfg.MultiEndpoints {
			meo.Endpoints = []string{emulatorAddr}
		}
	}

	// Set the number of channels to the default value if not specified.
	if cfg.GRPCgcpConfig.GetChannelPool() == nil || cfg.GRPCgcpConfig.GetChannelPool().GetMaxSize() == 0 {
		cfg.GRPCgcpConfig.ChannelPool = &grpcgcppb.ChannelPoolConfig{
			MinSize: numChannels,
			MaxSize: numChannels,
		}
	}
	// Set MinSize equal to MaxSize to create all the channels beforehand.
	cfg.GRPCgcpConfig.ChannelPool.MinSize = cfg.GRPCgcpConfig.ChannelPool.GetMaxSize()

	cfg.GRPCgcpConfig.ChannelPool.BindPickStrategy = grpcgcppb.ChannelPoolConfig_ROUND_ROBIN

	cfg.DialFunc = func(ctx context.Context, target string, dopts ...grpc.DialOption) (*grpc.ClientConn, error) {
		copts := opts

		for _, do := range dopts {
			copts = append(copts, option.WithGRPCDialOption(do))
		}

		allOpts := allClientOpts(1, config.Compression, config.EnableDirectAccess, copts...)

		// Overwrite endpoint and pool config.
		allOpts = append(allOpts,
			option.WithEndpoint(target),
			option.WithGRPCConnectionPool(1),
			option.WithGRPCConn(nil),
		)

		return gtransport.Dial(ctx, allOpts...)
	}

	gme, err := grpcgcp.NewGCPMultiEndpoint(cfg)
	return gme, err
}

// To use GCPMultiEndpoint in gtransport.Dial (via gtransport.WithConnPool option)
// we implement gtransport.ConnPool interface using this wrapper.
type gmeWrapper struct {
	*grpcgcp.GCPMultiEndpoint
}

// Make sure gmeWrapper implements ConnPool interface.
var _ gtransport.ConnPool = (*gmeWrapper)(nil)

func (gw *gmeWrapper) Conn() *grpc.ClientConn {
	// GCPMultiEndpoint does not expose any ClientConn.
	// This is safe because Cloud Spanner client doesn't use this function and instead
	// makes calls directly using Invoke and NewStream from the grpc.ClientConnInterface
	// which GCPMultiEndpoint implements.
	return nil
}

func (gw *gmeWrapper) Num() int {
	return int(gw.GCPMultiEndpoint.GCPConfig().GetChannelPool().GetMaxSize())
}

// ClientConfig has configurations for the client.
type ClientConfig struct {
	// NumChannels is the number of gRPC channels.
	// If zero, a reasonable default is used based on the execution environment.
	//
	// Deprecated: The Spanner client now uses a pool of gRPC connections. Use
	// option.WithGRPCConnectionPool(numConns) instead to specify the number of
	// connections the client should use. The client will default to a
	// reasonable default if this option is not specified.
	NumChannels int

	// SessionPoolConfig is the configuration for session pool.
	SessionPoolConfig

	// SessionLabels for the sessions created by this client.
	// See https://cloud.google.com/spanner/docs/reference/rpc/google.spanner.v1#session
	// for more info.
	SessionLabels map[string]string

	// QueryOptions is the configuration for executing a sql query.
	QueryOptions QueryOptions

	// ReadOptions is the configuration for reading rows from a database
	ReadOptions ReadOptions

	// ApplyOptions is the configuration for applying
	ApplyOptions []ApplyOption

	// TransactionOptions is the configuration for a transaction.
	TransactionOptions TransactionOptions

	// BatchWriteOptions is the configuration for a BatchWrite request.
	BatchWriteOptions BatchWriteOptions

	// CallOptions is the configuration for providing custom retry settings that
	// override the default values.
	CallOptions *vkit.CallOptions

	// UserAgent is the prefix to the user agent header. This is used to supply information
	// such as application name or partner tool.
	//
	// Internal Use Only: This field is for internal tracking purpose only,
	// setting the value for this config is not required.
	//
	// Recommended format: ``application-or-tool-ID/major.minor.version``.
	UserAgent string

	// DatabaseRole specifies the role to be assumed for all operations on the
	// database by this client.
	DatabaseRole string

	// DisableRouteToLeader specifies if all the requests of type read-write and PDML
	// need to be routed to the leader region.
	//
	// Default: false
	DisableRouteToLeader bool

	// Logger is the logger to use for this client. If it is nil, all logging
	// will be directed to the standard logger.
	Logger *log.Logger

	//
	// Sets the compression to use for all gRPC calls. The compressor must be a valid name.
	// This will enable compression both from the client to the
	// server and from the server to the client.
	//
	// Supported values are:
	//  gzip: Enable gzip compression
	//  identity: Disable compression
	//
	//  Default: identity
	Compression string

	// BatchTimeout specifies the timeout for a batch of sessions managed sessionClient.
	BatchTimeout time.Duration

	// ClientConfig options used to set the DirectedReadOptions for all ReadRequests
	// and ExecuteSqlRequests for the Client which indicate which replicas or regions
	// should be used for non-transactional reads or queries.
	DirectedReadOptions *sppb.DirectedReadOptions

	OpenTelemetryMeterProvider metric.MeterProvider

	// EnableEndToEndTracing indicates whether end to end tracing is enabled or not. If
	// it is enabled, trace spans will be created at Spanner layer. Enabling end to end
	// tracing requires OpenTelemetry to be set up. Simply enabling this option won't
	// generate traces at Spanner layer.
	//
	// Default: false
	EnableEndToEndTracing bool

	// DisableNativeMetrics indicates whether native metrics should be disabled or not.
	// If true, native metrics will not be emitted.
	//
	// Default: false
	DisableNativeMetrics bool

	// Default: false
	IsExperimentalHost bool

	// ClientContext is the default context for all requests made by the client.
	ClientContext *sppb.RequestOptions_ClientContext

	// EnableDirectAccess option is used to enable the directpath.
	// This setting is overridden by the GOOGLE_SPANNER_ENABLE_DIRECT_ACCESS
	// environment variable if it is set.
	//
	// Default: false
	EnableDirectAccess bool
}

type openTelemetryConfig struct {
	enabled                     bool
	meterProvider               metric.MeterProvider
	commonTraceStartOptions     []otrace.SpanStartOption
	attributeMap                []attribute.KeyValue
	attributeMapWithMultiplexed []attribute.KeyValue
	otMetricRegistration        metric.Registration
	openSessionCount            metric.Int64ObservableGauge
	getSessionTimeoutsCount     metric.Int64Counter
	acquiredSessionsCount       metric.Int64Counter
	releasedSessionsCount       metric.Int64Counter
	gfeLatency                  metric.Int64Histogram
	gfeHeaderMissingCount       metric.Int64Counter
}

func contextWithOutgoingMetadata(ctx context.Context, md metadata.MD, disableRouteToLeader bool) context.Context {
	existing, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		// Make sure that we only send one resource header.
		existing.Delete(resourcePrefixHeader)
		md = metadata.Join(existing, md)
	}
	if !disableRouteToLeader {
		md = metadata.Join(md, metadata.Pairs(routeToLeaderHeader, "true"))
	}
	return metadata.NewOutgoingContext(ctx, md)
}

// NewClient creates a client to a database. A valid database name has the
// form projects/PROJECT_ID/instances/INSTANCE_ID/databases/DATABASE_ID. It uses
// a default configuration.
func NewClient(ctx context.Context, database string, opts ...option.ClientOption) (*Client, error) {
	return NewClientWithConfig(ctx, database, ClientConfig{SessionPoolConfig: DefaultSessionPoolConfig, DisableRouteToLeader: false}, opts...)
}

// NewClientWithConfig creates a client to a database. A valid database name has
// the form projects/PROJECT_ID/instances/INSTANCE_ID/databases/DATABASE_ID.
func NewClientWithConfig(ctx context.Context, database string, config ClientConfig, opts ...option.ClientOption) (c *Client, err error) {
	return newClientWithConfig(ctx, database, config, nil, opts...)
}

type fallbackWrapper struct {
	*grpcgcp.GCPFallback
	primaryConn  gtransport.ConnPool
	fallbackConn gtransport.ConnPool
}

// Conn returns nil because GCPFallback hides the underlying ClientConn.
// The Spanner client handles this by using the interface methods (Invoke/NewStream).
func (fw *fallbackWrapper) Conn() *grpc.ClientConn {
	return nil
}

func (fw *fallbackWrapper) Num() int {
	return fw.primaryConn.Num()
}

func (fw *fallbackWrapper) Close() error {
	fw.GCPFallback.Close()
	err1 := fw.primaryConn.Close()
	err2 := fw.fallbackConn.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func newClientWithConfig(ctx context.Context, database string, config ClientConfig, gme *grpcgcp.GCPMultiEndpoint, opts ...option.ClientOption) (c *Client, err error) {
	// Validate database path.
	if err := validDatabaseName(database); err != nil {
		return nil, err
	}

	ctx, _ = startSpan(ctx, "NewClient")
	defer func() { endSpan(ctx, err) }()

	// Explicitly disable some gRPC experiments as they are not stable yet.
	gRPCPickFirstEnvVarName := "GRPC_EXPERIMENTAL_ENABLE_NEW_PICK_FIRST"
	if os.Getenv(gRPCPickFirstEnvVarName) == "" {
		err := os.Setenv(gRPCPickFirstEnvVarName, "false")
		if err != nil {
			logf(config.Logger, "Error overriding GRPC_EXPERIMENTAL_ENABLE_NEW_PICK_FIRST to false: %v. Ignoring.", err)
		}
	}

	// Append emulator options if SPANNER_EMULATOR_HOST has been set.
	if emulatorAddr := os.Getenv("SPANNER_EMULATOR_HOST"); emulatorAddr != "" {
		schemeRemoved := regexp.MustCompile("^(http://|https://|passthrough:///)").ReplaceAllString(emulatorAddr, "")
		emulatorOpts := []option.ClientOption{
			option.WithEndpoint("passthrough:///" + schemeRemoved),
			option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
			option.WithoutAuthentication(),
			internaloption.SkipDialSettingsValidation(),
		}
		opts = append(emulatorOpts, opts...)
	}

	// Prepare gRPC channels.
	hasNumChannelsConfig := config.NumChannels > 0
	if config.NumChannels == 0 {
		config.NumChannels = numChannels
	}

	var metricsProvider metric.MeterProvider
	if emulatorAddr := os.Getenv("SPANNER_EMULATOR_HOST"); emulatorAddr != "" {
		// Do not emit native metrics when emulator is being used
		metricsProvider = noop.NewMeterProvider()
	}
	// Check if native metrics are disabled via env.
	if disableNativeMetrics, _ := strconv.ParseBool(os.Getenv("SPANNER_DISABLE_BUILTIN_METRICS")); disableNativeMetrics {
		config.DisableNativeMetrics = true
	}
	if config.DisableNativeMetrics {
		// Do not emit native metrics when DisableNativeMetrics is set
		metricsProvider = noop.NewMeterProvider()
	}
	isAFEBuiltInMetricEnabled := strings.EqualFold("false", os.Getenv("SPANNER_DISABLE_AFE_SERVER_TIMING"))
	isGRPCBuiltInMetricsEnabled := strings.EqualFold("false", os.Getenv("SPANNER_DISABLE_DIRECT_ACCESS_GRPC_BUILTIN_METRICS"))
	// enable the AFE/GRPC built-in metrics if direct-path is enabled
	isDirectPathEnabled := config.EnableDirectAccess
	if enableDirectPathXdsString := os.Getenv("GOOGLE_SPANNER_ENABLE_DIRECT_ACCESS"); enableDirectPathXdsString != "" {
		isDirectPathEnabled, _ = strconv.ParseBool(enableDirectPathXdsString)
	}
	if isDirectPathEnabled {
		isAFEBuiltInMetricEnabled = true
		isGRPCBuiltInMetricsEnabled = true
	}
	// disable the AFE/GRPC built-in metrics if the env var is explicitly set
	if ok, _ := strconv.ParseBool(os.Getenv("SPANNER_DISABLE_AFE_SERVER_TIMING")); ok {
		isAFEBuiltInMetricEnabled = false
	}
	if ok, _ := strconv.ParseBool(os.Getenv("SPANNER_DISABLE_DIRECT_ACCESS_GRPC_BUILTIN_METRICS")); ok {
		isGRPCBuiltInMetricsEnabled = false
	}

	metricsTracerFactory, err := newBuiltinMetricsTracerFactory(ctx, database, config.Compression, isAFEBuiltInMetricEnabled, isGRPCBuiltInMetricsEnabled, metricsProvider, opts...)
	if err != nil {
		return nil, err
	}
	if len(metricsTracerFactory.clientOpts) > 0 {
		opts = append(opts, metricsTracerFactory.clientOpts...)
	}

	var pool gtransport.ConnPool
	var endpointClientOpts []option.ClientOption

	if gme != nil {
		// Use GCPMultiEndpoint if provided.
		pool = &gmeWrapper{gme}
		endpointClientOpts = append(endpointClientOpts, opts...)
	} else if isFallbackEnabled, _ := strconv.ParseBool(os.Getenv("GOOGLE_SPANNER_ENABLE_GCP_FALLBACK")); isFallbackEnabled && isDirectPathEnabled {
		var primaryConn gtransport.ConnPool
		var fallbackConn gtransport.ConnPool
		reqIDInjector := new(requestIDHeaderInjector)
		opts = append(opts,
			option.WithGRPCDialOption(grpc.WithChainStreamInterceptor(reqIDInjector.interceptStream)),
			option.WithGRPCDialOption(grpc.WithChainUnaryInterceptor(reqIDInjector.interceptUnary)),
		)
		allOpts := allClientOpts(config.NumChannels, config.Compression, config.EnableDirectAccess, opts...)
		endpointClientOpts = append(endpointClientOpts, allOpts...)
		primaryConn, err = gtransport.DialPool(ctx, allOpts...)
		if err != nil {
			return nil, err
		}

		fallbackConnOpts := append(allOpts, internaloption.EnableDirectPath(false))
		fallbackConn, err = gtransport.DialPool(ctx, fallbackConnOpts...)
		if err != nil {
			primaryConn.Close()
			return nil, err
		}

		if hasNumChannelsConfig && ((primaryConn.Num() != config.NumChannels) || (fallbackConn.Num() != config.NumChannels)) {
			primaryConn.Close()
			fallbackConn.Close()
			return nil, spannerErrorf(codes.InvalidArgument, "Connection pool mismatch: NumChannels=%v, primaryConn.Num()=%v, fallbackConn.Num()=%v", config.NumChannels, primaryConn.Num(), fallbackConn.Num())
		}

		fbOpts := grpcgcp.NewGCPFallbackOptions()
		fbOpts.EnableFallback = true
		fbOpts.ErrorRateThreshold = 1
		fbOpts.MinFailedCalls = 1

		if metricsTracerFactory != nil && metricsTracerFactory.meterProvider != nil {
			fbOpts.MeterProvider = metricsTracerFactory.meterProvider
		}

		gcpFallback, err := grpcgcp.NewGCPFallback(ctx, primaryConn, fallbackConn, fbOpts)
		if err != nil {
			primaryConn.Close()
			fallbackConn.Close()
			return nil, err
		}

		pool = &fallbackWrapper{gcpFallback, primaryConn, fallbackConn}
	} else {
		// Create gtransport ConnPool as usual if MultiEndpoint is not used.
		// gRPC options.

		// Add a unaryClientInterceptor and streamClientInterceptor.
		reqIDInjector := new(requestIDHeaderInjector)
		opts = append(opts,
			option.WithGRPCDialOption(grpc.WithChainStreamInterceptor(reqIDInjector.interceptStream)),
			option.WithGRPCDialOption(grpc.WithChainUnaryInterceptor(reqIDInjector.interceptUnary)),
		)

		allOpts := allClientOpts(config.NumChannels, config.Compression, config.EnableDirectAccess, opts...)
		endpointClientOpts = append(endpointClientOpts, allOpts...)
		pool, err = gtransport.DialPool(ctx, allOpts...)
		if err != nil {
			return nil, err
		}

		if hasNumChannelsConfig && pool.Num() != config.NumChannels {
			pool.Close()
			return nil, spannerErrorf(codes.InvalidArgument, "Connection pool mismatch: NumChannels=%v, WithGRPCConnectionPool=%v. Only set one of these options, or set both to the same value.", config.NumChannels, pool.Num())
		}
	}

	// TODO(loite): Remove as the original map cannot be changed by the user
	// anyways, and the client library is also not changing it.
	// Make a copy of labels.
	sessionLabels := make(map[string]string)
	for k, v := range config.SessionLabels {
		sessionLabels[k] = v
	}

	// Default configs for session pool.
	if config.MaxOpened == 0 {
		config.MaxOpened = uint64(pool.Num() * 100)
	}
	if config.MaxBurst == 0 {
		config.MaxBurst = DefaultSessionPoolConfig.MaxBurst
	}
	if config.BatchTimeout == 0 {
		config.BatchTimeout = time.Minute
	}

	md := metadata.Pairs(resourcePrefixHeader, database)
	if config.Compression == gzip.Name {
		md.Append(requestsCompressionHeader, gzip.Name)
	}
	// Append end to end tracing header if SPANNER_ENABLE_END_TO_END_TRACING
	// environment variable has been set or client has passed the opt-in
	// option in ClientConfig.
	endToEndTracingEnvironmentVariable := os.Getenv("SPANNER_ENABLE_END_TO_END_TRACING")
	if config.EnableEndToEndTracing || endToEndTracingEnvironmentVariable == "true" {
		md.Append(endToEndTracingHeader, "true")
	}

	if isAFEBuiltInMetricEnabled {
		md.Append(afeMetricHeader, "true")
	}

	// Multiplexed sessions are always enabled as the session pool has been removed.

	// Create a session client.
	sc := newSessionClient(pool, database, config.UserAgent, sessionLabels, config.DatabaseRole, config.DisableRouteToLeader, md, config.BatchTimeout, config.Logger, config.CallOptions)

	// Create an OpenTelemetry configuration
	otConfig, err := createOpenTelemetryConfig(ctx, config.OpenTelemetryMeterProvider, config.Logger, sc.id, database)
	if err != nil {
		// The error returned here will be due to database name parsing
		return nil, err
	}
	// To prevent data race in unit tests (ex: TestClient_SessionNotFound)
	sc.mu.Lock()
	sc.otConfig = otConfig
	sc.metricsTracerFactory = metricsTracerFactory
	sc.mu.Unlock()

	var locationRouter *locationRouter
	var sharedLocationAwareState *locationAwareState
	if isExperimentalLocationAPIEnabledForConfig(config) {
		sc.baseClientOpts = endpointClientOpts
		defaultEndpointAddress := ""
		if conn := pool.Conn(); conn != nil {
			sc.endpointAuthority = normalizeAuthorityTarget(conn.Target())
			defaultEndpointAddress = sc.endpointAuthority
		}
		// Some transport wrappers, such as GCPMultiEndpoint and GCPFallback,
		// intentionally do not expose a concrete default *grpc.ClientConn and
		// return nil from Conn(). In that case location-aware routing remains
		// enabled; we only skip deriving the default endpoint metadata used for
		// authority preservation and default-endpoint diagnostics.
		epCache := newEndpointClientCacheWithDefaultAddress(sc.createEndpointClient, defaultEndpointAddress)
		locationRouter = newLocationRouter(epCache)
		locationRouter.lifecycleManager = newEndpointLifecycleManager(epCache)
		locationRouter.finder.setLifecycleManager(locationRouter.lifecycleManager)
		sharedLocationAwareState = newLocationAwareState(
			nil,
			locationRouter,
			epCache,
			newEndpointOverloadCooldownTracker(),
		)
	}

	// Create a session manager.
	sp, err := newSessionManager(sc, config.SessionPoolConfig, sharedLocationAwareState)
	if err != nil {
		sc.close()
		return nil, err
	}

	if enableLogClientOptions() {
		projectID, _, _, _ := parseDatabaseName(database)
		logf(config.Logger, `
-----Client Options--------
Project ID: %v
Num of gRPC channels: %v
LAR enabled: %v
Directpath enabled: %v
End To End Tracing enabled: %v
Built-in metrics enabled: %v
gRPC metrics enabled: %v
Multiplexed session enabled: true
-----------------------------`,
			projectID, config.NumChannels, !config.DisableRouteToLeader, isDirectPathEnabled,
			config.EnableEndToEndTracing, !config.DisableNativeMetrics, isGRPCBuiltInMetricsEnabled)
	}
	c = &Client{
		sc:                   sc,
		sm:                   sp,
		logger:               config.Logger,
		qo:                   getQueryOptions(config.QueryOptions),
		ro:                   config.ReadOptions,
		ao:                   config.ApplyOptions,
		txo:                  config.TransactionOptions,
		bwo:                  config.BatchWriteOptions,
		ct:                   getCommonTags(sc),
		disableRouteToLeader: config.DisableRouteToLeader,
		dro:                  config.DirectedReadOptions,
		otConfig:             otConfig,
		metricsTracerFactory: metricsTracerFactory,
		clientContext:        config.ClientContext,
		locationRouter:       locationRouter,
	}
	return c, nil
}

func normalizeAuthorityTarget(target string) string {
	if idx := strings.Index(target, ":///"); idx >= 0 {
		return strings.TrimSuffix(target[idx+4:], "/")
	}
	if idx := strings.Index(target, "://"); idx >= 0 {
		return strings.TrimSuffix(target[idx+3:], "/")
	}
	return strings.TrimSuffix(target, "/")
}

// NewMultiEndpointClient is the same as NewMultiEndpointClientWithConfig with
// the default client configuration.
//
// A valid database name has the
// form projects/PROJECT_ID/instances/INSTANCE_ID/databases/DATABASE_ID.
func NewMultiEndpointClient(ctx context.Context, database string, gmeCfg *grpcgcp.GCPMultiEndpointOptions, opts ...option.ClientOption) (*Client, *grpcgcp.GCPMultiEndpoint, error) {
	return NewMultiEndpointClientWithConfig(ctx, database, ClientConfig{SessionPoolConfig: DefaultSessionPoolConfig, DisableRouteToLeader: false}, gmeCfg, opts...)
}

// NewMultiEndpointClientWithConfig creates a client to a database using GCPMultiEndpoint.
//
// The purposes of GCPMultiEndpoint are:
//
//   - Fallback to an alternative endpoint (host:port) when the original
//     endpoint is completely unavailable.
//   - Be able to route a Cloud Spanner call to a specific group of endpoints.
//   - Be able to reconfigure endpoints in runtime.
//
// The GRPCgcpConfig and DialFunc in the GCPMultiEndpointOptions are optional
// and will be configured automatically.
//
// For GCPMultiEndpoint the number of channels is configured via MaxSize of the
// ChannelPool config in the GRPCgcpConfig.
//
// The GCPMultiEndpoint returned can be used to update the endpoints in runtime.
//
// A valid database name has the
// form projects/PROJECT_ID/instances/INSTANCE_ID/databases/DATABASE_ID.
func NewMultiEndpointClientWithConfig(ctx context.Context, database string, config ClientConfig, gmeCfg *grpcgcp.GCPMultiEndpointOptions, opts ...option.ClientOption) (c *Client, gme *grpcgcp.GCPMultiEndpoint, err error) {
	gme, err = createGCPMultiEndpoint(gmeCfg, config, opts...)
	if err != nil {
		return nil, nil, err
	}
	// Align number of channels.
	config.NumChannels = int(gme.GCPConfig().GetChannelPool().GetMaxSize())
	c, err = newClientWithConfig(ctx, database, config, gme, opts...)
	if err != nil {
		return nil, nil, err
	}
	return
}

// Combines the default options from the generated client, the default options
// of the hand-written client and the user options to one list of options.
// Precedence: userOpts > clientDefaultOpts > generatedDefaultOpts
func allClientOpts(numChannels int, compression string, enableDirectAccess bool, userOpts ...option.ClientOption) []option.ClientOption {
	generatedDefaultOpts := vkit.DefaultClientOptions()
	clientDefaultOpts := []option.ClientOption{
		option.WithGRPCConnectionPool(numChannels),
		option.WithUserAgent(fmt.Sprintf("spanner-go/v%s", internal.Version)),
		option.WithGRPCDialOption(grpc.WithChainUnaryInterceptor(addNativeMetricsInterceptor()...)),
		option.WithGRPCDialOption(grpc.WithChainStreamInterceptor(addStreamNativeMetricsInterceptor()...)),
		option.WithGRPCDialOption(grpc.WithKeepaliveParams(keepalive.ClientParameters{Time: 120 * time.Second})),
	}
	enableDirectPathXds := enableDirectAccess
	if enableDirectPathXdsString := os.Getenv("GOOGLE_SPANNER_ENABLE_DIRECT_ACCESS"); enableDirectPathXdsString != "" {
		enableDirectPathXds, _ = strconv.ParseBool(enableDirectPathXdsString)
	}

	if enableDirectPathXds {
		clientDefaultOpts = append(clientDefaultOpts, internaloption.AllowNonDefaultServiceAccount(true))
		clientDefaultOpts = append(clientDefaultOpts, internaloption.EnableDirectPath(true), internaloption.EnableDirectPathXds())
		if disableBoundToken, _ := strconv.ParseBool(os.Getenv("GOOGLE_SPANNER_DISABLE_DIRECT_ACCESS_BOUND_TOKEN")); !disableBoundToken {
			clientDefaultOpts = append(clientDefaultOpts, internaloption.AllowHardBoundTokens("ALTS"))
		}
	}
	if compression == "gzip" {
		userOpts = append(userOpts, option.WithGRPCDialOption(grpc.WithDefaultCallOptions(
			grpc.UseCompressor(gzip.Name))))
	}
	allDefaultOpts := append(generatedDefaultOpts, clientDefaultOpts...)
	return append(allDefaultOpts, userOpts...)
}

// metricsInterceptor is a gRPC unary client interceptor that records metrics for unary RPCs.
func metricsInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req interface{},
		reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		mt, ok := ctx.Value(metricsTracerKey).(*builtinMetricsTracer)
		if !ok {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		mt.method = method
		mt.currOp.incrementAttemptCount()
		mt.currOp.currAttempt = &attemptTracer{}
		mt.currOp.currAttempt.setStartTime(time.Now())
		if strings.HasPrefix(cc.Target(), "google-c2p") {
			mt.currOp.setDirectPathEnabled(true)
		}

		var md metadata.MD
		peerInfo := &peer.Peer{}
		opts = append(opts, grpc.Header(&md), grpc.Peer(peerInfo))
		err := invoker(ctx, method, req, reply, cc, opts...)

		statusCode, _ := status.FromError(err)
		mt.currOp.currAttempt.setStatus(statusCode.Code().String())
		mt.currOp.currAttempt.setDirectPathUsed(peer.NewContext(ctx, peerInfo))
		latencies := parseServerTimingHeader(md)
		span := otrace.SpanFromContext(ctx)
		setGFEAndAFESpanAttributes(span, latencies)
		mt.currOp.currAttempt.setServerTimingMetrics(latencies)
		recordAttemptCompletion(mt)
		return err
	}
}

// metricsInterceptor is a gRPC stream client interceptor that records metrics for stream RPCs.
func metricsStreamInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		s, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}
		mt, ok := ctx.Value(metricsTracerKey).(*builtinMetricsTracer)
		if ok && mt != nil {
			mt.method = method
			if strings.HasPrefix(cc.Target(), "google-c2p") {
				mt.currOp.setDirectPathEnabled(true)
			}
		}
		return s, nil
	}
}

// AddNativeMetricsInterceptor intercepts unary requests and records metrics for them.
func addNativeMetricsInterceptor() []grpc.UnaryClientInterceptor {
	unaryInterceptors := []grpc.UnaryClientInterceptor{}
	unaryInterceptors = append(unaryInterceptors, metricsInterceptor())
	return unaryInterceptors
}

// AddStreamNativeMetricsInterceptor intercepts stream requests and records metrics for them.
func addStreamNativeMetricsInterceptor() []grpc.StreamClientInterceptor {
	streamInterceptors := []grpc.StreamClientInterceptor{}
	streamInterceptors = append(streamInterceptors, metricsStreamInterceptor())
	return streamInterceptors
}

// getQueryOptions returns the query options overwritten by the environment
// variables if exist. The input parameter is the query options set by users
// via application-level configuration. If the environment variables are set,
// this will return the overwritten query options.
func getQueryOptions(opts QueryOptions) QueryOptions {
	if opts.Options == nil {
		opts.Options = &sppb.ExecuteSqlRequest_QueryOptions{}
	}
	opv := os.Getenv("SPANNER_OPTIMIZER_VERSION")
	if opv != "" {
		opts.Options.OptimizerVersion = opv
	}
	opsp := os.Getenv("SPANNER_OPTIMIZER_STATISTICS_PACKAGE")
	if opsp != "" {
		opts.Options.OptimizerStatisticsPackage = opsp
	}
	return opts
}

// Close closes the client.
func (c *Client) Close() {
	if c.metricsTracerFactory != nil {
		c.metricsTracerFactory.shutdown(context.Background())
	}
	if c.sm != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.sm.close(ctx)
	}
	if c.locationRouter != nil {
		c.locationRouter.Close()
	}
	c.sc.close()
}

// Single provides a read-only snapshot transaction optimized for the case
// where only a single read or query is needed.  This is more efficient than
// using ReadOnlyTransaction() for a single read or query.
//
// Single will use a strong TimestampBound by default. Use
// ReadOnlyTransaction.WithTimestampBound to specify a different
// TimestampBound. A non-strong bound can be used to reduce latency, or
// "time-travel" to prior versions of the database, see the documentation of
// TimestampBound for details.
func (c *Client) Single() *ReadOnlyTransaction {
	t := &ReadOnlyTransaction{singleUse: true}
	t.txReadOnly.sm = c.sm
	t.txReadOnly.txReadEnv = t
	t.txReadOnly.qo = c.qo
	t.txReadOnly.ro = c.ro
	t.txReadOnly.disableRouteToLeader = true
	t.txReadOnly.qo.DirectedReadOptions = c.dro
	t.txReadOnly.ro.DirectedReadOptions = c.dro
	t.txReadOnly.ro.LockHint = sppb.ReadRequest_LOCK_HINT_UNSPECIFIED
	t.txReadOnly.clientContext = c.clientContext

	t.ct = c.ct
	t.otConfig = c.otConfig
	return t
}

// ReadOnlyTransaction returns a ReadOnlyTransaction that can be used for
// multiple reads from the database.  You must call Close() when the
// ReadOnlyTransaction is no longer needed to release resources on the server.
//
// ReadOnlyTransaction will use a strong TimestampBound by default.  Use
// ReadOnlyTransaction.WithTimestampBound to specify a different
// TimestampBound.  A non-strong bound can be used to reduce latency, or
// "time-travel" to prior versions of the database, see the documentation of
// TimestampBound for details.
func (c *Client) ReadOnlyTransaction() *ReadOnlyTransaction {
	t := &ReadOnlyTransaction{
		singleUse:       false,
		txReadyOrClosed: make(chan struct{}),
	}
	t.txReadOnly.sm = c.sm
	t.txReadOnly.txReadEnv = t
	t.txReadOnly.qo = c.qo
	t.txReadOnly.ro = c.ro
	t.txReadOnly.disableRouteToLeader = true
	t.txReadOnly.qo.DirectedReadOptions = c.dro
	t.txReadOnly.ro.DirectedReadOptions = c.dro
	t.txReadOnly.ro.LockHint = sppb.ReadRequest_LOCK_HINT_UNSPECIFIED
	t.txReadOnly.clientContext = c.clientContext

	t.ct = c.ct
	t.otConfig = c.otConfig
	return t
}

// BatchReadOnlyTransaction returns a BatchReadOnlyTransaction that can be used
// for partitioned reads or queries from a snapshot of the database. This is
// useful in batch processing pipelines where one wants to divide the work of
// reading from the database across multiple machines.
//
// Note: This transaction does not use the underlying session pool but creates a
// new session each time, and the session is reused across clients.
//
// You should call Close() after the txn is no longer needed on local
// client, and call Cleanup() when the txn is finished for all clients, to free
// the session.
func (c *Client) BatchReadOnlyTransaction(ctx context.Context, tb TimestampBound) (*BatchReadOnlyTransaction, error) {
	var (
		tx  transactionID
		rts time.Time
		sh  *sessionHandle
		err error
	)

	// Always use multiplexed sessions for batch read operations.
	sh, err = c.sm.takeMultiplexed(ctx)
	if err != nil {
		return nil, err
	}

	// Begin transaction.
	res, err := sh.getClient().BeginTransaction(contextWithOutgoingMetadata(ctx, sh.getMetadata(), true), &sppb.BeginTransactionRequest{
		Session: sh.getID(),
		Options: &sppb.TransactionOptions{
			Mode: &sppb.TransactionOptions_ReadOnly_{
				ReadOnly: buildTransactionOptionsReadOnly(tb, true),
			},
		},
		RequestOptions: createRequestOptions(sppb.RequestOptions_PRIORITY_UNSPECIFIED, "", "", c.clientContext),
	})
	if err != nil {
		return nil, ToSpannerError(err)
	}
	tx = res.Id
	if res.ReadTimestamp != nil {
		rts = time.Unix(res.ReadTimestamp.Seconds, int64(res.ReadTimestamp.Nanos))
	}

	t := &BatchReadOnlyTransaction{
		ReadOnlyTransaction: ReadOnlyTransaction{
			tx:                       tx,
			txReadyOrClosed:          make(chan struct{}),
			state:                    txActive,
			rts:                      rts,
			isLongRunningTransaction: true,
		},
		ID: BatchReadOnlyTransactionID{
			tid: tx,
			sid: sh.getID(),
			rts: rts,
		},
	}
	t.txReadOnly.sm = c.sm
	t.txReadOnly.sh = sh
	t.txReadOnly.txReadEnv = t
	t.txReadOnly.qo = c.qo
	t.txReadOnly.ro = c.ro
	t.txReadOnly.disableRouteToLeader = true
	t.txReadOnly.qo.DirectedReadOptions = c.dro
	t.txReadOnly.ro.DirectedReadOptions = c.dro
	t.txReadOnly.ro.LockHint = sppb.ReadRequest_LOCK_HINT_UNSPECIFIED
	t.txReadOnly.clientContext = c.clientContext

	t.ct = c.ct
	t.otConfig = c.otConfig
	return t, nil
}

// BatchReadOnlyTransactionFromID reconstruct a BatchReadOnlyTransaction from
// BatchReadOnlyTransactionID
func (c *Client) BatchReadOnlyTransactionFromID(tid BatchReadOnlyTransactionID) *BatchReadOnlyTransaction {
	s, err := c.sc.sessionWithID(tid.sid)
	if err != nil {
		logf(c.logger, "unexpected error: %v\nThis is an indication of an internal error in the Spanner client library.", err)
		// Use an invalid session. Preferably, this method should just return
		// the error instead of this, but that would mean an API change.
		s = &session{}
	}
	sh := &sessionHandle{session: s}

	t := &BatchReadOnlyTransaction{
		ReadOnlyTransaction: ReadOnlyTransaction{
			tx:                       tid.tid,
			txReadyOrClosed:          make(chan struct{}),
			state:                    txActive,
			rts:                      tid.rts,
			isLongRunningTransaction: true,
		},
		ID: tid,
	}
	t.txReadOnly.sm = c.sm
	t.txReadOnly.sh = sh
	t.txReadOnly.txReadEnv = t
	t.txReadOnly.qo = c.qo
	t.txReadOnly.ro = c.ro
	t.txReadOnly.disableRouteToLeader = true
	t.txReadOnly.qo.DirectedReadOptions = c.dro
	t.txReadOnly.ro.DirectedReadOptions = c.dro
	t.txReadOnly.ro.LockHint = sppb.ReadRequest_LOCK_HINT_UNSPECIFIED
	t.txReadOnly.clientContext = c.clientContext

	t.ct = c.ct
	t.otConfig = c.otConfig
	return t
}

type transactionInProgressKey struct{}

func checkNestedTxn(ctx context.Context) error {
	if ctx.Value(transactionInProgressKey{}) != nil {
		return spannerErrorf(codes.FailedPrecondition, "Cloud Spanner does not support nested transactions")
	}
	return nil
}

// ReadWriteTransaction executes a read-write transaction, with retries as
// necessary.
//
// The function f will be called one or more times. It must not maintain
// any state between calls.
//
// If the transaction cannot be committed or if f returns an ABORTED error,
// ReadWriteTransaction will call f again. It will continue to call f until the
// transaction can be committed or the Context times out or is cancelled.  If f
// returns an error other than ABORTED, ReadWriteTransaction will abort the
// transaction and return the error.
//
// To limit the number of retries, set a deadline on the Context rather than
// using a fixed limit on the number of attempts. ReadWriteTransaction will
// retry as needed until that deadline is met.
//
// See https://godoc.org/cloud.google.com/go/spanner#ReadWriteTransaction for
// more details.
func (c *Client) ReadWriteTransaction(ctx context.Context, f func(context.Context, *ReadWriteTransaction) error) (commitTimestamp time.Time, err error) {
	ctx, _ = startSpan(ctx, "ReadWriteTransaction", c.otConfig.commonTraceStartOptions...)
	defer func() { endSpan(ctx, err) }()
	resp, err := c.rwTransaction(ctx, f, TransactionOptions{})
	return resp.CommitTs, err
}

// ReadWriteTransactionWithOptions executes a read-write transaction with
// configurable options, with retries as necessary.
//
// ReadWriteTransactionWithOptions is a configurable ReadWriteTransaction.
//
// See https://godoc.org/cloud.google.com/go/spanner#ReadWriteTransaction for
// more details.
func (c *Client) ReadWriteTransactionWithOptions(ctx context.Context, f func(context.Context, *ReadWriteTransaction) error, options TransactionOptions) (resp CommitResponse, err error) {
	ctx, _ = startSpan(ctx, "ReadWriteTransactionWithOptions", c.otConfig.commonTraceStartOptions...)
	defer func() { endSpan(ctx, err) }()
	resp, err = c.rwTransaction(ctx, f, options)
	return resp, err
}

func (c *Client) rwTransaction(ctx context.Context, f func(context.Context, *ReadWriteTransaction) error, options TransactionOptions) (resp CommitResponse, err error) {
	if err := checkNestedTxn(ctx); err != nil {
		return resp, err
	}
	var (
		sh      *sessionHandle
		t       *ReadWriteTransaction
		attempt = 0
	)
	defer func() {
		if sh != nil {
			sh.recycle()
		}
	}()
	err = runWithRetryOnAbortedOrFailedInlineBegin(ctx, func(ctx context.Context) error {
		var (
			err error
		)
		if sh == nil || sh.getID() == "" || sh.getClient() == nil {
			sh, err = c.sm.takeMultiplexed(ctx)
			if err != nil {
				return err
			}
		}
		initTx := func(t *ReadWriteTransaction) {
			t.txReadOnly.sm = c.sm
			t.txReadOnly.txReadEnv = t
			t.txReadOnly.qo = c.qo
			t.txReadOnly.ro = c.ro
			t.txReadOnly.disableRouteToLeader = c.disableRouteToLeader
			t.txReadOnly.clientContext = c.clientContext
			t.wb = []*Mutation{}
			t.txOpts = c.txo.merge(options)
			t.txReadOnly.clientContext = mergeClientContext(c.clientContext, t.txOpts.ClientContext)

			t.ct = c.ct
			t.otConfig = c.otConfig
		}
		if t.shouldExplicitBegin(attempt, options) {
			if t == nil {
				t = &ReadWriteTransaction{
					txReadyOrClosed: make(chan struct{}),
				}
			}
			initTx(t)
			// Make sure we set the current session handle before calling BeginTransaction.
			// Note that the t.begin(ctx) call could change the session that is being used by the transaction, as the
			// BeginTransaction RPC invocation will be retried on a new session if it returns SessionNotFound.
			t.txReadOnly.sh = sh
			if err = t.begin(ctx, nil); err != nil {
				if attempt > 0 {
					trace.TracePrintf(ctx, nil, "Error while BeginTransaction during retrying a ReadWrite transaction: %v", ToSpannerError(err))
				} else {
					trace.TracePrintf(ctx, nil, "Error during the initial BeginTransaction for a ReadWrite transaction: %v", ToSpannerError(err))
				}
				return ToSpannerError(err)
			}
		} else {
			var previousTx transactionID
			if t != nil {
				previousTx = t.previousTx
			}
			t = &ReadWriteTransaction{
				txReadyOrClosed: make(chan struct{}),
				previousTx:      previousTx,
			}
			t.txReadOnly.sh = sh
			initTx(t)
		}
		attempt++

		trace.TracePrintf(ctx, map[string]interface{}{"transactionSelector": t.getTransactionSelector().String()},
			"Starting transaction attempt")

		resp, err = t.runInTransaction(ctx, f)
		return err
	})
	return resp, err
}

// applyOption controls the behavior of Client.Apply.
type applyOption struct {
	// If atLeastOnce == true, Client.Apply will execute the mutations on Cloud
	// Spanner at least once.
	atLeastOnce bool
	// transactionTag will be included with the CommitRequest.
	transactionTag string
	// priority is the RPC priority that is used for the commit operation.
	priority sppb.RequestOptions_Priority
	// If excludeTxnFromChangeStreams == true, mutations from this Client.Apply
	// will not be recorded in allowed tracking change streams with DDL option
	// allow_txn_exclusion=true.
	excludeTxnFromChangeStreams bool
	// commitOptions is the commit options to use for the commit operation.
	commitOptions CommitOptions
	// It defines the isolationLevel for the RW transactions
	isolationLevel sppb.TransactionOptions_IsolationLevel
}

// An ApplyOption is an optional argument to Apply.
type ApplyOption func(*applyOption)

// ApplyAtLeastOnce returns an ApplyOption that removes replay protection.
//
// With this option, Apply may attempt to apply mutations more than once; if
// the mutations are not idempotent, this may lead to a failure being reported
// when the mutation was applied more than once. For example, an insert may
// fail with ALREADY_EXISTS even though the row did not exist before Apply was
// called. For this reason, most users of the library will prefer not to use
// this option.  However, ApplyAtLeastOnce requires only a single RPC, whereas
// Apply's default replay protection may require an additional RPC.  So this
// option may be appropriate for latency sensitive and/or high throughput blind
// writing.
func ApplyAtLeastOnce() ApplyOption {
	return func(ao *applyOption) {
		ao.atLeastOnce = true
	}
}

// TransactionTag returns an ApplyOption that will include the given tag as a
// transaction tag for a write-only transaction.
func TransactionTag(tag string) ApplyOption {
	return func(ao *applyOption) {
		ao.transactionTag = tag
	}
}

// Priority returns an ApplyOptions that sets the RPC priority to use for the
// commit operation.
func Priority(priority sppb.RequestOptions_Priority) ApplyOption {
	return func(ao *applyOption) {
		ao.priority = priority
	}
}

// ExcludeTxnFromChangeStreams returns an ApplyOptions that sets whether to exclude recording this commit operation from allowed tracking change streams.
func ExcludeTxnFromChangeStreams() ApplyOption {
	return func(ao *applyOption) {
		ao.excludeTxnFromChangeStreams = true
	}
}

// IsolationLevel returns an ApplyOptions that sets which isolationLevel for RW transaction
func IsolationLevel(isolationLevel sppb.TransactionOptions_IsolationLevel) ApplyOption {
	return func(ao *applyOption) {
		ao.isolationLevel = isolationLevel
	}
}

// ApplyCommitOptions returns an ApplyOption that sets the commit options to use for the commit operation.
func ApplyCommitOptions(co CommitOptions) ApplyOption {
	return func(ao *applyOption) {
		ao.commitOptions = co
	}
}

// Apply applies a list of mutations atomically to the database.
func (c *Client) Apply(ctx context.Context, ms []*Mutation, opts ...ApplyOption) (commitTimestamp time.Time, err error) {
	ao := &applyOption{}

	for _, opt := range c.ao {
		opt(ao)
	}

	for _, opt := range opts {
		opt(ao)
	}

	ctx, _ = startSpan(ctx, "Apply", c.otConfig.commonTraceStartOptions...)
	defer func() { endSpan(ctx, err) }()

	if !ao.atLeastOnce {
		resp, err := c.ReadWriteTransactionWithOptions(ctx, func(ctx context.Context, t *ReadWriteTransaction) error {
			return t.BufferWrite(ms)
		}, TransactionOptions{CommitPriority: ao.priority, TransactionTag: ao.transactionTag, ExcludeTxnFromChangeStreams: ao.excludeTxnFromChangeStreams, CommitOptions: ao.commitOptions, IsolationLevel: ao.isolationLevel})
		return resp.CommitTs, err
	}
	t := &writeOnlyTransaction{sm: c.sm, commitPriority: ao.priority, transactionTag: ao.transactionTag, disableRouteToLeader: c.disableRouteToLeader, excludeTxnFromChangeStreams: ao.excludeTxnFromChangeStreams, commitOptions: ao.commitOptions, isolationLevel: ao.isolationLevel, clientContext: c.clientContext}
	return t.applyAtLeastOnce(ctx, ms...)
}

// BatchWriteOptions provides options for a BatchWriteRequest.
type BatchWriteOptions struct {
	// Priority is the RPC priority to use for this request.
	Priority sppb.RequestOptions_Priority

	// The transaction tag to use for this request.
	TransactionTag string

	// If excludeTxnFromChangeStreams == true, modifications from all transactions
	// in this batch write request will not be recorded in allowed tracking
	// change treams with DDL option allow_txn_exclusion=true.
	ExcludeTxnFromChangeStreams bool

	// ClientContext contains client-owned context information to be passed with the batch write request.
	ClientContext *sppb.RequestOptions_ClientContext
}

// merge combines two BatchWriteOptions such that the input parameter will have higher
// order of precedence.
func (bwo BatchWriteOptions) merge(opts BatchWriteOptions) BatchWriteOptions {
	merged := BatchWriteOptions{
		TransactionTag:              bwo.TransactionTag,
		Priority:                    bwo.Priority,
		ExcludeTxnFromChangeStreams: bwo.ExcludeTxnFromChangeStreams || opts.ExcludeTxnFromChangeStreams,
		ClientContext:               mergeClientContext(bwo.ClientContext, opts.ClientContext),
	}
	if opts.TransactionTag != "" {
		merged.TransactionTag = opts.TransactionTag
	}
	if opts.Priority != sppb.RequestOptions_PRIORITY_UNSPECIFIED {
		merged.Priority = opts.Priority
	}
	return merged
}

// BatchWriteResponseIterator is an iterator over BatchWriteResponse structures returned from BatchWrite RPC.
type BatchWriteResponseIterator struct {
	ctx                context.Context
	stream             sppb.Spanner_BatchWriteClient
	err                error
	meterTracerFactory *builtinMetricsTracerFactory
	rpc                func(ctx context.Context) (sppb.Spanner_BatchWriteClient, error)
	release            func(error)
	cancel             func()
}

// Next returns the next result. Its second return value is iterator.Done if
// there are no more results. Once Next returns Done, all subsequent calls
// will return Done.
func (r *BatchWriteResponseIterator) Next() (*sppb.BatchWriteResponse, error) {
	mt := r.meterTracerFactory.createBuiltinMetricsTracer(r.ctx)
	defer func() {
		if mt.method != "" {
			statusCode, _ := convertToGrpcStatusErr(r.err)
			mt.currOp.setStatus(statusCode.String())
			recordOperationCompletion(&mt)
		}
	}()
	for {
		// Stream finished or in error state.
		if r.err != nil {
			return nil, r.err
		}

		// RPC not made yet.
		if r.stream == nil {
			r.stream, r.err = r.rpc(r.ctx)
			continue
		}

		// Read from the stream.
		var response *sppb.BatchWriteResponse
		response, r.err = r.stream.Recv()

		// Return an item.
		if r.err == nil {
			return response, nil
		}

		// Stream finished.
		if r.err == io.EOF {
			r.err = iterator.Done
			return nil, r.err
		}
	}
}

// Stop terminates the iteration. It should be called after you finish using the
// iterator.
func (r *BatchWriteResponseIterator) Stop() {
	if r.stream != nil {
		err := r.err
		if err == iterator.Done {
			err = nil
		}
		defer trace.EndSpan(r.ctx, err)
	}
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	if r.release != nil {
		r.release(r.err)
		r.release = nil
	}
	if r.err == nil {
		r.err = spannerErrorf(codes.FailedPrecondition, "Next called after Stop")
	}
}

// Do calls the provided function once in sequence for each item in the
// iteration. If the function returns a non-nil error, Do immediately returns
// that error.
//
// If there are no items in the iterator, Do will return nil without calling the
// provided function.
//
// Do always calls Stop on the iterator.
func (r *BatchWriteResponseIterator) Do(f func(r *sppb.BatchWriteResponse) error) error {
	defer r.Stop()
	for {
		row, err := r.Next()
		switch err {
		case iterator.Done:
			return nil
		case nil:
			if err = f(row); err != nil {
				return err
			}
		default:
			return err
		}
	}
}

// BatchWrite applies a list of mutation groups in a collection of efficient
// transactions. The mutation groups are applied non-atomically in an
// unspecified order and thus, they must be independent of each other. Partial
// failure is possible, i.e., some mutation groups may have been applied
// successfully, while some may have failed. The results of individual batches
// are streamed into the response as the batches are applied.
//
// BatchWrite requests are not replay protected, meaning that each mutation
// group may be applied more than once. Replays of non-idempotent mutations
// may have undesirable effects. For example, replays of an insert mutation
// may produce an already exists error or if you use generated or commit
// timestamp-based keys, it may result in additional rows being added to the
// mutation's table. We recommend structuring your mutation groups to be
// idempotent to avoid this issue.
func (c *Client) BatchWrite(ctx context.Context, mgs []*MutationGroup) *BatchWriteResponseIterator {
	return c.BatchWriteWithOptions(ctx, mgs, BatchWriteOptions{})
}

// BatchWriteWithOptions is same as BatchWrite. It accepts additional options to customize the request.
func (c *Client) BatchWriteWithOptions(ctx context.Context, mgs []*MutationGroup, opts BatchWriteOptions) *BatchWriteResponseIterator {
	ctx, _ = startSpan(ctx, "BatchWrite", c.otConfig.commonTraceStartOptions...)

	var err error
	defer func() {
		trace.EndSpan(ctx, err)
	}()

	opts = c.bwo.merge(opts)

	mgsPb, err := mutationGroupsProto(mgs)
	if err != nil {
		return &BatchWriteResponseIterator{meterTracerFactory: c.metricsTracerFactory, err: err}
	}

	var sh *sessionHandle
	sh, err = c.sm.takeMultiplexed(ctx)
	if err != nil {
		return &BatchWriteResponseIterator{meterTracerFactory: c.metricsTracerFactory, err: err}
	}

	rpc := func(ct context.Context) (sppb.Spanner_BatchWriteClient, error) {
		var md metadata.MD
		stream, rpcErr := sh.getClient().BatchWrite(contextWithOutgoingMetadata(ct, sh.getMetadata(), c.disableRouteToLeader), &sppb.BatchWriteRequest{
			Session:                     sh.getID(),
			MutationGroups:              mgsPb,
			RequestOptions:              createRequestOptions(opts.Priority, "", opts.TransactionTag, mergeClientContext(c.clientContext, opts.ClientContext)),
			ExcludeTxnFromChangeStreams: opts.ExcludeTxnFromChangeStreams,
		}, gax.WithGRPCOptions(grpc.Header(&md)))

		if getGFELatencyMetricsFlag() && md != nil && c.ct != nil {
			if metricErr := createContextAndCaptureGFELatencyMetrics(ct, c.ct, md, "BatchWrite"); metricErr != nil {
				trace.TracePrintf(ct, nil, "Error in recording GFE Latency. Try disabling and rerunning. Error: %v", err)
			}
		}
		if metricErr := recordGFELatencyMetricsOT(ct, md, "BatchWrite", c.otConfig); metricErr != nil {
			trace.TracePrintf(ct, nil, "Error in recording GFE Latency through OpenTelemetry. Error: %v", err)
		}
		return stream, rpcErr
	}

	release := func(err error) {
		if sh == nil {
			return
		}
		sh.recycle()
	}

	ctx, cancel := context.WithCancel(ctx)
	ctx, _ = startSpan(ctx, "BatchWriteResponseIterator", c.otConfig.commonTraceStartOptions...)
	return &BatchWriteResponseIterator{
		ctx:                ctx,
		meterTracerFactory: c.metricsTracerFactory,
		rpc:                rpc,
		release:            release,
		cancel:             cancel,
	}
}

// logf logs the given message to the given logger, or the standard logger if
// the given logger is nil.
func logf(logger *log.Logger, format string, v ...interface{}) {
	if logger == nil {
		log.Printf(format, v...)
	} else {
		logger.Printf(format, v...)
	}
}

// parseServerTimingHeader extracts server timing metrics from gRPC metadata into a map
func parseServerTimingHeader(md metadata.MD) map[string]time.Duration {
	metrics := make(map[string]time.Duration)
	if md == nil {
		return metrics
	}

	serverTiming := md.Get(serverTimingHeaderKey)
	if len(serverTiming) == 0 {
		return metrics
	}

	for _, timing := range serverTiming {
		matches := serverTimingPattern.FindAllStringSubmatch(timing, -1)
		for _, match := range matches {
			if len(match) == 3 { // full match + 2 capture groups
				metricName := match[1]
				duration, err := strconv.ParseFloat(match[2], 64)
				if err == nil {
					metrics[metricName] = time.Duration(duration*1000) * time.Microsecond
				}
			}
		}
	}
	return metrics
}

// enableLogClientOptions returns true if the environment variable for enabling has been set to true, or if the
// environment variable for disabling has been set to false. It returns false by default if no env var has been set.
// The function uses two environment variables because this function was initially added with a default return value of
// true, which caused the config to always be logged. This again caused unnecessary log spamming.
func enableLogClientOptions() bool {
	if enableLogString, found := os.LookupEnv("GOOGLE_CLOUD_SPANNER_ENABLE_LOG_CLIENT_OPTIONS"); found {
		if enableLog, err := strconv.ParseBool(enableLogString); err == nil {
			return enableLog
		}
	}
	if disableLogString, found := os.LookupEnv("GOOGLE_CLOUD_SPANNER_DISABLE_LOG_CLIENT_OPTIONS"); found {
		if disableLog, err := strconv.ParseBool(disableLogString); err == nil {
			return !disableLog
		}
	}
	return false
}
