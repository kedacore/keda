package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

	URL                 string `keda:"name=url,                      order=authParams;triggerMetadata"`
	AuthType            string `keda:"name=authType,                 order=authParams;resolvedEnv, optional"`
	Username            string `keda:"name=username,                 order=authParams;resolvedEnv, optional"`
	Password            string `keda:"name=password,                 order=authParams;resolvedEnv, optional"`
	AccessToken         string `keda:"name=accessToken,              order=authParams;resolvedEnv, optional"`
	BrowserName         string `keda:"name=browserName,              order=triggerMetadata, optional"`
	SessionBrowserName  string `keda:"name=sessionBrowserName,       order=triggerMetadata, optional"`
	BrowserVersion      string `keda:"name=browserVersion,           order=triggerMetadata, optional"`
	PlatformName        string `keda:"name=platformName,             order=triggerMetadata, optional"`
	ActivationThreshold int64  `keda:"name=activationThreshold,      order=triggerMetadata, optional"`
	UnsafeSsl           bool   `keda:"name=unsafeSsl,                order=triggerMetadata, default=false"`
	NodeMaxSessions     int64  `keda:"name=nodeMaxSessions,          order=triggerMetadata, default=1"`
	Capabilities        string `keda:"name=capabilities,        	 order=triggerMetadata, optional"`

	TargetValue int64
}

type SeleniumResponse struct {
	Data Data `json:"data"`
}

type Data struct {
	Grid         Grid         `json:"grid"`
	NodesInfo    NodesInfo    `json:"nodesInfo"`
	SessionsInfo SessionsInfo `json:"sessionsInfo"`
}

type Grid struct {
	SessionCount int64 `json:"sessionCount"`
	MaxSession   int64 `json:"maxSession"`
	TotalSlots   int64 `json:"totalSlots"`
}

type NodesInfo struct {
	Nodes Nodes `json:"nodes"`
}

type SessionsInfo struct {
	SessionQueueRequests []string `json:"sessionQueueRequests"`
}

type Nodes []struct {
	ID           string   `json:"id"`
	Status       string   `json:"status"`
	SessionCount int64    `json:"sessionCount"`
	MaxSession   int64    `json:"maxSession"`
	SlotCount    int64    `json:"slotCount"`
	Stereotypes  string   `json:"stereotypes"`
	Sessions     Sessions `json:"sessions"`
}

type ReservedNodes struct {
	ID         string `json:"id"`
	MaxSession int64  `json:"maxSession"`
	SlotCount  int64  `json:"slotCount"`
}

type Sessions []struct {
	ID           string `json:"id"`
	Capabilities string `json:"capabilities"`
	Slot         Slot   `json:"slot"`
}

type Slot struct {
	ID         string `json:"id"`
	Stereotype string `json:"stereotype"`
}

type Stereotypes []struct {
	Slots      int64                  `json:"slots"`
	Stereotype map[string]interface{} `json:"stereotype"`
}

var ExtensionCapabilitiesPrefixes = []string{"goog:", "moz:", "ms:", "se:"}
var FunctionCapabilitiesPrefixes = []string{"se:downloadsEnabled"}

// Follow pattern in https://github.com/SeleniumHQ/selenium/blob/trunk/java/src/org/openqa/selenium/grid/data/DefaultSlotMatcher.java
func filterCapabilities(capabilities map[string]interface{}) map[string]interface{} {
	filteredCapabilities := make(map[string]interface{})

	for key, value := range capabilities {
		retain := true
		for _, excludePrefix := range ExtensionCapabilitiesPrefixes {
			if strings.HasPrefix(key, excludePrefix) {
				retain = false
				break
			}
		}
		for _, prefix := range FunctionCapabilitiesPrefixes {
			if strings.HasPrefix(key, prefix) {
				retain = true
				break
			}
		}
		if retain {
			filteredCapabilities[key] = value
		}
	}

	return filteredCapabilities
}

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

	return meta, nil
}

// No cleanup required for Selenium Grid scaler
func (s *seleniumGridScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

func (s *seleniumGridScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	newRequestNodes, onGoingSessions, err := s.getSessionsQueueLength(ctx, s.logger)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error requesting selenium grid endpoint: %w", err)
	}

	metric := GenerateMetricInMili(metricName, float64(newRequestNodes+onGoingSessions))

	return []external_metrics.ExternalMetricValue{metric}, (newRequestNodes + onGoingSessions) > s.metadata.ActivationThreshold, nil
}

func (s *seleniumGridScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("selenium-grid-%s-%s-%s", s.metadata.BrowserName, s.metadata.BrowserVersion, s.metadata.PlatformName))
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

func (s *seleniumGridScaler) getSessionsQueueLength(ctx context.Context, logger logr.Logger) (int64, int64, error) {
	body, err := json.Marshal(map[string]string{
		"query": "{ grid { sessionCount, maxSession, totalSlots }, nodesInfo { nodes { id, status, sessionCount, maxSession, slotCount, stereotypes, sessions { id, capabilities, slot { id, stereotype } } } }, sessionsInfo { sessionQueueRequests } }",
	})

	if err != nil {
		return -1, -1, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.metadata.URL, bytes.NewBuffer(body))
	if err != nil {
		return -1, -1, err
	}

	if (s.metadata.AuthType == "" || strings.EqualFold(s.metadata.AuthType, "Basic")) && s.metadata.Username != "" && s.metadata.Password != "" {
		req.SetBasicAuth(s.metadata.Username, s.metadata.Password)
	} else if !strings.EqualFold(s.metadata.AuthType, "Basic") && s.metadata.AccessToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", s.metadata.AuthType, s.metadata.AccessToken))
	}

	res, err := s.httpClient.Do(req)
	if err != nil {
		return -1, -1, err
	}

	if res.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Selenium Grid returned response status code: %d", res.StatusCode)
		logger.Error(errors.New(msg), msg)
		return -1, -1, errors.New(msg)
	}

	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Error when reading Selenium Grid response body: %s", err))
		return -1, -1, err
	}
	newRequestNodes, onGoingSession, err := getCountFromSeleniumResponse(b, s.metadata.BrowserName, s.metadata.BrowserVersion, s.metadata.SessionBrowserName, s.metadata.PlatformName, s.metadata.NodeMaxSessions, s.metadata.Capabilities, logger)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Error when getting count from Selenium Grid response: %s", err))
		return -1, -1, err
	}
	return newRequestNodes, onGoingSession, nil
}

func getCapability(capability map[string]interface{}, key string) string {
	value, ok := capability[key]
	if ok {
		return value.(string)
	}
	return ""
}

func getBrowserName(capability map[string]interface{}) string {
	return getCapability(capability, "browserName")
}

func getBrowserVersion(capability map[string]interface{}) string {
	return getCapability(capability, "browserVersion")
}

func getPlatformName(capability map[string]interface{}) string {
	return getCapability(capability, "platformName")
}

func countMatchingSlotsStereotypes(stereotypes Stereotypes, browserName string, browserVersion string, sessionBrowserName string, platformName string, capabilities map[string]interface{}) int64 {
	var matchingSlots int64
	for _, stereotype := range stereotypes {
		if checkStereotypeCapabilitiesMatch(stereotype.Stereotype, browserName, browserVersion, sessionBrowserName, platformName, capabilities) {
			matchingSlots += stereotype.Slots
		}
	}
	return matchingSlots
}

func countMatchingSessions(sessions Sessions, browserName string, browserVersion string, sessionBrowserName string, platformName string, capabilities map[string]interface{}, logger logr.Logger) int64 {
	var matchingSessions int64
	for _, session := range sessions {
		var capability map[string]interface{}
		if err := json.Unmarshal([]byte(session.Slot.Stereotype), &capability); err == nil {
			if checkStereotypeCapabilitiesMatch(capability, browserName, browserVersion, sessionBrowserName, platformName, capabilities) {
				matchingSessions++
			}
		} else {
			logger.Error(err, fmt.Sprintf("Error when unmarshaling session capabilities: %s", err))
		}
	}
	return matchingSessions
}

func extensionCapabilitiesMatch(stereotype map[string]interface{}, capabilities map[string]interface{}) bool {
	capabilities = filterCapabilities(capabilities)
	if len(capabilities) == 0 {
		return true
	}
	for key, value := range capabilities {
		if stereotypeValue, ok := stereotype[key]; !ok || stereotypeValue != value {
			return false
		}
	}
	return true
}

// This function checks if the request capabilities match the scaler metadata
func checkRequestCapabilitiesMatch(request map[string]interface{}, browserName string, browserVersion string, _ string, platformName string, capabilities map[string]interface{}) bool {
	// Check if browserName matches
	_browserName := getBrowserName(request)
	browserNameMatch := (_browserName == "" && browserName == "") ||
		strings.EqualFold(browserName, _browserName)

	// Check if browserVersion matches
	_browserVersion := getBrowserVersion(request)
	browserVersionMatch := (_browserVersion == "" && browserVersion == "") ||
		(_browserVersion != "" && strings.HasPrefix(browserVersion, _browserVersion))

	// Check if platformName matches
	_platformName := getPlatformName(request)
	platformNameMatch := (_platformName == "" || strings.EqualFold("any", _platformName) || strings.EqualFold(platformName, _platformName)) &&
		(platformName == "" || platformName == "any" || strings.EqualFold(platformName, _platformName))

	return browserNameMatch && browserVersionMatch && platformNameMatch && extensionCapabilitiesMatch(request, capabilities)
}

// This function checks if Node stereotypes or ongoing sessions match the scaler metadata
func checkStereotypeCapabilitiesMatch(capability map[string]interface{}, browserName string, browserVersion string, sessionBrowserName string, platformName string, capabilities map[string]interface{}) bool {
	// Check if browserName matches
	_browserName := getBrowserName(capability)
	browserNameMatch := (_browserName == "" && browserName == "") ||
		strings.EqualFold(browserName, _browserName) ||
		strings.EqualFold(sessionBrowserName, _browserName)

	// Check if browserVersion matches
	_browserVersion := getBrowserVersion(capability)
	browserVersionMatch := (_browserVersion == "" && browserVersion == "") ||
		(_browserVersion != "" && strings.HasPrefix(browserVersion, _browserVersion))

	// Check if platformName matches
	_platformVersion := getPlatformName(capability)
	platformNameMatch := (_platformVersion == "" || strings.EqualFold("any", _platformVersion) || strings.EqualFold(platformName, _platformVersion)) &&
		(platformName == "" || platformName == "any" || strings.EqualFold(platformName, _platformVersion))

	return browserNameMatch && browserVersionMatch && platformNameMatch && extensionCapabilitiesMatch(capability, capabilities)
}

func checkNodeReservedSlots(reservedNodes []ReservedNodes, nodeID string, availableSlots int64) int64 {
	for _, reservedNode := range reservedNodes {
		if strings.EqualFold(reservedNode.ID, nodeID) {
			return reservedNode.SlotCount
		}
	}
	return availableSlots
}

func updateOrAddReservedNode(reservedNodes []ReservedNodes, nodeID string, slotCount int64, maxSession int64) []ReservedNodes {
	for i, reservedNode := range reservedNodes {
		if strings.EqualFold(reservedNode.ID, nodeID) {
			// Update remaining available slots for the reserved node
			reservedNodes[i].SlotCount = slotCount
			return reservedNodes
		}
	}
	// Add new reserved node if not found
	return append(reservedNodes, ReservedNodes{ID: nodeID, SlotCount: slotCount, MaxSession: maxSession})
}

func getCountFromSeleniumResponse(b []byte, browserName string, browserVersion string, sessionBrowserName string, platformName string, nodeMaxSessions int64, _capabilities string, logger logr.Logger) (int64, int64, error) {
	// Track number of available slots of existing Nodes in the Grid can be reserved for the matched requests
	var availableSlots int64
	// Track number of matched requests in the sessions queue will be served by this scaler
	var queueSlots int64

	var seleniumResponse = SeleniumResponse{}
	if err := json.Unmarshal(b, &seleniumResponse); err != nil {
		return 0, 0, err
	}
	capabilities := map[string]interface{}{}
	if _capabilities != "" {
		if err := json.Unmarshal([]byte(_capabilities), &capabilities); err != nil {
			logger.Error(err, fmt.Sprintf("Error when unmarshaling trigger metadata 'capabilities': %s", err))
			return 0, 0, err
		}
	}

	var sessionQueueRequests = seleniumResponse.Data.SessionsInfo.SessionQueueRequests
	var nodes = seleniumResponse.Data.NodesInfo.Nodes
	// Track list of existing Nodes that have available slots for the matched requests
	var reservedNodes []ReservedNodes
	// Track list of new Nodes will be scaled up with number of available slots following scaler parameter `nodeMaxSessions`
	var newRequestNodes []ReservedNodes
	var onGoingSessions int64
	for requestIndex, sessionQueueRequest := range sessionQueueRequests {
		var isRequestMatched bool
		var requestCapability map[string]interface{}
		if err := json.Unmarshal([]byte(sessionQueueRequest), &requestCapability); err == nil {
			if checkRequestCapabilitiesMatch(requestCapability, browserName, browserVersion, sessionBrowserName, platformName, capabilities) {
				queueSlots++
				isRequestMatched = true
			}
		} else {
			logger.Error(err, fmt.Sprintf("Error when unmarshaling sessionQueueRequest capability: %s", err))
		}

		var isRequestReserved bool
		// Check if the matched request can be assigned to available slots of existing Nodes in the Grid
		for _, node := range nodes {
			// Check if node is UP and has available slots (maxSession > sessionCount)
			if isRequestMatched && strings.EqualFold(node.Status, "UP") && checkNodeReservedSlots(reservedNodes, node.ID, node.MaxSession-node.SessionCount) > 0 {
				var stereotypes = Stereotypes{}
				var availableSlotsMatch int64
				if err := json.Unmarshal([]byte(node.Stereotypes), &stereotypes); err == nil {
					// Count available slots that match the request capability and scaler metadata
					availableSlotsMatch += countMatchingSlotsStereotypes(stereotypes, browserName, browserVersion, sessionBrowserName, platformName, capabilities)
				} else {
					logger.Error(err, fmt.Sprintf("Error when unmarshaling node stereotypes: %s", err))
				}
				if availableSlotsMatch == 0 {
					continue
				}
				// Count ongoing sessions that match the request capability and scaler metadata
				var currentSessionsMatch = countMatchingSessions(node.Sessions, browserName, browserVersion, sessionBrowserName, platformName, capabilities, logger)
				// Count remaining available slots can be reserved for this request
				var availableSlotsCanBeReserved = checkNodeReservedSlots(reservedNodes, node.ID, node.MaxSession-node.SessionCount)
				// Reserve one available slot for the request if available slots match is greater than current sessions match
				if availableSlotsMatch > currentSessionsMatch {
					availableSlots++
					reservedNodes = updateOrAddReservedNode(reservedNodes, node.ID, availableSlotsCanBeReserved-1, node.MaxSession)
					isRequestReserved = true
					break
				}
			}
		}
		// Check if the matched request can be assigned to available slots of new Nodes will be scaled up, since the scaler parameter `nodeMaxSessions` can be greater than 1
		if isRequestMatched && !isRequestReserved {
			for _, newRequestNode := range newRequestNodes {
				if newRequestNode.SlotCount > 0 {
					newRequestNodes = updateOrAddReservedNode(newRequestNodes, newRequestNode.ID, newRequestNode.SlotCount-1, nodeMaxSessions)
					isRequestReserved = true
					break
				}
			}
		}
		// Check if a new Node should be scaled up to reserve for the matched request
		if isRequestMatched && !isRequestReserved {
			newRequestNodes = updateOrAddReservedNode(newRequestNodes, string(rune(requestIndex)), nodeMaxSessions-1, nodeMaxSessions)
		}
	}

	// Count ongoing sessions across all nodes that match the scaler metadata
	for _, node := range nodes {
		onGoingSessions += countMatchingSessions(node.Sessions, browserName, browserVersion, sessionBrowserName, platformName, capabilities, logger)
	}

	return int64(len(newRequestNodes)), onGoingSessions, nil
}
