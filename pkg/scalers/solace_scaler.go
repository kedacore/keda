package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	solaceExtMetricType = "External"
	solaceScalerID      = "solace"

	// REST ENDPOINT String Patterns
	solaceSempQueryFieldURLSuffix = "?select=msgs,msgSpoolUsage,averageRxMsgRate"
	solaceSempEndpointURLTemplate = "%s/%s/%s/monitor/msgVpns/%s/%ss/%s" + solaceSempQueryFieldURLSuffix
	solaceSempVpnStateURLTemplate = "%s/%s/%s/monitor/msgVpns/%s" + "?select=state"

	// SEMP REST API Context
	solaceAPIName            = "SEMP"
	solaceAPIVersion         = "v2"
	solaceAPIObjectTypeQueue = "queue"

	// YAML Configuration Metadata Field Names
	// Broker Identifiers
	solaceMetaSempBaseURL = "solaceSempBaseURL"

	// Credential Identifiers
	solaceMetaUsername        = "username"
	solaceMetaPassword        = "password"
	solaceMetaUsernameFromEnv = "usernameFromEnv"
	solaceMetaPasswordFromEnv = "passwordFromEnv"

	// Target Object Identifiers
	solaceMetaMsgVpn    = "messageVpn"
	solaceMetaQueueName = "queueName"

	// Metric Targets
	solaceMetaMsgCountTarget      = "messageCountTarget"
	solaceMetaMsgSpoolUsageTarget = "messageSpoolUsageTarget"
	solaceMetaMsgRxRateTarget     = "messageReceiveRateTarget"

	// Metric Activation Targets
	solaceMetaActivationMsgCountTarget      = "activationMessageCountTarget"
	solaceMetaActivationMsgSpoolUsageTarget = "activationMessageSpoolUsageTarget"
	solaceMetaActivationMsgRxRateTarget     = "activationMessageReceiveRateTarget"

	// Trigger type identifiers
	solaceTriggermsgcount      = "msgcount"
	solaceTriggermsgspoolusage = "msgspoolusage"
	solaceTriggermsgrxrate     = "msgrcvrate"
)

// SolaceMetricValues is the struct for Observed Metric Values
type SolaceMetricValues struct {
	//	Observed Message Count
	msgCount int
	//	Observed Message Spool Usage
	msgSpoolUsage int
	//  Observed Message Received Rate
	msgRcvRate int
}

type SolaceScaler struct {
	metricType v2.MetricTargetType
	metadata   *SolaceMetadata
	httpClient *http.Client
	logger     logr.Logger
	curHostIdx int
}

type SolaceMetadata struct {
	// Scaler index
	triggerIndex int

	SolaceMetaSempBaseURL string `keda:"name=solaceSempBaseURL,  order=triggerMetadata"`

	// Full SEMP URL to target queue (CONSTRUCTED IN CODE)
	endpointURLsList []string
	vpnStateURLsList []string

	// Solace Message VPN
	MessageVpn string `keda:"name=messageVpn,   order=triggerMetadata"`
	QueueName  string `keda:"name=queueName,    order=triggerMetadata"`

	// Basic Auth Username
	Username string `keda:"name=username, order=authParams;triggerMetadata;resolvedEnv"`
	// Basic Auth Password
	Password string `keda:"name=password, order=authParams;triggerMetadata;resolvedEnv"`

	// Target Message Count
	MsgCountTarget      int64 `keda:"name=messageCountTarget,       order=triggerMetadata, optional"`
	MsgSpoolUsageTarget int64 `keda:"name=messageSpoolUsageTarget,  order=triggerMetadata, optional"` // Spool Use Target in Megabytes
	MsgRxRateTarget     int64 `keda:"name=messageReceiveRateTarget, order=triggerMetadata, optional"` // Ingress Rate Target per consumer in msgs/second

	// Activation Target Message Count
	ActivationMsgCountTarget      int `keda:"name=activationMessageCountTarget,       order=triggerMetadata, default=0"`
	ActivationMsgSpoolUsageTarget int `keda:"name=activationMessageSpoolUsageTarget,  order=triggerMetadata, default=0"` // Spool Use Target in Megabytes
	ActivationMsgRxRateTarget     int `keda:"name=activationMessageReceiveRateTarget, order=triggerMetadata, default=0"` // Ingress Rate Target per consumer in msgs/second
}

func (s *SolaceMetadata) Validate() error {
	//	Check that we have at least one positive target value for the scaler
	if s.MsgCountTarget < 1 && s.MsgSpoolUsageTarget < 1 && s.MsgRxRateTarget < 1 {
		return fmt.Errorf("no target value found in the scaler configuration")
	}

	// Convert Megabyte values to Bytes
	s.MsgSpoolUsageTarget = s.MsgSpoolUsageTarget * 1024 * 1024
	s.ActivationMsgSpoolUsageTarget = s.ActivationMsgSpoolUsageTarget * 1024 * 1024

	return nil
}

// SEMP API Response Root Struct for VPNs
type solaceVpnSEMPResponse struct {
	Data solaceVpnSEMPData  `json:"data"`
	Meta solaceSEMPMetadata `json:"meta"`
}

// SEMP API Response Queue Data Struct
type solaceVpnSEMPData struct {
	State string `json:"state"`
}

// SEMP API Response Root Struct for Queues
type solaceQueueSEMPResponse struct {
	Collections solaceQueueSEMPCollections `json:"collections"`
	Data        solaceQueueSEMPData        `json:"data"`
	Meta        solaceSEMPMetadata         `json:"meta"`
}

// SEMP API Response Collections Struct
type solaceQueueSEMPCollections struct {
	Msgs solaceSEMPMessages `json:"msgs"`
}

// SEMP API Response Queue Data Struct
type solaceQueueSEMPData struct {
	MsgSpoolUsage int `json:"msgSpoolUsage"`
	MsgRcvRate    int `json:"averageRxMsgRate"`
}

// SEMP API Messages Struct
type solaceSEMPMessages struct {
	Count int `json:"count"`
}

// SEMP API Metadata Struct
type solaceSEMPMetadata struct {
	ResponseCode int `json:"responseCode"`
}

// NewSolaceScaler is the constructor for SolaceScaler
func NewSolaceScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	// Create HTTP Client
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, solaceScalerID+"_scaler")

	// Parse Solace Metadata
	solaceMetadata, err := parseSolaceMetadata(config)
	if err != nil {
		logger.Error(err, "Error parsing Solace Trigger Metadata or missing values")
		return nil, err
	}

	return &SolaceScaler{
		metricType: metricType,
		metadata:   solaceMetadata,
		httpClient: httpClient,
		logger:     logger,
		curHostIdx: 0,
	}, nil
}

// Called by constructor
func parseSolaceMetadata(config *scalersconfig.ScalerConfig) (*SolaceMetadata, error) {
	meta := &SolaceMetadata{}
	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing prometheus metadata: %w", err)
	}
	meta.triggerIndex = config.TriggerIndex

	// Format Solace SEMP Queue Endpoint (REST URL)
	sempURLs := strings.Split(meta.SolaceMetaSempBaseURL, ",")
	for i := 0; i < len(sempURLs); i++ {
		sempURL := strings.TrimSpace(sempURLs[i])
		meta.endpointURLsList = append(meta.endpointURLsList, fmt.Sprintf(
			solaceSempEndpointURLTemplate,
			sempURL,
			solaceAPIName,
			solaceAPIVersion,
			meta.MessageVpn,
			solaceAPIObjectTypeQueue,
			url.QueryEscape(meta.QueueName),
		))
		meta.vpnStateURLsList = append(meta.vpnStateURLsList, fmt.Sprintf(
			solaceSempVpnStateURLTemplate,
			sempURL,
			solaceAPIName,
			solaceAPIVersion,
			url.QueryEscape(meta.MessageVpn),
		))
	}

	return meta, nil
}

// INTERFACE METHOD
// DEFINE METRIC FOR SCALING
// CURRENT SUPPORTED METRICS ARE:
// - QUEUE MESSAGE COUNT (msgCount)
// - QUEUE SPOOL USAGE   (msgSpoolUsage in MBytes)
// METRIC IDENTIFIER HAS THE SIGNATURE:
// - solace-[Queue_Name]-[metric_type]
// e.g. solace-QUEUE1-msgCount
func (s *SolaceScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	var metricSpecList []v2.MetricSpec
	// Message Count Target Spec
	if s.metadata.MsgCountTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-%s-%s", s.metadata.QueueName, solaceTriggermsgcount))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.metadata.MsgCountTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceExtMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	// Message Spool Usage Target Spec
	if s.metadata.MsgSpoolUsageTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-%s-%s", s.metadata.QueueName, solaceTriggermsgspoolusage))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.metadata.MsgSpoolUsageTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceExtMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	// Message Receive Rate Target Spec
	if s.metadata.MsgRxRateTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-%s-%s", s.metadata.QueueName, solaceTriggermsgrxrate))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.metadata.MsgRxRateTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceExtMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	return metricSpecList
}

func (s *SolaceScaler) getVpnState(ctx context.Context, sempURL string) (string, error) {
	var httpClient = s.httpClient
	var sempResponse solaceVpnSEMPResponse

	// Define HTTP Request
	request, err := http.NewRequestWithContext(ctx, "GET", sempURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed attempting request to solace semp api: %s url: %s", err.Error(), sempURL)
	}

	// Add HTTP Auth and Headers
	request.SetBasicAuth(s.metadata.Username, s.metadata.Password)
	request.Header.Set("Content-Type", "application/json")

	// Call Solace SEMP API
	response, err := httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("call to solace semp api failed: %s, url: %s", err.Error(), sempURL)
	}
	defer response.Body.Close()

	// Check HTTP Status Code
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", fmt.Errorf("semp request http status code: %d - %s, url: %s", response.StatusCode, response.Status, sempURL)
	}

	// Decode SEMP Response and Test
	if err := json.NewDecoder(response.Body).Decode(&sempResponse); err != nil {
		return "", fmt.Errorf("failed to read semp response body: %s, url: %s", err.Error(), sempURL)
	}

	if sempResponse.Meta.ResponseCode < 200 || sempResponse.Meta.ResponseCode > 299 {
		return "", fmt.Errorf("solace semp api returned error status: %d, URL: %s", sempResponse.Meta.ResponseCode, sempURL)
	}

	return sempResponse.Data.State, nil
}

func (s *SolaceScaler) getQueueMetrics(ctx context.Context, sempURL string) (SolaceMetricValues, error) {
	var httpClient = s.httpClient
	var sempResponse solaceQueueSEMPResponse
	var metricValues SolaceMetricValues

	// Define HTTP Request
	request, err := http.NewRequestWithContext(ctx, "GET", sempURL, nil)
	if err != nil {
		return SolaceMetricValues{}, fmt.Errorf("failed attempting request to solace semp api: %s, url: %s", err.Error(), sempURL)
	}

	// Add HTTP Auth and Headers
	request.SetBasicAuth(s.metadata.Username, s.metadata.Password)
	request.Header.Set("Content-Type", "application/json")

	// Call Solace SEMP API
	response, err := httpClient.Do(request)
	if err != nil {
		return SolaceMetricValues{}, fmt.Errorf("call to solace semp api failed: %s, url: %s", err.Error(), sempURL)
	}
	defer response.Body.Close()

	// Check HTTP Status Code
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return SolaceMetricValues{}, fmt.Errorf("semp request http status code: %d - %s, url: %s", response.StatusCode, response.Status, sempURL)
	}

	// Decode SEMP Response and Test
	if err := json.NewDecoder(response.Body).Decode(&sempResponse); err != nil {
		return SolaceMetricValues{}, fmt.Errorf("failed to read semp response body: %s, url: %s", err.Error(), sempURL)
	}

	if sempResponse.Meta.ResponseCode < 200 || sempResponse.Meta.ResponseCode > 299 {
		return SolaceMetricValues{}, fmt.Errorf("solace semp api returned error status: %d, url: %s", sempResponse.Meta.ResponseCode, sempURL)
	}

	metricValues.msgCount = sempResponse.Collections.Msgs.Count
	metricValues.msgSpoolUsage = sempResponse.Data.MsgSpoolUsage
	metricValues.msgRcvRate = sempResponse.Data.MsgRcvRate

	return metricValues, nil
}

// returns SolaceMetricValues struct populated from broker  SEMP endpoint
func (s *SolaceScaler) getSolaceQueueMetricsFromSEMP(ctx context.Context) (SolaceMetricValues, error) {
	var metricValues SolaceMetricValues
	var errorList []string

	// RETRIEVE METRICS FROM SOLACE SEMP API
	for i := 0; i < len(s.metadata.endpointURLsList); i++ {
		idx := (s.curHostIdx + i) % len(s.metadata.endpointURLsList)
		sempQueueURL := s.metadata.endpointURLsList[idx]
		sempVpnStateURL := s.metadata.vpnStateURLsList[idx]

		vpnState, err := s.getVpnState(ctx, sempVpnStateURL)
		if err != nil {
			errorList = append(errorList, "Host "+strconv.Itoa(idx+1)+" Error: "+err.Error())
			continue
		}

		if vpnState != "up" {
			errorList = append(errorList, "Host "+strconv.Itoa(idx+1)+" Error: Message vpn is not up ("+vpnState+") url: "+sempVpnStateURL)
			continue
		}

		metricValues, err = s.getQueueMetrics(ctx, sempQueueURL)
		if err != nil {
			errorList = append(errorList, "Host "+strconv.Itoa(idx+1)+" Error: "+err.Error())
			continue
		}

		// Found active broker, update currHostIdx if it has changed
		if idx != s.curHostIdx {
			s.curHostIdx = idx
		}

		return metricValues, nil
	}

	return SolaceMetricValues{}, fmt.Errorf("unable to collect metrics, error(s): %s", strings.Join(errorList, "\n  "))
}

// INTERFACE METHOD
// Call SEMP API to retrieve metrics
// returns value for named metric
// returns true if queue messageCount > 0 || msgSpoolUsage > 0
func (s *SolaceScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	var metricValues, mv SolaceMetricValues
	var mve error

	if mv, mve = s.getSolaceQueueMetricsFromSEMP(ctx); mve != nil {
		return []external_metrics.ExternalMetricValue{}, false, mve
	}
	metricValues = mv

	var metric external_metrics.ExternalMetricValue
	switch {
	case strings.HasSuffix(metricName, solaceTriggermsgcount):
		metric = GenerateMetricInMili(metricName, float64(metricValues.msgCount))
	case strings.HasSuffix(metricName, solaceTriggermsgspoolusage):
		metric = GenerateMetricInMili(metricName, float64(metricValues.msgSpoolUsage))
	case strings.HasSuffix(metricName, solaceTriggermsgrxrate):
		metric = GenerateMetricInMili(metricName, float64(metricValues.msgRcvRate))
	default:
		// Should never end up here
		err := fmt.Errorf("unidentified metric: %s", metricName)
		s.logger.Error(err, "returning error to calling app")
		return []external_metrics.ExternalMetricValue{}, false, err
	}
	return []external_metrics.ExternalMetricValue{metric},
		metricValues.msgCount > s.metadata.ActivationMsgCountTarget ||
			metricValues.msgSpoolUsage > s.metadata.ActivationMsgSpoolUsageTarget ||
			metricValues.msgRcvRate > s.metadata.ActivationMsgRxRateTarget,
		nil
}

// Do Nothing - Satisfies Interface
func (s *SolaceScaler) Close(context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}
