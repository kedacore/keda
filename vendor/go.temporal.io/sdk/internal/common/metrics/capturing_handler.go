// The MIT License
//
// Copyright (c) 2021 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// This file contains test helpers only. They are not private because they are used by other tests.

type capturedInfo struct {
	sliceLock sync.RWMutex // Only governs slice access, not what's in the slice
	counters  []*CapturedCounter
	gauges    []*CapturedGauge
	timers    []*CapturedTimer
}

// CapturingHandler is a Handler that retains counted values locally.
type CapturingHandler struct {
	*capturedInfo
	// Never changed once created
	tags map[string]string
}

var _ Handler = &CapturingHandler{}

// NewCapturingHandler creates a new CapturingHandler.
func NewCapturingHandler() *CapturingHandler { return &CapturingHandler{capturedInfo: &capturedInfo{}} }

// Clear removes all known metrics from the root handler.
func (c *CapturingHandler) Clear() {
	c.sliceLock.Lock()
	defer c.sliceLock.Unlock()
	c.counters = nil
	c.gauges = nil
	c.timers = nil
}

// WithTags implements Handler.WithTags.
func (c *CapturingHandler) WithTags(tags map[string]string) Handler {
	ret := &CapturingHandler{capturedInfo: c.capturedInfo, tags: make(map[string]string)}
	for k, v := range c.tags {
		ret.tags[k] = v
	}
	for k, v := range tags {
		ret.tags[k] = v
	}
	return ret
}

// Counter implements Handler.Counter.
func (c *CapturingHandler) Counter(name string) Counter {
	c.sliceLock.Lock()
	defer c.sliceLock.Unlock()
	// Try to find one or create otherwise
	var ret *CapturedCounter
	for _, counter := range c.counters {
		if counter.Name == name && counter.equalTags(c.tags) {
			ret = counter
			break
		}
	}
	if ret == nil {
		ret = &CapturedCounter{CapturedMetricMeta: CapturedMetricMeta{Name: name, Tags: c.tags}}
		c.counters = append(c.counters, ret)
	}
	return ret
}

// Counters returns shallow copy of the local counters. New counters will not
// get added here, but the value within the counter may still change.
func (c *CapturingHandler) Counters() []*CapturedCounter {
	c.sliceLock.RLock()
	defer c.sliceLock.RUnlock()
	ret := make([]*CapturedCounter, len(c.counters))
	copy(ret, c.counters)
	return ret
}

// Gauge implements Handler.Gauge.
func (c *CapturingHandler) Gauge(name string) Gauge {
	c.sliceLock.Lock()
	defer c.sliceLock.Unlock()
	// Try to find one or create otherwise
	var ret *CapturedGauge
	for _, gauge := range c.gauges {
		if gauge.Name == name && gauge.equalTags(c.tags) {
			ret = gauge
			break
		}
	}
	if ret == nil {
		ret = &CapturedGauge{CapturedMetricMeta: CapturedMetricMeta{Name: name, Tags: c.tags}}
		c.gauges = append(c.gauges, ret)
	}
	return ret
}

// Gauges returns shallow copy of the local gauges. New gauges will not get
// added here, but the value within the gauge may still change.
func (c *CapturingHandler) Gauges() []*CapturedGauge {
	c.sliceLock.RLock()
	defer c.sliceLock.RUnlock()
	ret := make([]*CapturedGauge, len(c.gauges))
	copy(ret, c.gauges)
	return ret
}

// Timer implements Handler.Timer.
func (c *CapturingHandler) Timer(name string) Timer {
	c.sliceLock.Lock()
	defer c.sliceLock.Unlock()
	// Try to find one or create otherwise
	var ret *CapturedTimer
	for _, timer := range c.timers {
		if timer.Name == name && timer.equalTags(c.tags) {
			ret = timer
			break
		}
	}
	if ret == nil {
		ret = &CapturedTimer{CapturedMetricMeta: CapturedMetricMeta{Name: name, Tags: c.tags}}
		c.timers = append(c.timers, ret)
	}
	return ret
}

// Timers returns shallow copy of the local timers. New timers will not get
// added here, but the value within the timer may still change.
func (c *CapturingHandler) Timers() []*CapturedTimer {
	c.sliceLock.RLock()
	defer c.sliceLock.RUnlock()
	ret := make([]*CapturedTimer, len(c.timers))
	copy(ret, c.timers)
	return ret
}

// CapturedMetricMeta is common information for captured metrics. These fields
// should never by mutated.
type CapturedMetricMeta struct {
	Name string
	Tags map[string]string
}

func (c *CapturedMetricMeta) equalTags(other map[string]string) bool {
	if len(c.Tags) != len(other) {
		return false
	}
	for k, v := range c.Tags {
		if otherV, ok := other[k]; !ok || otherV != v {
			return false
		}
	}
	return true
}

// CapturedCounter atomically implements Counter and provides an atomic getter.
type CapturedCounter struct {
	CapturedMetricMeta
	value int64
}

// Inc implements Counter.Inc.
func (c *CapturedCounter) Inc(d int64) { atomic.AddInt64(&c.value, d) }

// Value atomically returns the current value.
func (c *CapturedCounter) Value() int64 { return atomic.LoadInt64(&c.value) }

// CapturedGauge atomically implements Gauge and provides an atomic getter.
type CapturedGauge struct {
	CapturedMetricMeta
	value     float64
	valueLock sync.RWMutex
}

// Update implements Gauge.Update.
func (c *CapturedGauge) Update(d float64) {
	c.valueLock.Lock()
	defer c.valueLock.Unlock()
	c.value = d
}

// Value atomically returns the current value.
func (c *CapturedGauge) Value() float64 {
	c.valueLock.RLock()
	defer c.valueLock.RUnlock()
	return c.value
}

// CapturedTimer atomically implements Timer and provides an atomic getter.
type CapturedTimer struct {
	CapturedMetricMeta
	value int64
}

// Record implements Timer.Record.
func (c *CapturedTimer) Record(d time.Duration) { atomic.StoreInt64(&c.value, int64(d)) }

// Value atomically returns the current value.
func (c *CapturedTimer) Value() time.Duration { return time.Duration(atomic.LoadInt64(&c.value)) }
