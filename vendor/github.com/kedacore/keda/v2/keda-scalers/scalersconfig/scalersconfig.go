/*
Copyright 2024 The KEDA Authors

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

package scalersconfig

import (
	"time"

	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

// ScalerConfig contains config fields common for all scalers
type ScalerConfig struct {
	// ScalableObjectName specifies name of the ScaledObject/ScaledJob that owns this scaler
	ScalableObjectName string

	// ScalableObjectNamespace specifies name of the ScaledObject/ScaledJob that owns this scaler
	ScalableObjectNamespace string

	// ScalableObjectType specifies whether this Scaler is owned by ScaledObject or ScaledJob
	ScalableObjectType string

	// The timeout to be used on all HTTP requests from the controller
	GlobalHTTPTimeout time.Duration

	// Name of the trigger
	TriggerName string

	// Trigger type (name of the trigger, also the scaler name)
	TriggerType string

	// Marks whether we should query metrics only during the polling interval
	// Any requests for metrics in between are read from the cache
	TriggerUseCachedMetrics bool

	// TriggerMetadata
	TriggerMetadata map[string]string

	// ResolvedEnv
	ResolvedEnv map[string]string

	// AuthParams
	AuthParams map[string]string

	// PodIdentity
	PodIdentity kedav1alpha1.AuthPodIdentity

	// TriggerIndex
	TriggerIndex int

	// TriggerUniqueKey for the scaler across KEDA. Useful to identify uniquely the scaler, eg: AWS credentials cache
	TriggerUniqueKey string

	// MetricType
	MetricType v2.MetricTargetType

	// When we use the scaler for composite scaler, we shouldn't require the value because it'll be ignored
	AsMetricSource bool

	// For events
	Recorder record.EventRecorder

	// ScaledObject
	ScaledObject runtime.Object
}
