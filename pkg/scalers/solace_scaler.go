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

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	solaceExtMetricType = "External"
	solaceScalerID      = "solace"

	// REST ENDPOINT String Patterns
	solaceSempQueryFieldURLSuffix = "?select=msgs,msgSpoolUsage,averageRxMsgRate"
	solaceSempEndpointURLTemplate = "%s/%s/%s/monitor/msgVpns/%s/%ss/%s" + solaceSempQueryFieldURLSuffix

	// SEMP REST API Context
	solaceAPIName            = "SEMP"
	solaceAPIVersion         = "v2"
	solaceAPIObjectTypeQueue = "queue"

	// Log Message Templates
	solaceFoundMetaFalse = "required Field %s NOT FOUND in Solace Metadata"

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
}

type SolaceMetadata struct {
	// Full SEMP URL to target queue (CONSTRUCTED IN CODE)
	endpointURL   string
	solaceSempURL string

	// Solace Message VPN
	messageVpn string
	queueName  string

	// Basic Auth Username
	username string
	// Basic Auth Password
	password string

	// Target Message Count
	msgCountTarget      int64
	msgSpoolUsageTarget int64 // Spool Use Target in Megabytes
	msgRxRateTarget     int64 // Ingress Rate Target per consumer in msgs/second

	// Activation Target Message Count
	activationMsgCountTarget      int
	activationMsgSpoolUsageTarget int // Spool Use Target in Megabytes
	activationMsgRxRateTarget     int // Ingress Rate Target per consumer in msgs/second
	// Scaler index
	scalerIndex int
}

// SEMP API Response Root Struct
type solaceSEMPResponse struct {
	Collections solaceSEMPCollections `json:"collections"`
	Data        solaceSEMPData        `json:"data"`
	Meta        solaceSEMPMetadata    `json:"meta"`
}

// SEMP API Response Collections Struct
type solaceSEMPCollections struct {
	Msgs solaceSEMPMessages `json:"msgs"`
}

// SEMP API Response Queue Data Struct
type solaceSEMPData struct {
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
func NewSolaceScaler(config *ScalerConfig) (Scaler, error) {
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
	}, nil
}

// Called by constructor
func parseSolaceMetadata(config *ScalerConfig) (*SolaceMetadata, error) {
	meta := SolaceMetadata{}
	//	GET THE SEMP API ENDPOINT
	if val, ok := config.TriggerMetadata[solaceMetaSempBaseURL]; ok && val != "" {
		meta.solaceSempURL = val
	} else {
		return nil, fmt.Errorf(solaceFoundMetaFalse, solaceMetaSempBaseURL)
	}
	//	GET Message VPN
	if val, ok := config.TriggerMetadata[solaceMetaMsgVpn]; ok && val != "" {
		meta.messageVpn = val
	} else {
		return nil, fmt.Errorf(solaceFoundMetaFalse, solaceMetaMsgVpn)
	}
	//	GET Queue Name
	if val, ok := config.TriggerMetadata[solaceMetaQueueName]; ok && val != "" {
		meta.queueName = val
	} else {
		return nil, fmt.Errorf(solaceFoundMetaFalse, solaceMetaQueueName)
	}

	//	GET METRIC TARGET VALUES
	//	GET msgCountTarget
	meta.msgCountTarget = 0
	if val, ok := config.TriggerMetadata[solaceMetaMsgCountTarget]; ok && val != "" {
		if msgCount, err := strconv.ParseInt(val, 10, 64); err == nil {
			meta.msgCountTarget = msgCount
		} else {
			return nil, fmt.Errorf("can't parse [%s], not a valid integer: %w", solaceMetaMsgCountTarget, err)
		}
	}
	//	GET msgSpoolUsageTarget
	meta.msgSpoolUsageTarget = 0
	if val, ok := config.TriggerMetadata[solaceMetaMsgSpoolUsageTarget]; ok && val != "" {
		if msgSpoolUsage, err := strconv.ParseInt(val, 10, 64); err == nil {
			meta.msgSpoolUsageTarget = msgSpoolUsage * 1024 * 1024
		} else {
			return nil, fmt.Errorf("can't parse [%s], not a valid integer: %w", solaceMetaMsgSpoolUsageTarget, err)
		}
	}
	//  GET msgRcvRateTarget
	meta.msgRxRateTarget = 0
	if val, ok := config.TriggerMetadata[solaceMetaMsgRxRateTarget]; ok && val != "" {
		if msgRcvRate, err := strconv.ParseInt(val, 10, 64); err == nil {
			meta.msgRxRateTarget = msgRcvRate
		} else {
			return nil, fmt.Errorf("can't parse [%s], not a valid integer: %w", solaceMetaMsgRxRateTarget, err)
		}
	}

	//	Check that we have at least one positive target value for the scaler
	if meta.msgCountTarget < 1 && meta.msgSpoolUsageTarget < 1 && meta.msgRxRateTarget < 1 {
		return nil, fmt.Errorf("no target value found in the scaler configuration")
	}

	//	GET ACTIVATION METRIC TARGET VALUES
	//	GET activationMsgCountTarget
	meta.activationMsgCountTarget = 0
	if val, ok := config.TriggerMetadata[solaceMetaActivationMsgCountTarget]; ok && val != "" {
		if activationMsgCountTarget, err := strconv.Atoi(val); err == nil {
			meta.activationMsgCountTarget = activationMsgCountTarget
		} else {
			return nil, fmt.Errorf("can't parse [%s], not a valid integer: %w", solaceMetaActivationMsgCountTarget, err)
		}
	}
	//	GET activationMsgSpoolUsageTarget
	meta.activationMsgSpoolUsageTarget = 0
	if val, ok := config.TriggerMetadata[solaceMetaActivationMsgSpoolUsageTarget]; ok && val != "" {
		if activationMsgSpoolUsageTarget, err := strconv.Atoi(val); err == nil {
			meta.activationMsgSpoolUsageTarget = activationMsgSpoolUsageTarget * 1024 * 1024
		} else {
			return nil, fmt.Errorf("can't parse [%s], not a valid integer: %w", solaceMetaActivationMsgSpoolUsageTarget, err)
		}
	}
	meta.activationMsgRxRateTarget = 0
	if val, ok := config.TriggerMetadata[solaceMetaActivationMsgRxRateTarget]; ok && val != "" {
		if activationMsgRxRateTarget, err := strconv.Atoi(val); err == nil {
			meta.activationMsgRxRateTarget = activationMsgRxRateTarget
		} else {
			return nil, fmt.Errorf("can't parse [%s], not a valid integer: %w", solaceMetaActivationMsgRxRateTarget, err)
		}
	}

	// Format Solace SEMP Queue Endpoint (REST URL)
	meta.endpointURL = fmt.Sprintf(
		solaceSempEndpointURLTemplate,
		meta.solaceSempURL,
		solaceAPIName,
		solaceAPIVersion,
		meta.messageVpn,
		solaceAPIObjectTypeQueue,
		url.QueryEscape(meta.queueName),
	)

	// Get Credentials
	var e error
	if meta.username, meta.password, e = getSolaceSempCredentials(config); e != nil {
		return nil, e
	}

	meta.scalerIndex = config.ScalerIndex

	return &meta, nil
}

func getSolaceSempCredentials(config *ScalerConfig) (u string, p string, err error) {
	//	GET CREDENTIALS
	//	The username must be a valid broker ADMIN user identifier with read access to SEMP for the broker, VPN, and relevant objects
	//	The scaler will attempt to acquire username and then password independently. For each:
	//	- Search K8S Secret (Encoded)
	//	- Search environment variable specified by config at 'usernameFromEnv' / 'passwordFromEnv'
	//	- Search 'username' / 'password' fields (Clear Text)
	//	Get username
	if usernameSecret, ok := config.AuthParams[solaceMetaUsername]; ok && usernameSecret != "" {
		u = usernameSecret
	} else if usernameFromEnv, ok := config.TriggerMetadata[solaceMetaUsernameFromEnv]; ok && usernameFromEnv != "" {
		if resolvedUser, ok := config.ResolvedEnv[config.TriggerMetadata[solaceMetaUsernameFromEnv]]; ok && resolvedUser != "" {
			u = resolvedUser
		} else {
			return "", "", fmt.Errorf("username could not be resolved from the environment variable: %s", usernameFromEnv)
		}
	} else if usernameClear, ok := config.TriggerMetadata[solaceMetaUsername]; ok && usernameClear != "" {
		u = usernameClear
	} else {
		return "", "", fmt.Errorf("username is required and not found in K8Secret, environment, or clear text")
	}
	//	Get Password
	if passwordSecret, ok := config.AuthParams[solaceMetaPassword]; ok && passwordSecret != "" {
		p = passwordSecret
	} else if passwordEnv, ok := config.TriggerMetadata[solaceMetaPasswordFromEnv]; ok && passwordEnv != "" {
		if resolvedPassword, ok := config.ResolvedEnv[config.TriggerMetadata[solaceMetaPasswordFromEnv]]; ok && resolvedPassword != "" {
			p = resolvedPassword
		} else {
			return "", "", fmt.Errorf("password could not be resolved from the environment variable: %s", passwordEnv)
		}
	} else if passwordClear, ok := config.TriggerMetadata[solaceMetaPassword]; ok && passwordClear != "" {
		p = passwordClear
	} else {
		return "", "", fmt.Errorf("password is required and not found in K8Secret, environment, or clear text")
	}
	return u, p, nil
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
	if s.metadata.msgCountTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-%s-%s", s.metadata.queueName, solaceTriggermsgcount))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.metadata.msgCountTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceExtMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	// Message Spool Usage Target Spec
	if s.metadata.msgSpoolUsageTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-%s-%s", s.metadata.queueName, solaceTriggermsgspoolusage))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.metadata.msgSpoolUsageTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceExtMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	// Message Receive Rate Target Spec
	if s.metadata.msgRxRateTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-%s-%s", s.metadata.queueName, solaceTriggermsgrxrate))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.metadata.msgRxRateTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceExtMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	return metricSpecList
}

// returns SolaceMetricValues struct populated from broker  SEMP endpoint
func (s *SolaceScaler) getSolaceQueueMetricsFromSEMP(ctx context.Context) (SolaceMetricValues, error) {
	var scaledMetricEndpointURL = s.metadata.endpointURL
	var httpClient = s.httpClient
	var sempResponse solaceSEMPResponse
	var metricValues SolaceMetricValues

	//	RETRIEVE METRICS FROM SOLACE SEMP API
	//	Define HTTP Request
	request, err := http.NewRequestWithContext(ctx, "GET", scaledMetricEndpointURL, nil)
	if err != nil {
		return SolaceMetricValues{}, fmt.Errorf("failed attempting request to solace semp api: %w", err)
	}

	//	Add HTTP Auth and Headers
	request.SetBasicAuth(s.metadata.username, s.metadata.password)
	request.Header.Set("Content-Type", "application/json")

	//	Call Solace SEMP API
	response, err := httpClient.Do(request)
	if err != nil {
		return SolaceMetricValues{}, fmt.Errorf("call to solace semp api failed: %w", err)
	}
	defer response.Body.Close()

	// Check HTTP Status Code
	if response.StatusCode < 200 || response.StatusCode > 299 {
		sempError := fmt.Errorf("semp request http status code: %s - %s", strconv.Itoa(response.StatusCode), response.Status)
		return SolaceMetricValues{}, sempError
	}

	// Decode SEMP Response and Test
	if err := json.NewDecoder(response.Body).Decode(&sempResponse); err != nil {
		return SolaceMetricValues{}, fmt.Errorf("failed to read semp response body: %w", err)
	}
	if sempResponse.Meta.ResponseCode < 200 || sempResponse.Meta.ResponseCode > 299 {
		return SolaceMetricValues{}, fmt.Errorf("solace semp api returned error status: %d", sempResponse.Meta.ResponseCode)
	}

	// Set Return Values
	metricValues.msgCount = sempResponse.Collections.Msgs.Count
	metricValues.msgSpoolUsage = sempResponse.Data.MsgSpoolUsage
	metricValues.msgRcvRate = sempResponse.Data.MsgRcvRate
	return metricValues, nil
}

// INTERFACE METHOD
// Call SEMP API to retrieve metrics
// returns value for named metric
// returns true if queue messageCount > 0 || msgSpoolUsage > 0
func (s *SolaceScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	var metricValues, mv SolaceMetricValues
	var mve error
	if mv, mve = s.getSolaceQueueMetricsFromSEMP(ctx); mve != nil {
		s.logger.Error(mve, "call to semp endpoint failed")
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
		metricValues.msgCount > s.metadata.activationMsgCountTarget ||
			metricValues.msgSpoolUsage > s.metadata.activationMsgSpoolUsageTarget ||
			metricValues.msgRcvRate > s.metadata.activationMsgRxRateTarget,
		nil
}

// Do Nothing - Satisfies Interface
func (s *SolaceScaler) Close(context.Context) error {
	return nil
}
