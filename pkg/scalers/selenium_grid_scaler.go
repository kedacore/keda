package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type seleniumGridScaler struct {
	metadata *seleniumGridScalerMetadata
	client   *http.Client
}

type seleniumGridScalerMetadata struct {
	url            string
	browserName    string
	targetValue    int64
	browserVersion string
}

type seleniumResponse struct {
	Data data `json:"data"`
}

type data struct {
	SessionsInfo sessionsInfo `json:"sessionsInfo"`
}

type sessionsInfo struct {
	SessionQueueRequests []string          `json:"sessionQueueRequests"`
	Sessions             []seleniumSession `json:"sessions"`
}

type seleniumSession struct {
	ID           string `json:"id"`
	Capabilities string `json:"capabilities"`
	NodeID       string `json:"nodeId"`
}

type capability struct {
	BrowserName    string `json:"browserName"`
	BrowserVersion string `json:"browserVersion"`
}

const (
	DefaultBrowserVersion string = "latest"
)

func NewSeleniumGridScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseSeleniumGridScalerMetadata(config)

	if err != nil {
		return nil, fmt.Errorf("error parsing selenium grid metadata: %s", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout)

	return &seleniumGridScaler{
		metadata: meta,
		client:   httpClient,
	}, nil
}

func parseSeleniumGridScalerMetadata(config *ScalerConfig) (*seleniumGridScalerMetadata, error) {
	meta := seleniumGridScalerMetadata{
		targetValue: 1,
	}

	if val, ok := config.TriggerMetadata["url"]; ok {
		meta.url = val
	} else {
		return nil, fmt.Errorf("no selenium grid url given in metadata")
	}

	if val, ok := config.TriggerMetadata["browserName"]; ok {
		meta.browserName = val
	} else {
		return nil, fmt.Errorf("no browser name given in metadata")
	}

	if val, ok := config.TriggerMetadata["browserVersion"]; ok && val != "" {
		meta.browserVersion = val
	} else {
		meta.browserVersion = DefaultBrowserVersion
	}

	return &meta, nil
}

// No cleanup required for selenium grid scaler
func (s *seleniumGridScaler) Close() error {
	return nil
}

func (s *seleniumGridScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	v, err := s.getSessionsCount()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, fmt.Errorf("error requesting selenium grid endpoint: %s", err)
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *v,
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *seleniumGridScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetValue := resource.NewQuantity(s.metadata.targetValue, resource.DecimalSI)
	metricName := kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s", "seleniumgrid", s.metadata.browserName, s.metadata.browserVersion))
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

func (s *seleniumGridScaler) IsActive(ctx context.Context) (bool, error) {
	v, err := s.getSessionsCount()
	if err != nil {
		return false, err
	}

	return v.AsApproximateFloat64() > 0.0, nil
}

func (s *seleniumGridScaler) getSessionsCount() (*resource.Quantity, error) {
	body, err := json.Marshal(map[string]string{
		"query": "{ sessionsInfo { sessionQueueRequests, sessions { id, capabilities, nodeId } } }",
	})

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", s.metadata.url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	res, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("selenium grid returned %d", res.StatusCode)
		return nil, errors.New(msg)
	}

	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	v, err := getCountFromSeleniumResponse(b, s.metadata.browserName, s.metadata.browserVersion)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func getCountFromSeleniumResponse(b []byte, browserName string, browserVersion string) (*resource.Quantity, error) {
	var count int64
	var seleniumResponse = seleniumResponse{}

	if err := json.Unmarshal(b, &seleniumResponse); err != nil {
		return nil, err
	}

	var sessionQueueRequests = seleniumResponse.Data.SessionsInfo.SessionQueueRequests
	for _, sessionQueueRequest := range sessionQueueRequests {
		var capability = capability{}
		if err := json.Unmarshal([]byte(sessionQueueRequest), &capability); err == nil {
			if capability.BrowserName == browserName {
				if strings.HasPrefix(capability.BrowserVersion, browserVersion) {
					count++
				} else if capability.BrowserVersion == "" && browserVersion == DefaultBrowserVersion {
					count++
				}
			}
		}
	}

	var sessions = seleniumResponse.Data.SessionsInfo.Sessions
	for _, session := range sessions {
		var capability = capability{}
		if err := json.Unmarshal([]byte(session.Capabilities), &capability); err == nil {
			if capability.BrowserName == browserName {
				if strings.HasPrefix(capability.BrowserVersion, browserVersion) {
					count++
				} else if browserVersion == DefaultBrowserVersion {
					count++
				}
			}
		}
	}

	return resource.NewQuantity(count, resource.DecimalSI), nil
}
