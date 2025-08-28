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
	"fmt"
	"sync"
	"time"

	"github.com/expr-lang/expr/vm"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/client-go/tools/record"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

var log = logf.Log.WithName("scalers_cache")

type ScalersCache struct {
	ScaledObject             *kedav1alpha1.ScaledObject
	Scalers                  []ScalerBuilder
	ScalableObjectGeneration int64
	Recorder                 record.EventRecorder
	CompiledFormula          *vm.Program
	mutex                    sync.RWMutex
}

type ScalerBuilder struct {
	Scaler       scalers.Scaler
	ScalerConfig scalersconfig.ScalerConfig
	Factory      func() (scalers.Scaler, *scalersconfig.ScalerConfig, error)
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

	if index < 0 || index >= len(c.Scalers) {
		return ScalerBuilder{}, fmt.Errorf("scaler with id %d not found. Len = %d", index, len(c.Scalers))
	}

	return c.Scalers[index], nil
}

// GetPushScalers returns array of push scalers stored in the cache
func (c *ScalersCache) GetPushScalers() []scalers.PushScaler {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	var result []scalers.PushScaler
	for _, s := range c.Scalers {
		if ps, ok := s.Scaler.(scalers.PushScaler); ok {
			result = append(result, ps)
		}
	}
	return result
}

// Close closes all scalers in the cache
func (c *ScalersCache) Close(ctx context.Context) {
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

// GetMetricSpecForScaling returns metrics specs for all scalers in the cache
func (c *ScalersCache) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	var spec []v2.MetricSpec
	for _, s := range c.Scalers {
		spec = append(spec, s.Scaler.GetMetricSpecForScaling(ctx)...)
	}
	return spec
}

// GetMetricSpecForScalingForScaler returns metrics spec for a scaler identified by the metric name
func (c *ScalersCache) GetMetricSpecForScalingForScaler(ctx context.Context, index int) ([]v2.MetricSpec, error) {
	var err error

	sb, err := c.getScalerBuilder(index)
	if err != nil {
		return nil, err
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
	sb, err := c.getScalerBuilder(index)
	if err != nil {
		return nil, false, -1, err
	}
	startTime := time.Now()
	metric, activity, err := sb.Scaler.GetMetricsAndActivity(ctx, metricName)
	if err == nil {
		return metric, activity, time.Since(startTime), nil
	}

	ns, err := c.refreshScaler(ctx, index)
	if err != nil {
		return nil, false, -1, err
	}
	startTime = time.Now()
	metric, activity, err = ns.GetMetricsAndActivity(ctx, metricName)
	return metric, activity, time.Since(startTime), err
}

func (c *ScalersCache) refreshScaler(ctx context.Context, index int) (scalers.Scaler, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if index < 0 || index >= len(c.Scalers) {
		return nil, fmt.Errorf("scaler with id %d not found. Len = %d", index, len(c.Scalers))
	}

	oldSb := c.Scalers[index]

	newScaler, sConfig, err := oldSb.Factory()
	if err != nil {
		return nil, err
	}

	c.Scalers[index] = ScalerBuilder{
		Scaler:       newScaler,
		ScalerConfig: *sConfig,
		Factory:      oldSb.Factory,
	}

	oldSb.Scaler.Close(ctx)

	return newScaler, nil
}
