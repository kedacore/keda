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

package reconciler

import (
	"fmt"

	"strings"

	"github.com/knative/serving/pkg/utils"
)

const suffix = "-service"

// GetK8sServiceFullname returns service full name
func GetK8sServiceFullname(name string, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.%s", name, namespace, utils.GetClusterDomainName())
}

// GetServingK8SServiceNameForObj returns the service name for the object
func GetServingK8SServiceNameForObj(name string) string {
	return name + suffix
}

// GetServingRevisionNameForK8sService returns the revision name from the service name
func GetServingRevisionNameForK8sService(name string) string {
	return strings.TrimSuffix(name, suffix)
}
