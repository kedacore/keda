package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type seleniumGridScaler struct {
	metricType v2.MetricTargetType
	metadata   *seleniumGridScalerMetadata
	httpClient *http.Client
	logger     logr.Logger
}

type seleniumGridScalerMetadata struct {
	triggerIndex int

	URL                   string `keda:"name=url,                     order=triggerMetadata;authParams"`
	BrowserName           string `keda:"name=browserName,             order=triggerMetadata"`
	SessionBrowserName    string `keda:"name=sessionBrowserName,      order=triggerMetadata, optional"`
	ActivationThreshold   int64  `keda:"name=activationThreshold,     order=triggerMetadata, optional"`
	BrowserVersion        string `keda:"name=browserVersion,          order=triggerMetadata, optional, default=latest"`
	UnsafeSsl             bool   `keda:"name=unsafeSsl,               order=triggerMetadata, optional, default=false"`
	PlatformName          string `keda:"name=platformName,            order=triggerMetadata, optional, default=linux"`
	SessionsPerNode       int64  `keda:"name=sessionsPerNode,         order=triggerMetadata, optional, default=1"`
	SetSessionsFromHub    bool   `keda:"name=setSessionsFromHub,      order=triggerMetadata, optional, default=false"`
	SessionBrowserVersion string `keda:"name=sessionBrowserVersion,   order=triggerMetadata, optional"`

	TargetValue int64
}

type seleniumResponse struct {
	Data data `json:"data"`
}

type data struct {
	Grid         grid         `json:"grid"`
	NodesInfo    nodesInfo    `json:"nodesInfo"`
	SessionsInfo sessionsInfo `json:"sessionsInfo"`
}

type grid struct {
	MaxSession int `json:"maxSession"`
	NodeCount  int `json:"nodeCount"`
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
	PlatformName   string `json:"platformName"`
}

type nodesInfo struct {
	Nodes []nodes `json:"nodes"`
}

type nodes struct {
	Stereotypes string `json:"stereotypes"`
}

type stereotype struct {
	Slots      int64      `json:"slots"`
	Stereotype capability `json:"stereotype"`
}

const (
	DefaultBrowserVersion string = "latest"
	DefaultPlatformName   string = "linux"
)

func NewSeleniumGridScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "selenium_grid_scaler")

	meta, err := parseSeleniumGridScalerMetadata(config)

	if err != nil {
		return nil, fmt.Errorf("error parsing selenium grid metadata: %w", err)
	}

	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, meta.UnsafeSsl)

	return &seleniumGridScaler{
		metricType: metricType,
		metadata:   meta,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

func parseSeleniumGridScalerMetadata(config *scalersconfig.ScalerConfig) (*seleniumGridScalerMetadata, error) {
	meta := &seleniumGridScalerMetadata{
		TargetValue: 1,
	}

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing prometheus metadata: %w", err)
	}

	meta.triggerIndex = config.TriggerIndex

	if meta.SessionBrowserName == "" {
		meta.SessionBrowserName = meta.BrowserName
	}
	if meta.SessionBrowserVersion == "" {
		meta.SessionBrowserVersion = meta.BrowserVersion
	}
	return meta, nil
}

// No cleanup required for selenium grid scaler
func (s *seleniumGridScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

func (s *seleniumGridScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	sessions, err := s.getSessionsCount(ctx, s.logger)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error requesting selenium grid endpoint: %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(sessions))

	return []external_metrics.ExternalMetricValue{metric}, sessions > s.metadata.ActivationThreshold, nil
}

func (s *seleniumGridScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("seleniumgrid-%s", s.metadata.BrowserName))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetValue),
	}
	metricSpec := v2.MetricSpec{
		External: externalMetric, Type: externalMetricType,
	}
	return []v2.MetricSpec{metricSpec}
}

func (s *seleniumGridScaler) getSessionsCount(ctx context.Context, logger logr.Logger) (int64, error) {
	body, err := json.Marshal(map[string]string{
		"query": "{ grid { maxSession, nodeCount }, nodesInfo { nodes { stereotypes } }, sessionsInfo { sessionQueueRequests, sessions { id, capabilities, nodeId } } }",
	})

	if err != nil {
		return -1, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.metadata.URL, bytes.NewBuffer(body))
	if err != nil {
		return -1, err
	}

	res, err := s.httpClient.Do(req)
	if err != nil {
		return -1, err
	}

	if res.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("selenium grid returned %d", res.StatusCode)
		return -1, errors.New(msg)
	}

	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return -1, err
	}
	v, err := getCountFromSeleniumResponse(b, s.metadata.BrowserName, s.metadata.BrowserVersion, s.metadata.SessionBrowserName, s.metadata.PlatformName, s.metadata.SessionsPerNode, s.metadata.SetSessionsFromHub, s.metadata.SessionBrowserVersion, logger)
	if err != nil {
		return -1, err
	}
	return v, nil
}

func getCountFromSeleniumResponse(b []byte, browserName string, browserVersion string, sessionBrowserName string, platformName string, sessionsPerNode int64, setSessionsFromHub bool, sessionBrowserVersion string, logger logr.Logger) (int64, error) {
	var count int64
	var slots int64
	var seleniumResponse = seleniumResponse{}

	if err := json.Unmarshal(b, &seleniumResponse); err != nil {
		return 0, err
	}

	if setSessionsFromHub {
		var nodes = seleniumResponse.Data.NodesInfo.Nodes
	slots:
		for _, node := range nodes {
			var stereotypes = []stereotype{}
			if err := json.Unmarshal([]byte(node.Stereotypes), &stereotypes); err == nil {
				for _, stereotype := range stereotypes {
					if stereotype.Stereotype.BrowserName == browserName {
						var platformNameMatches = stereotype.Stereotype.PlatformName == "" || strings.EqualFold(stereotype.Stereotype.PlatformName, platformName)
						if strings.HasPrefix(stereotype.Stereotype.BrowserVersion, browserVersion) && platformNameMatches {
							slots = stereotype.Slots
							break slots
						} else if len(strings.TrimSpace(stereotype.Stereotype.BrowserVersion)) == 0 && browserVersion == DefaultBrowserVersion && platformNameMatches {
							slots = stereotype.Slots
							break slots
						}
					}
				}
			} else {
				logger.Error(err, fmt.Sprintf("Error when unmarshalling stereotypes: %s", err))
			}
		}
	}

	var sessionQueueRequests = seleniumResponse.Data.SessionsInfo.SessionQueueRequests
	for _, sessionQueueRequest := range sessionQueueRequests {
		var capability = capability{}
		if err := json.Unmarshal([]byte(sessionQueueRequest), &capability); err == nil {
			if capability.BrowserName == browserName {
				var platformNameMatches = capability.PlatformName == "" || strings.EqualFold(capability.PlatformName, platformName)
				if strings.HasPrefix(capability.BrowserVersion, browserVersion) && platformNameMatches {
					count++
				} else if len(strings.TrimSpace(capability.BrowserVersion)) == 0 && browserVersion == DefaultBrowserVersion && platformNameMatches {
					count++
				}
			}
		} else {
			logger.Error(err, fmt.Sprintf("Error when unmarshaling session queue requests: %s", err))
		}
	}

	var sessions = seleniumResponse.Data.SessionsInfo.Sessions
	for _, session := range sessions {
		var capability = capability{}
		if err := json.Unmarshal([]byte(session.Capabilities), &capability); err == nil {
			var platformNameMatches = capability.PlatformName == "" || strings.EqualFold(capability.PlatformName, platformName)
			if capability.BrowserName == sessionBrowserName {
				if strings.HasPrefix(capability.BrowserVersion, sessionBrowserVersion) && platformNameMatches {
					count++
				} else if browserVersion == DefaultBrowserVersion && platformNameMatches {
					count++
				}
			}
		} else {
			logger.Error(err, fmt.Sprintf("Error when unmarshaling sessions info: %s", err))
		}
	}

	var gridMaxSession = int64(seleniumResponse.Data.Grid.MaxSession)
	var gridNodeCount = int64(seleniumResponse.Data.Grid.NodeCount)

	if setSessionsFromHub {
		if slots == 0 {
			slots = sessionsPerNode
		}
		var floatCount = float64(count) / float64(slots)
		count = int64(math.Ceil(floatCount))
	} else if gridMaxSession > 0 && gridNodeCount > 0 {
		// Get count, convert count to next highest int64
		var floatCount = float64(count) / (float64(gridMaxSession) / float64(gridNodeCount))
		count = int64(math.Ceil(floatCount))
	}

	return count, nil
}
