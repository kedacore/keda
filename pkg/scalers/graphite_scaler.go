package scalers

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	url_pkg "net/url"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kedacore/keda/v2/pkg/scalers/authentication"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	grapServerAddress = "serverAddress"
	grapMetricName    = "metricName"
	grapQuery         = "query"
	grapThreshold     = "threshold"
	grapqueryTime     = "queryTime"
)

type graphiteScaler struct {
	metadata *graphiteMetadata
}

type graphiteMetadata struct {
	serverAddress string
	metricName    string
	query         string
	threshold     int
	from          string

	// basic auth
	enableBasicAuth bool
	username        string
	password        string // +optional
}

var graphiteLog = logf.Log.WithName("graphite_scaler")

// NewGraphiteScaler creates a new graphiteScaler
func NewGraphiteScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseGraphiteMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing graphite metadata: %s", err)
	}
	return &graphiteScaler{
		metadata: meta,
	}, nil
}

func parseGraphiteMetadata(config *ScalerConfig) (*graphiteMetadata, error) {
	meta := graphiteMetadata{}

	if val, ok := config.TriggerMetadata[grapServerAddress]; ok && val != "" {
		meta.serverAddress = val
	} else {
		return nil, fmt.Errorf("no %s given", grapServerAddress)
	}

	if val, ok := config.TriggerMetadata[grapQuery]; ok && val != "" {
		meta.query = val
	} else {
		return nil, fmt.Errorf("no %s given", grapQuery)
	}

	if val, ok := config.TriggerMetadata[grapMetricName]; ok && val != "" {
		meta.metricName = val
	} else {
		return nil, fmt.Errorf("no %s given", grapMetricName)
	}

	if val, ok := config.TriggerMetadata[grapqueryTime]; ok && val != "" {
		meta.from = val
	} else {
		return nil, fmt.Errorf("no %s given", grapqueryTime)
	}

	if val, ok := config.TriggerMetadata[grapThreshold]; ok && val != "" {
		t, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %s", grapThreshold, err)
		}

		meta.threshold = t
	}

	authModes, ok := config.TriggerMetadata["authModes"]
	// no authMode specified
	if !ok {
		return &meta, nil
	}

	authTypes := strings.Split(authModes, ",")
	for _, t := range authTypes {
		authType := authentication.Type(strings.TrimSpace(t))
		switch authType {
		case authentication.BasicAuthType:
			if len(config.AuthParams["username"]) == 0 {
				return nil, errors.New("no username given")
			}

			meta.username = config.AuthParams["username"]
			// password is optional. For convenience, many application implement basic auth with
			// username as apikey and password as empty
			meta.password = config.AuthParams["password"]
			meta.enableBasicAuth = true
		default:
			return nil, fmt.Errorf("err incorrect value for authMode is given: %s", t)
		}
	}

	return &meta, nil
}

func (s *graphiteScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := s.ExecuteGrapQuery()
	if err != nil {
		graphiteLog.Error(err, "error executing graphite query")
		return false, err
	}

	return val > 0, nil
}

func (s *graphiteScaler) Close() error {
	return nil
}

func (s *graphiteScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricValue := resource.NewQuantity(int64(s.metadata.threshold), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s", "graphite", s.metadata.serverAddress, s.metadata.metricName)),
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

func (s *graphiteScaler) ExecuteGrapQuery() (float64, error) {
	client := &http.Client{}
	queryEscaped := url_pkg.QueryEscape(s.metadata.query)
	url := fmt.Sprintf("%s/render?from=%s&target=%s&format=json", s.metadata.serverAddress, s.metadata.from, queryEscaped)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return -1, err
	}
	if s.metadata.enableBasicAuth {
		req.SetBasicAuth(s.metadata.username, s.metadata.password)
	}
	r, err := client.Do(req)
	if err != nil {
		return -1, err
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return -1, err
	}
	r.Body.Close()

	result := gjson.GetBytes(b, "0.datapoints.#.0")
	var v float64 = -1
	for _, valur := range result.Array() {
		if valur.String() != "" {
			if float64(valur.Int()) > v {
				v = float64(valur.Int())
			}
		}
	}
	return v, nil
}

func (s *graphiteScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := s.ExecuteGrapQuery()
	if err != nil {
		graphiteLog.Error(err, "error executing graphite query")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(val), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
