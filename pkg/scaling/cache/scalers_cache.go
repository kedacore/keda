/*
Copyright 2021 The KEDA Authors

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
	"fmt"
	"sync"
	"time"

	"github.com/expr-lang/expr/vm"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

var log = logf.Log.WithName("scalers_cache")

// ErrCacheClosed is returned by reader methods when the cache has been closed.
var ErrCacheClosed = errors.New("scalers cache is closed")

type ScalersCache struct {
	ScaledObjectUpdateLock   sync.RWMutex // serializes updates to the ScaledObject
	ScaledObject             *kedav1alpha1.ScaledObject
	Scalers                  []ScalerBuilder
	ScalableObjectGeneration int64
	Recorder                 events.EventRecorder
	CompiledFormula          *vm.Program
	ReaderDrainBudget        time.Duration
	mutex                    sync.RWMutex
	closed                   bool
	activeReaders            sync.WaitGroup
}

// acquireReader either reserves an activeReaders slot or returns ErrCacheClosed if the cache has been closed. The returned release function should be called by defer statement.
// If ReaderDrainBudget > 0, the slot is auto-released by a timer if release is not called within the budget - this keeps activeReaders.Wait() bounded even if a reader is stuck in a third-party SDK that ignores ctx.
// If ReaderDrainBudget <= 0 means "no budget"; the slot is held until release is called.
func (c *ScalersCache) acquireReader() (release func(), err error) {
	c.mutex.RLock()
	if c.closed {
		c.mutex.RUnlock()
		return nil, ErrCacheClosed
	}
	c.activeReaders.Add(1)
	c.mutex.RUnlock()

	var once sync.Once
	done := func() { once.Do(c.activeReaders.Done) }

	if c.ReaderDrainBudget <= 0 {
		return done, nil
	}

	timer := time.AfterFunc(c.ReaderDrainBudget, func() {
		fired := false
		once.Do(func() {
			c.activeReaders.Done()
			fired = true
		})
		if fired {
			log.Info("scaler reader exceeded ReaderDrainBudget; releasing activeReaders slot to avoid blocking cache.Close", "budget", c.ReaderDrainBudget)
		}
	})

	return func() {
		timer.Stop()
		done()
	}, nil
}

type ScalerBuilder struct {
	Scaler            scalers.Scaler
	ScalerConfig      scalersconfig.ScalerConfig
	Factory           func() (scalers.Scaler, *scalersconfig.ScalerConfig, error)
	CachedMetricSpecs []v2.MetricSpec
}

func cloneMetricSpecs(specs []v2.MetricSpec) []v2.MetricSpec {
	if specs == nil {
		return nil
	}

	cloned := make([]v2.MetricSpec, 0, len(specs))
	for i := range specs {
		spec := specs[i]
		cloned = append(cloned, *spec.DeepCopy())
	}
	return cloned
}

// GetScalers returns array of scalers and scaler config stored in the cache
func (c *ScalersCache) GetScalers() ([]scalers.Scaler, []scalersconfig.ScalerConfig) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	scalersList := make([]scalers.Scaler, 0, len(c.Scalers))
	configsList := make([]scalersconfig.ScalerConfig, 0, len(c.Scalers))
	for _, s := range c.Scalers {
		scalersList = append(scalersList, s.Scaler)
		configsList = append(configsList, s.ScalerConfig)
	}

	return scalersList, configsList
}

// getScalerBuilder returns a ScalerBuilder stored in the cache
func (c *ScalersCache) getScalerBuilder(index int) (ScalerBuilder, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.closed {
		return ScalerBuilder{}, ErrCacheClosed
	}

	if index < 0 || index >= len(c.Scalers) {
		return ScalerBuilder{}, fmt.Errorf("scaler with id %d not found. Len = %d", index, len(c.Scalers))
	}

	return c.Scalers[index], nil
}

// PushScalerWithTriggerIndex pairs a push scaler with its trigger index.
type PushScalerWithTriggerIndex struct {
	Scaler       scalers.PushScaler
	TriggerIndex int
}

// GetPushScalers returns push scalers with their trigger indices.
func (c *ScalersCache) GetPushScalers() []PushScalerWithTriggerIndex {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	var result []PushScalerWithTriggerIndex
	for _, s := range c.Scalers {
		if ps, ok := s.Scaler.(scalers.PushScaler); ok {
			result = append(result, PushScalerWithTriggerIndex{Scaler: ps, TriggerIndex: s.ScalerConfig.TriggerIndex})
		}
	}
	return result
}

// Close closes all scalers in the cache. It is idempotent and waits for active readers to finish before tearing down the underlying scalers.
func (c *ScalersCache) Close(ctx context.Context) {
	c.mutex.Lock()
	if c.closed {
		c.mutex.Unlock()
		return
	}
	c.closed = true
	c.mutex.Unlock()

	c.activeReaders.Wait()

	c.mutex.Lock()
	scalers := c.Scalers
	c.Scalers = nil
	c.mutex.Unlock()

	for _, s := range scalers {
		err := s.Scaler.Close(ctx)
		if err != nil {
			log.Error(err, "error closing scaler", "scaler", s)
		}
	}
}

// GetMetricSpecForScaling returns metrics specs for all scalers in the cache.
// If a scaler has cached metric specs from StreamMetricSpec, those take precedence.
func (c *ScalersCache) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	var spec []v2.MetricSpec
	for _, s := range c.Scalers {
		if s.CachedMetricSpecs != nil {
			spec = append(spec, cloneMetricSpecs(s.CachedMetricSpecs)...)
		} else {
			spec = append(spec, s.Scaler.GetMetricSpecForScaling(ctx)...)
		}
	}
	return spec
}

// GetMetricSpecForScalingForScaler returns metrics spec for a scaler identified by the metric name.
// If the scaler has cached metric specs from StreamMetricSpec, those take precedence.
func (c *ScalersCache) GetMetricSpecForScalingForScaler(ctx context.Context, index int) ([]v2.MetricSpec, error) {
	release, err := c.acquireReader()
	if err != nil {
		return nil, err
	}
	defer release()

	sb, err := c.getScalerBuilder(index)
	if err != nil {
		return nil, err
	}

	if sb.CachedMetricSpecs != nil {
		return cloneMetricSpecs(sb.CachedMetricSpecs), nil
	}

	metricSpecs := sb.Scaler.GetMetricSpecForScaling(ctx)

	// no metric spec returned for a scaler -> this could signal error during connection to the scaler
	// usually in case this is an external scaler
	// let's try to refresh the scaler and query metrics spec again
	if len(metricSpecs) < 1 {
		var ns scalers.Scaler
		ns, err = c.refreshScaler(ctx, index)
		if err == nil {
			metricSpecs = ns.GetMetricSpecForScaling(ctx)
			if len(metricSpecs) < 1 {
				err = fmt.Errorf("got empty metric spec")
			}
		}
	}

	return metricSpecs, err
}

// GetMetricsAndActivityForScaler returns metric value, activity and latency for a scaler identified by the metric name
// and by the input index (from the list of scalers in this ScaledObject)
func (c *ScalersCache) GetMetricsAndActivityForScaler(ctx context.Context, index int, metricName string) ([]external_metrics.ExternalMetricValue, bool, time.Duration, error) {
	release, err := c.acquireReader()
	if err != nil {
		return nil, false, -1, err
	}
	defer release()

	sb, err := c.getScalerBuilder(index)
	if err != nil {
		return nil, false, -1, err
	}
	requestCtx := metricscollector.BuildScalerRequestCtx(ctx, sb.ScalerConfig, metricName)
	startTime := time.Now()
	metric, activity, err := sb.Scaler.GetMetricsAndActivity(requestCtx, metricName)
	if err == nil {
		return metric, activity, time.Since(startTime), nil
	}

	ns, err := c.refreshScaler(ctx, index)
	if err != nil {
		return nil, false, -1, err
	}
	newSb, err := c.getScalerBuilder(index)
	if err != nil {
		return nil, false, -1, err
	}
	requestCtx = metricscollector.BuildScalerRequestCtx(ctx, newSb.ScalerConfig, metricName)
	startTime = time.Now()
	metric, activity, err = ns.GetMetricsAndActivity(requestCtx, metricName)
	return metric, activity, time.Since(startTime), err
}

// UpdateMetricSpecForScaler replaces the cached metric specs for a given scaler
// index, but only when the cache still belongs to the ScaledObject identified by
// uid and generation. The identity guard prevents a stale streaming watcher
// (bound to an older ScaledObject generation, or to an object that was deleted
// and recreated under the same name) from overwriting the specs of an unrelated
// cache. A cache rebuilt for the same UID and generation (e.g. after a scaler
// error) keeps the same identity, so legitimate cache invalidation still
// receives streamed updates.
//
// The identity check and the write are performed under the same lock so the
// validated cache cannot silently become stale between the two operations. It
// reports whether the update was applied (false if the cache is closed, the
// index is out of range, or the identity does not match).
func (c *ScalersCache) UpdateMetricSpecForScaler(index int, specs []v2.MetricSpec, uid types.UID, generation int64) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed || index < 0 || index >= len(c.Scalers) {
		return false
	}

	if c.ScaledObject == nil || c.ScaledObject.UID != uid || c.ScalableObjectGeneration != generation {
		return false
	}

	c.Scalers[index].CachedMetricSpecs = cloneMetricSpecs(specs)
	return true
}

func (c *ScalersCache) refreshScaler(ctx context.Context, index int) (scalers.Scaler, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return nil, ErrCacheClosed
	}

	if index < 0 || index >= len(c.Scalers) {
		return nil, fmt.Errorf("scaler with id %d not found. Len = %d", index, len(c.Scalers))
	}

	oldSb := c.Scalers[index]

	newScaler, sConfig, err := oldSb.Factory()
	if err != nil {
		return nil, err
	}

	c.Scalers[index] = ScalerBuilder{
		Scaler:            newScaler,
		ScalerConfig:      *sConfig,
		Factory:           oldSb.Factory,
		CachedMetricSpecs: cloneMetricSpecs(oldSb.CachedMetricSpecs),
	}

	oldSb.Scaler.Close(ctx)

	return newScaler, nil
}
