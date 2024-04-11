/*
Copyright 2023 The KEDA Authors

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

package eventdata

import (
	"time"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
)

// EventData will save all event info and handler info for retry.
type EventData struct {
	Namespace      string
	ObjectName     string
	ObjectType     string
	CloudEventType eventingv1alpha1.CloudEventType
	Reason         string
	Message        string
	Time           time.Time
	HandlerKey     string
	RetryTimes     int
	Err            error
}
