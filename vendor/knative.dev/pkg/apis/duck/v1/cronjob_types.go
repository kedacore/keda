/*
Copyright 2021 The Knative Authors

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

package v1

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck/ducktypes"
)

// +genduck

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CronJob is a wrapper around CronJob resource, which supports our interfaces
// for webhooks
type CronJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec batchv1.CronJobSpec `json:"spec,omitempty"`
}

// Verify CronJob resources meet duck contracts.
var (
	_ apis.Validatable      = (*CronJob)(nil)
	_ apis.Defaultable      = (*CronJob)(nil)
	_ apis.Listable         = (*CronJob)(nil)
	_ ducktypes.Populatable = (*CronJob)(nil)
)

// GetFullType implements duck.Implementable
func (c *CronJob) GetFullType() ducktypes.Populatable {
	return &CronJob{}
}

// Populate implements duck.Populatable
func (c *CronJob) Populate() {
	c.Spec = batchv1.CronJobSpec{
		JobTemplate: batchv1.JobTemplateSpec{
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:  "container-name",
							Image: "container-image:latest",
						}},
					},
				},
			},
		},
	}
}

// GetListType implements apis.Listable
func (c *CronJob) GetListType() runtime.Object {
	return &CronJob{}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CronJobList is a list of CronJob resources
type CronJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CronJob `json:"items"`
}
