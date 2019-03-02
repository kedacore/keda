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
	"github.com/knative/serving/pkg/apis/serving"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// makeLabels constructs the labels we will apply to K8s resources.
func makeLabels(revision *v1alpha1.Revision) map[string]string {
	labels := make(map[string]string, len(revision.ObjectMeta.Labels)+3)

	labels[serving.RevisionLabelKey] = revision.Name
	labels[serving.RevisionUID] = string(revision.UID)

	for k, v := range revision.ObjectMeta.Labels {
		labels[k] = v
	}

	// If users don't specify an app: label we will automatically
	// populate it with the revision name to get the benefit of richer
	// tracing information.
	if _, ok := labels[AppLabelKey]; !ok {
		labels[AppLabelKey] = revision.Name
	}
	return labels
}

// makeSelector constructs the Selector we will apply to K8s resources.
func makeSelector(revision *v1alpha1.Revision) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			serving.RevisionUID: string(revision.UID),
		},
	}
}

// makeAnnotations creates the annotations we will apply to
// child resource of the given revision.
func makeAnnotations(revision *v1alpha1.Revision) map[string]string {
	annotations := make(map[string]string, len(revision.ObjectMeta.Annotations))
	for k, v := range revision.ObjectMeta.Annotations {
		// Don't propagate known-volatile annotations on the Revision
		// (e.g. our lastPinned heartbeat) to the Deployment or Pods.
		if k == serving.RevisionLastPinnedAnnotationKey {
			continue
		}
		annotations[k] = v
	}
	return annotations
}
