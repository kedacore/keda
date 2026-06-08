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
	"math"
	"sync"
	"time"
)

const defaultEWMADecayTime = 10 * time.Second

type ewmaLatencyTracker struct {
	mu sync.Mutex

	fixedAlpha *float64
	now        func() time.Time
	tau        time.Duration

	scoreMicros      float64
	initialized      bool
	lastUpdatedNanos int64
}

func newEWMALatencyTracker() *ewmaLatencyTracker {
	return newEWMALatencyTrackerWithOptions(defaultEWMADecayTime, time.Now)
}

func newEWMALatencyTrackerWithAlpha(alpha float64, now func() time.Time) *ewmaLatencyTracker {
	if alpha <= 0 || alpha > 1 {
		panic("alpha must be in (0, 1]")
	}
	alphaCopy := alpha
	if now == nil {
		now = time.Now
	}
	return &ewmaLatencyTracker{
		fixedAlpha: &alphaCopy,
		now:        now,
	}
}

func newEWMALatencyTrackerWithOptions(decayTime time.Duration, now func() time.Time) *ewmaLatencyTracker {
	if decayTime <= 0 {
		panic("decayTime must be > 0")
	}
	if now == nil {
		now = time.Now
	}
	return &ewmaLatencyTracker{
		now: now,
		tau: decayTime,
	}
}

func (t *ewmaLatencyTracker) hasScore() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.initialized
}

func (t *ewmaLatencyTracker) scoreValue() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.initialized {
		return math.MaxFloat64
	}
	return t.scoreMicros
}

func (t *ewmaLatencyTracker) update(latency time.Duration) {
	latencyMicros := float64(latency) / float64(time.Microsecond)
	now := t.now().UnixNano()

	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.initialized {
		t.scoreMicros = latencyMicros
		t.initialized = true
		t.lastUpdatedNanos = now
		return
	}

	alpha := t.calculateAlphaLocked(now)
	t.scoreMicros = alpha*latencyMicros + (1-alpha)*t.scoreMicros
	t.lastUpdatedNanos = now
}

func (t *ewmaLatencyTracker) recordError(penalty time.Duration) {
	t.update(penalty)
}

func (t *ewmaLatencyTracker) calculateAlphaLocked(nowNanos int64) float64 {
	if t.fixedAlpha != nil {
		return *t.fixedAlpha
	}
	deltaNanos := nowNanos - t.lastUpdatedNanos
	if deltaNanos <= 0 {
		return 1
	}
	alpha := 1 - math.Exp(-float64(deltaNanos)/float64(t.tau))
	if alpha < 0 {
		return 0
	}
	if alpha > 1 {
		return 1
	}
	return alpha
}
