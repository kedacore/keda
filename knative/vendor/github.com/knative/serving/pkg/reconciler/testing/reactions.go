/*
Copyright 2018 The Knative Authors.

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

	"k8s.io/apimachinery/pkg/runtime"
	clientgotesting "k8s.io/client-go/testing"

	"github.com/knative/pkg/apis"
)

// InduceFailure is used in conjunction with TableTest's WithReactors field.
// Tests that want to induce a failure in a row of a TableTest would add:
//   WithReactors: []clientgotesting.ReactionFunc{
//      // Makes calls to create revisions return an error.
//      InduceFailure("create", "revisions"),
//   },
func InduceFailure(verb, resource string) clientgotesting.ReactionFunc {
	return func(action clientgotesting.Action) (handled bool, ret runtime.Object, err error) {
		if !action.Matches(verb, resource) {
			return false, nil, nil
		}
		return true, nil, fmt.Errorf("inducing failure for %s %s", action.GetVerb(), action.GetResource().Resource)
	}
}

func ValidateCreates(action clientgotesting.Action) (handled bool, ret runtime.Object, err error) {
	got := action.(clientgotesting.CreateAction).GetObject()
	obj, ok := got.(apis.Validatable)
	if !ok {
		return false, nil, nil
	}
	if err := obj.Validate(); err != nil {
		return true, nil, err
	}
	return false, nil, nil
}

func ValidateUpdates(action clientgotesting.Action) (handled bool, ret runtime.Object, err error) {
	got := action.(clientgotesting.UpdateAction).GetObject()
	obj, ok := got.(apis.Validatable)
	if !ok {
		return false, nil, nil
	}
	if err := obj.Validate(); err != nil {
		return true, nil, err
	}
	return false, nil, nil
}
