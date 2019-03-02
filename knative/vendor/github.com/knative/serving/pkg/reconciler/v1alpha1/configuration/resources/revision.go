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
	"fmt"

	"github.com/knative/pkg/kmeta"
	"github.com/knative/serving/pkg/apis/serving"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// MakeRevision creates a revision object from configuration and build reference.
func MakeRevision(config *v1alpha1.Configuration, buildRef *corev1.ObjectReference) *v1alpha1.Revision {
	// Start from the ObjectMeta/Spec inlined in the Configuration resources.
	rev := &v1alpha1.Revision{
		ObjectMeta: config.Spec.RevisionTemplate.ObjectMeta,
		Spec:       config.Spec.RevisionTemplate.Spec,
	}
	// Populate the Namespace and Name.
	rev.Namespace = config.Namespace
	rev.GenerateName = config.Name + "-"

	UpdateRevisionLabels(rev, config)

	// Populate the Configuration Generation annotation.
	if rev.Annotations == nil {
		rev.Annotations = make(map[string]string)
	}

	// Populate OwnerReferences so that deletes cascade.
	rev.OwnerReferences = append(rev.OwnerReferences, *kmeta.NewControllerRef(config))

	// Fill in buildRef if build is involved
	rev.Spec.BuildRef = buildRef

	return rev
}

// UpdateRevisionLabels sets the revisions labels given a Configuration.
func UpdateRevisionLabels(rev *v1alpha1.Revision, config *v1alpha1.Configuration) {
	if rev.Labels == nil {
		rev.Labels = make(map[string]string)
	}

	for _, key := range []string{
		serving.ConfigurationLabelKey,
		serving.ServiceLabelKey,
		serving.ConfigurationGenerationLabelKey,
		serving.DeprecatedConfigurationMetadataGenerationLabelKey,
	} {
		rev.Labels[key] = RevisionLabelValueForKey(key, config)
	}
}

// RevisionLabelValueForKey returns the label value for the given key.
func RevisionLabelValueForKey(key string, config *v1alpha1.Configuration) string {
	switch key {
	case serving.ConfigurationLabelKey:
		return config.Name
	case serving.ServiceLabelKey:
		return config.Labels[serving.ServiceLabelKey]
	case serving.ConfigurationGenerationLabelKey:
		return fmt.Sprintf("%d", config.Generation)
	case serving.DeprecatedConfigurationMetadataGenerationLabelKey:
		return fmt.Sprintf("%d", config.Generation)
	}

	return ""
}
