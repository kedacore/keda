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

package resources

import (
	"context"

	"github.com/knative/pkg/logging"
	"github.com/knative/serving/pkg/apis/autoscaling/v1alpha1"
	"github.com/knative/serving/pkg/autoscaler"
)

// MakeMetric constructs a Metric resource from a PodAutoscaler taking
// into account the PA's ContainerConcurrency and the relevant
// autoscaling annotation.
func MakeMetric(ctx context.Context, pa *v1alpha1.PodAutoscaler, config *autoscaler.Config) *autoscaler.Metric {
	logger := logging.FromContext(ctx)

	target := config.TargetConcurrency(pa.Spec.ContainerConcurrency)
	if mt, ok := pa.Target(); ok {
		annotationTarget := float64(mt)
		if annotationTarget > target {
			// If the annotation target would cause the autoscaler to maintain
			// more requests per pod than the container can handle, we ignore
			// the annotation and use a containerConcurrency based target instead.
			logger.Warnf("Ignoring target of %v because it would underprovision the Revision.", annotationTarget)
		} else {
			logger.Debugf("Using target of %v", annotationTarget)
			target = annotationTarget
		}
	}
	return &autoscaler.Metric{
		ObjectMeta: pa.ObjectMeta,
		Spec: autoscaler.MetricSpec{
			TargetConcurrency: target,
		},
	}
}
