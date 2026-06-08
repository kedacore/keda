/*
Copyright 2026 Google LLC

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
	"sync"
	"time"

	vkit "cloud.google.com/go/spanner/apiv1"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var suppressResourceExhaustedRetryOption = newSuppressRetryCodesOption(codes.ResourceExhausted)
var suppressEndpointRetryOptions = newSuppressRetryCodesOption(codes.ResourceExhausted, codes.Unavailable)

type locationAwareState struct {
	clientPool              []spannerClient
	router                  *locationRouter
	endpointCache           channelEndpointCache
	defaultAffinityEndpoint channelEndpoint
	defaultEndpointAddress  string
	endpointCooldowns       *endpointOverloadCooldownTracker
}

func newLocationAwareState(
	clientPool []spannerClient,
	router *locationRouter,
	endpointCache channelEndpointCache,
	endpointCooldowns *endpointOverloadCooldownTracker,
) *locationAwareState {
	var defaultAffinityEndpoint channelEndpoint = &passthroughChannelEndpoint{address: ""}
	if endpointCache != nil && endpointCache.DefaultChannel() != nil {
		defaultAffinityEndpoint = endpointCache.DefaultChannel()
	}
	if endpointCooldowns == nil {
		endpointCooldowns = newEndpointOverloadCooldownTracker()
	}
	return &locationAwareState{
		clientPool:              clientPool,
		router:                  router,
		endpointCache:           endpointCache,
		defaultAffinityEndpoint: defaultAffinityEndpoint,
		defaultEndpointAddress:  defaultAffinityEndpoint.Address(),
		endpointCooldowns:       endpointCooldowns,
	}
}

func (s *locationAwareState) defaultClient(idx int) spannerClient {
	if s == nil || idx < 0 || idx >= len(s.clientPool) {
		return nil
	}
	return s.clientPool[idx]
}

// locationAwareSpannerClient is a thin spannerClient adapter that routes RPCs
// using shared client-level location-aware state while preserving the chosen
// default pooled client for the current request.
type locationAwareSpannerClient struct {
	state                   *locationAwareState
	defaultClientIndex      int
	defaultClient           spannerClient
	router                  *locationRouter
	endpointCache           channelEndpointCache
	defaultAffinityEndpoint channelEndpoint
	defaultEndpointAddress  string
	endpointCooldowns       *endpointOverloadCooldownTracker
}

var _ spannerClient = (*locationAwareSpannerClient)(nil)

// asGRPCSpannerClient extracts the underlying *grpcSpannerClient from a
// spannerClient, handling the locationAwareSpannerClient wrapper.
func asGRPCSpannerClient(c spannerClient) *grpcSpannerClient {
	if gsc, ok := c.(*grpcSpannerClient); ok {
		return gsc
	}
	if lac, ok := c.(*locationAwareSpannerClient); ok {
		return asGRPCSpannerClient(lac.defaultClient)
	}
	return nil
}

func newLocationAwareSpannerClient(defaultClient spannerClient, router *locationRouter, endpointCache channelEndpointCache) *locationAwareSpannerClient {
	return newIndexedLocationAwareSpannerClient(
		newLocationAwareState([]spannerClient{defaultClient}, router, endpointCache, nil),
		0,
	)
}

func newIndexedLocationAwareSpannerClient(state *locationAwareState, defaultClientIndex int) *locationAwareSpannerClient {
	if state == nil {
		return &locationAwareSpannerClient{defaultClientIndex: defaultClientIndex}
	}
	defaultClient := state.defaultClient(defaultClientIndex)
	return &locationAwareSpannerClient{
		state:                   state,
		defaultClientIndex:      defaultClientIndex,
		defaultClient:           defaultClient,
		router:                  state.router,
		endpointCache:           state.endpointCache,
		defaultAffinityEndpoint: state.defaultAffinityEndpoint,
		defaultEndpointAddress:  state.defaultEndpointAddress,
		endpointCooldowns:       state.endpointCooldowns,
	}
}

func (c *locationAwareSpannerClient) affinityTrackingEndpoint(ep channelEndpoint) channelEndpoint {
	if ep != nil {
		return ep
	}
	return c.defaultAffinityEndpoint
}

func (c *locationAwareSpannerClient) onRequestRouted(ep channelEndpoint) {
	if c == nil || c.router == nil || c.router.lifecycleManager == nil || ep == nil || !ep.IsHealthy() {
		return
	}
	c.router.lifecycleManager.recordRealTraffic(ep.Address())
}

func (c *locationAwareSpannerClient) maybeMarkEndpointCoolingDown(ep channelEndpoint, err error) {
	if c == nil || ep == nil {
		return
	}
	if !shouldCooldownEndpointOnRetry(status.Code(err)) {
		return
	}
	if ep.Address() == c.defaultEndpointAddress {
		return
	}
	if c.endpointCooldowns != nil {
		c.endpointCooldowns.recordFailure(ep.Address())
	}
}

func shouldCooldownEndpointOnRetry(code codes.Code) bool {
	return code == codes.ResourceExhausted || code == codes.Unavailable
}

func (c *locationAwareSpannerClient) maybeRecordEndpointErrorPenalty(ep channelEndpoint, operationUID uint64, preferLeader bool, err error) {
	if c == nil || ep == nil || operationUID == 0 {
		return
	}
	if !shouldCooldownEndpointOnRetry(status.Code(err)) || ep.Address() == c.defaultEndpointAddress {
		return
	}
	endpointLatencyRegistryRecordError(operationUID, preferLeader, ep.Address())
}

func (c *locationAwareSpannerClient) recordEndpointLatency(ep channelEndpoint, operationUID uint64, preferLeader bool, startedAt time.Time) {
	if c == nil || ep == nil || operationUID == 0 || ep.Address() == c.defaultEndpointAddress {
		return
	}
	endpointLatencyRegistryRecordLatency(operationUID, preferLeader, ep.Address(), time.Since(startedAt))
}

func (c *locationAwareSpannerClient) rerouteErrorMarker(ep channelEndpoint, operationUID uint64, preferLeader bool) func(error) {
	var once sync.Once
	return func(err error) {
		if !shouldCooldownEndpointOnRetry(status.Code(err)) {
			return
		}
		once.Do(func() {
			c.maybeMarkEndpointCoolingDown(ep, err)
			c.maybeRecordEndpointErrorPenalty(ep, operationUID, preferLeader, err)
		})
	}
}

func (c *locationAwareSpannerClient) reroutedCallOptions(base []gax.CallOption, logicalRequestKey string, attempt uint32, mark func(error), suppressUnavailable bool) []gax.CallOption {
	extraOptions := 2
	if logicalRequestKey != "" && attempt > 1 {
		extraOptions++
	}
	opts := make([]gax.CallOption, 0, len(base)+extraOptions)
	opts = append(opts, base...)
	if logicalRequestKey != "" && attempt > 1 {
		opts = append(opts, logicalRequestIDWrap{logicalKey: logicalRequestKey}.withNextRetryAttempt(attempt))
	}
	opts = append(opts, resourceExhaustedMarkerOption{mark: mark})
	if suppressUnavailable {
		opts = append(opts, suppressEndpointRetryOptions)
	} else {
		opts = append(opts, suppressResourceExhaustedRetryOption)
	}
	return opts
}

func (c *locationAwareSpannerClient) maybeWaitForReroute(ctx context.Context, ep channelEndpoint, lastRoutedAddress string) (bool, error) {
	if c == nil || lastRoutedAddress == "" {
		return false, nil
	}
	if c.endpointCooldowns == nil {
		return false, nil
	}
	if ep != nil && (ep.Address() == "" || ep.Address() != lastRoutedAddress) {
		return false, nil
	}
	wait := c.endpointCooldowns.remainingCooldown(lastRoutedAddress)
	if wait <= 0 {
		return false, nil
	}
	if err := gax.Sleep(ctx, wait); err != nil {
		return true, err
	}
	return true, nil
}

func (c *locationAwareSpannerClient) observeExecuteSQLResponse(req *spannerpb.ExecuteSqlRequest, resp *spannerpb.ResultSet, ep channelEndpoint) {
	c.router.observeResultSet(resp)
	if txMeta := resp.GetMetadata().GetTransaction(); txMeta != nil && len(txMeta.GetId()) > 0 {
		if isReadOnlyBegin, readOnlyStrong := readOnlyBeginFromSelector(req.GetTransaction()); isReadOnlyBegin {
			c.router.trackReadOnlyTransaction(string(txMeta.GetId()), readOnlyStrong)
		} else if isReadWriteBeginFromSelector(req.GetTransaction()) {
			c.router.setTransactionAffinity(string(txMeta.GetId()), c.affinityTrackingEndpoint(ep))
		}
	}
}

func (c *locationAwareSpannerClient) observeReadResponse(req *spannerpb.ReadRequest, resp *spannerpb.ResultSet, ep channelEndpoint) {
	c.router.observeResultSet(resp)
	if txMeta := resp.GetMetadata().GetTransaction(); txMeta != nil && len(txMeta.GetId()) > 0 {
		if isReadOnlyBegin, readOnlyStrong := readOnlyBeginFromSelector(req.GetTransaction()); isReadOnlyBegin {
			c.router.trackReadOnlyTransaction(string(txMeta.GetId()), readOnlyStrong)
		} else if isReadWriteBeginFromSelector(req.GetTransaction()) {
			c.router.setTransactionAffinity(string(txMeta.GetId()), c.affinityTrackingEndpoint(ep))
		}
	}
}

func (c *locationAwareSpannerClient) observeBeginTransactionResponse(req *spannerpb.BeginTransactionRequest, resp *spannerpb.Transaction, ep channelEndpoint) {
	c.router.observeTransaction(resp)
	if len(resp.GetId()) > 0 {
		if isReadOnly, readOnlyStrong := readOnlyBeginFromTransactionOptions(req.GetOptions()); isReadOnly {
			c.router.trackReadOnlyTransaction(string(resp.GetId()), readOnlyStrong)
		} else {
			c.router.setTransactionAffinity(string(resp.GetId()), c.affinityTrackingEndpoint(ep))
		}
	}
}

// clientForEndpoint resolves a channelEndpoint to a spannerClient, falling
// back to the default client if the endpoint is nil, unhealthy, or has no
// associated client.
func (c *locationAwareSpannerClient) clientForEndpoint(ep channelEndpoint) spannerClient {
	if ep == nil || !ep.IsHealthy() {
		return c.defaultClient
	}
	client := c.endpointCache.ClientFor(ep)
	if client == nil {
		return c.defaultClient
	}
	c.onRequestRouted(ep)
	return client
}

// affinityClient returns the spannerClient for a given transaction ID based on
// affinity, falling back to the default client.
func (c *locationAwareSpannerClient) affinityClient(txID []byte) spannerClient {
	return c.affinityClientWithCooldownTracker(txID, nil)
}

func (c *locationAwareSpannerClient) affinityEndpoint(txID []byte, cooldowns *endpointOverloadCooldownTracker) channelEndpoint {
	if len(txID) == 0 {
		return nil
	}
	ep := c.router.getTransactionAffinity(string(txID))
	if ep != nil && isEndpointCoolingDown(cooldowns, ep.Address()) {
		return nil
	}
	if ep != nil && !ep.IsHealthy() && c.router != nil && c.router.lifecycleManager != nil {
		c.router.lifecycleManager.requestEndpointRecreation(ep.Address())
	}
	if ep != nil && !ep.IsHealthy() {
		return nil
	}
	return ep
}

func (c *locationAwareSpannerClient) affinityClientWithCooldownTracker(txID []byte, cooldowns *endpointOverloadCooldownTracker) spannerClient {
	ep := c.affinityEndpoint(txID, cooldowns)
	return c.clientForEndpoint(ep)
}

// --- Pass-through methods ---

func (c *locationAwareSpannerClient) CallOptions() *vkit.CallOptions {
	return c.defaultClient.CallOptions()
}

func (c *locationAwareSpannerClient) Close() error {
	return nil
}

func (c *locationAwareSpannerClient) Connection() *grpc.ClientConn {
	return c.defaultClient.Connection()
}

func (c *locationAwareSpannerClient) CreateSession(ctx context.Context, req *spannerpb.CreateSessionRequest, opts ...gax.CallOption) (*spannerpb.Session, error) {
	return c.defaultClient.CreateSession(ctx, req, opts...)
}

func (c *locationAwareSpannerClient) BatchCreateSessions(ctx context.Context, req *spannerpb.BatchCreateSessionsRequest, opts ...gax.CallOption) (*spannerpb.BatchCreateSessionsResponse, error) {
	return c.defaultClient.BatchCreateSessions(ctx, req, opts...)
}

func (c *locationAwareSpannerClient) GetSession(ctx context.Context, req *spannerpb.GetSessionRequest, opts ...gax.CallOption) (*spannerpb.Session, error) {
	return c.defaultClient.GetSession(ctx, req, opts...)
}

func (c *locationAwareSpannerClient) ListSessions(ctx context.Context, req *spannerpb.ListSessionsRequest, opts ...gax.CallOption) *vkit.SessionIterator {
	return c.defaultClient.ListSessions(ctx, req, opts...)
}

func (c *locationAwareSpannerClient) DeleteSession(ctx context.Context, req *spannerpb.DeleteSessionRequest, opts ...gax.CallOption) error {
	return c.defaultClient.DeleteSession(ctx, req, opts...)
}

func (c *locationAwareSpannerClient) ExecuteBatchDml(ctx context.Context, req *spannerpb.ExecuteBatchDmlRequest, opts ...gax.CallOption) (*spannerpb.ExecuteBatchDmlResponse, error) {
	return c.defaultClient.ExecuteBatchDml(ctx, req, opts...)
}

func (c *locationAwareSpannerClient) PartitionQuery(ctx context.Context, req *spannerpb.PartitionQueryRequest, opts ...gax.CallOption) (*spannerpb.PartitionResponse, error) {
	return c.defaultClient.PartitionQuery(ctx, req, opts...)
}

func (c *locationAwareSpannerClient) PartitionRead(ctx context.Context, req *spannerpb.PartitionReadRequest, opts ...gax.CallOption) (*spannerpb.PartitionResponse, error) {
	return c.defaultClient.PartitionRead(ctx, req, opts...)
}

func (c *locationAwareSpannerClient) BatchWrite(ctx context.Context, req *spannerpb.BatchWriteRequest, opts ...gax.CallOption) (spannerpb.Spanner_BatchWriteClient, error) {
	return c.defaultClient.BatchWrite(ctx, req, opts...)
}

// --- Routed RPCs ---

func (c *locationAwareSpannerClient) StreamingRead(ctx context.Context, req *spannerpb.ReadRequest, opts ...gax.CallOption) (spannerpb.Spanner_StreamingReadClient, error) {
	logicalRequestKey := logicalRequestKeyFromCallOptions(opts)
	cooldowns := c.endpointCooldowns
	preferLeader := preferLeaderFromSelector(req.GetTransaction())
	lastRoutedAddress := ""
	for attempt := uint32(1); ; attempt++ {
		ep := c.router.prepareReadRequestWithCooldownTracker(ctx, req, cooldowns)
		operationUID := req.GetRoutingHint().GetOperationUid()
		if waited, err := c.maybeWaitForReroute(ctx, ep, lastRoutedAddress); err != nil {
			return nil, err
		} else if waited {
			continue
		}
		client := c.clientForEndpoint(ep)
		usedDefaultEndpoint := client == c.defaultClient
		markRetryableError := c.rerouteErrorMarker(ep, operationUID, preferLeader)
		currentOpts := c.reroutedCallOptions(opts, logicalRequestKey, attempt, markRetryableError, !usedDefaultEndpoint)
		if ep != nil {
			ep.IncrementActiveRequests()
		}
		startedAt := time.Now()
		stream, err := client.StreamingRead(ctx, req, currentOpts...)
		if err == nil {
			isReadOnlyBegin, readOnlyStrong := readOnlyBeginFromSelector(req.GetTransaction())
			return newAffinityTrackingStream(
				stream,
				c.router,
				c.affinityTrackingEndpoint(ep),
				isReadOnlyBegin,
				readOnlyStrong,
				isReadWriteBeginFromSelector(req.GetTransaction()),
				func() { c.recordEndpointLatency(ep, operationUID, preferLeader, startedAt) },
				markRetryableError,
				func() {
					if ep != nil {
						ep.DecrementActiveRequests()
					}
				},
			), nil
		}
		if ep != nil {
			ep.DecrementActiveRequests()
		}
		markRetryableError(err)
		if !shouldCooldownEndpointOnRetry(status.Code(err)) || ep == nil || usedDefaultEndpoint {
			return nil, err
		}
		lastRoutedAddress = ep.Address()
	}
}

func (c *locationAwareSpannerClient) Read(ctx context.Context, req *spannerpb.ReadRequest, opts ...gax.CallOption) (*spannerpb.ResultSet, error) {
	logicalRequestKey := logicalRequestKeyFromCallOptions(opts)
	cooldowns := c.endpointCooldowns
	preferLeader := preferLeaderFromSelector(req.GetTransaction())
	lastRoutedAddress := ""
	for attempt := uint32(1); ; attempt++ {
		ep := c.router.prepareReadRequestWithCooldownTracker(ctx, req, cooldowns)
		operationUID := req.GetRoutingHint().GetOperationUid()
		if waited, err := c.maybeWaitForReroute(ctx, ep, lastRoutedAddress); err != nil {
			return nil, err
		} else if waited {
			continue
		}
		client := c.clientForEndpoint(ep)
		usedDefaultEndpoint := client == c.defaultClient
		markRetryableError := c.rerouteErrorMarker(ep, operationUID, preferLeader)
		currentOpts := c.reroutedCallOptions(opts, logicalRequestKey, attempt, markRetryableError, !usedDefaultEndpoint)
		if ep != nil {
			ep.IncrementActiveRequests()
		}
		startedAt := time.Now()
		resp, err := client.Read(ctx, req, currentOpts...)
		if ep != nil {
			ep.DecrementActiveRequests()
		}
		if err == nil {
			c.recordEndpointLatency(ep, operationUID, preferLeader, startedAt)
			c.observeReadResponse(req, resp, ep)
			return resp, nil
		}
		markRetryableError(err)
		if !shouldCooldownEndpointOnRetry(status.Code(err)) || ep == nil || usedDefaultEndpoint {
			return nil, err
		}
		lastRoutedAddress = ep.Address()
	}
}

func (c *locationAwareSpannerClient) ExecuteStreamingSql(ctx context.Context, req *spannerpb.ExecuteSqlRequest, opts ...gax.CallOption) (spannerpb.Spanner_ExecuteStreamingSqlClient, error) {
	logicalRequestKey := logicalRequestKeyFromCallOptions(opts)
	cooldowns := c.endpointCooldowns
	preferLeader := preferLeaderFromSelector(req.GetTransaction())
	lastRoutedAddress := ""
	for attempt := uint32(1); ; attempt++ {
		ep := c.router.prepareExecuteSQLRequestWithCooldownTracker(ctx, req, cooldowns)
		operationUID := req.GetRoutingHint().GetOperationUid()
		if waited, err := c.maybeWaitForReroute(ctx, ep, lastRoutedAddress); err != nil {
			return nil, err
		} else if waited {
			continue
		}
		client := c.clientForEndpoint(ep)
		usedDefaultEndpoint := client == c.defaultClient
		markRetryableError := c.rerouteErrorMarker(ep, operationUID, preferLeader)
		currentOpts := c.reroutedCallOptions(opts, logicalRequestKey, attempt, markRetryableError, !usedDefaultEndpoint)
		if ep != nil {
			ep.IncrementActiveRequests()
		}
		startedAt := time.Now()
		stream, err := client.ExecuteStreamingSql(ctx, req, currentOpts...)
		if err == nil {
			isReadOnlyBegin, readOnlyStrong := readOnlyBeginFromSelector(req.GetTransaction())
			return newAffinityTrackingStream(
				stream,
				c.router,
				c.affinityTrackingEndpoint(ep),
				isReadOnlyBegin,
				readOnlyStrong,
				isReadWriteBeginFromSelector(req.GetTransaction()),
				func() { c.recordEndpointLatency(ep, operationUID, preferLeader, startedAt) },
				markRetryableError,
				func() {
					if ep != nil {
						ep.DecrementActiveRequests()
					}
				},
			), nil
		}
		if ep != nil {
			ep.DecrementActiveRequests()
		}
		markRetryableError(err)
		if !shouldCooldownEndpointOnRetry(status.Code(err)) || ep == nil || usedDefaultEndpoint {
			return nil, err
		}
		lastRoutedAddress = ep.Address()
	}
}

func (c *locationAwareSpannerClient) ExecuteSql(ctx context.Context, req *spannerpb.ExecuteSqlRequest, opts ...gax.CallOption) (*spannerpb.ResultSet, error) {
	logicalRequestKey := logicalRequestKeyFromCallOptions(opts)
	cooldowns := c.endpointCooldowns
	preferLeader := preferLeaderFromSelector(req.GetTransaction())
	lastRoutedAddress := ""
	for attempt := uint32(1); ; attempt++ {
		ep := c.router.prepareExecuteSQLRequestWithCooldownTracker(ctx, req, cooldowns)
		operationUID := req.GetRoutingHint().GetOperationUid()
		if waited, err := c.maybeWaitForReroute(ctx, ep, lastRoutedAddress); err != nil {
			return nil, err
		} else if waited {
			continue
		}
		client := c.clientForEndpoint(ep)
		usedDefaultEndpoint := client == c.defaultClient
		markRetryableError := c.rerouteErrorMarker(ep, operationUID, preferLeader)
		currentOpts := c.reroutedCallOptions(opts, logicalRequestKey, attempt, markRetryableError, !usedDefaultEndpoint)
		if ep != nil {
			ep.IncrementActiveRequests()
		}
		startedAt := time.Now()
		resp, err := client.ExecuteSql(ctx, req, currentOpts...)
		if ep != nil {
			ep.DecrementActiveRequests()
		}
		if err == nil {
			c.recordEndpointLatency(ep, operationUID, preferLeader, startedAt)
			c.observeExecuteSQLResponse(req, resp, ep)
			return resp, nil
		}
		markRetryableError(err)
		if !shouldCooldownEndpointOnRetry(status.Code(err)) || ep == nil || usedDefaultEndpoint {
			return nil, err
		}
		lastRoutedAddress = ep.Address()
	}
}

func (c *locationAwareSpannerClient) BeginTransaction(ctx context.Context, req *spannerpb.BeginTransactionRequest, opts ...gax.CallOption) (*spannerpb.Transaction, error) {
	logicalRequestKey := logicalRequestKeyFromCallOptions(opts)
	cooldowns := c.endpointCooldowns
	preferLeader := preferLeaderFromTransactionOptions(req.GetOptions())
	operationUID := req.GetRoutingHint().GetOperationUid()
	lastRoutedAddress := ""
	for attempt := uint32(1); ; attempt++ {
		ep := c.router.prepareBeginTransactionRequestWithCooldownTracker(ctx, req, cooldowns)
		if waited, err := c.maybeWaitForReroute(ctx, ep, lastRoutedAddress); err != nil {
			return nil, err
		} else if waited {
			continue
		}
		client := c.clientForEndpoint(ep)
		usedDefaultEndpoint := client == c.defaultClient
		markRetryableError := c.rerouteErrorMarker(ep, operationUID, preferLeader)
		currentOpts := c.reroutedCallOptions(opts, logicalRequestKey, attempt, markRetryableError, !usedDefaultEndpoint)
		if ep != nil {
			ep.IncrementActiveRequests()
		}
		startedAt := time.Now()
		resp, err := client.BeginTransaction(ctx, req, currentOpts...)
		if ep != nil {
			ep.DecrementActiveRequests()
		}
		if err == nil {
			c.recordEndpointLatency(ep, operationUID, preferLeader, startedAt)
			c.observeBeginTransactionResponse(req, resp, ep)
			return resp, nil
		}
		markRetryableError(err)
		if !shouldCooldownEndpointOnRetry(status.Code(err)) || ep == nil || usedDefaultEndpoint {
			return nil, err
		}
		lastRoutedAddress = ep.Address()
	}
}

// --- Affinity RPCs ---

func (c *locationAwareSpannerClient) Commit(ctx context.Context, req *spannerpb.CommitRequest, opts ...gax.CallOption) (*spannerpb.CommitResponse, error) {
	cooldowns := c.endpointCooldowns
	ep := c.router.prepareCommitRequestWithCooldownTracker(ctx, req, cooldowns)
	if txID := req.GetTransactionId(); len(txID) > 0 {
		if affinityEndpoint := c.affinityEndpoint(txID, cooldowns); affinityEndpoint != nil {
			ep = affinityEndpoint
		}
	}
	markRetryableError := c.rerouteErrorMarker(ep, 0, false)
	client := c.clientForEndpoint(ep)
	resp, err := client.Commit(ctx, req, appendResourceExhaustedMarkerOptions(opts, markRetryableError, true)...)
	markRetryableError(err)
	c.router.observeCommitResponse(resp)
	c.router.clearTransactionAffinity(string(req.GetTransactionId()))
	return resp, err
}

func (c *locationAwareSpannerClient) Rollback(ctx context.Context, req *spannerpb.RollbackRequest, opts ...gax.CallOption) error {
	ep := c.affinityEndpoint(req.GetTransactionId(), c.endpointCooldowns)
	markRetryableError := c.rerouteErrorMarker(ep, 0, false)
	client := c.clientForEndpoint(ep)
	err := client.Rollback(ctx, req, appendResourceExhaustedMarkerOptions(opts, markRetryableError, true)...)
	markRetryableError(err)
	c.router.clearTransactionAffinity(string(req.GetTransactionId()))
	return err
}

// affinityTrackingStream wraps a streaming RPC client to intercept Recv()
// calls and record transaction affinity from the first PartialResultSet that
// contains a transaction ID.
type affinityTrackingStream struct {
	grpc.ClientStream
	router             *locationRouter
	affinityEndpoint   channelEndpoint
	trackReadOnlyBegin bool
	readOnlyStrong     bool
	trackAffinity      bool
	once               sync.Once
	errorOnce          sync.Once
	latencyOnce        sync.Once
	doneOnce           sync.Once
	inner              streamingClient
	onFirstResponse    func()
	onError            func(error)
	onDone             func()
}

func (s *affinityTrackingStream) finish() {
	s.doneOnce.Do(func() {
		if s.onDone != nil {
			s.onDone()
		}
	})
}

// streamingClient is the shared interface implemented by both
// StreamingRead and ExecuteStreamingSql response streams.
type streamingClient interface {
	Recv() (*spannerpb.PartialResultSet, error)
	grpc.ClientStream
}

func newAffinityTrackingStream(
	inner streamingClient,
	router *locationRouter,
	affinityEndpoint channelEndpoint,
	trackReadOnlyBegin bool,
	readOnlyStrong bool,
	trackAffinity bool,
	onFirstResponse func(),
	onError func(error),
	onDone func(),
) *affinityTrackingStream {
	return &affinityTrackingStream{
		ClientStream:       inner,
		router:             router,
		affinityEndpoint:   affinityEndpoint,
		trackReadOnlyBegin: trackReadOnlyBegin,
		readOnlyStrong:     readOnlyStrong,
		trackAffinity:      trackAffinity,
		inner:              inner,
		onFirstResponse:    onFirstResponse,
		onError:            onError,
		onDone:             onDone,
	}
}

func (s *affinityTrackingStream) Recv() (*spannerpb.PartialResultSet, error) {
	prs, err := s.inner.Recv()
	if err != nil {
		s.finish()
		s.errorOnce.Do(func() {
			if s.onError != nil {
				s.onError(err)
			}
		})
		return nil, err
	}
	s.latencyOnce.Do(func() {
		if s.onFirstResponse != nil {
			s.onFirstResponse()
		}
	})
	// Record transaction metadata from the first PartialResultSet that contains
	// a transaction ID.
	if txMeta := prs.GetMetadata().GetTransaction(); txMeta != nil && len(txMeta.GetId()) > 0 {
		txID := string(txMeta.GetId())
		s.once.Do(func() {
			if s.trackReadOnlyBegin {
				s.router.trackReadOnlyTransaction(txID, s.readOnlyStrong)
				return
			}
			if s.trackAffinity {
				s.router.setTransactionAffinity(txID, s.affinityEndpoint)
			}
		})
	}
	// Observe cache updates from every PartialResultSet.
	s.router.observePartialResultSet(prs)
	return prs, nil
}

func readOnlyBeginFromSelector(selector *spannerpb.TransactionSelector) (bool, bool) {
	if selector == nil {
		return false, false
	}
	begin := selector.GetBegin()
	if begin == nil || begin.GetReadOnly() == nil {
		return false, false
	}
	return true, begin.GetReadOnly().GetStrong()
}

func isReadWriteBeginFromSelector(selector *spannerpb.TransactionSelector) bool {
	if selector == nil {
		return false
	}
	begin := selector.GetBegin()
	return begin != nil && begin.GetReadOnly() == nil
}

func readOnlyBeginFromTransactionOptions(options *spannerpb.TransactionOptions) (bool, bool) {
	if options == nil || options.GetReadOnly() == nil {
		return false, false
	}
	return true, options.GetReadOnly().GetStrong()
}
