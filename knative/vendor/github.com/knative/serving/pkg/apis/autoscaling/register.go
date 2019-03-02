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

package autoscaling

const (
	InternalGroupName = "autoscaling.internal.knative.dev"

	GroupName = "autoscaling.knative.dev"

	// ClassAnnotationKey is the annotation for the explicit class of autoscaler
	// that a particular resource has opted into. For example,
	//   autoscaling.knative.dev/class: foo
	// This uses a different domain because unlike the resource, it is user-facing.
	ClassAnnotationKey = GroupName + "/class"
	// KPA is Knative Horizontal Pod Autoscaler
	KPA = "kpa.autoscaling.knative.dev"
	// HPA is Kubernetes Horizontal Pod Autoscaler
	HPA = "hpa.autoscaling.knative.dev"

	// MinScaleAnnotationKey is the annotation to specify the minimum number of Pods
	// the PodAutoscaler should provision. For example,
	//   autoscaling.knative.dev/minScale: "1"
	MinScaleAnnotationKey = GroupName + "/minScale"
	// MinScaleAnnotationKey is the annotation to specify the maximum number of Pods
	// the PodAutoscaler should provision. For example,
	//   autoscaling.knative.dev/maxScale: "10"
	MaxScaleAnnotationKey = GroupName + "/maxScale"

	// MetricAnnotationKey is the annotation to specify what metric the PodAutoscaler
	// should be scaled on. For example,
	//   autoscaling.knative.dev/metric: cpu
	MetricAnnotationKey = GroupName + "/metric"
	// Concurrency is the number of requests in-flight at any given time.
	Concurrency = "concurrency"
	// CPU is the amount of the requested cpu actually being consumed by the Pod.
	CPU = "cpu"

	// TargetAnnotationKey is the annotation to specify what metric value the
	// PodAutoscaler should attempt to maintain. For example,
	//   autoscaling.knative.dev/metric: cpu
	//   autoscaling.knative.dev/target: 75   # target 75% cpu utilization
	TargetAnnotationKey = GroupName + "/target"

	// KPALabelKey is the label key attached to a K8s Service to hint to the KPA
	// which services/endpoints should trigger reconciles.
	KPALabelKey = GroupName + "/kpa"
)
