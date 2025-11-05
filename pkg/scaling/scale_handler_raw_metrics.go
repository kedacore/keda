/*
Copyright 2025 The KEDA Authors

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

package scaling

import (
	"context"
	"os"
	"slices"
	"strings"
	"time"

	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kedacore/keda/v2/pkg/metricsservice/api"
)

var (
	rawMetricsLog = logf.Log.WithName("raw_metrics")

	// sendRawMetrics is a feature flag that enables sending the raw metric values to all subscribed parties
	sendRawMetrics = os.Getenv("RAW_METRICS_GRPC_PROTOCOL") == "enabled"
	// sendRawMetricsMode defines the mode of sending raw metrics
	sendRawMetricsMode = parseRawMetricsMode()
)

// RawMetricsMode defines the different modes for sending raw metrics
type RawMetricsMode int

const (
	// RawMetricsDisabled - no raw metrics are sent
	RawMetricsDisabled RawMetricsMode = iota
	// RawMetricsAll - send all raw metrics (both HPA and polling) - default behavior
	RawMetricsAll
	// RawMetricsHPA - only send raw metrics from HPA requests
	RawMetricsHPA
	// RawMetricsPollingInterval - only send raw metrics from polling interval
	RawMetricsPollingInterval
)

// parseRawMetricsMode parses the RAW_METRICS_MODE environment variable
// Valid values:
//   - "all" or "" (empty): Send all raw metrics (both HPA and polling) - default behavior
//   - "hpa": Send raw metrics only from HPA requests
//   - "pollinginterval": Send raw metrics only from polling interval
//
// Any other value defaults to all
func parseRawMetricsMode() RawMetricsMode {
	if !sendRawMetrics {
		return RawMetricsDisabled
	}

	mode := strings.ToLower(os.Getenv("RAW_METRICS_MODE"))
	switch mode {
	// Default to "all" if not set
	case "all", "":
		return RawMetricsAll
	case "pollinginterval":
		return RawMetricsPollingInterval
	case "hpa":
		return RawMetricsHPA
	default:
		rawMetricsLog.Info("Unknown RAW_METRICS_MODE value, defaulting to all", "mode", mode)
		return RawMetricsAll
	}
}

// shouldSendRawMetrics determines if raw metrics should be sent based on the mode
func shouldSendRawMetrics(mode RawMetricsMode) bool {
	if !sendRawMetrics {
		return false
	}

	if sendRawMetricsMode == RawMetricsAll {
		return true
	}
	return sendRawMetricsMode == mode
}

type metricMeta struct {
	TriggerName      string
	ScaledObjectName string
	Namespace        string
}

type RawMetrics struct {
	Meta   metricMeta
	Values []external_metrics.ExternalMetricValue
}

type RawMetricSubscriptions struct {
	id            string
	subscriptions []metricMeta
	rawMetrics    chan RawMetrics
	done          chan bool
}

const (
	// KEDA will keep this many measurements on the channel (each collected in 15s interval), then sendTimeout will start ticking
	// if client isn't able to consume the message within that period, it will be automatically unsubscribed
	rawMetricsChannelCapacity = 4

	// timeout after which all the metrics for the subscriber will be unsubscribed (if these are not being consumed by client)
	unsubscribeTimeout = 1 * time.Minute
)

func (h *scaleHandler) UnsubscribeMetric(_ context.Context, subscriber string, sor *api.ScaledObjectRef) bool {
	if sor == nil || len(sor.MetricName) == 0 || len(sor.Namespace) == 0 || len(sor.Name) == 0 {
		return false
	}
	toDel := toMetricMeta(sor)
	log.Info("Unsubscribing metric", "subscriber", subscriber, "metric", toDel)
	h.subsLock.Lock()
	defer h.subsLock.Unlock()
	if _, found := h.rawMetricsSubscriptions[subscriber]; !found || !h.isSubscribed(subscriber, sor) {
		log.V(10).Info("Unsubscribing metric - noop")
		return false
	}

	// 1. update metricToSubscriptions map
	if allSubsOfMetric, found := h.metricToSubscriptions[toDel]; found {
		updatedAllSubs := slices.DeleteFunc(allSubsOfMetric, func(item *RawMetricSubscriptions) bool {
			return item != nil && item.id == subscriber
		})
		if len(updatedAllSubs) == 0 {
			delete(h.metricToSubscriptions, toDel)
		} else {
			h.metricToSubscriptions[toDel] = updatedAllSubs
		}
	}

	// 2. update rawMetricsSubscriptions map
	remainingMetricsOfSub := slices.DeleteFunc(h.rawMetricsSubscriptions[subscriber].subscriptions, func(item metricMeta) bool {
		return item.equal(toDel)
	})

	// (3). subscriber has no other metrics -> close the channels
	if len(remainingMetricsOfSub) == 0 {
		select {
		case h.rawMetricsSubscriptions[subscriber].done <- true:
			close(h.rawMetricsSubscriptions[subscriber].done)
		default:
		}
		delete(h.rawMetricsSubscriptions, subscriber)
	} else {
		h.rawMetricsSubscriptions[subscriber].subscriptions = remainingMetricsOfSub
	}
	log.V(10).Info("All subscribed metrics", "subscriber", subscriber, "subscribersMetrics", remainingMetricsOfSub)
	return true
}

func (h *scaleHandler) SubscribeMetric(_ context.Context, subscriber string, sor *api.ScaledObjectRef) bool {
	if sor == nil || len(sor.MetricName) == 0 || len(sor.Namespace) == 0 || len(sor.Name) == 0 {
		return false
	}
	toAdd := toMetricMeta(sor)
	log.Info("Subscribing metric", "subscriber", subscriber, "metric", toAdd)
	h.subsLock.Lock()
	defer h.subsLock.Unlock()

	if _, found := h.rawMetricsSubscriptions[subscriber]; !found {
		h.rawMetricsSubscriptions[subscriber] = makeEmptySubscriptions(subscriber)
	} else if h.isSubscribed(subscriber, sor) {
		log.V(10).Info("Subscribing metric - noop")
		return true
	}

	// 1. update rawMetricsSubscriptions map
	h.rawMetricsSubscriptions[subscriber].subscriptions = append(h.rawMetricsSubscriptions[subscriber].subscriptions, toAdd)

	// 2. update metricToSubscriptions map
	allSubsOfMetric, found := h.metricToSubscriptions[toAdd]
	if !found {
		h.metricToSubscriptions[toAdd] = make([]*RawMetricSubscriptions, 1)
	}
	if !slices.ContainsFunc(allSubsOfMetric, func(sub *RawMetricSubscriptions) bool {
		return sub != nil && sub.id == subscriber
	}) {
		h.metricToSubscriptions[toAdd] = append(h.metricToSubscriptions[toAdd], h.rawMetricsSubscriptions[subscriber])
	}

	mm := h.rawMetricsSubscriptions[subscriber].subscriptions
	log.V(10).Info("All subscribed metrics", "subscriber", subscriber, "subscribersMetrics", mm)
	return false
}

func (h *scaleHandler) GetRawMetricsChan(subscriber string) (chan RawMetrics, chan bool) {
	h.subsLock.Lock()
	defer h.subsLock.Unlock()
	if _, found := h.rawMetricsSubscriptions[subscriber]; !found {
		h.rawMetricsSubscriptions[subscriber] = makeEmptySubscriptions(subscriber)
	}
	return h.rawMetricsSubscriptions[subscriber].rawMetrics, h.rawMetricsSubscriptions[subscriber].done
}

func (h *scaleHandler) sendWhenSubscribed(soName, ns, triggerName string, metrics []external_metrics.ExternalMetricValue) {
	mm := metricMeta{
		TriggerName:      triggerName,
		ScaledObjectName: soName,
		Namespace:        ns,
	}
	var targets []RawMetricSubscriptions
	h.subsLock.RLock()
	for _, t := range h.metricToSubscriptions[mm] {
		if t != nil {
			targets = append(targets, *t)
		}
	}
	h.subsLock.RUnlock()

	if len(targets) > 0 {
		vals := make([]external_metrics.ExternalMetricValue, len(metrics))
		copy(vals, metrics)
		msg := RawMetrics{Meta: mm, Values: vals}

		for _, t := range targets {
			select {
			case t.rawMetrics <- msg:
				// sent ok
			case val := <-t.done:
				// subscriber closed
				if val {
					close(t.rawMetrics)
				}
			case <-time.After(unsubscribeTimeout):
				// timed out -> drop & unsubscribe
				log.V(10).Info("Raw metric channel has reached its capacity and timeout has passed."+
					" Unsubscribing all the metrics for this subscriber.",
					"subscriber", t.id,
					"subscribedMetricsCount", len(t.subscriptions),
					"channelCapacity", rawMetricsChannelCapacity,
					"timeout", unsubscribeTimeout)
				for _, sub := range t.subscriptions {
					h.UnsubscribeMetric(context.Background(), t.id, &api.ScaledObjectRef{
						Name:       sub.ScaledObjectName,
						Namespace:  sub.Namespace,
						MetricName: sub.TriggerName,
					})
				}
			}
		}
	}
}

// make sure this is being called only from critical section
func (h *scaleHandler) isSubscribed(subscriber string, sor *api.ScaledObjectRef) bool {
	toFind := toMetricMeta(sor)
	return slices.ContainsFunc(h.rawMetricsSubscriptions[subscriber].subscriptions, func(mm metricMeta) bool {
		return mm.equal(toFind)
	})
}

func (a metricMeta) equal(b metricMeta) bool {
	return a.TriggerName == b.TriggerName && a.ScaledObjectName == b.ScaledObjectName && a.Namespace == b.Namespace
}

func toMetricMeta(sor *api.ScaledObjectRef) metricMeta {
	return metricMeta{
		TriggerName:      sor.MetricName,
		ScaledObjectName: sor.Name,
		Namespace:        sor.Namespace,
	}
}

func makeEmptySubscriptions(id string) *RawMetricSubscriptions {
	return &RawMetricSubscriptions{
		id:            id,
		subscriptions: []metricMeta{},
		rawMetrics:    make(chan RawMetrics, rawMetricsChannelCapacity),
		done:          make(chan bool, 1),
	}
}
