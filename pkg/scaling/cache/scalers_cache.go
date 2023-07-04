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
	"time"

	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/client-go/tools/record"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	externalscaling "github.com/kedacore/keda/v2/pkg/externalscaling"
	"github.com/kedacore/keda/v2/pkg/scalers"
)

var log = logf.Log.WithName("scalers_cache")

type ScalersCache struct {
	ScaledObject                   *kedav1alpha1.ScaledObject
	Scalers                        []ScalerBuilder
	ScalableObjectGeneration       int64
	Recorder                       record.EventRecorder
	ExternalCalculationGrpcClients []ExternalCalculationClient
}

type ExternalCalculationClient struct {
	Name      string
	Client    *externalscaling.GrpcClient
	Connected bool
}

type ScalerBuilder struct {
	Scaler       scalers.Scaler
	ScalerConfig scalers.ScalerConfig
	Factory      func() (scalers.Scaler, *scalers.ScalerConfig, error)
}

// GetScalers returns array of scalers and scaler config stored in the cache
func (c *ScalersCache) GetScalers() ([]scalers.Scaler, []scalers.ScalerConfig) {
	scalersList := make([]scalers.Scaler, 0, len(c.Scalers))
	configsList := make([]scalers.ScalerConfig, 0, len(c.Scalers))
	for _, s := range c.Scalers {
		scalersList = append(scalersList, s.Scaler)
		configsList = append(configsList, s.ScalerConfig)
	}

	return scalersList, configsList
}

// GetPushScaler returns array of push scalers stored in the cache
func (c *ScalersCache) GetPushScalers() []scalers.PushScaler {
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
	for _, client := range c.ExternalCalculationGrpcClients {
		err := client.Client.CloseConnection()
		if err != nil {
			log.Error(err, fmt.Sprintf("couldn't close grpc connection for externalCalculator '%s'", client.Name))
		} else {
			log.V(0).Info(fmt.Sprintf("successfully closed grpc connection for externalCalculator '%s'", client.Name))
		}
	}
	scalers := c.Scalers
	c.Scalers = nil
	for _, s := range scalers {
		err := s.Scaler.Close(ctx)
		if err != nil {
			log.Error(err, "error closing scaler", "scaler", s)
		}
	}
}

// GetMetricSpecForScaling returns metrics specs for all scalers in the cache
func (c *ScalersCache) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	var spec []v2.MetricSpec
	for _, s := range c.Scalers {
		spec = append(spec, s.Scaler.GetMetricSpecForScaling(ctx)...)
	}
	return spec
}

// GetMetricSpecForScalingForScaler returns metrics spec for a scaler identified by the metric name
func (c *ScalersCache) GetMetricSpecForScalingForScaler(ctx context.Context, index int) ([]v2.MetricSpec, error) {
	var err error

	scalersList, _ := c.GetScalers()
	if index < 0 || index >= len(scalersList) {
		return nil, fmt.Errorf("scaler with id %d not found. Len = %d", index, len(c.Scalers))
	}

	metricSpecs := scalersList[index].GetMetricSpecForScaling(ctx)

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
func (c *ScalersCache) GetMetricsAndActivityForScaler(ctx context.Context, index int, metricName string) ([]external_metrics.ExternalMetricValue, bool, int64, error) {
	if index < 0 || index >= len(c.Scalers) {
		return nil, false, -1, fmt.Errorf("scaler with id %d not found. Len = %d", index, len(c.Scalers))
	}
	startTime := time.Now()
	metric, activity, err := c.Scalers[index].Scaler.GetMetricsAndActivity(ctx, metricName)
	if err == nil {
		return metric, activity, time.Since(startTime).Milliseconds(), nil
	}

	ns, err := c.refreshScaler(ctx, index)
	if err != nil {
		return nil, false, -1, err
	}
	startTime = time.Now()
	metric, activity, err = ns.GetMetricsAndActivity(ctx, metricName)
	return metric, activity, time.Since(startTime).Milliseconds(), err
}

func (c *ScalersCache) refreshScaler(ctx context.Context, id int) (scalers.Scaler, error) {
	if id < 0 || id >= len(c.Scalers) {
		return nil, fmt.Errorf("scaler with id %d not found, len = %d, cache has been probably already invalidated", id, len(c.Scalers))
	}

	sb := c.Scalers[id]
	defer sb.Scaler.Close(ctx)
	ns, sConfig, err := sb.Factory()
	if err != nil {
		return nil, err
	}

	if id < 0 || id >= len(c.Scalers) {
		return nil, fmt.Errorf("scaler with id %d not found, len = %d, cache has been probably already invalidated", id, len(c.Scalers))
	}
	c.Scalers[id] = ScalerBuilder{
		Scaler:       ns,
		ScalerConfig: *sConfig,
		Factory:      sb.Factory,
	}

	return ns, nil
}
