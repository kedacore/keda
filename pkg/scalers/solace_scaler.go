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
	solace_EXT_METRIC_TYPE = "External"
	solace_SCALER_ID       = "solace"

	// REST ENDPOINT String Patterns
	solace_BROKER_BASE_URL_TEMPLATE   = "%s://%s:%s"
	solace_SEMP_ENDPOINT_URL_TEMPLATE = "%s/%s/%s/monitor/msgVpns/%s/%ss/%s"
	// SEMP REST API Context
	solace_API_NAME          = "SEMP"
	solace_API_VERSION       = "v2"
	solace_API_DFLT_OBJ_TYPE = "queue"

	// Log Message Templates
	solace_FOUND_META_TRUE  = "Field %s Found in Solace Metadata; Value=%v"
	solace_FOUND_META_FALSE = "Required Field %s NOT FOUND in Solace Metadata"

	// YAML Configuration Metadata Field Names
	// Broker Identifiers
	solace_META_brokerBaseUrl  = "brokerBaseUrl"
	solace_META_brokerProtocol = "brokerProtocol"
	solace_META_brokerHostname = "brokerHostname"
	solace_META_brokerPort     = "brokerPort"
	// Credential Identifiers
	solace_META_username    = "username"
	solace_META_password    = "password"
	solace_META_usernameEnv = "usernameEnv"
	solace_META_passwordEnv = "passwordEnv"
	// Target Object Identifiers
	solace_META_msgVpn    = "msgVpn"
	solace_META_queueName = "queueName"
	// Metric Targets
	solace_META_msgCountTarget      = "msgCountTarget"
	solace_META_msgSpoolUsageTarget = "msgSpoolUsageTarget"

	// Trigger type identifiers
	solace_TRIG_msgcount      = "msgcount"
	solace_TRIG_msgspoolusage = "msgspoolusage"
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
	endpointUrl string
	// Protocol-Host-Port: http://host-name:12345
	brokerBaseUrl  string
	brokerProtocol string // Used if brokerBaseUrl not present
	brokerHostname string // Used if brokerBaseUrl not present
	brokerPort     string // Used if brokerBaseUrl not present
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
var solaceLog = logf.Log.WithName(solace_SCALER_ID + "_scaler")

//	Constructor for SolaceScaler
func NewSolaceScaler(config *ScalerConfig) (Scaler, error) {

	// Create HTTP Client
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout)

	// Parse Solace Metadata
	solaceMetadata, err := parseSolaceMetadata(config)
	if err != nil {
		solaceLog.Error(err, "Error parsing Solace Trigger Metadata or missing values")
		return nil, err //fmt.Errorf("Error parsing Solace Trigger metadata: %s", err)
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
	//	First look for brokerBaseUrl in config; Use components if not found
	if val, ok := config.TriggerMetadata[solace_META_brokerBaseUrl]; ok && val != "" {
		meta.brokerBaseUrl = val
	} else {
		//	IF brokerBaseUrl is not present, then get components
		//	GET Protocol
		if val, ok := config.TriggerMetadata[solace_META_brokerProtocol]; ok && (val == "https" || val == "http") {
			meta.brokerProtocol = val
		} else {
			return nil, fmt.Errorf(solace_FOUND_META_FALSE, solace_META_brokerProtocol)
		}
		//	GET Hostname
		if val, ok := config.TriggerMetadata[solace_META_brokerHostname]; ok && val != "" {
			meta.brokerHostname = val
		} else {
			return nil, fmt.Errorf(solace_FOUND_META_FALSE, solace_META_brokerHostname)
		}
		//	GET Port
		if val, ok := config.TriggerMetadata[solace_META_brokerPort]; ok && val != "" {
			if _, err := strconv.Atoi(val); err == nil {
				meta.brokerPort = val
			} else {
				return nil, fmt.Errorf("Can't parse brokerPort, not a valid integer: %s", err)
			}
		} else {
			return nil, fmt.Errorf(solace_FOUND_META_FALSE, solace_META_brokerPort)
		}
		// Format Solace Broker Base URL from components
		meta.brokerBaseUrl = fmt.Sprintf(solace_BROKER_BASE_URL_TEMPLATE, meta.brokerProtocol, meta.brokerHostname, meta.brokerPort)
	}

	//	GET Message VPN
	if val, ok := config.TriggerMetadata[solace_META_msgVpn]; ok && val != "" {
		meta.messageVpn = val
	} else {
		return nil, fmt.Errorf(solace_FOUND_META_FALSE, solace_META_msgVpn)
	}

	//	GET Queue Name
	if val, ok := config.TriggerMetadata[solace_META_queueName]; ok && val != "" {
		meta.queueName = val
	} else {
		return nil, fmt.Errorf(solace_FOUND_META_FALSE, solace_META_queueName)
	}

	//	GET METRIC TARGET VALUES
	//	GET msgCountTarget
	if val, ok := config.TriggerMetadata[solace_META_msgCountTarget]; ok && val != "" {
		if msgCount, err := strconv.Atoi(val); err == nil {
			meta.msgCountTarget = msgCount
		} else {
			return nil, fmt.Errorf("Can't parse [%s], not a valid integer: %s", solace_META_msgCountTarget, err)
		}
	}
	//	GET msgSpoolUsageTarget
	if val, ok := config.TriggerMetadata[solace_META_msgSpoolUsageTarget]; ok && val != "" {
		if msgSpoolUsage, err := strconv.Atoi(val); err == nil {
			meta.msgSpoolUsageTarget = msgSpoolUsage
		} else {
			return nil, fmt.Errorf("Can't parse [%s], not a valid integer: %s", solace_META_msgSpoolUsageTarget, err)
		}
	}

	//	Check that we have at least one positive target value for the scaler
	if meta.msgCountTarget < 1 && meta.msgSpoolUsageTarget < 1 {
		return nil, fmt.Errorf("No Target Value found in the Scaler Configuration")
	}

	// Format Solace SEMP Queue Endpoint (REST URL)
	meta.endpointUrl = fmt.Sprintf(
		solace_SEMP_ENDPOINT_URL_TEMPLATE,
		meta.brokerBaseUrl,
		solace_API_NAME,
		solace_API_VERSION,
		meta.messageVpn,
		solace_API_DFLT_OBJ_TYPE,
		meta.queueName)

	/*	GET CREDENTIALS
		The username must be a valid broker ADMIN user identifier with read access to SEMP for the broker, VPN, and relevant objects
		The scaler will attempt to acquire username and then password independently. For each:
		- Search K8S Secret (Encoded)
		- Search environment variable specified by config at 'usernameEnv' / 'passwordEnv'
		- Search 'username' / 'password' fields (Clear Text)
	*/
	//	Get username
	if usernameSecret, ok := config.AuthParams[solace_META_username]; ok && usernameSecret != "" {
		meta.username = usernameSecret
	} else if usernameEnv, ok := config.TriggerMetadata[solace_META_usernameEnv]; ok && usernameEnv != "" {
		if resolvedUser, ok := config.ResolvedEnv[config.TriggerMetadata[solace_META_usernameEnv]]; ok && resolvedUser != "" {
			meta.username = resolvedUser
		} else {
			return nil, fmt.Errorf("username could not be resolved from the environment variable: %s", usernameEnv)
		}
	} else if usernameClear, ok := config.TriggerMetadata[solace_META_username]; ok && usernameClear != "" {
		meta.username = usernameClear
	} else {
		return nil, fmt.Errorf("username is required and not found in K8Secret, environment, or clear text")
	}
	//	Get Password
	if passwordSecret, ok := config.AuthParams[solace_META_password]; ok && passwordSecret != "" {
		meta.password = passwordSecret
	} else if passwordEnv, ok := config.TriggerMetadata[solace_META_passwordEnv]; ok && passwordEnv != "" {
		if resolvedPassword, ok := config.ResolvedEnv[config.TriggerMetadata[solace_META_passwordEnv]]; ok && resolvedPassword != "" {
			meta.password = resolvedPassword
		} else {
			return nil, fmt.Errorf("password could not be resolved from the environment variable: %s", passwordEnv)
		}
	} else if passwordClear, ok := config.TriggerMetadata[solace_META_password]; ok && passwordClear != "" {
		meta.password = passwordClear
	} else {
		return nil, fmt.Errorf("password is required and not found in K8Secret, environment, or clear text")
	}

	return &meta, nil
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
				Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s-%s", solace_SCALER_ID, s.metadata.messageVpn, s.metadata.queueName, solace_TRIG_msgcount)),
			},
			Target: v2beta2.MetricTarget{
				Type:         v2beta2.AverageValueMetricType,
				AverageValue: targetMetricValue,
			},
		}
		metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: solace_EXT_METRIC_TYPE}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	// Message Spool Usage Target Spec
	if s.metadata.msgSpoolUsageTarget > 0 {
		targetMetricValue := resource.NewQuantity(int64(s.metadata.msgSpoolUsageTarget), resource.DecimalSI)
		externalMetric := &v2beta2.ExternalMetricSource{
			Metric: v2beta2.MetricIdentifier{
				Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s-%s", solace_SCALER_ID, s.metadata.messageVpn, s.metadata.queueName, solace_TRIG_msgspoolusage)),
			},
			Target: v2beta2.MetricTarget{
				Type:         v2beta2.AverageValueMetricType,
				AverageValue: targetMetricValue,
			},
		}
		metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: solace_EXT_METRIC_TYPE}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	return metricSpecList
}

//	returns SolaceMetricValues struct populated from broker  SEMP endpoint
func (s *SolaceScaler) getSolaceQueueMetricsFromSEMP() (SolaceMetricValues, error) {

	var scaledMetricEndpointUrl string = s.metadata.endpointUrl
	var httpClient *http.Client = s.httpClient
	var sempResponse solaceSEMPResponse
	var metricValues SolaceMetricValues

	//	RETRIEVE METRICS FROM SOLACE SEMP API
	//	Define HTTP Request
	request, err := http.NewRequest("GET", scaledMetricEndpointUrl, nil)
	if err != nil {
		return SolaceMetricValues{}, fmt.Errorf("Failed attempting request to Solace SEMP API: %s", err)
	}
	//	Add HTTP Auth and Headers
	request.SetBasicAuth(s.metadata.username, s.metadata.password)
	request.Header.Set("Content-Type", "application/json")
	//	Call Solace SEMP API
	response, err := httpClient.Do(request)
	if err != nil {
		return SolaceMetricValues{}, fmt.Errorf("Call to Solace SEMP API failed: %s", err)
	}
	defer response.Body.Close()

	// Check HTTP Status Code
	if response.StatusCode < 200 || response.StatusCode > 299 {
		sempError := fmt.Errorf("SEMP Request HTTP Status Code: %s - %s", strconv.Itoa(response.StatusCode), response.Status)
		return SolaceMetricValues{}, sempError
	}

	// Decode SEMP Response and Test
	if err := json.NewDecoder(response.Body).Decode(&sempResponse); err != nil {
		return SolaceMetricValues{}, fmt.Errorf("Failed to read SEMP response body: %s", err)
	}
	if sempResponse.Meta.ResponseCode < 200 && sempResponse.Meta.ResponseCode > 299 {
		return SolaceMetricValues{}, fmt.Errorf("Solace SEMP API returned error status: %d", sempResponse.Meta.ResponseCode)
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

	var metricValues SolaceMetricValues
	if val, err := s.getSolaceQueueMetricsFromSEMP(); err != nil {
		solaceLog.Error(err, fmt.Sprintf("Call to SEMP Endpoint Failed"))
		return []external_metrics.ExternalMetricValue{}, err
	} else {
		metricValues = val
	}

	var metric external_metrics.ExternalMetricValue
	switch {
	case strings.HasSuffix(metricName, solace_TRIG_msgcount):
		metric = external_metrics.ExternalMetricValue{
			MetricName: metricName,
			Value:      *resource.NewQuantity(int64(metricValues.msgCount), resource.DecimalSI),
			Timestamp:  metav1.Now(),
		}
	case strings.HasSuffix(metricName, solace_TRIG_msgspoolusage):
		metric = external_metrics.ExternalMetricValue{
			MetricName: metricName,
			Value:      *resource.NewQuantity(int64(metricValues.msgSpoolUsage), resource.DecimalSI),
			Timestamp:  metav1.Now(),
		}
	default:
		// Should never end up here
		err := fmt.Errorf("Unidentified Metric: %s", metricName)
		solaceLog.Error(err, "Returning Error to calling app")
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
		solaceLog.Error(err, "Call to SEMP Endpoint Failed")
		return false, err
	}
	return (metricValues.msgCount > 0 || metricValues.msgSpoolUsage > 0), nil
}

// Do Nothing - Satisfies Interface
func (s *SolaceScaler) Close() error {
	return nil
}
