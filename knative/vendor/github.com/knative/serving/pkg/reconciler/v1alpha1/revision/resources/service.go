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
	"github.com/knative/pkg/kmeta"
	"github.com/knative/serving/pkg/apis/autoscaling"
	"github.com/knative/serving/pkg/apis/serving"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/revision/resources/names"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// MakeK8sService creates a Kubernetes Service that targets all pods with the same
// serving.RevisionLabelKey label. Traffic is routed to queue-proxy port.
func MakeK8sService(rev *v1alpha1.Revision) *corev1.Service {
	labels := makeLabels(rev)
	labels[autoscaling.KPALabelKey] = names.KPA(rev)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            names.K8sService(rev),
			Namespace:       rev.Namespace,
			Labels:          labels,
			Annotations:     makeAnnotations(rev),
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(rev)},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       ServicePortName(rev),
				Protocol:   corev1.ProtocolTCP,
				Port:       ServicePort,
				TargetPort: intstr.FromString(v1alpha1.RequestQueuePortName),
			}, {
				Name:       MetricsPortName,
				Protocol:   corev1.ProtocolTCP,
				Port:       MetricsPort,
				TargetPort: intstr.FromString(v1alpha1.RequestQueueMetricsPortName),
			}},
			Selector: map[string]string{
				serving.RevisionLabelKey: rev.Name,
			},
		},
	}
}

func ServicePortName(rev *v1alpha1.Revision) string {
	if rev.GetProtocol() == v1alpha1.RevisionProtocolH2C {
		return ServicePortNameH2C
	}

	return ServicePortNameHTTP1
}
