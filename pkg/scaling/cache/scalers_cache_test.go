/*
Copyright 2025 The KEDA Authors

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

package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.uber.org/atomic"
	v2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

const (
	testCacheUID        = types.UID("test-cache-uid")
	testCacheGeneration = int64(1)
)

func TestBuildScalerRequestCtx(t *testing.T) {
	RegisterTestingT(t)

	sb := ScalerBuilder{
		ScalerConfig: scalersconfig.ScalerConfig{
			TriggerType:             "prometheus",
			TriggerName:             "my-trigger",
			ScalableObjectNamespace: "my-namespace",
			ScalableObjectName:      "my-scaled-object",
		},
	}

	ctx := metricscollector.BuildScalerRequestCtx(context.Background(), sb.ScalerConfig, "my-metric")

	Expect(ctx.Value(metricscollector.ScalerContextKey)).To(Equal("prometheus"))
	Expect(ctx.Value(metricscollector.TriggerNameContextKey)).To(Equal("my-trigger"))
	Expect(ctx.Value(metricscollector.MetricNameContextKey)).To(Equal("my-metric"))
	Expect(ctx.Value(metricscollector.NamespaceContextKey)).To(Equal("my-namespace"))
	Expect(ctx.Value(metricscollector.ScaledResourceContextKey)).To(Equal("my-scaled-object"))
}

func TestEmptyScalersCache(t *testing.T) {
	RegisterTestingT(t)

	cache := &ScalersCache{
		Scalers: make([]ScalerBuilder, 0),
	}
	go func() {
		scalers, configs := cache.GetScalers()
		Expect(scalers).To(BeEmpty())
		Expect(configs).To(BeEmpty())
	}()

	go func() {
		pushScalers := cache.GetPushScalers()
		Expect(pushScalers).To(BeEmpty())
	}()

	go func() {
		metrics := cache.GetMetricSpecForScaling(context.Background())
		Expect(metrics).To(BeEmpty())
	}()

	go func() {
		metrics, err := cache.GetMetricSpecForScalingForScaler(context.Background(), 0)
		Expect(err).To(Not(BeNil()))
		Expect(metrics).To(BeNil())
	}()

	go func() {
		cache.Close(context.Background())
	}()
}

type fakeScaler struct {
	release               chan struct{}
	entered               chan struct{}
	enterOnce             sync.Once
	getMetricsCompletedAt *atomic.Int64
	closeCalledAt         *atomic.Int64
	closeCount            *atomic.Int32
}

func newFakeScaler(release chan struct{}) *fakeScaler {
	return &fakeScaler{
		release:               release,
		entered:               make(chan struct{}),
		getMetricsCompletedAt: atomic.NewInt64(0),
		closeCalledAt:         atomic.NewInt64(0),
		closeCount:            atomic.NewInt32(0),
	}
}

var _ scalers.Scaler = (*fakeScaler)(nil)

func (f *fakeScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	if f.release != nil {
		f.enterOnce.Do(func() { close(f.entered) })
		<-f.release
	}
	f.getMetricsCompletedAt.Store(time.Now().UnixNano())
	return []external_metrics.ExternalMetricValue{{MetricName: metricName}}, true, nil
}

func (f *fakeScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	return []v2.MetricSpec{{
		External: &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{Name: "fake"},
		},
	}}
}

func (f *fakeScaler) Close(_ context.Context) error {
	f.closeCount.Add(1)
	f.closeCalledAt.Store(time.Now().UnixNano())
	return nil
}

func newCacheWithScaler(s scalers.Scaler) *ScalersCache {
	return &ScalersCache{
		ScaledObject: &kedav1alpha1.ScaledObject{
			ObjectMeta: metav1.ObjectMeta{
				UID:        testCacheUID,
				Generation: testCacheGeneration,
			},
		},
		ScalableObjectGeneration: testCacheGeneration,
		Scalers: []ScalerBuilder{{
			Scaler:       s,
			ScalerConfig: scalersconfig.ScalerConfig{},
			Factory: func() (scalers.Scaler, *scalersconfig.ScalerConfig, error) {
				return s, &scalersconfig.ScalerConfig{}, nil
			},
		}},
	}
}

func TestScalersCache_CloseIsIdempotent(t *testing.T) {
	scaler := newFakeScaler(nil)
	cache := newCacheWithScaler(scaler)

	cache.Close(context.Background())
	// Second call must be a no-op and must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("second Close panicked: %v", r)
		}
	}()
	cache.Close(context.Background())
	if got := scaler.closeCount.Load(); got != 1 {
		t.Fatalf("Scaler.Close called %d times, want 1", got)
	}
}

func TestScalersCache_GetMetricsAndActivityForScaler_AfterCloseReturnsErrCacheClosed(t *testing.T) {
	scaler := newFakeScaler(nil)
	cache := newCacheWithScaler(scaler)
	cache.Close(context.Background())

	metrics, active, latency, err := cache.GetMetricsAndActivityForScaler(context.Background(), 0, "fake")
	if !errors.Is(err, ErrCacheClosed) {
		t.Fatalf("err = %v, want ErrCacheClosed", err)
	}
	if metrics != nil {
		t.Errorf("metrics = %v, want nil", metrics)
	}
	if active {
		t.Errorf("active = true, want false")
	}
	if latency != -1 {
		t.Errorf("latency = %v, want -1", latency)
	}
}

func TestScalersCache_GetMetricSpecForScalingForScaler_AfterCloseReturnsErrCacheClosed(t *testing.T) {
	scaler := newFakeScaler(nil)
	cache := newCacheWithScaler(scaler)
	cache.Close(context.Background())

	specs, err := cache.GetMetricSpecForScalingForScaler(context.Background(), 0)
	if !errors.Is(err, ErrCacheClosed) {
		t.Fatalf("err = %v, want ErrCacheClosed", err)
	}
	if specs != nil {
		t.Errorf("specs = %v, want nil", specs)
	}
}

func TestScalersCache_CloseWaitsForInFlightReader(t *testing.T) {
	release := make(chan struct{})
	scaler := newFakeScaler(release)
	cache := newCacheWithScaler(scaler)

	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		_, _, _, err := cache.GetMetricsAndActivityForScaler(context.Background(), 0, "fake")
		if err != nil {
			t.Errorf("reader got unexpected error: %v", err)
		}
	}()

	select {
	case <-scaler.entered:
	case <-time.After(time.Second):
		t.Fatal("reader did not enter GetMetricsAndActivity")
	}

	closeReturned := make(chan struct{})
	go func() {
		defer close(closeReturned)
		cache.Close(context.Background())
	}()

	select {
	case <-closeReturned:
		t.Fatal("Close returned while in-flight reader was still active")
	case <-time.After(50 * time.Millisecond):
	}
	if got := scaler.closeCount.Load(); got != 0 {
		t.Fatalf("Scaler.Close ran before in-flight reader completed (count=%d)", got)
	}

	close(release)

	select {
	case <-readerDone:
	case <-time.After(time.Second):
		t.Fatal("reader did not finish")
	}
	select {
	case <-closeReturned:
	case <-time.After(time.Second):
		t.Fatal("Close did not return")
	}

	if got := scaler.closeCount.Load(); got != 1 {
		t.Fatalf("Scaler.Close called %d times, want 1", got)
	}
	if scaler.closeCalledAt.Load() < scaler.getMetricsCompletedAt.Load() {
		t.Fatal("Scaler.Close ran before reader's GetMetricsAndActivity returned")
	}
}

// A reader stuck in a scaler that ignores ctx (modeled here as a fakeScaler
// whose release channel is never closed) must not be allowed to block Close
// past ReaderDrainBudget. The timer attached at acquireReader fires, releases
// the activeReaders slot, and Close proceeds.
func TestScalersCache_ReaderDrainBudgetUnblocksCloseOnStuckReader(t *testing.T) {
	release := make(chan struct{})
	defer close(release)

	scaler := newFakeScaler(release)
	cache := newCacheWithScaler(scaler)
	cache.ReaderDrainBudget = 50 * time.Millisecond

	go func() {
		_, _, _, _ = cache.GetMetricsAndActivityForScaler(context.Background(), 0, "fake")
	}()
	select {
	case <-scaler.entered:
	case <-time.After(time.Second):
		t.Fatal("reader did not enter GetMetricsAndActivity")
	}

	start := time.Now()
	closeReturned := make(chan struct{})
	go func() {
		defer close(closeReturned)
		cache.Close(context.Background())
	}()

	select {
	case <-closeReturned:
	case <-time.After(time.Second):
		t.Fatal("Close did not return within 1s; ReaderDrainBudget was not honored")
	}

	elapsed := time.Since(start)
	if elapsed < 50*time.Millisecond {
		t.Fatalf("Close returned in %v, before the configured budget", elapsed)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("Close returned in %v, far longer than the configured 50ms budget", elapsed)
	}
	if got := scaler.closeCount.Load(); got != 1 {
		t.Fatalf("Scaler.Close called %d times, want 1 (Close must proceed after budget)", got)
	}
}

// With ReaderDrainBudget at its zero value (no budget), a stuck reader keeps
// Close blocked indefinitely - equivalent to the post-#7737 behavior before
// any timeboxing. This is the documented opt-in semantics for callers that
// don't set the budget.
func TestScalersCache_NoBudgetLeavesReaderUnbounded(t *testing.T) {
	release := make(chan struct{})
	scaler := newFakeScaler(release)
	cache := newCacheWithScaler(scaler)
	// cache.ReaderDrainBudget left at the zero value.

	go func() {
		_, _, _, _ = cache.GetMetricsAndActivityForScaler(context.Background(), 0, "fake")
	}()
	select {
	case <-scaler.entered:
	case <-time.After(time.Second):
		t.Fatal("reader did not enter GetMetricsAndActivity")
	}

	closeReturned := make(chan struct{})
	go func() {
		defer close(closeReturned)
		cache.Close(context.Background())
	}()

	select {
	case <-closeReturned:
		t.Fatal("Close returned while reader was stuck and ReaderDrainBudget=0")
	case <-time.After(150 * time.Millisecond):
	}

	close(release)
	select {
	case <-closeReturned:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after reader released")
	}
}

// A reader that returns before the budget elapses must release the slot via
// the natural defer path (timer is stopped), not via the timer goroutine. The
// activeReaders counter is decremented exactly once regardless.
func TestScalersCache_FastReaderDoesNotTripBudgetTimer(t *testing.T) {
	scaler := newFakeScaler(nil) // never blocks
	cache := newCacheWithScaler(scaler)
	cache.ReaderDrainBudget = time.Second

	for i := 0; i < 10; i++ {
		if _, _, _, err := cache.GetMetricsAndActivityForScaler(context.Background(), 0, "fake"); err != nil {
			t.Fatalf("reader %d got unexpected error: %v", i, err)
		}
	}

	// If sync.Once weren't gating Done(), Close() would either panic on a
	// negative WaitGroup counter or block forever. Either way it'd fail here.
	closeReturned := make(chan struct{})
	go func() {
		defer close(closeReturned)
		cache.Close(context.Background())
	}()
	select {
	case <-closeReturned:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after fast readers completed")
	}
}

func TestScalersCache_ConcurrentCloseAndRead(t *testing.T) {
	const iterations = 200
	for i := 0; i < iterations; i++ {
		scaler := newFakeScaler(nil)
		cache := newCacheWithScaler(scaler)

		var wg sync.WaitGroup
		wg.Add(2)

		var readErr error
		go func() {
			defer wg.Done()
			_, _, _, readErr = cache.GetMetricsAndActivityForScaler(context.Background(), 0, "fake")
		}()
		go func() {
			defer wg.Done()
			cache.Close(context.Background())
		}()
		wg.Wait()

		if readErr != nil && !errors.Is(readErr, ErrCacheClosed) {
			t.Fatalf("iteration %d: unexpected error from concurrent read: %v", i, readErr)
		}
	}
}

func TestScalersCache_UpdateMetricSpecForScaler(t *testing.T) {
	scaler := newFakeScaler(nil)
	c := newCacheWithScaler(scaler)

	newSpecs := []v2.MetricSpec{{
		External: &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name:     "updated-metric",
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"source": "stream"}},
			},
		},
		Type: "External",
	}}

	if !c.UpdateMetricSpecForScaler(0, newSpecs, testCacheUID, testCacheGeneration) {
		t.Fatal("UpdateMetricSpecForScaler should report true for a valid index")
	}

	// Mutate the caller-owned slice after storing it; the cache must keep its own deep copy.
	newSpecs[0].External.Metric.Name = "mutated"
	newSpecs[0].External.Metric.Selector.MatchLabels["source"] = "mutated"

	specs := c.GetMetricSpecForScaling(context.Background())
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if specs[0].External.Metric.Name != "updated-metric" {
		t.Fatalf("expected updated-metric, got %s", specs[0].External.Metric.Name)
	}
	if specs[0].External.Metric.Selector.MatchLabels["source"] != "stream" {
		t.Fatalf("expected selector source=stream, got %q", specs[0].External.Metric.Selector.MatchLabels["source"])
	}
}

func TestScalersCache_UpdateMetricSpecForScaler_InvalidIndex(t *testing.T) {
	scaler := newFakeScaler(nil)
	c := newCacheWithScaler(scaler)

	if c.UpdateMetricSpecForScaler(-1, []v2.MetricSpec{}, testCacheUID, testCacheGeneration) {
		t.Fatal("UpdateMetricSpecForScaler should report false for a negative index")
	}
	if c.UpdateMetricSpecForScaler(5, []v2.MetricSpec{}, testCacheUID, testCacheGeneration) {
		t.Fatal("UpdateMetricSpecForScaler should report false for an out-of-range index")
	}
}

func TestScalersCache_UpdateMetricSpecForScaler_IdentityMismatch(t *testing.T) {
	scaler := newFakeScaler(nil)
	c := newCacheWithScaler(scaler)

	specs := []v2.MetricSpec{{
		External: &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{Name: "updated-metric"},
		},
		Type: "External",
	}}

	if c.UpdateMetricSpecForScaler(0, specs, types.UID("other-uid"), testCacheGeneration) {
		t.Fatal("UpdateMetricSpecForScaler should report false for a mismatched UID")
	}
	if c.UpdateMetricSpecForScaler(0, specs, testCacheUID, testCacheGeneration+1) {
		t.Fatal("UpdateMetricSpecForScaler should report false for a mismatched generation")
	}

	if got := c.GetMetricSpecForScaling(context.Background()); len(got) != 1 || got[0].External.Metric.Name != "fake" {
		t.Fatalf("cache specs must be untouched on identity mismatch, got %+v", got)
	}
}

func TestScalersCache_GetMetricSpecForScalingForScaler_UsesCachedSpecs(t *testing.T) {
	scaler := newFakeScaler(nil)
	c := newCacheWithScaler(scaler)

	cachedSpecs := []v2.MetricSpec{{
		External: &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name:     "cached-metric",
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"owner": "cache"}},
			},
		},
		Type: "External",
	}}
	c.UpdateMetricSpecForScaler(0, cachedSpecs, testCacheUID, testCacheGeneration)

	specs, err := c.GetMetricSpecForScalingForScaler(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec, got %d", len(specs))
	}
	if specs[0].External.Metric.Name != "cached-metric" {
		t.Fatalf("expected cached-metric, got %s", specs[0].External.Metric.Name)
	}

	// Mutating the returned spec must not bleed back into the cache.
	specs[0].External.Metric.Name = "mutated"
	specs[0].External.Metric.Selector.MatchLabels["owner"] = "mutated"

	specsAgain, err := c.GetMetricSpecForScalingForScaler(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error on second read: %v", err)
	}
	if specsAgain[0].External.Metric.Name != "cached-metric" {
		t.Fatalf("expected cached-metric on second read, got %s", specsAgain[0].External.Metric.Name)
	}
	if specsAgain[0].External.Metric.Selector.MatchLabels["owner"] != "cache" {
		t.Fatalf("expected selector owner=cache on second read, got %q", specsAgain[0].External.Metric.Selector.MatchLabels["owner"])
	}
}
