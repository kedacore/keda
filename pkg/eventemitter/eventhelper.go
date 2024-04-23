/*
Copyright 2024 The KEDA Authors

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

package eventemitter

import (
	"fmt"

	"github.com/kedacore/keda/v2/pkg/eventemitter/eventdata"
	"github.com/kedacore/keda/v2/pkg/util"
)

var (
	kedaNamespace, _ = util.GetClusterObjectNamespace()
)

// generateCloudEventSource generates the CloudEventSource for CloudEvent from the given clusterName and kedaNamespace
func generateCloudEventSource(clusterName string) string {
	return fmt.Sprintf("/%s/%s/keda", clusterName, kedaNamespace)
}

// generateCloudEventSubject generates the CloudEventSubject for CloudEvent from the given clusterName, objectNamespace, objectType and objectName
func generateCloudEventSubject(clusterName string, objectNamespace string, objectType string, objectName string) string {
	return fmt.Sprintf("/%s/%s/%s/%s", clusterName, objectNamespace, objectType, objectName)
}

// generateCloudEventSubjectFromEventData generates the CloudEventSubject for CloudEvent from the given clusterName and eventData
func generateCloudEventSubjectFromEventData(clusterName string, eventData eventdata.EventData) string {
	return generateCloudEventSubject(clusterName, eventData.Namespace, eventData.ObjectType, eventData.ObjectName)
}
