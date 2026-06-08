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
	"sync/atomic"
	"time"

	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type channelFinder struct {
	updateMu sync.Mutex

	databaseID  atomic.Uint64
	recipeCache *keyRecipeCache
	rangeCache  *keyRangeCache

	coalescingMu    sync.Mutex
	pendingUpdates  []*sppb.CacheUpdate
	flushScheduled  bool
	coalescingDelay time.Duration
	scheduleFlush   func(time.Duration, func())
}

const cacheUpdateCoalescingWindow = 5 * time.Millisecond

func defaultChannelFinderFlushScheduler(delay time.Duration, fn func()) {
	time.AfterFunc(delay, fn)
}

func newChannelFinder(endpointCache channelEndpointCache) *channelFinder {
	return &channelFinder{
		recipeCache:     newKeyRecipeCache(),
		rangeCache:      newKeyRangeCache(endpointCache),
		coalescingDelay: cacheUpdateCoalescingWindow,
		scheduleFlush:   defaultChannelFinderFlushScheduler,
	}
}

func (f *channelFinder) useDeterministicRandom() {
	f.rangeCache.useDeterministicRandom()
}

func (f *channelFinder) setLifecycleManager(lifecycleManager *endpointLifecycleManager) {
	if f == nil {
		return
	}
	f.rangeCache.setLifecycleManager(lifecycleManager)
}

func (f *channelFinder) setCoalescingDelayForTest(delay time.Duration) {
	if f == nil {
		return
	}
	f.coalescingMu.Lock()
	defer f.coalescingMu.Unlock()
	f.coalescingDelay = delay
}

func (f *channelFinder) setFlushSchedulerForTest(schedule func(time.Duration, func())) {
	if f == nil {
		return
	}
	if schedule == nil {
		schedule = defaultChannelFinderFlushScheduler
	}
	f.coalescingMu.Lock()
	defer f.coalescingMu.Unlock()
	f.scheduleFlush = schedule
}

func (f *channelFinder) update(update *sppb.CacheUpdate) {
	if update == nil {
		return
	}
	f.updateMu.Lock()
	defer f.updateMu.Unlock()
	f.applyUpdateLocked(update)
}

func (f *channelFinder) applyUpdates(updates []*sppb.CacheUpdate) {
	if len(updates) == 0 {
		return
	}
	f.updateMu.Lock()
	defer f.updateMu.Unlock()

	for _, update := range updates {
		f.applyUpdateLocked(update)
	}
}

func (f *channelFinder) applyUpdateLocked(update *sppb.CacheUpdate) {
	if update == nil {
		return
	}
	currentID := f.databaseID.Load()
	if currentID != update.GetDatabaseId() {
		if currentID != 0 {
			f.recipeCache.clear()
			f.rangeCache.clear()
		}
		f.databaseID.Store(update.GetDatabaseId())
	}
	if update.GetKeyRecipes() != nil {
		f.recipeCache.addRecipes(update.GetKeyRecipes())
	}
	f.rangeCache.addRanges(update)
}

func (f *channelFinder) updateAsync(update *sppb.CacheUpdate) {
	if !f.shouldProcessUpdate(update) {
		return
	}
	f.enqueueCoalescedUpdate(update)
}

func (f *channelFinder) shouldProcessUpdate(update *sppb.CacheUpdate) bool {
	if update == nil {
		return false
	}
	// Apply any material cache update and let applyUpdateLocked handle a database
	// switch by clearing stale state before storing the new database ID. For
	// database-ID-only messages, only process them when they indicate that this
	// finder has switched to a different database; database IDs are treated as
	// identity values, not an ordered sequence.
	if f.isMaterialUpdate(update) {
		return true
	}
	updateDatabaseID := update.GetDatabaseId()
	return updateDatabaseID != 0 && f.databaseID.Load() != updateDatabaseID
}

func (*channelFinder) isMaterialUpdate(update *sppb.CacheUpdate) bool {
	if update == nil {
		return false
	}
	return len(update.GetGroup()) > 0 ||
		len(update.GetRange()) > 0 ||
		(update.GetKeyRecipes() != nil && len(update.GetKeyRecipes().GetRecipe()) > 0)
}

func (f *channelFinder) enqueueCoalescedUpdate(update *sppb.CacheUpdate) {
	if f == nil || update == nil {
		return
	}

	f.coalescingMu.Lock()
	f.pendingUpdates = append(f.pendingUpdates, cloneCacheUpdate(update))
	if f.flushScheduled {
		f.coalescingMu.Unlock()
		return
	}
	f.flushScheduled = true
	delay := f.coalescingDelay
	scheduleFlush := f.scheduleFlush
	f.coalescingMu.Unlock()

	scheduleFlush(delay, f.flushCoalescedUpdates)
}

func (f *channelFinder) flushCoalescedUpdates() {
	if f == nil {
		return
	}
	f.coalescingMu.Lock()
	updates := f.pendingUpdates
	f.pendingUpdates = nil
	f.flushScheduled = false
	f.coalescingMu.Unlock()

	f.applyUpdates(updates)
}

func cloneCacheUpdate(update *sppb.CacheUpdate) *sppb.CacheUpdate {
	if update == nil {
		return nil
	}
	return cloneProto(update)
}

func cloneProto[M interface{ ProtoReflect() protoreflect.Message }](msg M) M {
	if any(msg) == nil {
		var zero M
		return zero
	}
	return proto.Clone(msg).(M)
}

func (f *channelFinder) findServerRead(ctx context.Context, req *sppb.ReadRequest, preferLeader bool) channelEndpoint {
	return f.findServerReadWithCooldownTracker(ctx, req, preferLeader, nil)
}

func (f *channelFinder) findServerReadWithCooldownTracker(ctx context.Context, req *sppb.ReadRequest, preferLeader bool, cooldowns *endpointOverloadCooldownTracker) channelEndpoint {
	if req == nil {
		return nil
	}
	f.recipeCache.computeReadKeys(req)
	hint := ensureReadRoutingHint(req)
	return f.fillRoutingHintWithCooldownTracker(ctx, preferLeader, rangeModeCoveringSplit, req.GetDirectedReadOptions(), hint, cooldowns)
}

func (f *channelFinder) findServerReadWithTransaction(ctx context.Context, req *sppb.ReadRequest) channelEndpoint {
	if req == nil {
		return nil
	}
	return f.findServerRead(ctx, req, preferLeaderFromSelector(req.GetTransaction()))
}

func (f *channelFinder) findServerExecuteSQL(ctx context.Context, req *sppb.ExecuteSqlRequest, preferLeader bool) channelEndpoint {
	return f.findServerExecuteSQLWithCooldownTracker(ctx, req, preferLeader, nil)
}

func (f *channelFinder) findServerExecuteSQLWithCooldownTracker(ctx context.Context, req *sppb.ExecuteSqlRequest, preferLeader bool, cooldowns *endpointOverloadCooldownTracker) channelEndpoint {
	if req == nil {
		return nil
	}
	f.recipeCache.computeQueryKeys(req)
	hint := ensureExecuteSQLRoutingHint(req)
	return f.fillRoutingHintWithCooldownTracker(ctx, preferLeader, rangeModePickRandom, req.GetDirectedReadOptions(), hint, cooldowns)
}

func (f *channelFinder) findServerExecuteSQLWithTransaction(ctx context.Context, req *sppb.ExecuteSqlRequest) channelEndpoint {
	if req == nil {
		return nil
	}
	return f.findServerExecuteSQL(ctx, req, preferLeaderFromSelector(req.GetTransaction()))
}

func (f *channelFinder) findServerBeginTransaction(ctx context.Context, req *sppb.BeginTransactionRequest) channelEndpoint {
	return f.findServerBeginTransactionWithCooldownTracker(ctx, req, nil)
}

func (f *channelFinder) findServerBeginTransactionWithCooldownTracker(ctx context.Context, req *sppb.BeginTransactionRequest, cooldowns *endpointOverloadCooldownTracker) channelEndpoint {
	if req == nil || req.GetMutationKey() == nil {
		return nil
	}
	return f.routeMutationWithCooldownTracker(ctx, req.GetMutationKey(), preferLeaderFromTransactionOptions(req.GetOptions()), ensureBeginTransactionRoutingHint(req), cooldowns)
}

func (f *channelFinder) fillCommitRoutingHint(ctx context.Context, req *sppb.CommitRequest) channelEndpoint {
	return f.fillCommitRoutingHintWithCooldownTracker(ctx, req, nil)
}

func (f *channelFinder) fillCommitRoutingHintWithCooldownTracker(ctx context.Context, req *sppb.CommitRequest, cooldowns *endpointOverloadCooldownTracker) channelEndpoint {
	if req == nil {
		return nil
	}
	mutation := selectMutationProtoForRouting(req.GetMutations())
	if mutation == nil {
		return nil
	}
	return f.routeMutationWithCooldownTracker(ctx, mutation, true, ensureCommitRoutingHint(req), cooldowns)
}

func (f *channelFinder) routeMutation(ctx context.Context, mutation *sppb.Mutation, preferLeader bool, hint *sppb.RoutingHint) channelEndpoint {
	return f.routeMutationWithCooldownTracker(ctx, mutation, preferLeader, hint, nil)
}

func (f *channelFinder) routeMutationWithCooldownTracker(ctx context.Context, mutation *sppb.Mutation, preferLeader bool, hint *sppb.RoutingHint, cooldowns *endpointOverloadCooldownTracker) channelEndpoint {
	if mutation == nil || hint == nil {
		return nil
	}
	f.recipeCache.applySchemaGeneration(hint)
	target := f.recipeCache.mutationToTargetRange(mutation)
	if target == nil {
		return nil
	}
	f.recipeCache.applyTargetRange(hint, target)
	return f.fillRoutingHintWithCooldownTracker(ctx, preferLeader, rangeModeCoveringSplit, &sppb.DirectedReadOptions{}, hint, cooldowns)
}

func (f *channelFinder) fillRoutingHint(ctx context.Context, preferLeader bool, mode rangeMode, directedReadOptions *sppb.DirectedReadOptions, hint *sppb.RoutingHint) channelEndpoint {
	return f.fillRoutingHintWithCooldownTracker(ctx, preferLeader, mode, directedReadOptions, hint, nil)
}

func (f *channelFinder) fillRoutingHintWithCooldownTracker(ctx context.Context, preferLeader bool, mode rangeMode, directedReadOptions *sppb.DirectedReadOptions, hint *sppb.RoutingHint, cooldowns *endpointOverloadCooldownTracker) channelEndpoint {
	if hint == nil {
		return nil
	}
	databaseID := f.databaseID.Load()
	if databaseID == 0 {
		return nil
	}
	hint.DatabaseId = databaseID
	return f.rangeCache.fillRoutingHintWithCooldownTracker(ctx, preferLeader, mode, directedReadOptions, hint, cooldowns)
}

func preferLeaderFromSelector(selector *sppb.TransactionSelector) bool {
	if selector == nil {
		return true
	}
	switch s := selector.GetSelector().(type) {
	case *sppb.TransactionSelector_Begin:
		if s.Begin == nil || s.Begin.GetReadOnly() == nil {
			return true
		}
		return s.Begin.GetReadOnly().GetStrong()
	case *sppb.TransactionSelector_SingleUse:
		if s.SingleUse == nil || s.SingleUse.GetReadOnly() == nil {
			return true
		}
		return s.SingleUse.GetReadOnly().GetStrong()
	default:
		return true
	}
}

func preferLeaderFromTransactionOptions(options *sppb.TransactionOptions) bool {
	if options == nil || options.GetReadOnly() == nil {
		return true
	}
	return options.GetReadOnly().GetStrong()
}
