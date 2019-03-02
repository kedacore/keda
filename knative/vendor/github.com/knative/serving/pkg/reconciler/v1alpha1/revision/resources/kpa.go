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
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/knative/pkg/kmeta"
	kpa "github.com/knative/serving/pkg/apis/autoscaling/v1alpha1"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/revision/resources/names"
)

func MakeKPA(rev *v1alpha1.Revision) *kpa.PodAutoscaler {
	return &kpa.PodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:            names.KPA(rev),
			Namespace:       rev.Namespace,
			Labels:          makeLabels(rev),
			Annotations:     makeAnnotations(rev),
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(rev)},
		},
		Spec: kpa.PodAutoscalerSpec{
			ContainerConcurrency: rev.Spec.ContainerConcurrency,
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       names.Deployment(rev),
			},
			ServiceName: names.K8sService(rev),
		},
	}
}
