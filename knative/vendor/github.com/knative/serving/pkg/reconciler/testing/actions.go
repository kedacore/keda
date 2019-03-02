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

	clientgotesting "k8s.io/client-go/testing"
)

type Actions struct {
	Creates           []clientgotesting.CreateAction
	Updates           []clientgotesting.UpdateAction
	Deletes           []clientgotesting.DeleteAction
	DeleteCollections []clientgotesting.DeleteCollectionAction
	Patches           []clientgotesting.PatchAction
}

type ActionRecorder interface {
	Actions() []clientgotesting.Action
}

type ActionRecorderList []ActionRecorder

func (l ActionRecorderList) ActionsByVerb() (Actions, error) {
	var a Actions

	for _, recorder := range l {
		for _, action := range recorder.Actions() {
			switch action.GetVerb() {
			case "create":
				a.Creates = append(a.Creates,
					action.(clientgotesting.CreateAction))
			case "update":
				a.Updates = append(a.Updates,
					action.(clientgotesting.UpdateAction))
			case "delete":
				a.Deletes = append(a.Deletes,
					action.(clientgotesting.DeleteAction))
			case "delete-collection":
				a.DeleteCollections = append(a.DeleteCollections,
					action.(clientgotesting.DeleteCollectionAction))
			case "patch":
				a.Patches = append(a.Patches,
					action.(clientgotesting.PatchAction))
			case "list": // avoid 'unexpected verb list' error
			case "watch": // avoid 'unexpected verb watch' error
			default:
				return a, fmt.Errorf("unexpected verb %v: %+v", action.GetVerb(), action)
			}
		}
	}

	return a, nil
}
