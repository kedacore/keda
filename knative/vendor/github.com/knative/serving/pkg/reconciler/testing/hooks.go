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

// Package testing includes utilities for testing controllers.
package testing

import (
	"errors"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	kubetesting "k8s.io/client-go/testing"
)

// HookResult is the return value of hook functions.
type HookResult bool

const (
	// HookComplete indicates the hook function completed, and WaitForHooks should
	// not wait for it.
	HookComplete HookResult = true
	// HookIncomplete indicates the hook function is incomplete, and WaitForHooks
	// should wait for it to complete.
	HookIncomplete HookResult = false
)

/*
CreateHookFunc is a function for handling a Create hook. Its runtime.Object
parameter will be the Kubernetes resource created. The resource can be cast
to its actual type like this:

		pod := obj.(*v1.Pod)

A return value of true marks the hook as completed. Returning false allows
the hook to run again when the next resource of the requested type is
created.
*/
type CreateHookFunc func(runtime.Object) HookResult

/*
UpdateHookFunc is a function for handling an update hook. its runtime.Object
parameter will be the Kubernetes resource updated. The resource can be cast
to its actual type like this:

		pod := obj.(*v1.Pod)

A return value of true marks the hook as completed. Returning false allows
the hook to run again when the next resource of the requested type is
updated.
*/
type UpdateHookFunc func(runtime.Object) HookResult

/*
DeleteHookFunc is a function for handling a delete hook. Its name parameter will
be the name of the resource deleted. The resource itself is not available to
the reactor.
*/
type DeleteHookFunc func(string) HookResult

/*
Hooks is a utility struct that simplifies controller testing with fake
clients. A Hooks struct allows attaching hook functions to actions (create,
update, delete) on a specified resource type within a fake client and ensuring
that all hooks complete in a timely manner.
*/
type Hooks struct {
	completionCh    chan int32
	completionIndex int32
}

// NewHooks returns a Hooks struct that can be used to attach hooks to one or
// more fake clients and wait for all hooks to complete.
// TODO(grantr): Allow validating that a hook never fires
func NewHooks() *Hooks {
	return &Hooks{
		completionCh:    make(chan int32, 100),
		completionIndex: -1,
	}
}

// OnCreate attaches a create hook to the given Fake. The hook function is
// executed every time a resource of the given type is created.
func (h *Hooks) OnCreate(fake *kubetesting.Fake, resource string, rf CreateHookFunc) {
	index := atomic.AddInt32(&h.completionIndex, 1)
	fake.PrependReactor("create", resource, func(a kubetesting.Action) (bool, runtime.Object, error) {
		obj := a.(kubetesting.CreateActionImpl).Object
		if rf(obj) == HookComplete {
			h.completionCh <- index
		}
		return false, nil, nil
	})
}

// OnUpdate attaches an update hook to the given Fake. The hook function is
// executed every time a resource of the given type is updated.
func (h *Hooks) OnUpdate(fake *kubetesting.Fake, resource string, rf UpdateHookFunc) {
	index := atomic.AddInt32(&h.completionIndex, 1)
	fake.PrependReactor("update", resource, func(a kubetesting.Action) (bool, runtime.Object, error) {
		obj := a.(kubetesting.UpdateActionImpl).Object
		if rf(obj) == HookComplete {
			h.completionCh <- index
		}
		return false, nil, nil
	})
}

// OnDelete attaches a delete hook to the given Fake. The hook function is
// executed every time a resource of the given type is deleted.
func (h *Hooks) OnDelete(fake *kubetesting.Fake, resource string, rf DeleteHookFunc) {
	index := atomic.AddInt32(&h.completionIndex, 1)
	fake.PrependReactor("delete", resource, func(a kubetesting.Action) (bool, runtime.Object, error) {
		name := a.(kubetesting.DeleteActionImpl).Name
		if rf(name) == HookComplete {
			h.completionCh <- index
		}
		return false, nil, nil
	})
}

// WaitForHooks waits until all attached hooks have returned true at least once.
// If the given timeout expires before that happens, an error is returned.
func (h *Hooks) WaitForHooks(timeout time.Duration) error {
	ci := int(atomic.LoadInt32(&h.completionIndex))
	if ci == -1 {
		return nil
	}
	// Convert index to count.
	ci++
	timer := time.After(timeout)
	hookCompletions := map[int32]HookResult{}
	for {
		select {
		case i := <-h.completionCh:
			hookCompletions[i] = HookComplete
			if len(hookCompletions) == ci {
				atomic.StoreInt32(&h.completionIndex, -1)
				return nil
			}
		case <-timer:
			return errors.New("timed out waiting for hooks to complete")
		}
	}
}
