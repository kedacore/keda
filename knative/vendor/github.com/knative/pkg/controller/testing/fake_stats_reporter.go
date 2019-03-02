/*
Copyright 2017 The Knative Authors

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

package testing

import (
	"sync"
	"time"
)

// FakeStatsReporter is a fake implementation of StatsReporter
type FakeStatsReporter struct {
	queueDepths   []int64
	reconcileData []FakeReconcileStatData
	Lock          sync.Mutex
}

// FakeReconcileStatData is used to record the calls to ReportReconcile
type FakeReconcileStatData struct {
	Duration     time.Duration
	Key, Success string
}

// ReportQueueDepth records the call and returns success.
func (r *FakeStatsReporter) ReportQueueDepth(v int64) error {
	r.Lock.Lock()
	defer r.Lock.Unlock()
	r.queueDepths = append(r.queueDepths, v)
	return nil
}

// ReportReconcile records the call and returns success.
func (r *FakeStatsReporter) ReportReconcile(duration time.Duration, key, success string) error {
	r.Lock.Lock()
	defer r.Lock.Unlock()
	r.reconcileData = append(r.reconcileData, FakeReconcileStatData{duration, key, success})
	return nil
}

// GetQueueDepths returns the recorded queue depth values
func (r *FakeStatsReporter) GetQueueDepths() []int64 {
	r.Lock.Lock()
	defer r.Lock.Unlock()
	return r.queueDepths
}

// GetReconcileData returns the recorded reconcile data
func (r *FakeStatsReporter) GetReconcileData() []FakeReconcileStatData {
	r.Lock.Lock()
	defer r.Lock.Unlock()
	return r.reconcileData
}
