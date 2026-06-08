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
	"math/rand"
	"sync"
	"time"
)

const (
	defaultEndpointOverloadInitialCooldown = 10 * time.Second
	defaultEndpointOverloadMaxCooldown     = time.Minute
	defaultEndpointOverloadResetAfter      = 10 * time.Minute
)

type endpointOverloadCooldownState struct {
	consecutiveFailures int
	cooldownUntil       time.Time
	lastFailureAt       time.Time
}

// endpointOverloadCooldownTracker keeps routed endpoints out of selection for a
// short period after RESOURCE_EXHAUSTED so the router can try another replica.
type endpointOverloadCooldownTracker struct {
	mu              sync.RWMutex
	entries         map[string]endpointOverloadCooldownState
	initialCooldown time.Duration
	maxCooldown     time.Duration
	resetAfter      time.Duration
	now             func() time.Time
	randInt63n      func(int64) int64
}

func newEndpointOverloadCooldownTracker() *endpointOverloadCooldownTracker {
	return newEndpointOverloadCooldownTrackerWithOptions(
		defaultEndpointOverloadInitialCooldown,
		defaultEndpointOverloadMaxCooldown,
		defaultEndpointOverloadResetAfter,
		time.Now,
		rand.Int63n,
	)
}

func newEndpointOverloadCooldownTrackerWithOptions(
	initialCooldown time.Duration,
	maxCooldown time.Duration,
	resetAfter time.Duration,
	now func() time.Time,
	randInt63n func(int64) int64,
) *endpointOverloadCooldownTracker {
	if initialCooldown <= 0 {
		initialCooldown = defaultEndpointOverloadInitialCooldown
	}
	if maxCooldown <= 0 {
		maxCooldown = defaultEndpointOverloadMaxCooldown
	}
	if maxCooldown < initialCooldown {
		maxCooldown = initialCooldown
	}
	if resetAfter <= 0 {
		resetAfter = defaultEndpointOverloadResetAfter
	}
	if now == nil {
		now = time.Now
	}
	if randInt63n == nil {
		randInt63n = rand.Int63n
	}
	return &endpointOverloadCooldownTracker{
		entries:         make(map[string]endpointOverloadCooldownState),
		initialCooldown: initialCooldown,
		maxCooldown:     maxCooldown,
		resetAfter:      resetAfter,
		now:             now,
		randInt63n:      randInt63n,
	}
}

func isEndpointCoolingDown(cooldowns *endpointOverloadCooldownTracker, address string) bool {
	return cooldowns != nil && cooldowns.isCoolingDown(address)
}

func (t *endpointOverloadCooldownTracker) isCoolingDown(address string) bool {
	if t == nil || address == "" {
		return false
	}

	now := t.now()

	t.mu.RLock()
	state, ok := t.entries[address]
	t.mu.RUnlock()
	if !ok {
		return false
	}
	if state.cooldownUntil.After(now) {
		return true
	}

	if now.Sub(state.lastFailureAt) < t.resetAfter {
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	state, ok = t.entries[address]
	if !ok {
		return false
	}
	if state.cooldownUntil.After(now) {
		return true
	}
	if now.Sub(state.lastFailureAt) >= t.resetAfter {
		delete(t.entries, address)
	}
	return false
}

func (t *endpointOverloadCooldownTracker) remainingCooldown(address string) time.Duration {
	if t == nil || address == "" {
		return 0
	}

	now := t.now()

	t.mu.RLock()
	state, ok := t.entries[address]
	t.mu.RUnlock()
	if !ok {
		return 0
	}
	if state.cooldownUntil.After(now) {
		return state.cooldownUntil.Sub(now)
	}
	return 0
}

func (t *endpointOverloadCooldownTracker) recordFailure(address string) {
	if t == nil || address == "" {
		return
	}

	now := t.now()

	t.mu.Lock()
	defer t.mu.Unlock()

	state := t.entries[address]
	if state.lastFailureAt.IsZero() || now.Sub(state.lastFailureAt) >= t.resetAfter {
		state.consecutiveFailures = 0
	}
	state.consecutiveFailures++
	state.lastFailureAt = now
	state.cooldownUntil = now.Add(t.cooldownForFailures(state.consecutiveFailures))
	t.entries[address] = state
}

func (t *endpointOverloadCooldownTracker) cooldownForFailures(failures int) time.Duration {
	cooldown := t.initialCooldown
	for i := 1; i < failures; i++ {
		if cooldown > t.maxCooldown/2 {
			cooldown = t.maxCooldown
			break
		}
		cooldown *= 2
	}
	cooldownNanos := int64(cooldown)
	if cooldownNanos < 1 {
		cooldownNanos = 1
	}
	floorNanos := cooldownNanos / 2
	if floorNanos < 1 {
		floorNanos = 1
	}
	rangeSize := cooldownNanos - floorNanos + 1
	if rangeSize < 1 {
		rangeSize = 1
	}
	return time.Duration(floorNanos + t.randInt63n(rangeSize))
}

func (t *endpointOverloadCooldownTracker) pruneStaleEntries(maxAge time.Duration) {
	if t == nil || maxAge <= 0 {
		return
	}

	now := t.now()

	t.mu.Lock()
	defer t.mu.Unlock()

	for address, state := range t.entries {
		if state.cooldownUntil.After(now) {
			continue
		}
		if now.Sub(state.lastFailureAt) >= maxAge {
			delete(t.entries, address)
		}
	}
}
