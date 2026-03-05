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

package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	url_pkg "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	// ionosMonitoringPrometheusPath is the Mimir-compatible Prometheus instant query path.
	ionosMonitoringPrometheusPath = "prometheus/api/v1/query"
)

type ionosMonitoringScaler struct {
	metricType v2.MetricTargetType
	metadata   *ionosMonitoringMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type ionosMonitoringMetadata struct {
	// Host is the pipeline httpEndpoint returned when creating an IONOS Monitoring pipeline,
	// e.g. https://123456789-metrics.987654321.monitoring.de-txl.ionos.com
	Host string `keda:"name=host, order=triggerMetadata"`

	// APIKey is the pipeline key credential used to authenticate against the monitoring endpoint.
	// Must be supplied via TriggerAuthentication (authParams).
	APIKey string `keda:"name=apiKey, order=authParams"`

	// Query is a PromQL expression evaluated as an instant query.
	Query string `keda:"name=query, order=triggerMetadata"`

	// Threshold is the metric value at which scaling occurs.
	Threshold float64 `keda:"name=threshold, order=triggerMetadata"`

	// ActivationThreshold is the minimum value required to activate the scaler (default 0).
	ActivationThreshold float64 `keda:"name=activationThreshold, order=triggerMetadata, optional"`

	// IgnoreNullValues controls whether an empty query result returns 0 (true) or an error (false).
	IgnoreNullValues bool `keda:"name=ignoreNullValues, order=triggerMetadata, default=true"`

	TriggerIndex int
}

// ionosPromQueryResult maps the Prometheus-compatible instant query response.
type ionosPromQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct{}      `json:"metric"`
			Value  []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// NewIONOSMonitoringScaler creates a new ionosMonitoringScaler.
func NewIONOSMonitoringScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "ionos_monitoring_scaler")

	meta, err := parseIONOSMonitoringMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing IONOS Monitoring metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	logger.Info("Initializing IONOS Monitoring Scaler", "host", meta.Host)

	return &ionosMonitoringScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

func parseIONOSMonitoringMetadata(config *scalersconfig.ScalerConfig) (*ionosMonitoringMetadata, error) {
	meta := &ionosMonitoringMetadata{}
	meta.TriggerIndex = config.TriggerIndex
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing IONOS Monitoring metadata: %w", err)
	}
	return meta, nil
}

func (s *ionosMonitoringScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

func (s *ionosMonitoringScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, kedautil.NormalizeString("ionos-monitoring")),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.Threshold),
	}
	return []v2.MetricSpec{{External: externalMetric, Type: externalMetricType}}
}

func (s *ionosMonitoringScaler) executeQuery(ctx context.Context) (float64, error) {
	t := time.Now().UTC().Format(time.RFC3339)
	queryEscaped := url_pkg.QueryEscape(s.metadata.Query)
	queryURL := fmt.Sprintf("%s/%s?query=%s&time=%s",
		strings.TrimRight(s.metadata.Host, "/"),
		ionosMonitoringPrometheusPath,
		queryEscaped,
		t,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return -1, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.metadata.APIKey))

	r, err := s.httpClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer r.Body.Close()

	if r.StatusCode < 200 || r.StatusCode > 299 {
		_, _ = io.Copy(io.Discard, r.Body)
		return -1, fmt.Errorf("IONOS Monitoring query API returned status %d", r.StatusCode)
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		return -1, err
	}

	var result ionosPromQueryResult
	if err := json.Unmarshal(b, &result); err != nil {
		return -1, fmt.Errorf("error parsing IONOS Monitoring query response: %w", err)
	}

	if result.Status != "success" {
		return -1, fmt.Errorf("IONOS Monitoring query returned status %q", result.Status)
	}

	if len(result.Data.Result) == 0 {
		if s.metadata.IgnoreNullValues {
			return 0, nil
		}
		return -1, errors.New("IONOS Monitoring query returned empty result")
	}

	if len(result.Data.Result) > 1 {
		return -1, fmt.Errorf("IONOS Monitoring query %q returned multiple elements", s.metadata.Query)
	}

	values := result.Data.Result[0].Value
	if len(values) < 2 {
		if s.metadata.IgnoreNullValues {
			return 0, nil
		}
		return -1, fmt.Errorf("IONOS Monitoring query %q returned no values", s.metadata.Query)
	}

	valStr, ok := values[1].(string)
	if !ok {
		return -1, errors.New("IONOS Monitoring query returned a non-string metric value")
	}

	v, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return -1, fmt.Errorf("error converting IONOS Monitoring metric value %q: %w", valStr, err)
	}

	if math.IsInf(v, 0) {
		if s.metadata.IgnoreNullValues {
			return 0, nil
		}
		return -1, fmt.Errorf("IONOS Monitoring query returned infinite value")
	}

	return v, nil
}

func (s *ionosMonitoringScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	val, err := s.executeQuery(ctx)
	if err != nil {
		s.logger.Error(err, "error executing IONOS Monitoring query")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, val)
	return []external_metrics.ExternalMetricValue{metric}, val > s.metadata.ActivationThreshold, nil
}
