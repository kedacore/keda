/*
Copyright 2021 The KEDA Authors

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

package prommetrics

// Server an HTTP serving instance to track metrics
type Server interface {
	NewServer(address string, pattern string)
	RecordScalerError(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, err error)
	RecordScalerMetric(namespace string, scaledObject string, scaler string, scalerIndex int, metric string, value int64)
	RecordScalerObjectError(namespace string, scaledObject string, err error)
}
