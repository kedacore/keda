/*
Copyright 2023 The KEDA Authors

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

package scalers

import (
	"context"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	mqttMetricType = "External"
)

// mqttScalerMetadata holds the parsed trigger configuration for a
// ScaledObject using `type: mqtt`.
type mqttScalerMetadata struct {
	triggerIndex int

	BrokerAddress string `keda:"name=brokerAddress, order=triggerMetadata"`
	Topic         string `keda:"name=topic, order=triggerMetadata"`
	WindowSeconds int    `keda:"name=windowSeconds, order=triggerMetadata, optional, default=30"`
	QueryValue    int64  `keda:"name=queryValue, order=triggerMetadata"`
	QoS           int    `keda:"name=qos, order=triggerMetadata, optional, default=1"`

	ConnectRetryIntervalSeconds int  `keda:"name=connectRetryIntervalSeconds, order=triggerMetadata, optional, default=2"`
	MaxReconnectIntervalSeconds int  `keda:"name=maxReconnectIntervalSeconds, order=triggerMetadata, optional, default=60"`
	EnableTLS                   bool `keda:"name=enableTLS, order=triggerMetadata, optional"`
	UnsafeSsl                   bool `keda:"name=unsafeSsl, order=triggerMetadata, optional"`

	Username    string `keda:"name=username, order=authParams, optional"`
	Password    string `keda:"name=password, order=authParams, optional"`
	Ca          string `keda:"name=ca, order=authParams, optional"`
	Cert        string `keda:"name=cert, order=authParams, optional"`
	Key         string `keda:"name=key, order=authParams, optional"`
	KeyPassword string `keda:"name=keyPassword, order=authParams, optional"`
}

func parseMqttScalerMetadata(config *scalersconfig.ScalerConfig) (mqttScalerMetadata, error) {
	meta := mqttScalerMetadata{triggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(&meta); err != nil {
		return meta, fmt.Errorf("error parsing mqtt scaler metadata: %w", err)
	}
	return meta, nil
}

// messageWindow is a thread-safe sliding window over recent MQTT
// message arrivals, plus a time-bounded "retained message seen" signal.
//
// It is written to from the MQTT client's own callback goroutine and
// read from GetMetricsAndActivity, which KEDA calls independently and
// concurrently (both the KEDA operator's own polling loop and the
// Kubernetes metrics adapter serving the HPA call this on their own
// schedules) — hence the mutex.
type messageWindow struct {
	mu             sync.Mutex
	timestamps     []time.Time
	window         time.Duration
	lastRetainedAt time.Time // zero value means "never seen"
}

func newMessageWindow(window time.Duration) *messageWindow {
	return &messageWindow{window: window}
}

func (w *messageWindow) record() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.timestamps = append(w.timestamps, time.Now())
}

func (w *messageWindow) markRetained() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastRetainedAt = time.Now()
}

// count prunes timestamps older than the window and returns the current
// message count, plus whether a retained message has been seen within
// the window. Both values are derived from wall-clock time rather than
// being consumed on read, so multiple independent, concurrent callers
// all get a consistent answer regardless of call order.
func (w *messageWindow) count() (int64, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	cutoff := time.Now().Add(-w.window)

	i := 0
	for i < len(w.timestamps) && w.timestamps[i].Before(cutoff) {
		i++
	}
	w.timestamps = w.timestamps[i:]

	retainedActive := !w.lastRetainedAt.IsZero() && w.lastRetainedAt.After(cutoff)
	return int64(len(w.timestamps)), retainedActive
}
