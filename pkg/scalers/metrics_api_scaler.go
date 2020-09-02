package scalers

import (
	"context"
	"errors"
	"fmt"
	kedautil "github.com/kedacore/keda/pkg/util"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"net/http"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
)

type metricsAPIScaler struct {
	metadata *metricsAPIScalerMetadata
}

type metricsAPIScalerMetadata struct {
	targetValue   int
	url           string
	valueLocation string
}

var httpLog = logf.Log.WithName("metrics_api_scaler")

// NewMetricsAPIScaler creates a new HTTP scaler
func NewMetricsAPIScaler(resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	meta, err := metricsAPIMetadata(resolvedEnv, metadata, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing metric API metadata: %s", err)
	}
	return &metricsAPIScaler{metadata: meta}, nil
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
		meta.url = val
	} else {
		return nil, fmt.Errorf("no url given in metadata")
	}

	if val, ok := metadata["valueLocation"]; ok {
		meta.valueLocation = val
	} else {
		return nil, fmt.Errorf("no valueLocation given in metadata")
	}

	return &meta, nil
}

// GetValueFromResponse uses provided valueLocation to access the numeric value in provided body
func GetValueFromResponse(body []byte, valueLocation string) (int64, error) {
	r := gjson.GetBytes(body, valueLocation)
	if r.Type != gjson.Number {
		msg := fmt.Sprintf("valueLocation must point to value of type number got: %s", r.Type.String())
		return 0, errors.New(msg)
	}
	return int64(r.Num), nil
}

func (s *metricsAPIScaler) getMetricValue() (int64, error) {
	r, err := http.Get(s.metadata.url)
	if err != nil {
		return 0, err
	}
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return 0, err
	}
	v, err := GetValueFromResponse(b, s.metadata.valueLocation)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// Close does nothing in case of metricsAPIScaler
func (s *metricsAPIScaler) Close() error {
	return nil
}

// IsActive returns true if there are pending messages to be processed
func (s *metricsAPIScaler) IsActive(ctx context.Context) (bool, error) {
	v, err := s.getMetricValue()
	if err != nil {
		httpLog.Error(err, fmt.Sprintf("Error when checking metric value: %s", err))
		return false, err
	}

	return v > 0.0, nil
}

// GetMetricSpecForScaling returns the MetricSpec for the Horizontal Pod Autoscaler
func (s *metricsAPIScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetValue := resource.NewQuantity(int64(s.metadata.targetValue), resource.DecimalSI)
	metricName := fmt.Sprintf("%s-%s-%s", "http", kedautil.NormalizeString(s.metadata.url), s.metadata.valueLocation)
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
	v, err := s.getMetricValue()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error requesting metrics endpoint: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(v, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
