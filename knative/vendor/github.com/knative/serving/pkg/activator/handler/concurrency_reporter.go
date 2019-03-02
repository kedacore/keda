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

package handler

import (
	"time"

	"github.com/knative/pkg/system"
	"github.com/knative/serving/pkg/autoscaler"
)

// ConcurrencyReporter reports stats based on incoming requests and ticks.
type ConcurrencyReporter struct {
	podName string

	// Ticks with every request arrived/completed respectively
	reqChan chan ReqEvent
	// Ticks with every stat report request
	reportChan <-chan time.Time
	// Stat reporting channel
	statChan chan *autoscaler.StatMessage

	clock system.Clock
}

// NewConcurrencyReporter creates a ConcurrencyReporter which listens to incoming
// ReqEvents on reqChan and ticks on reportChan and reports stats on statChan.
func NewConcurrencyReporter(podName string, reqChan chan ReqEvent, reportChan <-chan time.Time, statChan chan *autoscaler.StatMessage) *ConcurrencyReporter {
	return NewConcurrencyReporterWithClock(podName, reqChan, reportChan, statChan, system.RealClock{})
}

// NewConcurrencyReporterWithClock instantiates a new concurrency reporter
// which uses the passed clock.
func NewConcurrencyReporterWithClock(podName string, reqChan chan ReqEvent, reportChan <-chan time.Time, statChan chan *autoscaler.StatMessage, clock system.Clock) *ConcurrencyReporter {
	return &ConcurrencyReporter{
		podName:    podName,
		reqChan:    reqChan,
		reportChan: reportChan,
		statChan:   statChan,
		clock:      clock,
	}
}

func (cr *ConcurrencyReporter) report(now time.Time, key string, concurrency, requestCount int32) {
	stat := autoscaler.Stat{
		Time:                      &now,
		PodName:                   cr.podName,
		AverageConcurrentRequests: float64(concurrency),
		RequestCount:              requestCount,
	}

	// Send the stat to another goroutine to transmit
	// so we can continue bucketing stats.
	cr.statChan <- &autoscaler.StatMessage{
		Key:  key,
		Stat: stat,
	}
}

// Run runs until stopCh is closed and processes events on all incoming channels
func (cr *ConcurrencyReporter) Run(stopCh <-chan struct{}) {
	// Contains the number of in-flight requests per-key
	outstandingRequestsPerKey := make(map[string]int32)
	// Contains the number of incoming requests in the current
	// reporting period, per key.
	incomingRequestsPerKey := make(map[string]int32)

	for {
		select {
		case event := <-cr.reqChan:
			switch event.EventType {
			case ReqIn:
				incomingRequestsPerKey[event.Key]++

				// Report the first request for a key immediately.
				if _, ok := outstandingRequestsPerKey[event.Key]; !ok {
					cr.report(cr.clock.Now(), event.Key, 1, incomingRequestsPerKey[event.Key])
				}
				outstandingRequestsPerKey[event.Key]++
			case ReqOut:
				outstandingRequestsPerKey[event.Key]--
			}
		case now := <-cr.reportChan:
			for key, concurrency := range outstandingRequestsPerKey {
				if concurrency == 0 {
					delete(outstandingRequestsPerKey, key)
				} else {
					cr.report(now, key, concurrency, incomingRequestsPerKey[key])
				}
			}

			incomingRequestsPerKey = make(map[string]int32)
		case <-stopCh:
			return
		}
	}
}
