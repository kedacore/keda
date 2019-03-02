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

package testing

import (
	"fmt"

	"k8s.io/client-go/tools/record"
)

// EventList exports all events during reconciliation through fake event recorder
// with event channel with buffer of given size.
type EventList struct {
	Recorder *record.FakeRecorder
}

// Events iterates over events received from channel in fake event recorder and returns all.
func (l EventList) Events() []string {
	close(l.Recorder.Events)
	events := []string{}
	for e := range l.Recorder.Events {
		events = append(events, e)
	}
	return events
}

// Eventf formats as FakeRecorder does.
func Eventf(eventType, reason, messageFmt string, args ...interface{}) string {
	return fmt.Sprintf(eventType+" "+reason+" "+messageFmt, args...)
}
