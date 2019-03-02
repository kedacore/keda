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

package names

import (
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
)

func Deployment(rev *v1alpha1.Revision) string {
	return rev.Name + "-deployment"
}

func ImageCache(rev *v1alpha1.Revision) string {
	return rev.Name + "-cache"
}

func KPA(rev *v1alpha1.Revision) string {
	// We want the KPA's "key" to match the revision,
	// to simplify the transition to the KPA.
	return rev.Name
}

func K8sService(rev *v1alpha1.Revision) string {
	return rev.Name + "-service"
}

func FluentdConfigMap(rev *v1alpha1.Revision) string {
	return rev.Name + "-fluentd"
}
