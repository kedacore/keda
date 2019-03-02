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

package autoscaler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/knative/serving/pkg/apis/serving"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/reconciler"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"go.uber.org/zap"
)

const (
	httpClientTimeout = 3 * time.Second
)

// StatsScraper defines the interface for collecting Revision metrics
type StatsScraper interface {
	// Scrape scrapes the Revision queue metric endpoint then sends it as a
	// StatMessage to the given channel.
	Scrape(statsCh chan<- *StatMessage)
}

// cacheDisabledClient is a http client with cache disabled. It is shared by
// every goruntime for a revision scraper.
var cacheDisabledClient = &http.Client{
	Transport: &http.Transport{
		// Do not use the cached connection
		DisableKeepAlives: true,
	},
	Timeout: httpClientTimeout,
}

// ServiceScraper scrapes Revision metrics via a K8S service by sampling. Which
// pod to be picked up to serve the request is decided by K8S. Please see
// https://kubernetes.io/docs/concepts/services-networking/network-policies/
// for details.
type ServiceScraper struct {
	httpClient *http.Client
	url        string
	metricKey  string
	logger     *zap.SugaredLogger
}

// NewServiceScraper creates a new StatsScraper for the Revision which
// the given Metric is responsible for.
func NewServiceScraper(metric *Metric, logger *zap.SugaredLogger) (*ServiceScraper, error) {
	return newServiceScraperWithClient(metric, logger, cacheDisabledClient)
}

func newServiceScraperWithClient(metric *Metric, logger *zap.SugaredLogger, httpClient *http.Client) (*ServiceScraper, error) {
	revName := metric.Labels[serving.RevisionLabelKey]
	if revName == "" {
		return nil, fmt.Errorf("no Revision label found for Metric %s", metric.Name)
	}

	serviceName := reconciler.GetServingK8SServiceNameForObj(revName)
	return &ServiceScraper{
		httpClient: httpClient,
		url:        fmt.Sprintf("http://%s.%s:%d/metrics", serviceName, metric.Namespace, v1alpha1.RequestQueueMetricsPort),
		metricKey:  NewMetricKey(metric.Namespace, metric.Name),
		logger:     logger,
	}, nil
}

// Scrape call the destination service then send it
// to the given stats chanel
func (s *ServiceScraper) Scrape(statsCh chan<- *StatMessage) {
	stat, err := s.scrapeViaURL()
	if err != nil {
		s.logger.Errorw("Failed to get metrics", zap.Error(err))
		return
	}

	s.sendStatMessage(*stat, statsCh)
}

func (s *ServiceScraper) scrapeViaURL() (*Stat, error) {
	req, err := http.NewRequest("GET", s.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("GET request for URL %q returned HTTP status %v", s.url, resp.StatusCode)
	}

	return extractData(resp.Body)
}

func extractData(body io.Reader) (*Stat, error) {
	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(body)
	if err != nil {
		return nil, fmt.Errorf("Reading text format failed: %v", err)
	}

	now := time.Now()
	stat := Stat{
		Time: &now,
	}

	if pMetric := getPrometheusMetric(metricFamilies, "queue_average_concurrent_requests"); pMetric != nil {
		// The autoscaler should decide what to do with a Stat with empty pod name
		for _, label := range pMetric.Label {
			if *label.Name == "destination_pod" {
				stat.PodName = *label.Value
				break
			}
		}
		stat.AverageConcurrentRequests = *pMetric.Gauge.Value
	} else {
		return nil, errors.New("Could not find value for queue_average_concurrent_requests in response")
	}

	if pMetric := getPrometheusMetric(metricFamilies, "queue_operations_per_second"); pMetric != nil {
		stat.RequestCount = int32(*pMetric.Gauge.Value)
	} else {
		return nil, errors.New("Could not find value for queue_operations_per_second in response")
	}

	return &stat, nil
}

// getPrometheusMetric returns the point of the first Metric of the MetricFamily
// with the given key from the given map. If there is no such MetricFamily or it
// has no Metrics, then returns nil.
func getPrometheusMetric(metricFamilies map[string]*dto.MetricFamily, key string) *dto.Metric {
	if metric, ok := metricFamilies[key]; ok && metric != nil && len(metric.Metric) != 0 {
		return metric.Metric[0]
	}

	return nil
}

func (s *ServiceScraper) sendStatMessage(stat Stat, statsCh chan<- *StatMessage) {
	sm := &StatMessage{
		Stat: stat,
		Key:  s.metricKey,
	}
	statsCh <- sm
}
