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
	"regexp"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

// KeyOrDie returns the string key of the Kubernetes object or panics if a key
// cannot be generated.
func KeyOrDie(obj interface{}) string {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		panic(err)
	}
	return key
}

// ExpectNormalEventDelivery returns a hook function that can be passed to a
// Hooks.OnCreate() call to verify that an event of type Normal was created
// matching the given regular expression. For this expectation to be effective
// the test must also call Hooks.WaitForHooks().
func ExpectNormalEventDelivery(t *testing.T, messageRegexp string) CreateHookFunc {
	t.Helper()
	wantRegexp, err := regexp.Compile(messageRegexp)
	if err != nil {
		t.Fatalf("Invalid regular expression: %v", err)
	}
	return func(obj runtime.Object) HookResult {
		t.Helper()
		event := obj.(*corev1.Event)
		if !wantRegexp.MatchString(event.Message) {
			return HookIncomplete
		}
		t.Logf("Got an event message matching %q: %q", wantRegexp, event.Message)
		if got, want := event.Type, corev1.EventTypeNormal; got != want {
			t.Errorf("unexpected event Type: %q expected: %q", got, want)
		}
		return HookComplete
	}
}

// ExpectWarningEventDelivery returns a hook function that can be passed to a
// Hooks.OnCreate() call to verify that an event of type Warning was created
// matching the given regular expression. For this expectation to be effective
// the test must also call Hooks.WaitForHooks().
func ExpectWarningEventDelivery(t *testing.T, messageRegexp string) CreateHookFunc {
	t.Helper()
	wantRegexp, err := regexp.Compile(messageRegexp)
	if err != nil {
		t.Fatalf("Invalid regular expression: %v", err)
	}
	return func(obj runtime.Object) HookResult {
		t.Helper()
		event := obj.(*corev1.Event)
		if !wantRegexp.MatchString(event.Message) {
			return HookIncomplete
		}
		t.Logf("Got an event message matching %q: %q", wantRegexp, event.Message)
		if got, want := event.Type, corev1.EventTypeWarning; got != want {
			t.Errorf("unexpected event Type: %q expected: %q", got, want)
		}
		return HookComplete
	}
}
