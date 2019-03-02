/*
Copyright 2018 The Knative Authors

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

package autoscaler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"
	kpa "github.com/knative/serving/pkg/apis/autoscaling/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Enough buffer to store scale requests generated every 2
	// seconds while an http request is taking the full timeout of 5
	// second.
	scaleBufferSize = 10
)

// Metric is a resource which observes the request load of a Revision and
// recommends a number of replicas to run.
// +k8s:deepcopy-gen=true
type Metric struct {
	metav1.ObjectMeta
	Spec   MetricSpec
	Status MetricStatus
}

// MetricSpec is the parameters in which the Revision should scaled.
type MetricSpec struct {
	TargetConcurrency float64
}

// MetricStatus is the current scale recommendation.
type MetricStatus struct {
	DesiredScale int32
}

// UniScaler records statistics for a particular Metric and proposes the scale for the Metric's target based on those statistics.
type UniScaler interface {
	// Record records the given statistics.
	Record(context.Context, Stat)

	// Scale either proposes a number of replicas or skips proposing. The proposal is requested at the given time.
	// The returned boolean is true if and only if a proposal was returned.
	Scale(context.Context, time.Time) (int32, bool)

	// Update reconfigures the UniScaler according to the MetricSpec.
	Update(MetricSpec) error
}

// UniScalerFactory creates a UniScaler for a given PA using the given dynamic configuration.
type UniScalerFactory func(*Metric, *DynamicConfig) (UniScaler, error)

// scalerRunner wraps a UniScaler and a channel for implementing shutdown behavior.
type scalerRunner struct {
	scaler UniScaler
	stopCh chan struct{}
	pokeCh chan struct{}

	// mux guards access to metric
	mux    sync.RWMutex
	metric Metric
}

func (sr *scalerRunner) getLatestScale() int32 {
	sr.mux.RLock()
	defer sr.mux.RUnlock()
	return sr.metric.Status.DesiredScale
}

func (sr *scalerRunner) updateLatestScale(new int32) bool {
	sr.mux.Lock()
	defer sr.mux.Unlock()
	if sr.metric.Status.DesiredScale != new {
		sr.metric.Status.DesiredScale = new
		return true
	}
	return false
}

// NewMetricKey identifies a UniScaler in the multiscaler. Stats send in
// are identified and routed via this key.
func NewMetricKey(namespace string, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

// MultiScaler maintains a collection of Uniscalers.
type MultiScaler struct {
	scalers       map[string]*scalerRunner
	scalersMutex  sync.RWMutex
	scalersStopCh <-chan struct{}

	dynConfig *DynamicConfig

	uniScalerFactory UniScalerFactory

	logger *zap.SugaredLogger

	watcher      func(string)
	watcherMutex sync.RWMutex
}

// NewMultiScaler constructs a MultiScaler.
func NewMultiScaler(dynConfig *DynamicConfig, stopCh <-chan struct{}, uniScalerFactory UniScalerFactory, logger *zap.SugaredLogger) *MultiScaler {
	logger.Debugf("Creating MultiScaler with configuration %#v", dynConfig)
	return &MultiScaler{
		scalers:          make(map[string]*scalerRunner),
		scalersStopCh:    stopCh,
		dynConfig:        dynConfig,
		uniScalerFactory: uniScalerFactory,
		logger:           logger,
	}
}

// Get return the current Metric.
func (m *MultiScaler) Get(ctx context.Context, namespace, name string) (*Metric, error) {
	key := NewMetricKey(namespace, name)
	m.scalersMutex.RLock()
	defer m.scalersMutex.RUnlock()
	scaler, exists := m.scalers[key]
	if !exists {
		// This GroupResource is a lie, but unfortunately this interface requires one.
		return nil, errors.NewNotFound(kpa.Resource("Metrics"), key)
	}
	scaler.mux.RLock()
	defer scaler.mux.RUnlock()
	return (&scaler.metric).DeepCopy(), nil
}

// Create instantiates the desired Metric.
func (m *MultiScaler) Create(ctx context.Context, metric *Metric) (*Metric, error) {
	m.scalersMutex.Lock()
	defer m.scalersMutex.Unlock()
	key := NewMetricKey(metric.Namespace, metric.Name)
	scaler, exists := m.scalers[key]
	if !exists {
		var err error
		scaler, err = m.createScaler(ctx, metric)
		if err != nil {
			return nil, err
		}
		m.scalers[key] = scaler
	}
	scaler.mux.RLock()
	defer scaler.mux.RUnlock()
	return (&scaler.metric).DeepCopy(), nil
}

// Update applied the desired MetricSpec to a currently running Metric.
func (m *MultiScaler) Update(ctx context.Context, metric *Metric) (*Metric, error) {
	key := NewMetricKey(metric.Namespace, metric.Name)
	m.scalersMutex.Lock()
	defer m.scalersMutex.Unlock()
	if scaler, exists := m.scalers[key]; exists {
		scaler.mux.Lock()
		defer scaler.mux.Unlock()
		scaler.metric = *metric
		scaler.scaler.Update(metric.Spec)
		return metric, nil
	}
	// This GroupResource is a lie, but unfortunately this interface requires one.
	return nil, errors.NewNotFound(kpa.Resource("Metrics"), key)
}

// Delete stops and removes a Metric.
func (m *MultiScaler) Delete(ctx context.Context, namespace, name string) error {
	key := NewMetricKey(namespace, name)
	m.scalersMutex.Lock()
	defer m.scalersMutex.Unlock()
	if scaler, exists := m.scalers[key]; exists {
		close(scaler.stopCh)
		delete(m.scalers, key)
	}
	return nil
}

// Watch registers a singleton function to call when MetricStatus is updated.
func (m *MultiScaler) Watch(fn func(string)) {
	m.watcherMutex.Lock()
	defer m.watcherMutex.Unlock()

	if m.watcher != nil {
		m.logger.Fatal("Multiple calls to Watch() not supported")
	}
	m.watcher = fn
}

// Inform sends an update to the registered watcher function, if it is set.
func (m *MultiScaler) Inform(event string) bool {
	m.watcherMutex.RLock()
	defer m.watcherMutex.RUnlock()

	if m.watcher != nil {
		m.watcher(event)
		return true
	}
	return false
}

// setScale directly sets the scale for a given metric key. This does not perform any ticking
// or updating of other scaler components.
func (m *MultiScaler) setScale(metricKey string, scale int32) bool {
	scaler, exists := m.scalers[metricKey]
	if !exists {
		return false
	}
	scaler.updateLatestScale(scale)
	return true
}

func (m *MultiScaler) createScaler(ctx context.Context, metric *Metric) (*scalerRunner, error) {

	scaler, err := m.uniScalerFactory(metric, m.dynConfig)
	if err != nil {
		return nil, err
	}

	stopCh := make(chan struct{})
	runner := &scalerRunner{
		scaler: scaler,
		stopCh: stopCh,
		metric: *metric,
		pokeCh: make(chan struct{}),
	}
	runner.metric.Status.DesiredScale = -1

	ticker := time.NewTicker(m.dynConfig.Current().TickInterval)

	scaleChan := make(chan int32, scaleBufferSize)

	go func() {
		for {
			select {
			case <-m.scalersStopCh:
				ticker.Stop()
				return
			case <-stopCh:
				ticker.Stop()
				return
			case <-ticker.C:
				m.tickScaler(ctx, scaler, scaleChan)
			case <-runner.pokeCh:
				m.tickScaler(ctx, scaler, scaleChan)
			}
		}
	}()

	metricKey := NewMetricKey(metric.Namespace, metric.Name)
	go func() {
		for {
			select {
			case <-m.scalersStopCh:
				return
			case <-stopCh:
				return
			case desiredScale := <-scaleChan:
				if runner.updateLatestScale(desiredScale) {
					m.Inform(metricKey)
				}
			}
		}
	}()

	return runner, nil
}

func (m *MultiScaler) tickScaler(ctx context.Context, scaler UniScaler, scaleChan chan<- int32) {
	logger := logging.FromContext(ctx)
	desiredScale, scaled := scaler.Scale(ctx, time.Now())

	if scaled {
		// Cannot scale negative.
		if desiredScale < 0 {
			logger.Errorf("Cannot scale: desiredScale %d < 0.", desiredScale)
			return
		}

		// Don't scale to zero if scale to zero is disabled.
		if desiredScale == 0 && !m.dynConfig.Current().EnableScaleToZero {
			logger.Warn("Cannot scale: Desired scale == 0 && EnableScaleToZero == false.")
			return
		}

		scaleChan <- desiredScale
	}
}

// RecordStat records some statistics for the given Metric.
func (m *MultiScaler) RecordStat(key string, stat Stat) {
	m.scalersMutex.RLock()
	defer m.scalersMutex.RUnlock()

	scaler, exists := m.scalers[key]
	if exists {
		logger := m.logger.With(zap.String(logkey.Key, key))
		ctx := logging.WithLogger(context.TODO(), logger)

		scaler.scaler.Record(ctx, stat)
		if scaler.getLatestScale() == 0 && stat.AverageConcurrentRequests != 0 {
			scaler.pokeCh <- struct{}{}
		}
	}
}
