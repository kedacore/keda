package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	kedautil "github.com/kedacore/keda/pkg/util"
	"io/ioutil"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"net/http"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"strings"
)

type metricsAPIScaler struct {
	metadata *metricsAPIScalerMetadata
}

type metricsAPIScalerMetadata struct {
	targetValue int
	url         string
	metricName  string
}

type metric struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

var httpLog = logf.Log.WithName("metrics_api_scaler")

// NewMetricsAPIScaler creates a new HTTP scaler
func NewMetricsAPIScaler(resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	meta, err := metricsAPIMetadata(resolvedEnv, metadata, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing metric API metadata: %s", err)
	}
	scaler := &metricsAPIScaler{metadata: meta}
	err = scaler.checkHealth()
	if err != nil {
		return nil, fmt.Errorf("error checking metric API health/ endpoint: %s", err)
	}

	return scaler, nil
}

func metricsAPIMetadata(resolvedEnv, metadata, authParams map[string]string) (*metricsAPIScalerMetadata, error) {
	meta := metricsAPIScalerMetadata{}

	if val, ok := metadata["targetValue"]; ok {
		targetValue, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("targetValue parsing error %s", err.Error())
		}
		meta.targetValue = targetValue
	} else {
		return nil, fmt.Errorf("no targetValue given in metadata")
	}

	if val, ok := metadata["url"]; ok {
		// remove ending / for better string formatting
		meta.url = strings.TrimSuffix(val, "/")
	} else {
		return nil, fmt.Errorf("no url given in metadata")
	}

	if val, ok := metadata["metricName"]; ok {
		meta.metricName = val
	} else {
		return nil, fmt.Errorf("no metricName given in metadata")
	}

	return &meta, nil
}

func (s *metricsAPIScaler) checkHealth() error {
	u := fmt.Sprintf("%s/health/", s.metadata.url)
	_, err := http.Get(u)
	return err
}

func (s *metricsAPIScaler) getMetricInfo() (*metric, error) {
	var m *metric
	u := fmt.Sprintf("%s/metrics/%s/", s.metadata.url, s.metadata.metricName)
	r, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Close does nothing in case of metricsAPIScaler
func (s *metricsAPIScaler) Close() error {
	return nil
}

// IsActive returns true if there are pending messages to be processed
func (s *metricsAPIScaler) IsActive(ctx context.Context) (bool, error) {
	m, err := s.getMetricInfo()
	if err != nil {
		httpLog.Error(err, fmt.Sprintf("Error when checking metric value: %s", err))
		return false, err
	}

	return m.Value > 0.0, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *metricsAPIScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetValue := resource.NewQuantity(int64(s.metadata.targetValue), resource.DecimalSI)
	metricName := fmt.Sprintf("%s-%s-%s", "http", kedautil.NormalizeString(s.metadata.url), s.metadata.metricName)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: metricName,
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *metricsAPIScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	m, err := s.getMetricInfo()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error requesting metrics endpoint: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(m.Value), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
