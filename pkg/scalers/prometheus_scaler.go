package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	url_pkg "net/url"
	"strconv"
	"time"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	promServerAddress = "serverAddress"
	promMetricName    = "metricName"
	promQuery         = "query"
	promThreshold     = "threshold"
)

type prometheusScaler struct {
	metadata   *prometheusMetadata
	httpClient *http.Client
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
func NewPrometheusScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parsePrometheusMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing prometheus metadata: %s", err)
	}

	return &prometheusScaler{
		metadata:   meta,
		httpClient: kedautil.CreateHTTPClient(config.GlobalHTTPTimeout),
	}, nil
}

func parsePrometheusMetadata(config *ScalerConfig) (*prometheusMetadata, error) {
	meta := prometheusMetadata{}

	if val, ok := config.TriggerMetadata[promServerAddress]; ok && val != "" {
		meta.serverAddress = val
	} else {
		return nil, fmt.Errorf("no %s given", promServerAddress)
	}

	if val, ok := config.TriggerMetadata[promQuery]; ok && val != "" {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no %s given", promQuery)
	}

	if val, ok := config.TriggerMetadata[promMetricName]; ok && val != "" {
		meta.metricName = val
	} else {
		return nil, fmt.Errorf("no %s given", promMetricName)
	}

	if val, ok := config.TriggerMetadata[promThreshold]; ok && val != "" {
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

func (s *prometheusScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(int64(s.metadata.threshold), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s", "prometheus", s.metadata.serverAddress, s.metadata.metricName)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricValue,
		},
	}
	metricSpec := v2beta2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2beta2.MetricSpec{metricSpec}
}

func (s *prometheusScaler) ExecutePromQuery() (float64, error) {
	t := time.Now().UTC().Format(time.RFC3339)
	queryEscaped := url_pkg.QueryEscape(s.metadata.query)
	url := fmt.Sprintf("%s/api/v1/query?query=%s&time=%s", s.metadata.serverAddress, queryEscaped, t)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return -1, err
	}
	r, err := s.httpClient.Do(req)
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

	// allow for zero element or single element result sets
	if len(result.Data.Result) == 0 {
		return 0, nil
	} else if len(result.Data.Result) > 1 {
		return -1, fmt.Errorf("prometheus query %s returned multiple elements", s.metadata.query)
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
