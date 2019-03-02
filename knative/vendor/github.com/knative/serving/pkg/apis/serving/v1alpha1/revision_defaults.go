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

package v1alpha1

const (
	// defaultTimeoutSeconds will be set if timeoutSeconds not specified.
	defaultTimeoutSeconds = 5 * 60
)

func (r *Revision) SetDefaults() {
	r.Spec.SetDefaults()
}

func (rs *RevisionSpec) SetDefaults() {
	// When ConcurrencyModel is specified but ContainerConcurrency
	// is not (0), use the ConcurrencyModel value.
	if rs.DeprecatedConcurrencyModel == RevisionRequestConcurrencyModelSingle && rs.ContainerConcurrency == 0 {
		rs.ContainerConcurrency = 1
	}

	if rs.TimeoutSeconds == 0 {
		rs.TimeoutSeconds = defaultTimeoutSeconds
	}

	vms := rs.Container.VolumeMounts
	for i := range vms {
		vms[i].ReadOnly = true
	}
}
