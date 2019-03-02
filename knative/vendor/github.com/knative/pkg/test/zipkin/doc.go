/*
Copyright 2019 The Knative Authors

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

/*
Package zipkin adds Zipkin tracing support that can be used in conjunction with
SpoofingClient to log zipkin traces for requests that have encountered server errors
i.e HTTP request that have HTTP status between 500 to 600.

This package exposes following methods:

	SetupZipkinTracing(*kubernetes.Clientset) error
		SetupZipkinTracing sets up zipkin tracing by setting up port-forwarding from
		localhost to zipkin pod on the cluster. On successful setup this method sets
		an internal flag zipkinTracingEnabled to true.

	CleanupZipkinTracingSetup() error
		CleanupZipkinTracingSetup cleans up zipkin tracing setup by cleaning up the
		port-forwarding setup by call to SetupZipkinTracing. This method also sets
		zipkinTracingEnabled flag to false.

A general flow for a Test Suite to use Zipkin Tracing support is as follows:

		1. Call SetupZipkinTracing(*kubernetes.Clientset) in TestMain.
		2. Use SpoofingClient to make HTTP requests.
		3. Call CleanupZipkinTracingSetup on cleanup after tests are executed.

*/
package zipkin
