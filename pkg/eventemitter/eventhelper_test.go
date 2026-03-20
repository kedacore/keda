/*
Copyright 2026 The KEDA Authors

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
	"testing"

	"github.com/kedacore/keda/v2/pkg/eventemitter/eventdata"
)

func TestGenerateCloudEventSource(t *testing.T) {
	originalNamespace := kedaNamespace
	t.Cleanup(func() { kedaNamespace = originalNamespace })
	kedaNamespace = "keda-system"

	got := generateCloudEventSource("my-cluster")
	want := "/my-cluster/keda-system/keda"

	if got != want {
		t.Errorf("generateCloudEventSource() = %q, want %q", got, want)
	}
}

func TestGenerateCloudEventSubject(t *testing.T) {
	tests := []struct {
		name            string
		clusterName     string
		objectNamespace string
		objectType      string
		objectName      string
		want            string
	}{
		{
			name:            "standard subject",
			clusterName:     "my-cluster",
			objectNamespace: "default",
			objectType:      "scaledobject",
			objectName:      "my-app",
			want:            "/my-cluster/default/scaledobject/my-app",
		},
		{
			name:            "empty fields",
			clusterName:     "",
			objectNamespace: "",
			objectType:      "",
			objectName:      "",
			want:            "////",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateCloudEventSubject(tt.clusterName, tt.objectNamespace, tt.objectType, tt.objectName)
			if got != tt.want {
				t.Errorf("generateCloudEventSubject() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateCloudEventSubjectFromEventData(t *testing.T) {
	ed := eventdata.EventData{
		Namespace:  "production",
		ObjectType: "scaledobject",
		ObjectName: "worker-scaler",
	}

	got := generateCloudEventSubjectFromEventData("cluster-1", ed)
	want := "/cluster-1/production/scaledobject/worker-scaler"

	if got != want {
		t.Errorf("generateCloudEventSubjectFromEventData() = %q, want %q", got, want)
	}
}
