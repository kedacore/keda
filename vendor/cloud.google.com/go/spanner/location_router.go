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
	"os"
	"strconv"
	"sync"

	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
)

const experimentalLocationAPIEnvVar = "GOOGLE_SPANNER_EXPERIMENTAL_LOCATION_API"

type locationRouter struct {
	finder           *channelFinder
	endpointCache    channelEndpointCache
	lifecycleManager *endpointLifecycleManager

	affinityMu                      sync.RWMutex
	transactionAffinity             map[string]channelEndpoint
	readOnlyTransactionPreferLeader map[string]bool
}

func isExperimentalLocationAPIEnabled() bool {
	return isExperimentalLocationAPIEnabledForConfig(ClientConfig{})
}

func isExperimentalLocationAPIEnabledForConfig(config ClientConfig) bool {
	locationAPIEnvValue := os.Getenv(experimentalLocationAPIEnvVar)
	if locationAPIEnvValue != "" {
		enabled, _ := strconv.ParseBool(locationAPIEnvValue)
		return enabled
	}
	return config.IsExperimentalHost
}

func newLocationRouter(endpointCache channelEndpointCache) *locationRouter {
	if endpointCache == nil {
		endpointCache = newPassthroughChannelEndpointCache()
	}
	return &locationRouter{
		finder:                          newChannelFinder(endpointCache),
		endpointCache:                   endpointCache,
		transactionAffinity:             make(map[string]channelEndpoint),
		readOnlyTransactionPreferLeader: make(map[string]bool),
	}
}

func (r *locationRouter) prepareReadRequest(ctx context.Context, req *sppb.ReadRequest) channelEndpoint {
	return r.prepareReadRequestWithCooldownTracker(ctx, req, nil)
}

func (r *locationRouter) prepareReadRequestWithCooldownTracker(ctx context.Context, req *sppb.ReadRequest, cooldowns *endpointOverloadCooldownTracker) channelEndpoint {
	if r == nil || req == nil {
		return nil
	}
	if txID := transactionIDFromSelector(req.GetTransaction()); txID != "" {
		if preferLeader, ok := r.getReadOnlyTransactionPreferLeader(txID); ok {
			return r.finder.findServerReadWithCooldownTracker(ctx, req, preferLeader, cooldowns)
		}
		if ep := r.getTransactionAffinity(txID); ep != nil && !isEndpointCoolingDown(cooldowns, ep.Address()) {
			return ep
		}
	}
	return r.finder.findServerReadWithCooldownTracker(ctx, req, preferLeaderFromSelector(req.GetTransaction()), cooldowns)
}

func (r *locationRouter) prepareExecuteSQLRequest(ctx context.Context, req *sppb.ExecuteSqlRequest) channelEndpoint {
	return r.prepareExecuteSQLRequestWithCooldownTracker(ctx, req, nil)
}

func (r *locationRouter) prepareExecuteSQLRequestWithCooldownTracker(ctx context.Context, req *sppb.ExecuteSqlRequest, cooldowns *endpointOverloadCooldownTracker) channelEndpoint {
	if r == nil || req == nil {
		return nil
	}
	if txID := transactionIDFromSelector(req.GetTransaction()); txID != "" {
		if preferLeader, ok := r.getReadOnlyTransactionPreferLeader(txID); ok {
			return r.finder.findServerExecuteSQLWithCooldownTracker(ctx, req, preferLeader, cooldowns)
		}
		if ep := r.getTransactionAffinity(txID); ep != nil && !isEndpointCoolingDown(cooldowns, ep.Address()) {
			return ep
		}
	}
	return r.finder.findServerExecuteSQLWithCooldownTracker(ctx, req, preferLeaderFromSelector(req.GetTransaction()), cooldowns)
}

func (r *locationRouter) prepareBeginTransactionRequest(ctx context.Context, req *sppb.BeginTransactionRequest) channelEndpoint {
	return r.prepareBeginTransactionRequestWithCooldownTracker(ctx, req, nil)
}

func (r *locationRouter) prepareBeginTransactionRequestWithCooldownTracker(ctx context.Context, req *sppb.BeginTransactionRequest, cooldowns *endpointOverloadCooldownTracker) channelEndpoint {
	if r == nil || req == nil {
		return nil
	}
	return r.finder.findServerBeginTransactionWithCooldownTracker(ctx, req, cooldowns)
}

func (r *locationRouter) prepareCommitRequest(ctx context.Context, req *sppb.CommitRequest) channelEndpoint {
	return r.prepareCommitRequestWithCooldownTracker(ctx, req, nil)
}

func (r *locationRouter) prepareCommitRequestWithCooldownTracker(ctx context.Context, req *sppb.CommitRequest, cooldowns *endpointOverloadCooldownTracker) channelEndpoint {
	if r == nil || req == nil {
		return nil
	}
	return r.finder.fillCommitRoutingHintWithCooldownTracker(ctx, req, cooldowns)
}

func (r *locationRouter) observePartialResultSet(prs *sppb.PartialResultSet) {
	if r == nil || prs == nil || prs.GetCacheUpdate() == nil {
		return
	}
	r.finder.updateAsync(prs.GetCacheUpdate())
}

func (r *locationRouter) observeResultSet(rs *sppb.ResultSet) {
	if r == nil || rs == nil || rs.GetCacheUpdate() == nil {
		return
	}
	r.finder.updateAsync(rs.GetCacheUpdate())
}

func (r *locationRouter) observeTransaction(tx *sppb.Transaction) {
	if r == nil || tx == nil || tx.GetCacheUpdate() == nil {
		return
	}
	r.finder.updateAsync(tx.GetCacheUpdate())
}

func (r *locationRouter) observeCommitResponse(resp *sppb.CommitResponse) {
	if r == nil || resp == nil || resp.GetCacheUpdate() == nil {
		return
	}
	r.finder.updateAsync(resp.GetCacheUpdate())
}

func (r *locationRouter) setTransactionAffinity(txID string, ep channelEndpoint) {
	if r == nil || txID == "" || ep == nil {
		return
	}
	r.affinityMu.Lock()
	defer r.affinityMu.Unlock()
	r.transactionAffinity[txID] = ep
}

func (r *locationRouter) trackReadOnlyTransaction(txID string, preferLeader bool) {
	if r == nil || txID == "" {
		return
	}
	r.affinityMu.Lock()
	defer r.affinityMu.Unlock()
	r.readOnlyTransactionPreferLeader[txID] = preferLeader
}

func (r *locationRouter) getReadOnlyTransactionPreferLeader(txID string) (bool, bool) {
	if r == nil || txID == "" {
		return false, false
	}
	r.affinityMu.RLock()
	defer r.affinityMu.RUnlock()
	preferLeader, ok := r.readOnlyTransactionPreferLeader[txID]
	return preferLeader, ok
}

func (r *locationRouter) isReadOnlyTransaction(txID string) bool {
	_, ok := r.getReadOnlyTransactionPreferLeader(txID)
	return ok
}

func (r *locationRouter) getTransactionAffinity(txID string) channelEndpoint {
	if r == nil || txID == "" {
		return nil
	}
	r.affinityMu.RLock()
	defer r.affinityMu.RUnlock()
	return r.transactionAffinity[txID]
}

func (r *locationRouter) clearTransactionAffinity(txID string) {
	if r == nil || txID == "" {
		return
	}
	r.affinityMu.Lock()
	defer r.affinityMu.Unlock()
	delete(r.transactionAffinity, txID)
	delete(r.readOnlyTransactionPreferLeader, txID)
}

func (r *locationRouter) Close() error {
	if r == nil {
		return nil
	}
	if r.lifecycleManager != nil {
		r.lifecycleManager.shutdown()
	}
	if r.endpointCache == nil {
		return nil
	}
	return r.endpointCache.Close()
}

func transactionIDFromSelector(selector *sppb.TransactionSelector) string {
	if selector == nil {
		return ""
	}
	if id := selector.GetId(); len(id) > 0 {
		return string(id)
	}
	return ""
}
