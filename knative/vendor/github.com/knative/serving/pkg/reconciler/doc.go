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

// Package reconciler defines implementations of the Reconciler interface
// defined at github.com/knative/pkg/controller.Reconciler.  These implement
// the basic workhorse functionality of controllers, while leaving the
// shared controller implementation to manage things like the workqueue.
//
// Despite defining a Reconciler, each of the packages here are expected to
// expose a controller constructor like:
//    func NewController(...) *controller.Impl { ... }
// These constructors will:
// 1. Construct the Reconciler,
// 2. Construct a controller.Impl with that Reconciler,
// 3. Wire the assorted informers this Reconciler watches to call appropriate
//   enqueue methods on the controller.
package reconciler
