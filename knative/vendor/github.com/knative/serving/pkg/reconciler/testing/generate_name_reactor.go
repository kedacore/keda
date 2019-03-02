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
	"sync/atomic"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	clientgotesting "k8s.io/client-go/testing"
)

// GenerateNameReactor will simulate the k8s API server
// and generate a name for resources who's metadata.generateName
// property is set. This happens only for CreateAction types
//
// This generator is deterministic (unliked k8s) and uses a global
// counter to help make test names predictable
type GenerateNameReactor struct {
	count int64
}

// Handles contains all the logic to generate the name and mutates
// the create action object
//
// This is a hack as 'React' is passed a DeepCopy of the action hence
// this is the only opportunity to 'mutate' the action in the
// ReactionChain and have to continue executing additional reactors
//
// We should push changes upstream to client-go to help us with
// mocking
func (r *GenerateNameReactor) Handles(action clientgotesting.Action) bool {
	create, ok := action.(clientgotesting.CreateAction)

	if !ok {
		return false
	}

	objMeta, err := meta.Accessor(create.GetObject())

	if err != nil {
		return false
	}

	if objMeta.GetName() != "" {
		return false
	}

	if objMeta.GetGenerateName() == "" {
		return false
	}

	val := atomic.AddInt64(&r.count, 1)

	objMeta.SetName(fmt.Sprintf("%s%05d", objMeta.GetGenerateName(), val))

	return false
}

// React is noop-function
func (r *GenerateNameReactor) React(action clientgotesting.Action) (handled bool, ret runtime.Object, err error) {
	return false, nil, nil
}

var _ clientgotesting.Reactor = (*GenerateNameReactor)(nil)

// PrependGenerateNameReactor will instrument a client-go testing Fake
// with a reactor that simulates 'generateName' functionality
func PrependGenerateNameReactor(f *clientgotesting.Fake) {
	f.ReactionChain = append([]clientgotesting.Reactor{&GenerateNameReactor{}}, f.ReactionChain...)
}
