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

/*
Package autoscaler calculates the number of pods necessary for the
desired level of concurrency per pod (stableConcurrencyPerPod). It
operates in two modes, stable mode and panic mode.

Stable mode calculates the average concurrency observed over the last
60 seconds and adjusts the observed pod count to achieve the target
value. Current observed pod count is the number of unique pod names
which show up in the last 60 seconds.

Panic mode calculates the average concurrency observed over the last 6
seconds and adjusts the observed pod count to achieve the stable
target value. Panic mode is engaged when the observed 6 second average
concurrency reaches 2x the target stable concurrency. Panic mode will
last at least 60 seconds--longer if the 2x threshold is repeatedly
breached. During panic mode the number of pods is never decreased in
order to prevent flapping.

Package autoscaler supports both single-tenant (one autoscaler per
revision) and multitenant (one autoscaler for all revisions) autoscalers;
config/controller.yaml determines which kind of autoscaler is used.
*/
package autoscaler
