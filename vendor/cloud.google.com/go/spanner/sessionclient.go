/*
Copyright 2019 Google LLC

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
	"log"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/internal/trace"
	vkit "cloud.google.com/go/spanner/apiv1"
	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"cloud.google.com/go/spanner/internal"
	"github.com/googleapis/gax-go/v2"
	"go.opencensus.io/tag"
	"google.golang.org/api/option"

	gtransport "google.golang.org/api/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

var cidGen = newClientIDGenerator()

const (
	routedKeepaliveTime    = 2 * time.Second
	routedKeepaliveTimeout = 20 * time.Second
)

type clientIDGenerator struct {
	mu  sync.Mutex
	ids map[string]int
}

func newClientIDGenerator() *clientIDGenerator {
	return &clientIDGenerator{ids: make(map[string]int)}
}

func (cg *clientIDGenerator) nextClientIDAndOrdinal(database string) (clientID string, nthClient int) {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	var id int
	if val, ok := cg.ids[database]; ok {
		id = val + 1
	} else {
		id = 1
	}
	cg.ids[database] = id
	return fmt.Sprintf("client-%d", id), id
}

func (cg *clientIDGenerator) nextID(database string) string {
	clientStrID, _ := cg.nextClientIDAndOrdinal(database)
	return clientStrID
}

// sessionConsumer is passed to the session creation methods and will receive
// the sessions that are created as they become available. A sessionConsumer
// implementation must be safe for concurrent use.
//
// The interface is implemented by sessionManager and is used for testing the
// sessionClient.
type sessionConsumer interface {
	// sessionReady is called when a session has been created and is ready for
	// use.
	sessionReady(ctx context.Context, s *session)

	// sessionCreationFailed is called when the creation of a session failed.
	sessionCreationFailed(ctx context.Context, err error)
}

// sessionClient creates sessions for a database. Each session will be
// affiliated with a gRPC channel. The session client now only supports
// creating multiplexed sessions.
type sessionClient struct {
	waitWorkers          sync.WaitGroup
	mu                   sync.Mutex
	closed               bool
	disableRouteToLeader bool

	connPool             gtransport.ConnPool
	database             string
	id                   string
	userAgent            string
	sessionLabels        map[string]string
	databaseRole         string
	md                   metadata.MD
	batchTimeout         time.Duration
	logger               *log.Logger
	callOptions          *vkit.CallOptions
	otConfig             *openTelemetryConfig
	metricsTracerFactory *builtinMetricsTracerFactory
	channelIDMap         map[*grpc.ClientConn]uint64

	// baseClientOpts holds the client options used for creating endpoint-specific
	// gRPC connections in location-aware routing.
	baseClientOpts []option.ClientOption
	// endpointAuthority preserves the default endpoint authority for routed
	// endpoint clients so TLS/SNI continues to use the original host identity.
	endpointAuthority string

	// These fields are for request-id propagation.
	nthClient int
	// nthRequest shall always be incremented on every fresh request.
	nthRequest *atomic.Uint32
}

// newSessionClient creates a session client to use for a database.
func newSessionClient(connPool gtransport.ConnPool, database, userAgent string, sessionLabels map[string]string, databaseRole string, disableRouteToLeader bool, md metadata.MD, batchTimeout time.Duration, logger *log.Logger, callOptions *vkit.CallOptions) *sessionClient {
	clientID, nthClient := cidGen.nextClientIDAndOrdinal(database)
	return &sessionClient{
		connPool:             connPool,
		database:             database,
		userAgent:            userAgent,
		id:                   clientID,
		sessionLabels:        sessionLabels,
		databaseRole:         databaseRole,
		disableRouteToLeader: disableRouteToLeader,
		md:                   md,
		batchTimeout:         batchTimeout,
		logger:               logger,
		callOptions:          callOptions,

		nthClient:  nthClient,
		nthRequest: new(atomic.Uint32),
	}
}

func (sc *sessionClient) close() error {
	defer sc.waitWorkers.Wait()

	var err error
	func() {
		sc.mu.Lock()
		defer sc.mu.Unlock()

		sc.closed = true
		err = sc.connPool.Close()
	}()
	return err
}

// createSession creates one session for the database of the sessionClient. The
// session is created using one synchronous RPC.
func (sc *sessionClient) createSession(ctx context.Context) (*session, error) {
	sc.mu.Lock()
	if sc.closed {
		sc.mu.Unlock()
		return nil, spannerErrorf(codes.FailedPrecondition, "SessionClient is closed")
	}
	sc.mu.Unlock()
	client, err := sc.nextClient()
	if err != nil {
		return nil, err
	}

	var md metadata.MD
	sid, err := client.CreateSession(contextWithOutgoingMetadata(ctx, sc.md, sc.disableRouteToLeader), &sppb.CreateSessionRequest{
		Database: sc.database,
		Session:  &sppb.Session{Labels: sc.sessionLabels, CreatorRole: sc.databaseRole},
	}, gax.WithGRPCOptions(grpc.Header(&md)))

	if getGFELatencyMetricsFlag() && md != nil {
		_, instance, database, err := parseDatabaseName(sc.database)
		if err != nil {
			return nil, ToSpannerError(err)
		}
		ctxGFE, err := tag.New(ctx,
			tag.Upsert(tagKeyClientID, sc.id),
			tag.Upsert(tagKeyDatabase, database),
			tag.Upsert(tagKeyInstance, instance),
			tag.Upsert(tagKeyLibVersion, internal.Version),
		)
		if err != nil {
			trace.TracePrintf(ctx, nil, "Error in recording GFE Latency. Try disabling and rerunning. Error: %v", ToSpannerError(err))
		}
		err = captureGFELatencyStats(ctxGFE, md, "createSession")
		if err != nil {
			trace.TracePrintf(ctx, nil, "Error in recording GFE Latency. Try disabling and rerunning. Error: %v", ToSpannerError(err))
		}
	}
	if metricErr := recordGFELatencyMetricsOT(ctx, md, "createSession", sc.otConfig); metricErr != nil {
		trace.TracePrintf(ctx, nil, "Error in recording GFE Latency through OpenTelemetry. Error: %v", metricErr)
	}
	if err != nil {
		return nil, ToSpannerError(err)
	}
	return &session{client: client, id: sid.Name, createTime: time.Now(), md: sc.md, logger: sc.logger}, nil
}

func (sc *sessionClient) executeCreateMultiplexedSession(ctx context.Context, client spannerClient, md metadata.MD, consumer sessionConsumer) {
	ctx, _ = startSpan(ctx, "CreateSession", sc.otConfig.commonTraceStartOptions...)
	defer func() { endSpan(ctx, nil) }()
	trace.TracePrintf(ctx, nil, "Creating a multiplexed session")
	sc.mu.Lock()
	closed := sc.closed
	sc.mu.Unlock()
	if closed {
		err := spannerErrorf(codes.Canceled, "Session client closed")
		trace.TracePrintf(ctx, nil, "Session client closed while creating a multiplexed session: %v", err)
		consumer.sessionCreationFailed(ctx, err)
		return
	}
	if ctx.Err() != nil {
		trace.TracePrintf(ctx, nil, "Context error while creating a multiplexed session: %v", ctx.Err())
		consumer.sessionCreationFailed(ctx, ToSpannerError(ctx.Err()))
		return
	}
	var mdForGFELatency metadata.MD
	response, err := client.CreateSession(contextWithOutgoingMetadata(ctx, sc.md, sc.disableRouteToLeader), &sppb.CreateSessionRequest{
		Database: sc.database,
		// Multiplexed sessions do not support labels.
		Session: &sppb.Session{CreatorRole: sc.databaseRole, Multiplexed: true},
	}, gax.WithGRPCOptions(grpc.Header(&mdForGFELatency)))

	if getGFELatencyMetricsFlag() && mdForGFELatency != nil {
		_, instance, database, err := parseDatabaseName(sc.database)
		if err != nil {
			trace.TracePrintf(ctx, nil, "Error getting instance and database name: %v", err)
		}
		// Errors should not prevent initializing the session pool.
		ctxGFE, err := tag.New(ctx,
			tag.Upsert(tagKeyClientID, sc.id),
			tag.Upsert(tagKeyDatabase, database),
			tag.Upsert(tagKeyInstance, instance),
			tag.Upsert(tagKeyLibVersion, internal.Version),
		)
		if err != nil {
			trace.TracePrintf(ctx, nil, "Error in adding tags in CreateSession for GFE Latency: %v", err)
		}
		err = captureGFELatencyStats(ctxGFE, mdForGFELatency, "executeCreateSession")
		if err != nil {
			trace.TracePrintf(ctx, nil, "Error in Capturing GFE Latency and Header Missing count. Try disabling and rerunning. Error: %v", err)
		}
	}
	if metricErr := recordGFELatencyMetricsOT(ctx, mdForGFELatency, "executeCreateSession", sc.otConfig); metricErr != nil {
		trace.TracePrintf(ctx, nil, "Error in recording GFE Latency through OpenTelemetry. Error: %v", metricErr)
	}
	if err != nil {
		trace.TracePrintf(ctx, nil, "Error creating a multiplexed sessions: %v", err)
		consumer.sessionCreationFailed(ctx, ToSpannerError(err))
		return
	}
	consumer.sessionReady(ctx, &session{client: client, id: response.Name, createTime: time.Now(), md: md, logger: sc.logger})
	trace.TracePrintf(ctx, nil, "Finished creating multiplexed sessions")
}

func (sc *sessionClient) sessionWithID(id string) (*session, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	client, err := sc.nextClient()
	if err != nil {
		return nil, err
	}
	return &session{client: client, id: id, createTime: time.Now(), md: sc.md, logger: sc.logger}, nil
}

// nextClient returns the next gRPC client to use for session creation. The
// client is set on the session, and used by all subsequent gRPC calls on the
// session. Using the same channel for all gRPC calls for a session ensures the
// optimal usage of server side caches.
func (sc *sessionClient) nextClient() (spannerClient, error) {
	var clientOpt option.ClientOption
	var channelID uint64
	if _, ok := sc.connPool.(*gmeWrapper); ok {
		// Pass GCPMultiEndpoint as a pool.
		clientOpt = gtransport.WithConnPool(sc.connPool)
	} else if _, ok := sc.connPool.(*fallbackWrapper); ok {
		clientOpt = gtransport.WithConnPool(sc.connPool)
	} else {
		// Pick a grpc.ClientConn from a regular pool.
		conn := sc.connPool.Conn()

		// Retrieve the channelID for each spannerClient.
		// It is assumed that this method is invoked
		// under a lock already.
		var ok bool
		channelID, ok = sc.channelIDMap[conn]
		if !ok {
			if sc.channelIDMap == nil {
				sc.channelIDMap = make(map[*grpc.ClientConn]uint64)
			}
			channelID = uint64(len(sc.channelIDMap)) + 1
			sc.channelIDMap[conn] = channelID
		}

		clientOpt = option.WithGRPCConn(conn)
	}
	client, err := newGRPCSpannerClient(context.Background(), sc, channelID, clientOpt)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// createEndpointClient creates a new spannerClient for a specific server endpoint
// address. This is used by the location-aware routing feature to create direct
// connections to Spanner servers.
func (sc *sessionClient) createEndpointClient(ctx context.Context, address string) (spannerClient, error) {
	opts := make([]option.ClientOption, len(sc.baseClientOpts))
	copy(opts, sc.baseClientOpts)
	if sc.endpointAuthority != "" {
		opts = append(opts, option.WithGRPCDialOption(grpc.WithAuthority(sc.endpointAuthority)))
	}
	opts = append(opts, option.WithEndpoint(address))
	// Routed endpoint clients should keep a single connection per endpoint so
	// bypass traffic does not fan out into the parent's broader pool sizing.
	opts = append(opts, option.WithGRPCConnectionPool(1))
	opts = append(opts, option.WithGRPCDialOption(grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:    routedKeepaliveTime,
		Timeout: routedKeepaliveTimeout,
	})))
	return newGRPCSpannerClient(ctx, sc, 0, opts...)
}

// mergeCallOptions merges two CallOptions into one and the first argument has
// a lower order of precedence than the second one.
func mergeCallOptions(a *vkit.CallOptions, b *vkit.CallOptions) *vkit.CallOptions {
	res := &vkit.CallOptions{}
	resVal := reflect.ValueOf(res).Elem()
	aVal := reflect.ValueOf(a).Elem()
	bVal := reflect.ValueOf(b).Elem()

	t := aVal.Type()

	for i := 0; i < aVal.NumField(); i++ {
		fieldName := t.Field(i).Name

		aFieldVal := aVal.Field(i).Interface().([]gax.CallOption)
		bFieldVal := bVal.Field(i).Interface().([]gax.CallOption)

		merged := append(aFieldVal, bFieldVal...)
		resVal.FieldByName(fieldName).Set(reflect.ValueOf(merged))
	}
	return res
}
