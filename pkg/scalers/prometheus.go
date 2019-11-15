package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	promServerAddress = "serverAddress"
	promMetricName    = "metricName"
	promQuery         = "query"
	promThreshold     = "threshold"
)

type prometheusScaler struct {
	metadata *prometheusMetadata
}

type prometheusMetadata struct {
	serverAddress string
	metricName    string
	query         string
	threshold     int
}

type promQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

var prometheusLog = logf.Log.WithName("prometheus_scaler")

// NewPrometheusScaler creates a new prometheusScaler
func NewPrometheusScaler(resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, err := parsePrometheusMetadata(metadata, resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("error parsing prometheus metadata: %s", err)
	}

	return &prometheusScaler{
		metadata: meta,
	}, nil
}

func parsePrometheusMetadata(metadata, resolvedEnv map[string]string) (*prometheusMetadata, error) {
	meta := prometheusMetadata{}

	if val, ok := metadata[promServerAddress]; ok && val != "" {
		meta.serverAddress = val
	} else {
		return nil, fmt.Errorf("no %s given", promServerAddress)
	}

	if val, ok := metadata[promQuery]; ok && val != "" {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no %s given", promQuery)
	}

	if val, ok := metadata[promMetricName]; ok && val != "" {
		meta.metricName = val
	} else {
		return nil, fmt.Errorf("no %s given", promMetricName)
	}

	if val, ok := metadata[promThreshold]; ok && val != "" {
		t, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %s", promThreshold, err)
		}

		meta.threshold = t
	}

	return &meta, nil
}

func (s *prometheusScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := s.ExecutePromQuery()
	if err != nil {
		prometheusLog.Error(err, "error executing prometheus query")
		return false, err
	}

	return val > 0, nil
}

func (s *prometheusScaler) Close() error {
	return nil
}

func (s *prometheusScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	return []v2beta1.MetricSpec{
		{
			External: &v2beta1.ExternalMetricSource{
				TargetAverageValue: resource.NewQuantity(int64(s.metadata.threshold), resource.DecimalSI),
				MetricName:         s.metadata.metricName,
			},
			Type: externalMetricType,
		},
	}
}

func (s *prometheusScaler) ExecutePromQuery() (float64, error) {
	t := time.Now().UTC().Format(time.RFC3339)
	url := fmt.Sprintf("%s/api/v1/query?query=%s&time=%s", s.metadata.serverAddress, s.metadata.query, t)
	r, err := http.Get(url)
	if err != nil {
		return -1, err
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return -1, err
	}
	r.Body.Close()

	var result promQueryResult
	err = json.Unmarshal(b, &result)
	if err != nil {
		return -1, err
	}

	var v float64 = -1

	// only allow for single element result sets
	if len(result.Data.Result) == 0 {
		return -1, fmt.Errorf("Prometheus query %s returned empty", s.metadata.query)
	} else if len(result.Data.Result) > 1 {
		return -1, fmt.Errorf("Prometheus query %s returned multiple elements", s.metadata.query)
	}

	val := result.Data.Result[0].Value[1]
	if val != nil {
		s := val.(string)
		v, err = strconv.ParseFloat(s, 64)
		if err != nil {
			prometheusLog.Error(err, "Error converting prometheus value", "prometheus_value", s)
			return -1, err
		}
	}

	return v, nil
}

func (s *prometheusScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := s.ExecutePromQuery()
	if err != nil {
		prometheusLog.Error(err, "error executing prometheus query")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(val), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
