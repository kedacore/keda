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
	"time"
)

// FakeStatsReporter is a fake implementation of StatsReporter
type FakeStatsReporter struct {
	servicesReady map[string]int
}

func (r *FakeStatsReporter) ReportServiceReady(namespace, service string, d time.Duration) error {
	key := fmt.Sprintf("%s/%s", namespace, service)
	if r.servicesReady == nil {
		r.servicesReady = make(map[string]int)
	}
	r.servicesReady[key]++
	return nil
}

func (r *FakeStatsReporter) GetServiceReadyStats() map[string]int {
	return r.servicesReady
}
