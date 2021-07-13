package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	solaceExtMetricType = "External"
	solaceScalerID      = "solace"
	// REST ENDPOINT String Patterns
	solaceSempEndpointURLTemplate = "%s/%s/%s/monitor/msgVpns/%s/%ss/%s"
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
	solaceMetaUsername    = "username"
	solaceMetaPassword    = "password"
	solaceMetaUsernameEnv = "usernameEnv"
	solaceMetaPasswordEnv = "passwordEnv"
	// Target Object Identifiers
	solaceMetaMsgVpn    = "messageVpn"
	solaceMetaQueueName = "queueName"
	// Metric Targets
	solaceMetaMsgCountTarget      = "messageCountTarget"
	solaceMetaMsgSpoolUsageTarget = "messageSpoolUsageTarget"
	// Trigger type identifiers
	solaceTriggermsgcount      = "msgcount"
	solaceTriggermsgspoolusage = "msgspoolusage"
)

// Struct for Observed Metric Values
type SolaceMetricValues struct {
	//	Observed Message Count
	msgCount int
	//	Observed Message Spool Usage
	msgSpoolUsage int
}

type SolaceScaler struct {
	metadata   *SolaceMetadata
	httpClient *http.Client
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
	msgCountTarget      int
	msgSpoolUsageTarget int // Spool Use Target in Megabytes
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
}

// SEMP API Messages Struct
type solaceSEMPMessages struct {
	Count int `json:"count"`
}

// SEMP API Metadata Struct
type solaceSEMPMetadata struct {
	ResponseCode int `json:"responseCode"`
}

//	Solace Logger
var solaceLog = logf.Log.WithName(solaceScalerID + "_scaler")

//	Constructor for SolaceScaler
func NewSolaceScaler(config *ScalerConfig) (Scaler, error) {
	// Create HTTP Client
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout)

	// Parse Solace Metadata
	solaceMetadata, err := parseSolaceMetadata(config)
	if err != nil {
		solaceLog.Error(err, "Error parsing Solace Trigger Metadata or missing values")
		return nil, err
	}

	return &SolaceScaler{
		metadata:   solaceMetadata,
		httpClient: httpClient,
	}, nil
}

//	Called by constructor
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
	if val, ok := config.TriggerMetadata[solaceMetaMsgCountTarget]; ok && val != "" {
		if msgCount, err := strconv.Atoi(val); err == nil {
			meta.msgCountTarget = msgCount
		} else {
			return nil, fmt.Errorf("can't parse [%s], not a valid integer: %s", solaceMetaMsgCountTarget, err)
		}
	}
	//	GET msgSpoolUsageTarget
	if val, ok := config.TriggerMetadata[solaceMetaMsgSpoolUsageTarget]; ok && val != "" {
		if msgSpoolUsage, err := strconv.Atoi(val); err == nil {
			meta.msgSpoolUsageTarget = msgSpoolUsage * 1024 * 1024
		} else {
			return nil, fmt.Errorf("can't parse [%s], not a valid integer: %s", solaceMetaMsgSpoolUsageTarget, err)
		}
	}

	//	Check that we have at least one positive target value for the scaler
	if meta.msgCountTarget < 1 && meta.msgSpoolUsageTarget < 1 {
		return nil, fmt.Errorf("no target value found in the scaler configuration")
	}

	// Format Solace SEMP Queue Endpoint (REST URL)
	meta.endpointURL = fmt.Sprintf(
		solaceSempEndpointURLTemplate,
		meta.solaceSempURL,
		solaceAPIName,
		solaceAPIVersion,
		meta.messageVpn,
		solaceAPIObjectTypeQueue,
		meta.queueName)

	// Get Credentials
	var e error
	if meta.username, meta.password, e = getSolaceSempCredentials(config); e != nil {
		return nil, e
	}
	return &meta, nil
}

func getSolaceSempCredentials(config *ScalerConfig) (u string, p string, err error) {
	//	GET CREDENTIALS
	//	The username must be a valid broker ADMIN user identifier with read access to SEMP for the broker, VPN, and relevant objects
	//	The scaler will attempt to acquire username and then password independently. For each:
	//	- Search K8S Secret (Encoded)
	//	- Search environment variable specified by config at 'usernameEnv' / 'passwordEnv'
	//	- Search 'username' / 'password' fields (Clear Text)
	//	Get username
	if usernameSecret, ok := config.AuthParams[solaceMetaUsername]; ok && usernameSecret != "" {
		u = usernameSecret
	} else if usernameEnv, ok := config.TriggerMetadata[solaceMetaUsernameEnv]; ok && usernameEnv != "" {
		if resolvedUser, ok := config.ResolvedEnv[config.TriggerMetadata[solaceMetaUsernameEnv]]; ok && resolvedUser != "" {
			u = resolvedUser
		} else {
			return "", "", fmt.Errorf("username could not be resolved from the environment variable: %s", usernameEnv)
		}
	} else if usernameClear, ok := config.TriggerMetadata[solaceMetaUsername]; ok && usernameClear != "" {
		u = usernameClear
	} else {
		return "", "", fmt.Errorf("username is required and not found in K8Secret, environment, or clear text")
	}
	//	Get Password
	if passwordSecret, ok := config.AuthParams[solaceMetaPassword]; ok && passwordSecret != "" {
		p = passwordSecret
	} else if passwordEnv, ok := config.TriggerMetadata[solaceMetaPasswordEnv]; ok && passwordEnv != "" {
		if resolvedPassword, ok := config.ResolvedEnv[config.TriggerMetadata[solaceMetaPasswordEnv]]; ok && resolvedPassword != "" {
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

//	INTERFACE METHOD
//	DEFINE METRIC FOR SCALING
//	CURRENT SUPPORTED METRICS ARE:
//	- QUEUE MESSAGE COUNT (msgCount)
//	- QUEUE SPOOL USAGE   (msgSpoolUsage in MBytes)
//	METRIC IDENTIFIER HAS THE SIGNATURE:
//	- solace-[VPN_Name]-[Queue_Name]-[metric_type]
//	e.g. solace-myvpn-QUEUE1-msgCount
func (s *SolaceScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	var metricSpecList []v2beta2.MetricSpec
	// Message Count Target Spec
	if s.metadata.msgCountTarget > 0 {
		targetMetricValue := resource.NewQuantity(int64(s.metadata.msgCountTarget), resource.DecimalSI)
		externalMetric := &v2beta2.ExternalMetricSource{
			Metric: v2beta2.MetricIdentifier{
				Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s-%s", solaceScalerID, s.metadata.messageVpn, s.metadata.queueName, solaceTriggermsgcount)),
			},
			Target: v2beta2.MetricTarget{
				Type:         v2beta2.AverageValueMetricType,
				AverageValue: targetMetricValue,
			},
		}
		metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: solaceExtMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	// Message Spool Usage Target Spec
	if s.metadata.msgSpoolUsageTarget > 0 {
		targetMetricValue := resource.NewQuantity(int64(s.metadata.msgSpoolUsageTarget), resource.DecimalSI)
		externalMetric := &v2beta2.ExternalMetricSource{
			Metric: v2beta2.MetricIdentifier{
				Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s-%s", solaceScalerID, s.metadata.messageVpn, s.metadata.queueName, solaceTriggermsgspoolusage)),
			},
			Target: v2beta2.MetricTarget{
				Type:         v2beta2.AverageValueMetricType,
				AverageValue: targetMetricValue,
			},
		}
		metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: solaceExtMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	return metricSpecList
}

//	returns SolaceMetricValues struct populated from broker  SEMP endpoint
func (s *SolaceScaler) getSolaceQueueMetricsFromSEMP() (SolaceMetricValues, error) {
	var scaledMetricEndpointURL = s.metadata.endpointURL
	var httpClient = s.httpClient
	var sempResponse solaceSEMPResponse
	var metricValues SolaceMetricValues

	//	RETRIEVE METRICS FROM SOLACE SEMP API
	//	Define HTTP Request
	request, err := http.NewRequest("GET", scaledMetricEndpointURL, nil)
	if err != nil {
		return SolaceMetricValues{}, fmt.Errorf("failed attempting request to solace semp api: %s", err)
	}

	//	Add HTTP Auth and Headers
	request.SetBasicAuth(s.metadata.username, s.metadata.password)
	request.Header.Set("Content-Type", "application/json")

	//	Call Solace SEMP API
	response, err := httpClient.Do(request)
	if err != nil {
		return SolaceMetricValues{}, fmt.Errorf("call to solace semp api failed: %s", err)
	}
	defer response.Body.Close()

	// Check HTTP Status Code
	if response.StatusCode < 200 || response.StatusCode > 299 {
		sempError := fmt.Errorf("semp request http status code: %s - %s", strconv.Itoa(response.StatusCode), response.Status)
		return SolaceMetricValues{}, sempError
	}

	// Decode SEMP Response and Test
	if err := json.NewDecoder(response.Body).Decode(&sempResponse); err != nil {
		return SolaceMetricValues{}, fmt.Errorf("failed to read semp response body: %s", err)
	}
	if sempResponse.Meta.ResponseCode < 200 || sempResponse.Meta.ResponseCode > 299 {
		return SolaceMetricValues{}, fmt.Errorf("solace semp api returned error status: %d", sempResponse.Meta.ResponseCode)
	}

	// Set Return Values
	metricValues.msgCount = sempResponse.Collections.Msgs.Count
	metricValues.msgSpoolUsage = sempResponse.Data.MsgSpoolUsage
	return metricValues, nil
}

//	INTERFACE METHOD
//	Call SEMP API to retrieve metrics
//	returns value for named metric
func (s *SolaceScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	var metricValues, mv SolaceMetricValues
	var mve error
	if mv, mve = s.getSolaceQueueMetricsFromSEMP(); mve != nil {
		solaceLog.Error(mve, "call to semp endpoint failed")
		return []external_metrics.ExternalMetricValue{}, mve
	}
	metricValues = mv

	var metric external_metrics.ExternalMetricValue
	switch {
	case strings.HasSuffix(metricName, solaceTriggermsgcount):
		metric = external_metrics.ExternalMetricValue{
			MetricName: metricName,
			Value:      *resource.NewQuantity(int64(metricValues.msgCount), resource.DecimalSI),
			Timestamp:  metav1.Now(),
		}
	case strings.HasSuffix(metricName, solaceTriggermsgspoolusage):
		metric = external_metrics.ExternalMetricValue{
			MetricName: metricName,
			Value:      *resource.NewQuantity(int64(metricValues.msgSpoolUsage), resource.DecimalSI),
			Timestamp:  metav1.Now(),
		}
	default:
		// Should never end up here
		err := fmt.Errorf("unidentified metric: %s", metricName)
		solaceLog.Error(err, "returning error to calling app")
		return []external_metrics.ExternalMetricValue{}, err
	}
	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

//	INTERFACE METHOD
//	Call SEMP API to retrieve metrics
//	IsActive returns true if queue messageCount > 0 || msgSpoolUsage > 0
func (s *SolaceScaler) IsActive(ctx context.Context) (bool, error) {
	metricValues, err := s.getSolaceQueueMetricsFromSEMP()
	if err != nil {
		solaceLog.Error(err, "call to semp endpoint failed")
		return false, err
	}
	return (metricValues.msgCount > 0 || metricValues.msgSpoolUsage > 0), nil
}

// Do Nothing - Satisfies Interface
func (s *SolaceScaler) Close() error {
	return nil
}
