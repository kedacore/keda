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
	"bytes"
	"context"
	"hash/crc32"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	sppb "cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/grpc/connectivity"
)

const (
	maxLocalReplicaDistance        = 5
	defaultMinEntriesForRandomPick = 1000
	maxRangesPerPartition          = 1024
	groupCacheShardBits            = 12
	groupCacheShardCount           = 1 << groupCacheShardBits
	localLeaderSelectionCostBias   = 0.5
)

var crc32cTable = crc32.MakeTable(crc32.Castagnoli)

type rangeMode int

const (
	rangeModeCoveringSplit rangeMode = iota
	rangeModePickRandom
)

type keyRangeCache struct {
	endpointCache               channelEndpointCache
	updateMu                    sync.Mutex
	configMu                    sync.RWMutex
	lifecycleManager            *endpointLifecycleManager
	deterministicRandom         bool
	minEntriesForRandomPickHint int

	state atomic.Value // *keyRangeCacheState

	accessCounter atomic.Int64
}

type cachedTablet struct {
	tabletUID     uint64
	incarnation   []byte
	serverAddress string
	distance      uint32
	skip          bool
	role          sppb.Tablet_Role
	location      string

	endpoint atomic.Pointer[cachedTabletEndpointRef]
}

type eligibleReplica struct {
	tablet        *cachedTablet
	endpoint      channelEndpoint
	selectionCost float64
}

type routeSelectionState struct {
	sawMatchingReplica       bool
	sawCoolingDownReplica    bool
	sawNonCoolingDownReplica bool
	hasUnavailableReplica    bool
	hasUnroutableReplica     bool
}

func (s routeSelectionState) allCoolingDown() bool {
	return s.sawMatchingReplica && s.sawCoolingDownReplica && !s.sawNonCoolingDownReplica
}

type cachedGroup struct {
	groupUID uint64

	mu         sync.RWMutex
	generation []byte
	tablets    []*cachedTablet
	leaderIdx  int
}

type cachedTabletEndpointRef struct {
	endpoint channelEndpoint
}

type cachedRange struct {
	startKey   []byte
	limitKey   []byte
	groupUID   uint64
	splitID    uint64
	generation []byte
	lastAccess int64
}

type rangePartition struct {
	startKey []byte
	limitKey []byte
	ranges   []*cachedRange
}

type keyRangeCacheState struct {
	partitions  []*rangePartition
	groupShards [groupCacheShardCount]map[uint64]*cachedGroup
	groupCount  int
	rangeCount  int
}

type keyRangeCacheRoutingConfig struct {
	lifecycleManager        *endpointLifecycleManager
	deterministicRandom     bool
	minEntriesForRandomPick int
}

type keyRangeCacheStateBuilder struct {
	cache                 *keyRangeCache
	partitions            []*rangePartition
	groupShards           [groupCacheShardCount]map[uint64]*cachedGroup
	clonedGroupShards     [groupCacheShardCount]bool
	mutableGroups         map[uint64]struct{}
	overlappingRanges     int
	rangesInserted        int
	rangesRemoved         int
	clonedRangeShardCount int
	rangeShardSizeSum     int
	rangeShardSizeMax     int
	groupShardSizeSum     int
	groupShardSizeMax     int
	groupCount            int
	rangeCount            int
}

func newKeyRangeCache(endpointCache channelEndpointCache) *keyRangeCache {
	if endpointCache == nil {
		endpointCache = newPassthroughChannelEndpointCache()
	}
	cache := &keyRangeCache{
		endpointCache:               endpointCache,
		minEntriesForRandomPickHint: defaultMinEntriesForRandomPick,
	}
	cache.state.Store(&keyRangeCacheState{})
	return cache
}

func (c *keyRangeCache) useDeterministicRandom() {
	c.configMu.Lock()
	defer c.configMu.Unlock()
	c.deterministicRandom = true
}

func (c *keyRangeCache) setMinEntriesForRandomPick(value int) {
	c.configMu.Lock()
	defer c.configMu.Unlock()
	if value <= 0 {
		value = defaultMinEntriesForRandomPick
	}
	c.minEntriesForRandomPickHint = value
}

func (c *keyRangeCache) setLifecycleManager(lifecycleManager *endpointLifecycleManager) {
	c.configMu.Lock()
	defer c.configMu.Unlock()
	c.lifecycleManager = lifecycleManager
}

func (c *keyRangeCache) recordReplicaLatency(operationUID uint64, address string, latency time.Duration) {
	endpointLatencyRegistryRecordLatency(operationUID, false, address, latency)
}

func (c *keyRangeCache) recordReplicaError(operationUID uint64, address string) {
	endpointLatencyRegistryRecordError(operationUID, false, address)
}

func routingOperationUID(hint *sppb.RoutingHint) uint64 {
	if hint == nil {
		return 0
	}
	return hint.GetOperationUid()
}

func (c *keyRangeCache) loadState() *keyRangeCacheState {
	state, _ := c.state.Load().(*keyRangeCacheState)
	if state == nil {
		return &keyRangeCacheState{}
	}
	return state
}

func (c *keyRangeCache) loadRoutingConfig() keyRangeCacheRoutingConfig {
	c.configMu.RLock()
	defer c.configMu.RUnlock()
	minEntries := c.minEntriesForRandomPickHint
	if minEntries <= 0 {
		minEntries = defaultMinEntriesForRandomPick
	}
	return keyRangeCacheRoutingConfig{
		lifecycleManager:        c.lifecycleManager,
		deterministicRandom:     c.deterministicRandom,
		minEntriesForRandomPick: minEntries,
	}
}

func cloneCachedGroup(group *cachedGroup) *cachedGroup {
	if group == nil {
		return nil
	}
	group.mu.RLock()
	defer group.mu.RUnlock()
	cloned := &cachedGroup{
		groupUID:   group.groupUID,
		generation: append([]byte(nil), group.generation...),
		leaderIdx:  group.leaderIdx,
		tablets:    make([]*cachedTablet, 0, len(group.tablets)),
	}
	for _, tablet := range group.tablets {
		if tablet == nil {
			cloned.tablets = append(cloned.tablets, nil)
			continue
		}
		clonedTablet := &cachedTablet{
			tabletUID:     tablet.tabletUID,
			incarnation:   append([]byte(nil), tablet.incarnation...),
			serverAddress: tablet.serverAddress,
			distance:      tablet.distance,
			skip:          tablet.skip,
			role:          tablet.role,
			location:      tablet.location,
		}
		clonedTablet.storeEndpoint(tablet.loadEndpoint())
		cloned.tablets = append(cloned.tablets, clonedTablet)
	}
	return cloned
}

func (c *keyRangeCache) cloneState() *keyRangeCacheStateBuilder {
	current := c.loadState()
	builder := &keyRangeCacheStateBuilder{
		cache:         c,
		partitions:    append([]*rangePartition(nil), current.partitions...),
		mutableGroups: make(map[uint64]struct{}),
		groupCount:    current.groupCount,
		rangeCount:    current.rangeCount,
	}
	for shardIdx := range current.groupShards {
		builder.groupShards[shardIdx] = current.groupShards[shardIdx]
	}
	return builder
}

func (b *keyRangeCacheStateBuilder) snapshot() *keyRangeCacheState {
	return &keyRangeCacheState{
		partitions:  b.partitions,
		groupShards: b.groupShards,
		groupCount:  b.groupCount,
		rangeCount:  b.rangeCount,
	}
}

func groupShardIndex(groupUID uint64) int {
	return int(mixUint64(groupUID) & uint64(groupCacheShardCount-1))
}

func mixUint64(v uint64) uint64 {
	v ^= v >> 30
	v *= 0xbf58476d1ce4e5b9
	v ^= v >> 27
	v *= 0x94d049bb133111eb
	v ^= v >> 31
	return v
}

func (b *keyRangeCacheStateBuilder) cloneGroupShard(idx int) {
	if idx < 0 || idx >= groupCacheShardCount || b.clonedGroupShards[idx] {
		return
	}
	original := b.groupShards[idx]
	b.groupShardSizeSum += len(original)
	if len(original) > b.groupShardSizeMax {
		b.groupShardSizeMax = len(original)
	}
	if len(original) == 0 {
		b.groupShards[idx] = make(map[uint64]*cachedGroup)
		b.clonedGroupShards[idx] = true
		return
	}
	cloned := make(map[uint64]*cachedGroup, len(original))
	for groupUID, group := range original {
		cloned[groupUID] = group
	}
	b.groupShards[idx] = cloned
	b.clonedGroupShards[idx] = true
}

func (c *keyRangeCache) addRanges(cacheUpdate *sppb.CacheUpdate) {
	if cacheUpdate == nil {
		return
	}

	c.updateMu.Lock()
	defer c.updateMu.Unlock()

	builder := c.cloneState()
	newGroups := make([]*cachedGroup, 0, len(cacheUpdate.GetGroup()))
	for _, groupIn := range cacheUpdate.GetGroup() {
		newGroups = append(newGroups, builder.findOrInsertGroup(groupIn))
	}
	for _, rangeIn := range cacheUpdate.GetRange() {
		builder.replaceRangeIfNewer(rangeIn)
	}
	for _, group := range newGroups {
		builder.unrefGroup(group)
	}
	c.state.Store(builder.snapshot())
}

func (c *keyRangeCache) fillRoutingHint(ctx context.Context, preferLeader bool, mode rangeMode, directedReadOptions *sppb.DirectedReadOptions, hint *sppb.RoutingHint) channelEndpoint {
	return c.fillRoutingHintWithCooldownTracker(ctx, preferLeader, mode, directedReadOptions, hint, nil)
}

func (c *keyRangeCache) fillRoutingHintWithCooldownTracker(ctx context.Context, preferLeader bool, mode rangeMode, directedReadOptions *sppb.DirectedReadOptions, hint *sppb.RoutingHint, cooldowns *endpointOverloadCooldownTracker) channelEndpoint {
	if hint == nil || len(hint.GetKey()) == 0 {
		return nil
	}
	if directedReadOptions == nil {
		directedReadOptions = &sppb.DirectedReadOptions{}
	}

	state := c.loadState()
	cfg := c.loadRoutingConfig()
	targetRange := c.findRangeInState(state, hint.GetKey(), hint.GetLimitKey(), mode, cfg)
	if targetRange == nil {
		return nil
	}
	targetGroup := state.findGroup(targetRange.groupUID)
	if targetGroup == nil {
		return nil
	}

	hint.GroupUid = targetRange.groupUID
	hint.SplitId = targetRange.splitID
	hint.Key = append(hint.Key[:0], targetRange.startKey...)
	hint.LimitKey = append(hint.LimitKey[:0], targetRange.limitKey...)

	return targetGroup.fillRoutingHintWithCooldownTracker(ctx, c.endpointCache, cfg.lifecycleManager, cfg.deterministicRandom, preferLeader, directedReadOptions, hint, cooldowns)
}

func (c *keyRangeCache) clear() {
	c.updateMu.Lock()
	defer c.updateMu.Unlock()
	c.state.Store(&keyRangeCacheState{})
	c.accessCounter.Store(0)
}

func (c *keyRangeCache) size() int {
	return c.loadState().rangeCount
}

func newRangePartition(ranges []*cachedRange) *rangePartition {
	if len(ranges) == 0 {
		return nil
	}
	return &rangePartition{
		startKey: append([]byte(nil), ranges[0].startKey...),
		limitKey: append([]byte(nil), ranges[len(ranges)-1].limitKey...),
		ranges:   ranges,
	}
}

func buildRangePartitions(ranges []*cachedRange) []*rangePartition {
	if len(ranges) == 0 {
		return nil
	}
	sort.Slice(ranges, func(i, j int) bool {
		return bytes.Compare(ranges[i].startKey, ranges[j].startKey) < 0
	})
	partitions := make([]*rangePartition, 0, (len(ranges)+maxRangesPerPartition-1)/maxRangesPerPartition)
	for i := 0; i < len(ranges); i += maxRangesPerPartition {
		end := i + maxRangesPerPartition
		if end > len(ranges) {
			end = len(ranges)
		}
		chunk := append([]*cachedRange(nil), ranges[i:end]...)
		partitions = append(partitions, newRangePartition(chunk))
	}
	return partitions
}

func uniqueRangesFromPartitions(partitions []*rangePartition) []*cachedRange {
	if len(partitions) == 0 {
		return nil
	}
	total := 0
	for _, partition := range partitions {
		if partition != nil {
			total += len(partition.ranges)
		}
	}
	ranges := make([]*cachedRange, 0, total)
	for _, partition := range partitions {
		if partition == nil {
			continue
		}
		ranges = append(ranges, partition.ranges...)
	}
	return ranges
}

func findPartitionStartIndex(partitions []*rangePartition, key []byte) int {
	return sort.Search(len(partitions), func(i int) bool {
		return bytes.Compare(partitions[i].limitKey, key) > 0
	})
}

func findOverlappingPartitionWindow(partitions []*rangePartition, startKey, limitKey []byte) (int, int) {
	start := findPartitionStartIndex(partitions, startKey)
	if len(limitKey) == 0 {
		end := start
		if start < len(partitions) && bytes.Compare(partitions[start].startKey, startKey) <= 0 {
			end = start + 1
		}
		return start, end
	}
	end := start
	for end < len(partitions) && bytes.Compare(partitions[end].startKey, limitKey) < 0 {
		end++
	}
	return start, end
}

func (b *keyRangeCacheStateBuilder) recordTouchedPartitions(start, end int) {
	if start < 0 {
		start = 0
	}
	if end > len(b.partitions) {
		end = len(b.partitions)
	}
	for _, partition := range b.partitions[start:end] {
		if partition == nil {
			continue
		}
		size := len(partition.ranges)
		b.clonedRangeShardCount++
		b.rangeShardSizeSum += size
		if size > b.rangeShardSizeMax {
			b.rangeShardSizeMax = size
		}
	}
}

func (b *keyRangeCacheStateBuilder) replacePartitionWindow(start, end int, ranges []*cachedRange) {
	b.recordTouchedPartitions(start, end)
	rebuilt := buildRangePartitions(ranges)
	next := make([]*rangePartition, 0, len(b.partitions)-(end-start)+len(rebuilt))
	next = append(next, b.partitions[:start]...)
	next = append(next, rebuilt...)
	next = append(next, b.partitions[end:]...)
	b.partitions = next
}

func (c *keyRangeCache) shrinkTo(newSize int) {
	c.updateMu.Lock()
	defer c.updateMu.Unlock()
	builder := c.cloneState()
	if newSize <= 0 {
		c.state.Store(&keyRangeCacheState{})
		c.accessCounter.Store(0)
		return
	}
	if newSize >= builder.rangeCount {
		return
	}

	allRanges := uniqueRangesFromPartitions(builder.partitions)
	if newSize >= len(allRanges) {
		return
	}

	numToShrink := len(allRanges) - newSize
	numToSample := numToShrink * 2
	if numToSample > len(allRanges) {
		numToSample = len(allRanges)
	}

	perm := rand.Perm(len(allRanges))
	sampled := make([]*cachedRange, 0, numToSample)
	for i := 0; i < numToSample; i++ {
		sampled = append(sampled, allRanges[perm[i]])
	}
	sort.Slice(sampled, func(i, j int) bool {
		return sampled[i].lastAccess < sampled[j].lastAccess
	})

	evicted := make(map[*cachedRange]struct{}, numToShrink)
	for i := 0; i < numToShrink; i++ {
		evicted[sampled[i]] = struct{}{}
	}

	kept := make([]*cachedRange, 0, len(allRanges)-numToShrink)
	for _, r := range allRanges {
		if _, ok := evicted[r]; ok {
			continue
		}
		kept = append(kept, r)
	}
	builder.recordTouchedPartitions(0, len(builder.partitions))
	builder.partitions = buildRangePartitions(kept)
	builder.rangeCount = len(allRanges) - numToShrink
	c.state.Store(builder.snapshot())
}

func (c *keyRangeCache) accessTimeNow() int64 {
	return c.accessCounter.Add(1)
}

func (c *keyRangeCache) findRangeInState(state *keyRangeCacheState, key, limit []byte, mode rangeMode, cfg keyRangeCacheRoutingConfig) *cachedRange {
	if state == nil {
		return nil
	}
	ranges := c.lookupRangesForState(state, key, limit)
	low, high := 0, len(ranges)
	for low < high {
		mid := int(uint(low+high) >> 1)
		if bytes.Compare(ranges[mid].limitKey, key) > 0 {
			high = mid
		} else {
			low = mid + 1
		}
	}
	idx := low
	if idx >= len(ranges) {
		return nil
	}
	first := ranges[idx]
	startInRange := bytes.Compare(key, first.startKey) >= 0
	if len(limit) == 0 {
		if startInRange {
			atomic.StoreInt64(&first.lastAccess, c.accessTimeNow())
			return first
		}
		return nil
	}
	if startInRange && bytes.Compare(limit, first.limitKey) <= 0 {
		atomic.StoreInt64(&first.lastAccess, c.accessTimeNow())
		return first
	}
	if mode == rangeModeCoveringSplit {
		return nil
	}

	total := 0
	foundGap := !startInRange
	sampledIdx := idx
	lastLimit := first.startKey
	hitEnd := false

	i := idx
	for ; i < len(ranges); i++ {
		current := ranges[i]
		if bytes.Compare(lastLimit, current.startKey) != 0 {
			foundGap = true
			if bytes.Compare(current.startKey, limit) >= 0 {
				break
			}
		}
		total++
		if c.uniformRandom(total, key, limit, current.startKey, cfg.deterministicRandom) == 0 {
			sampledIdx = i
		}
		lastLimit = current.limitKey
		if bytes.Compare(lastLimit, limit) >= 0 || total >= cfg.minEntriesForRandomPick {
			break
		}
	}
	if i >= len(ranges) {
		hitEnd = true
	}
	if hitEnd {
		foundGap = true
	}
	if !foundGap || total >= cfg.minEntriesForRandomPick {
		selected := ranges[sampledIdx]
		atomic.StoreInt64(&selected.lastAccess, c.accessTimeNow())
		return selected
	}
	return nil
}

func (c *keyRangeCache) lookupRangesForState(state *keyRangeCacheState, key, limit []byte) []*cachedRange {
	if state == nil {
		return nil
	}
	start, end := findOverlappingPartitionWindow(state.partitions, key, limit)
	if start >= len(state.partitions) {
		return nil
	}
	if end <= start {
		return state.partitions[start].ranges
	}
	if end == start+1 {
		return state.partitions[start].ranges
	}
	ranges := uniqueRangesFromPartitions(state.partitions[start:end])
	sort.Slice(ranges, func(i, j int) bool {
		return bytes.Compare(ranges[i].limitKey, ranges[j].limitKey) < 0
	})
	return ranges
}

func (c *keyRangeCache) uniformRandom(n int, seed1, seed2, seed3 []byte, deterministic bool) int {
	if n <= 1 {
		return 0
	}
	if deterministic {
		data := make([]byte, 0, len(seed1)+len(seed2)+len(seed3))
		data = append(data, seed1...)
		data = append(data, seed2...)
		data = append(data, seed3...)
		return int(crc32.Checksum(data, crc32cTable) % uint32(n))
	}
	return rand.Intn(n)
}

func (b *keyRangeCacheStateBuilder) replaceRangeIfNewer(rangeIn *sppb.Range) {
	if rangeIn == nil {
		return
	}
	startKey := append([]byte(nil), rangeIn.GetStartKey()...)
	limitKey := append([]byte(nil), rangeIn.GetLimitKey()...)
	start, end := findOverlappingPartitionWindow(b.partitions, startKey, limitKey)
	touchedRanges := uniqueRangesFromPartitions(b.partitions[start:end])

	overlappingRanges := make([]*cachedRange, 0)
	rebuiltRanges := make([]*cachedRange, 0, len(touchedRanges)+3)
	for _, existing := range touchedRanges {
		if bytes.Compare(existing.limitKey, startKey) <= 0 || bytes.Compare(existing.startKey, limitKey) >= 0 {
			rebuiltRanges = append(rebuiltRanges, existing)
			continue
		}
		cmp := bytes.Compare(rangeIn.GetGeneration(), existing.generation)
		if cmp < 0 || (cmp == 0 && bytes.Equal(existing.startKey, startKey) && bytes.Equal(existing.limitKey, limitKey)) {
			return
		}
		overlappingRanges = append(overlappingRanges, existing)
	}
	b.overlappingRanges += len(overlappingRanges)

	if len(overlappingRanges) > 0 {
		sort.Slice(overlappingRanges, func(i, j int) bool {
			return bytes.Compare(overlappingRanges[i].startKey, overlappingRanges[j].startKey) < 0
		})
		first := overlappingRanges[0]
		if bytes.Compare(first.startKey, startKey) < 0 {
			rebuiltRanges = append(rebuiltRanges, &cachedRange{
				startKey:   append([]byte(nil), first.startKey...),
				limitKey:   append([]byte(nil), startKey...),
				groupUID:   first.groupUID,
				splitID:    first.splitID,
				generation: append([]byte(nil), first.generation...),
				lastAccess: first.lastAccess,
			})
			b.rangesInserted++
		}
		last := overlappingRanges[len(overlappingRanges)-1]
		if bytes.Compare(last.limitKey, limitKey) > 0 {
			rebuiltRanges = append(rebuiltRanges, &cachedRange{
				startKey:   append([]byte(nil), limitKey...),
				limitKey:   append([]byte(nil), last.limitKey...),
				groupUID:   last.groupUID,
				splitID:    last.splitID,
				generation: append([]byte(nil), last.generation...),
				lastAccess: last.lastAccess,
			})
			b.rangesInserted++
		}
		b.rangesRemoved += len(overlappingRanges)
	}

	rebuiltRanges = append(rebuiltRanges, &cachedRange{
		startKey:   startKey,
		limitKey:   limitKey,
		groupUID:   rangeIn.GetGroupUid(),
		splitID:    rangeIn.GetSplitId(),
		generation: append([]byte(nil), rangeIn.GetGeneration()...),
		lastAccess: b.cache.accessTimeNow(),
	})
	b.rangesInserted++

	b.rangeCount += len(rebuiltRanges) - len(touchedRanges)
	b.replacePartitionWindow(start, end, rebuiltRanges)
}

func (b *keyRangeCacheStateBuilder) findAndRefGroup(groupUID uint64) *cachedGroup {
	return b.findGroup(groupUID)
}

func (b *keyRangeCacheStateBuilder) findOrInsertGroup(groupIn *sppb.Group) *cachedGroup {
	if groupIn == nil {
		return nil
	}
	groupUID := groupIn.GetGroupUid()
	shardIdx := groupShardIndex(groupUID)
	b.cloneGroupShard(shardIdx)

	group, ok := b.groupShards[shardIdx][groupUID]
	if !ok {
		group = &cachedGroup{groupUID: groupUID, leaderIdx: -1}
		b.groupShards[shardIdx][groupUID] = group
		b.mutableGroups[groupUID] = struct{}{}
		b.groupCount++
	} else if _, mutable := b.mutableGroups[groupUID]; !mutable {
		group = cloneCachedGroup(group)
		b.groupShards[shardIdx][groupUID] = group
		b.mutableGroups[groupUID] = struct{}{}
	}
	group.update(groupIn)
	return group
}

func (b *keyRangeCacheStateBuilder) refGroup(group *cachedGroup) *cachedGroup {
	return group
}

func (b *keyRangeCacheStateBuilder) unrefGroup(group *cachedGroup) {
}

func (s *keyRangeCacheState) findGroup(groupUID uint64) *cachedGroup {
	if s == nil {
		return nil
	}
	shard := s.groupShards[groupShardIndex(groupUID)]
	if len(shard) == 0 {
		return nil
	}
	return shard[groupUID]
}

func (b *keyRangeCacheStateBuilder) findGroup(groupUID uint64) *cachedGroup {
	if b == nil {
		return nil
	}
	shard := b.groupShards[groupShardIndex(groupUID)]
	if len(shard) == 0 {
		return nil
	}
	return shard[groupUID]
}

func (t *cachedTablet) update(tabletIn *sppb.Tablet) {
	if tabletIn == nil {
		return
	}
	if t.tabletUID > 0 && bytes.Compare(t.incarnation, tabletIn.GetIncarnation()) > 0 {
		return
	}
	t.tabletUID = tabletIn.GetTabletUid()
	t.incarnation = append([]byte(nil), tabletIn.GetIncarnation()...)
	t.distance = tabletIn.GetDistance()
	t.skip = tabletIn.GetSkip()
	t.role = tabletIn.GetRole()
	t.location = tabletIn.GetLocation()
	if t.serverAddress != tabletIn.GetServerAddress() {
		t.serverAddress = tabletIn.GetServerAddress()
		t.storeEndpoint(nil)
	}
}

func (t *cachedTablet) loadEndpoint() channelEndpoint {
	if t == nil {
		return nil
	}
	ref := t.endpoint.Load()
	if ref == nil {
		return nil
	}
	return ref.endpoint
}

func (t *cachedTablet) storeEndpoint(endpoint channelEndpoint) {
	if t == nil {
		return
	}
	if endpoint == nil {
		t.endpoint.Store(nil)
		return
	}
	t.endpoint.Store(&cachedTabletEndpointRef{endpoint: endpoint})
}

func (t *cachedTablet) clearShutdownEndpoint() channelEndpoint {
	endpoint := t.loadEndpoint()
	if endpoint == nil {
		return nil
	}
	conn := endpoint.GetConn()
	if conn == nil {
		return endpoint
	}
	if conn.GetState() == connectivity.Shutdown {
		t.storeEndpoint(nil)
		return nil
	}
	return endpoint
}

func (t *cachedTablet) getOrLoadEndpointIfPresent(endpointCache channelEndpointCache) channelEndpoint {
	endpoint := t.clearShutdownEndpoint()
	if endpoint != nil || endpointCache == nil {
		return endpoint
	}
	endpoint = endpointCache.GetIfPresent(t.serverAddress)
	if endpoint != nil {
		t.storeEndpoint(endpoint)
	}
	return endpoint
}

func (t *cachedTablet) matches(directedReadOptions *sppb.DirectedReadOptions) bool {
	if directedReadOptions == nil {
		return t.distance <= maxLocalReplicaDistance
	}
	switch replicas := directedReadOptions.GetReplicas().(type) {
	case *sppb.DirectedReadOptions_IncludeReplicas_:
		for _, selection := range replicas.IncludeReplicas.GetReplicaSelections() {
			if t.matchesReplicaSelection(selection) {
				return true
			}
		}
		return false
	case *sppb.DirectedReadOptions_ExcludeReplicas_:
		for _, selection := range replicas.ExcludeReplicas.GetReplicaSelections() {
			if t.matchesReplicaSelection(selection) {
				return false
			}
		}
		return true
	default:
		return t.distance <= maxLocalReplicaDistance
	}
}

func (t *cachedTablet) matchesReplicaSelection(selection *sppb.DirectedReadOptions_ReplicaSelection) bool {
	if selection == nil {
		return true
	}
	if selection.GetLocation() != "" && selection.GetLocation() != t.location {
		return false
	}
	switch selection.GetType() {
	case sppb.DirectedReadOptions_ReplicaSelection_READ_WRITE:
		return t.role == sppb.Tablet_READ_WRITE || t.role == sppb.Tablet_ROLE_UNSPECIFIED
	case sppb.DirectedReadOptions_ReplicaSelection_READ_ONLY:
		return t.role == sppb.Tablet_READ_ONLY
	default:
		return true
	}
}

func (t *cachedTablet) shouldSkip(hint *sppb.RoutingHint) bool {
	return t.shouldSkipWithCooldownTracker(hint, nil)
}

func (t *cachedTablet) shouldSkipWithCooldownTracker(hint *sppb.RoutingHint, cooldowns *endpointOverloadCooldownTracker) bool {
	if hint == nil {
		return true
	}
	if t.skip || t.serverAddress == "" {
		hint.SkippedTabletUid = append(hint.SkippedTabletUid, &sppb.RoutingHint_SkippedTablet{
			TabletUid:   t.tabletUID,
			Incarnation: append([]byte(nil), t.incarnation...),
		})
		return true
	}
	if endpoint := t.clearShutdownEndpoint(); endpoint != nil && !endpoint.IsHealthy() {
		hint.SkippedTabletUid = append(hint.SkippedTabletUid, &sppb.RoutingHint_SkippedTablet{
			TabletUid:   t.tabletUID,
			Incarnation: append([]byte(nil), t.incarnation...),
		})
		return true
	}
	if isEndpointCoolingDown(cooldowns, t.serverAddress) {
		return true
	}
	return false
}

func (t *cachedTablet) shouldSkipForRouting(endpointCache channelEndpointCache, lifecycleManager *endpointLifecycleManager, hint *sppb.RoutingHint, cooldowns *endpointOverloadCooldownTracker, skippedTabletUIDs map[uint64]struct{}, pendingCreations map[string]struct{}, state *routeSelectionState) bool {
	if hint == nil {
		return true
	}
	if state != nil {
		state.sawMatchingReplica = true
	}
	if t.skip || t.serverAddress == "" {
		if state != nil {
			state.sawNonCoolingDownReplica = true
			state.hasUnroutableReplica = true
		}
		t.addSkippedTablet(hint, skippedTabletUIDs)
		return true
	}
	if isEndpointCoolingDown(cooldowns, t.serverAddress) {
		if state != nil {
			state.sawCoolingDownReplica = true
		}
		return true
	}
	if state != nil {
		state.sawNonCoolingDownReplica = true
	}

	endpoint := t.getOrLoadEndpointIfPresent(endpointCache)
	if endpoint == nil {
		if state != nil {
			state.hasUnavailableReplica = true
		}
		if pendingCreations != nil {
			pendingCreations[t.serverAddress] = struct{}{}
			if lifecycleManager != nil {
				lifecycleManager.requestEndpointRecreation(t.serverAddress)
			}
			return true
		}
		if lifecycleManager != nil {
			lifecycleManager.requestEndpointRecreation(t.serverAddress)
		}
		if t.maybeAddRecentTransientFailureSkip(lifecycleManager, hint, skippedTabletUIDs) {
			return true
		}
		return true
	}
	if endpoint.IsHealthy() {
		return false
	}

	if lifecycleManager != nil {
		lifecycleManager.requestEndpointRecreation(t.serverAddress)
	}
	if endpoint.IsTransientFailure() {
		if state != nil {
			state.hasUnavailableReplica = true
		}
		t.addSkippedTablet(hint, skippedTabletUIDs)
		return true
	}

	if state != nil {
		state.hasUnavailableReplica = true
	}
	if t.maybeAddRecentTransientFailureSkip(lifecycleManager, hint, skippedTabletUIDs) {
		return true
	}
	return true
}

func (t *cachedTablet) recordKnownTransientFailure(endpointCache channelEndpointCache, lifecycleManager *endpointLifecycleManager, hint *sppb.RoutingHint, cooldowns *endpointOverloadCooldownTracker, skippedTabletUIDs map[uint64]struct{}) {
	if hint == nil || t.skip || t.serverAddress == "" || isEndpointCoolingDown(cooldowns, t.serverAddress) {
		return
	}

	endpoint := t.getOrLoadEndpointIfPresent(endpointCache)
	if endpoint != nil && endpoint.IsTransientFailure() {
		t.addSkippedTablet(hint, skippedTabletUIDs)
		return
	}

	t.maybeAddRecentTransientFailureSkip(lifecycleManager, hint, skippedTabletUIDs)
}

func (t *cachedTablet) maybeAddRecentTransientFailureSkip(lifecycleManager *endpointLifecycleManager, hint *sppb.RoutingHint, skippedTabletUIDs map[uint64]struct{}) bool {
	if lifecycleManager == nil || !lifecycleManager.wasRecentlyEvictedTransientFailure(t.serverAddress) {
		return false
	}
	t.addSkippedTablet(hint, skippedTabletUIDs)
	return true
}

func (t *cachedTablet) addSkippedTablet(hint *sppb.RoutingHint, skippedTabletUIDs map[uint64]struct{}) {
	if hint == nil {
		return
	}
	if skippedTabletUIDs != nil {
		if _, ok := skippedTabletUIDs[t.tabletUID]; ok {
			return
		}
		skippedTabletUIDs[t.tabletUID] = struct{}{}
	}
	hint.SkippedTabletUid = append(hint.SkippedTabletUid, &sppb.RoutingHint_SkippedTablet{
		TabletUid:   t.tabletUID,
		Incarnation: append([]byte(nil), t.incarnation...),
	})
}

func (t *cachedTablet) pick(hint *sppb.RoutingHint) channelEndpoint {
	if hint != nil {
		hint.TabletUid = t.tabletUID
	}
	return t.loadEndpoint()
}

func (g *cachedGroup) update(groupIn *sppb.Group) {
	if groupIn == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	if bytes.Compare(groupIn.GetGeneration(), g.generation) > 0 {
		g.generation = append([]byte(nil), groupIn.GetGeneration()...)
		if idx := int(groupIn.GetLeaderIndex()); idx >= 0 && idx < len(groupIn.GetTablets()) {
			g.leaderIdx = idx
		} else {
			g.leaderIdx = -1
		}
	}

	if len(g.tablets) == len(groupIn.GetTablets()) {
		mismatch := false
		for i := range g.tablets {
			if g.tablets[i].tabletUID != groupIn.GetTablets()[i].GetTabletUid() {
				mismatch = true
				break
			}
		}
		if !mismatch {
			for i := range g.tablets {
				g.tablets[i].update(groupIn.GetTablets()[i])
			}
			return
		}
	}

	tabletByUID := make(map[uint64]*cachedTablet, len(g.tablets))
	for _, tablet := range g.tablets {
		tabletByUID[tablet.tabletUID] = tablet
	}
	newTablets := make([]*cachedTablet, 0, len(groupIn.GetTablets()))
	for _, tabletIn := range groupIn.GetTablets() {
		tablet := tabletByUID[tabletIn.GetTabletUid()]
		if tablet == nil {
			tablet = &cachedTablet{}
		}
		tablet.update(tabletIn)
		newTablets = append(newTablets, tablet)
	}
	g.tablets = newTablets
}

func (g *cachedGroup) hasLeaderLocked() bool {
	return g.leaderIdx >= 0 && g.leaderIdx < len(g.tablets)
}

func (g *cachedGroup) leaderLocked() *cachedTablet {
	if !g.hasLeaderLocked() {
		return nil
	}
	return g.tablets[g.leaderIdx]
}

func (g *cachedGroup) fillRoutingHint(ctx context.Context, endpointCache channelEndpointCache, preferLeader bool, directedReadOptions *sppb.DirectedReadOptions, hint *sppb.RoutingHint) channelEndpoint {
	return g.fillRoutingHintWithCooldownTracker(ctx, endpointCache, nil, false, preferLeader, directedReadOptions, hint, nil)
}

func (g *cachedGroup) fillRoutingHintWithCooldownTracker(ctx context.Context, endpointCache channelEndpointCache, lifecycleManager *endpointLifecycleManager, deterministicRandom bool, preferLeader bool, directedReadOptions *sppb.DirectedReadOptions, hint *sppb.RoutingHint, cooldowns *endpointOverloadCooldownTracker) channelEndpoint {
	pendingCreations := make(map[string]struct{})
	selected, state := g.fillRoutingHintAttempt(endpointCache, lifecycleManager, deterministicRandom, preferLeader, directedReadOptions, hint, cooldowns, pendingCreations)
	if selected != nil {
		return selected.pick(hint)
	}
	if state.allCoolingDown() {
		g.mu.RLock()
		selected = g.selectCoolingDownTabletLocked(endpointCache, deterministicRandom, preferLeader, directedReadOptions, hint)
		if selected != nil {
			g.recordKnownTransientFailuresLocked(endpointCache, lifecycleManager, selected, directedReadOptions, hint, cooldowns, skippedTabletUIDsFromHint(hint))
			g.mu.RUnlock()
			return selected.pick(hint)
		}
		g.mu.RUnlock()
	}
	if len(pendingCreations) == 0 || !shouldSynchronouslyWarmEndpoints(endpointCache) {
		return nil
	}
	warmPendingEndpoints(ctx, endpointCache, pendingCreations)
	selected, state = g.fillRoutingHintAttempt(endpointCache, lifecycleManager, deterministicRandom, preferLeader, directedReadOptions, hint, cooldowns, nil)
	if selected == nil {
		if !state.allCoolingDown() {
			return nil
		}
		g.mu.RLock()
		selected = g.selectCoolingDownTabletLocked(endpointCache, deterministicRandom, preferLeader, directedReadOptions, hint)
		if selected == nil {
			g.mu.RUnlock()
			return nil
		}
		g.recordKnownTransientFailuresLocked(endpointCache, lifecycleManager, selected, directedReadOptions, hint, cooldowns, skippedTabletUIDsFromHint(hint))
		g.mu.RUnlock()
	}
	return selected.pick(hint)
}

func shouldSynchronouslyWarmEndpoints(endpointCache channelEndpointCache) bool {
	if endpointCache == nil {
		return false
	}
	_, blocksOnGet := endpointCache.(*endpointClientCache)
	return !blocksOnGet
}

func (g *cachedGroup) fillRoutingHintAttempt(endpointCache channelEndpointCache, lifecycleManager *endpointLifecycleManager, deterministicRandom bool, preferLeader bool, directedReadOptions *sppb.DirectedReadOptions, hint *sppb.RoutingHint, cooldowns *endpointOverloadCooldownTracker, pendingCreations map[string]struct{}) (*cachedTablet, routeSelectionState) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if directedReadOptions == nil {
		directedReadOptions = &sppb.DirectedReadOptions{}
	}
	hasDirectedReadOptions := directedReadOptions.GetReplicas() != nil
	skippedTabletUIDs := skippedTabletUIDsFromHint(hint)
	var state routeSelectionState

	if !preferLeader || routingOperationUID(hint) > 0 {
		selected := g.selectScoreAwareTabletLocked(endpointCache, lifecycleManager, deterministicRandom, preferLeader, hasDirectedReadOptions, directedReadOptions, hint, cooldowns, skippedTabletUIDs, pendingCreations, &state)
		if selected != nil {
			g.recordKnownTransientFailuresLocked(endpointCache, lifecycleManager, selected, directedReadOptions, hint, cooldowns, skippedTabletUIDs)
		}
		return selected, state
	}

	leader := g.leaderLocked()
	if !hasDirectedReadOptions && leader != nil && leader.distance <= maxLocalReplicaDistance && !leader.shouldSkipForRouting(endpointCache, lifecycleManager, hint, cooldowns, skippedTabletUIDs, pendingCreations, &state) {
		g.recordKnownTransientFailuresLocked(endpointCache, lifecycleManager, leader, directedReadOptions, hint, cooldowns, skippedTabletUIDs)
		return leader, state
	}
	for _, tablet := range g.tablets {
		if !tablet.matches(directedReadOptions) {
			continue
		}
		if tablet.shouldSkipForRouting(endpointCache, lifecycleManager, hint, cooldowns, skippedTabletUIDs, pendingCreations, &state) {
			continue
		}
		g.recordKnownTransientFailuresLocked(endpointCache, lifecycleManager, tablet, directedReadOptions, hint, cooldowns, skippedTabletUIDs)
		return tablet, state
	}
	return nil, state
}

func (g *cachedGroup) selectScoreAwareTabletLocked(endpointCache channelEndpointCache, lifecycleManager *endpointLifecycleManager, deterministicRandom bool, preferLeader bool, hasDirectedReadOptions bool, directedReadOptions *sppb.DirectedReadOptions, hint *sppb.RoutingHint, cooldowns *endpointOverloadCooldownTracker, skippedTabletUIDs map[uint64]struct{}, pendingCreations map[string]struct{}, state *routeSelectionState) *cachedTablet {
	preferredLeader := g.localLeaderForScoreBiasLocked(hasDirectedReadOptions)
	candidates := make([]eligibleReplica, 0, len(g.tablets))
	for _, tablet := range g.tablets {
		if !tablet.matches(directedReadOptions) {
			continue
		}
		if tablet.shouldSkipForRouting(endpointCache, lifecycleManager, hint, cooldowns, skippedTabletUIDs, pendingCreations, state) {
			continue
		}
		endpoint := tablet.loadEndpoint()
		if endpoint == nil {
			continue
		}
		candidates = append(candidates, eligibleReplica{
			tablet:        tablet,
			endpoint:      endpoint,
			selectionCost: selectionCostForTablet(routingOperationUID(hint), preferLeader, endpoint, tablet, preferredLeader),
		})
	}
	selected := selectEligibleReplica(candidates, deterministicRandom)
	if selected == nil {
		return nil
	}
	return selected.tablet
}

func (g *cachedGroup) selectCoolingDownTabletLocked(endpointCache channelEndpointCache, deterministicRandom bool, preferLeader bool, directedReadOptions *sppb.DirectedReadOptions, hint *sppb.RoutingHint) *cachedTablet {
	hasDirectedReadOptions := directedReadOptions != nil && directedReadOptions.GetReplicas() != nil
	preferredLeader := g.localLeaderForScoreBiasLocked(hasDirectedReadOptions)
	candidates := make([]eligibleReplica, 0, len(g.tablets))
	for _, tablet := range g.tablets {
		if tablet == nil || !tablet.matches(directedReadOptions) || tablet.skip || tablet.serverAddress == "" {
			continue
		}
		endpoint := tablet.getOrLoadEndpointIfPresent(endpointCache)
		if endpoint == nil || !endpoint.IsHealthy() {
			continue
		}
		candidates = append(candidates, eligibleReplica{
			tablet:        tablet,
			endpoint:      endpoint,
			selectionCost: selectionCostForTablet(routingOperationUID(hint), preferLeader, endpoint, tablet, preferredLeader),
		})
	}
	selected := selectEligibleReplica(candidates, deterministicRandom)
	if selected == nil {
		return nil
	}
	return selected.tablet
}

func (g *cachedGroup) localLeaderForScoreBiasLocked(hasDirectedReadOptions bool) *cachedTablet {
	leader := g.leaderLocked()
	if hasDirectedReadOptions || leader == nil || leader.distance > maxLocalReplicaDistance {
		return nil
	}
	return leader
}

func selectionCostForTablet(operationUID uint64, preferLeader bool, endpoint channelEndpoint, tablet *cachedTablet, preferredLeader *cachedTablet) float64 {
	if tablet == nil {
		return 0
	}
	cost := endpointLatencyRegistrySelectionCost(operationUID, preferLeader, endpoint, tablet.serverAddress)
	if preferredLeader != nil && tablet == preferredLeader {
		return cost * localLeaderSelectionCostBias
	}
	return cost
}

func selectEligibleReplica(candidates []eligibleReplica, alwaysSelectBest bool) *eligibleReplica {
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) == 1 {
		return &candidates[0]
	}
	if alwaysSelectBest {
		best := &candidates[0]
		for i := 1; i < len(candidates); i++ {
			if candidates[i].selectionCost < best.selectionCost {
				best = &candidates[i]
			}
		}
		return best
	}

	selectedIndex := defaultPowerOfTwoReplicaSelector.chooseIndex(len(candidates), func(index int) float64 {
		return candidates[index].selectionCost
	})
	if selectedIndex < 0 || selectedIndex >= len(candidates) {
		return &candidates[0]
	}
	return &candidates[selectedIndex]
}

func warmPendingEndpoints(ctx context.Context, endpointCache channelEndpointCache, pendingCreations map[string]struct{}) {
	if endpointCache == nil || len(pendingCreations) == 0 {
		return
	}
	for address := range pendingCreations {
		endpointCache.Get(ctx, address)
	}
}

func (g *cachedGroup) recordKnownTransientFailuresLocked(endpointCache channelEndpointCache, lifecycleManager *endpointLifecycleManager, selected *cachedTablet, directedReadOptions *sppb.DirectedReadOptions, hint *sppb.RoutingHint, cooldowns *endpointOverloadCooldownTracker, skippedTabletUIDs map[uint64]struct{}) {
	for _, tablet := range g.tablets {
		if tablet == selected || !tablet.matches(directedReadOptions) {
			continue
		}
		tablet.recordKnownTransientFailure(endpointCache, lifecycleManager, hint, cooldowns, skippedTabletUIDs)
	}
}

func skippedTabletUIDsFromHint(hint *sppb.RoutingHint) map[uint64]struct{} {
	if hint == nil || len(hint.GetSkippedTabletUid()) == 0 {
		return make(map[uint64]struct{})
	}
	skippedTabletUIDs := make(map[uint64]struct{}, len(hint.GetSkippedTabletUid()))
	for _, skippedTablet := range hint.GetSkippedTabletUid() {
		skippedTabletUIDs[skippedTablet.GetTabletUid()] = struct{}{}
	}
	return skippedTabletUIDs
}
