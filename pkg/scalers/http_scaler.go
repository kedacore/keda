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

type httpScaler struct {
	metadata *httpScalerMetadata
}

type httpScalerMetadata struct {
	targetValue int
	apiURL      string
	metricName  string
}

type metric struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

var httpLog = logf.Log.WithName("http_scaler")

// NewHTTPScaler creates a new HTTP scaler
func NewHTTPScaler(resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	meta, err := parseHTTPMetadata(resolvedEnv, metadata, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing HTTP metadata: %s", err)
	}
	return &httpScaler{metadata: meta}, nil
}

func parseHTTPMetadata(resolvedEnv, metadata, authParams map[string]string) (*httpScalerMetadata, error) {
	meta := httpScalerMetadata{}

	if val, ok := metadata["targetValue"]; ok {
		targetValue, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("targetValue parsing error %s", err.Error())
		}
		meta.targetValue = targetValue
	} else {
		return nil, fmt.Errorf("no targetValue given in metadata")
	}

	if val, ok := metadata["apiURL"]; ok {
		// remove ending / for better string formatting
		meta.apiURL = strings.TrimSuffix(val, "/")
	} else {
		return nil, fmt.Errorf("no apiURL given in metadata")
	}

	if val, ok := metadata["metricName"]; ok {
		meta.metricName = val
	} else {
		return nil, fmt.Errorf("no metricName given in metadata")
	}

	return &meta, nil
}

func (s *httpScaler) checkHealth() error {
	u := fmt.Sprintf("%s/health/", s.metadata.apiURL)
	_, err := http.Get(u)
	return err
}

func (s *httpScaler) getMetricInfo() (*metric, error) {
	var m *metric
	u := fmt.Sprintf("%s/metrics/%s/", s.metadata.apiURL, s.metadata.metricName)
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

// Close does nothing in case of httpScaler
func (s *httpScaler) Close() error {
	return nil
}

// IsActive returns true if there are pending messages to be processed
func (s *httpScaler) IsActive(ctx context.Context) (bool, error) {
	err := s.checkHealth()
	if err != nil {
		httpLog.Error(err, fmt.Sprintf("Error when checking API health: %s", err))
		return false, err
	}
	return true, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *httpScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetValue := resource.NewQuantity(int64(s.metadata.targetValue), resource.DecimalSI)
	metricName := fmt.Sprintf("%s-%s", "http", kedautil.NormalizeString(s.metadata.apiURL), s.metadata.metricName)
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
func (s *httpScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
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
