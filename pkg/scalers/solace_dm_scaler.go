package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	// Metric Type
	solaceDMExternalMetricType = "External"
	// Target Client TxByteRate
	aggregatedClientTxByteRateTargetMetricName = "aggregatedClientTxByteRateTarget"
	// Target Client AverageTxByteRate
	aggregatedClientAverageTxByteRateTargetMetricName = "aggregatedClientAverageTxByteRateTarget"
	// Target Client TxMsgRate
	aggregatedClientTxMsgRateTargetMetricName = "aggregatedClientTxMsgRateTarget"
	// Target Client AverageTxMsgRate
	aggregatedClientAverageTxMsgRateTargetMetricName = "aggregatedClientAverageTxMsgRateTarget"
	// URL validation regex pattern
	urlValidationPattern = "https?://[-_.A-Za-z0-9]{2,255}(:[0-9]{2,6})?"
	// SEMP v1 URL Pattern
	sempURLPattern = "%s/SEMP"
	// D-1 Queue
	d1PriorityQueue = "D-1"
	// Unit of Work Byte Size
	unitOfWorkByteSize = 2048
)

// Package level variables
// Compile the regular expression
var re = regexp.MustCompile(urlValidationPattern)

/*************************************************************************/
/*** Scaler configuration ´Metadata´                                     */
/*************************************************************************/
type SolaceDMScalerConfiguration struct {
	// Scaler index
	triggerIndex int

	// SolaceSEMPBaseURL
	SolaceSEMPBaseURL string `keda:"name=solaceSempBaseURL, order=triggerMetadata"`
	// Basic Auth Username
	Username string `keda:"name=username, order=authParams;resolvedEnv"`
	// Basic Auth Password
	Password string `keda:"name=password, order=authParams;resolvedEnv"`

	// Message VPN
	MessageVpn string `keda:"name=messageVpn, order=triggerMetadata"`
	// Client Name Prefix
	ClientNamePattern string `keda:"name=clientNamePattern, order=triggerMetadata"`

	// UnsafeSSL
	UnsafeSSL bool `keda:"name=unsafeSSL, order=triggerMetadata, default=false"`

	// factor to multiply queued messages length
	// to increase weight on queued messages and scale faster
	QueuedMessagesFactor int64 `keda:"name=queuedMessagesFactor, order=triggerMetadata, default=3"`
	// Target Client TxByteRate
	AggregatedClientTxByteRateTarget int64 `keda:"name=aggregatedClientTxByteRateTarget, order=triggerMetadata, default=0"`
	// Target Client AverageTxByteRate
	AggregatedClientAverageTxByteRateTarget int64 `keda:"name=aggregatedClientAverageTxByteRateTarget, order=triggerMetadata, default=0"`
	// Target Client TxMsgRate
	AggregatedClientTxMsgRateTarget int64 `keda:"name=aggregatedClientTxMsgRateTarget, order=triggerMetadata, default=0"`
	// Target Client AverageTxMsgRate
	AggregatedClientAverageTxMsgRateTarget int64 `keda:"name=aggregatedClientAverageTxMsgRateTarget, order=triggerMetadata, default=0"`

	// Activation Client TxByteRate
	ActivationAggregatedClientTxByteRateTarget int `keda:"name=activationAggregatedClientTxByteRateTarget, order=triggerMetadata, default=0"`
	// Activation Target Average TxByteRate
	ActivationAggregatedClientAverageTxByteRateTarget int `keda:"name=activationAggregatedClientAverageTxByteRateTarget, order=triggerMetadata, default=0"`
	// Activation Client TxMsgRate
	ActivationAggregatedClientTxMsgRateTarget int `keda:"name=activationAggregatedClientTxMsgRateTarget, order=triggerMetadata, default=0"`
	// Activation Target Average TxMsgRate
	ActivationAggregatedClientAverageTxMsgRateTarget int `keda:"name=activationAggregatedClientAverageTxMsgRateTarget, order=triggerMetadata, default=0"`

	// Full SEMP URLs to get stats
	sempURL []string
}

func (s *SolaceDMScalerConfiguration) Validate() error {
	// Check each of the urls for: empty strings, valid url pattern
	urls := strings.Split(s.SolaceSEMPBaseURL, ",")

	for i, v := range urls {
		url := strings.TrimSpace(v)
		if len(url) == 0 {
			return fmt.Errorf("empty host url value found in the scaler configuration. Url[%d]: '%s'", i, url)
		}
		match := re.MatchString(url)
		if !match {
			return fmt.Errorf("invalid host url value found in the scaler configuration. Url[%d]: '%s'", i, url)
		}
	}

	if strings.Contains(s.ClientNamePattern, "*") {
		return fmt.Errorf("client-name-pattern should not contain '*'. ClientNamePattern: '%s'", s.ClientNamePattern)
	}

	if s.QueuedMessagesFactor < 1 || s.QueuedMessagesFactor > 100 {
		return fmt.Errorf("queued messages factor should be >0 and <=100. QueuedMessagesFactor: '%d'", s.QueuedMessagesFactor)
	}

	//	Check that we have at least one positive target value for the scaler
	if s.AggregatedClientTxByteRateTarget < 1 && s.AggregatedClientAverageTxByteRateTarget < 1 && s.AggregatedClientTxMsgRateTarget < 1 && s.AggregatedClientAverageTxMsgRateTarget < 1 {
		return errors.New("no target value found in the scaler configuration")
	}

	return nil
}

/*************************************************************************/
/*** Scaler Metric values                                                */
/*************************************************************************/
// SolaceMetricValues is the struct for Observed Metric Values
type SolaceDMScalerMetricValues struct {
	//	Observed clientTxByteRate
	AggregatedClientTxByteRate int64
	//	Observed clientAverageTxByteRate
	AggregatedClientAverageTxByteRate int64
	//	Observed clientTxMsgRate
	AggregatedClientTxMsgRate int64
	//	Observed clientAverageTxMsgRate
	AggregatedClientAverageTxMsgRate int64
	//	Observed Client queued messages count
	AggregatedClientD1QueueMsgCount int64
	//	Observed Client queue units of work
	AggregatedClientD1QueueUnitsOfWork int64
	//	Observed Client queue units of work ratio
	AggregatedClientD1QueueUnitsOfWorkRatio int64
}

func (c SolaceDMScalerMetricValues) String() string {
	out, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}

	return fmt.Sprint(string(out))
}

/*************************************************************************/
/*** Client Stats - structs                                              */
/*************************************************************************/
type CSRpcReply struct {
	XMLName       xml.Name        `xml:"rpc-reply"`
	SempVersion   string          `xml:"semp-version,attr"`
	RPC           CSRpc           `xml:"rpc"`
	ExecuteResult CSExecuteResult `xml:"execute-result"`
}
type CSRpc struct {
	XMLName xml.Name `xml:"rpc"`
	Show    CSShow   `xml:"show"`
}
type CSExecuteResult struct {
	XMLName xml.Name `xml:"execute-result"`
	Code    string   `xml:"code,attr"`
}
type CSShow struct {
	XMLName xml.Name `xml:"show"`
	Client  CSClient `xml:"client"`
}
type CSClient struct {
	XMLName              xml.Name               `xml:"client"`
	PrimaryVirtualRouter CSPrimaryVirtualRouter `xml:"primary-virtual-router"`
}
type CSPrimaryVirtualRouter struct {
	XMLName xml.Name    `xml:"primary-virtual-router"`
	Clients []CSClientD `xml:"client"`
}
type CSClientD struct {
	XMLName        xml.Name `xml:"client"`
	ClientAddress  string   `xml:"client-address"`
	Name           string   `xml:"name"`
	MessageVpn     string   `xml:"message-vpn"`
	SlowSubscriber bool     `xml:"slow-subscriber"`
	ClientUsername string   `xml:"client-username"`
	Stats          CSStats  `xml:"stats"`
}
type CSStats struct {
	XMLName              xml.Name `xml:"stats"`
	MsgRatePerSecond     int64    `xml:"current-egress-rate-per-second"`
	AvgMsgRatePerMinute  int64    `xml:"average-egress-rate-per-minute"`
	ByteRatePerSecond    int64    `xml:"current-egress-byte-rate-per-second"`
	AvgByteRatePerMinute int64    `xml:"average-egress-byte-rate-per-minute"`
}

func (c CSClientD) String() string {
	out, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}

	return fmt.Sprint(string(out))
}

/*************************************************************************/
/*** Client Stats Queues - structs                                       */
/*************************************************************************/
type CSQRpcReply struct {
	XMLName       xml.Name         `xml:"rpc-reply"`
	SempVersion   string           `xml:"semp-version,attr"`
	RPC           CSQRpc           `xml:"rpc"`
	ExecuteResult CSQExecuteResult `xml:"execute-result"`
}
type CSQRpc struct {
	XMLName xml.Name `xml:"rpc"`
	Show    CSQShow  `xml:"show"`
}
type CSQExecuteResult struct {
	XMLName xml.Name `xml:"execute-result"`
	Code    string   `xml:"code,attr"`
}
type CSQShow struct {
	XMLName xml.Name  `xml:"show"`
	Client  CSQClient `xml:"client"`
}
type CSQClient struct {
	XMLName              xml.Name                `xml:"client"`
	PrimaryVirtualRouter CSQPrimaryVirtualRouter `xml:"primary-virtual-router"`
}
type CSQPrimaryVirtualRouter struct {
	XMLName xml.Name     `xml:"primary-virtual-router"`
	Clients []CSQClientD `xml:"client"`
}
type CSQClientD struct {
	XMLName        xml.Name         `xml:"client"`
	ClientAddress  string           `xml:"client-address"`
	Name           string           `xml:"name"`
	MessageVpn     string           `xml:"message-vpn"`
	SlowSubscriber bool             `xml:"slow-subscriber"`
	ClientUsername string           `xml:"client-username"`
	ClientQueues   []CSQClientQueue `xml:"client-queue"`
}
type CSQClientQueue struct {
	XMLName           xml.Name `xml:"client-queue"`
	QueuePriority     string   `xml:"queue-priority"`
	LengthMsgs        int64    `xml:"length-msgs"`
	LengthWork        int64    `xml:"length-work"`
	HighWatermarkWork int64    `xml:"high-water-mark-work"`
	MaxWork           int64    `xml:"max-work"`
	DiscardsMsgs      int64    `xml:"discards-msgs"`
	DeliveredMsgs     int64    `xml:"delivered-msgs"`
}

func (c CSQClientD) String() string {
	out, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}

	return fmt.Sprint(string(out))
}

/*************************************************************************/
/*** Scaler                                                              */
/*************************************************************************/
type SolaceDMScaler struct {
	metricType    v2.MetricTargetType
	configuration *SolaceDMScalerConfiguration
	httpClient    *http.Client
	logger        logr.Logger
}

/*****/

// Constructor for SolaceDMScaler
func NewSolaceDMScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "solace-dm-scaler")

	// Parse Solace Metadata
	scalerConfig, err := parseSolaceDMConfiguration(config)
	if err != nil {
		logger.Error(err, "Error parsing Solace Trigger Metadata or missing values")
		return nil, err
	}

	// Create HTTP Client
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, scalerConfig.UnsafeSSL)

	scaler := &SolaceDMScaler{
		metricType:    metricType,
		configuration: scalerConfig,
		httpClient:    httpClient,
		logger:        logger,
	}

	return scaler, nil
}

func parseSolaceDMConfiguration(scalerConfig *scalersconfig.ScalerConfig) (*SolaceDMScalerConfiguration, error) {
	meta := &SolaceDMScalerConfiguration{}
	if err := scalerConfig.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing metadata: %w", err)
	}
	meta.triggerIndex = scalerConfig.TriggerIndex

	// initialize urls
	meta.sempURL = []string{}
	urls := strings.Split(meta.SolaceSEMPBaseURL, ",")

	for _, v := range urls {
		url := strings.TrimSpace(v)
		fullURL := fmt.Sprintf(sempURLPattern, url)
		meta.sempURL = append(meta.sempURL, fullURL)
	}

	return meta, nil
}

// GetMetricSpecForScaling returns the metrics based on which this scaler determines that the ScaleTarget scales. This is used to construct the HPA spec that is created for
// this scaled object. The labels used should match the selectors used in GetMetrics
func (s *SolaceDMScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	var metricSpecList []v2.MetricSpec
	var triggerIndex = s.configuration.triggerIndex
	var clientNamePattern = s.configuration.ClientNamePattern

	// Target Client TxByteRate
	if s.configuration.AggregatedClientTxByteRateTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-dm-%s-%s", clientNamePattern, aggregatedClientTxByteRateTargetMetricName))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(triggerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.configuration.AggregatedClientTxByteRateTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceDMExternalMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	//
	// Target Client AverageTxByteRate
	if s.configuration.AggregatedClientAverageTxByteRateTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-dm-%s-%s", clientNamePattern, aggregatedClientAverageTxByteRateTargetMetricName))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(triggerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.configuration.AggregatedClientAverageTxByteRateTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceDMExternalMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	//
	// Target Client TxMsgRate
	if s.configuration.AggregatedClientTxMsgRateTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-dm-%s-%s", clientNamePattern, aggregatedClientTxMsgRateTargetMetricName))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(triggerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.configuration.AggregatedClientTxMsgRateTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceDMExternalMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	//
	// Target Client AverageTxMsgRate
	if s.configuration.AggregatedClientAverageTxMsgRateTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-dm-%s-%s", clientNamePattern, aggregatedClientAverageTxMsgRateTargetMetricName))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(triggerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.configuration.AggregatedClientAverageTxMsgRateTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceDMExternalMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}

	return metricSpecList
}

// GetMetricsAndActivity returns the metric values and activity for a metric Name
func (s *SolaceDMScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	metricValues, _ := &SolaceDMScalerMetricValues{}, error(nil)

	s.logger.V(1).Info(fmt.Sprintf("* MetricName: '%s'", metricName))

	err := s.getClientStats(ctx, metricValues)
	if err != nil {
		s.logger.Error(err, "call to semp endpoint (client stats) failed")
		return []external_metrics.ExternalMetricValue{}, false, err
	}
	err = s.getClientStatQueues(ctx, metricValues)

	if err != nil {
		s.logger.Error(err, "call to semp endpoint (client queues stats) failed")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	// Use the queued messages in D-1 queue to add them to the other metrics!
	metricValues.AggregatedClientTxMsgRate += (metricValues.AggregatedClientD1QueueMsgCount * s.configuration.QueuedMessagesFactor)
	metricValues.AggregatedClientAverageTxMsgRate += (metricValues.AggregatedClientD1QueueMsgCount * s.configuration.QueuedMessagesFactor)
	metricValues.AggregatedClientTxByteRate += (metricValues.AggregatedClientD1QueueUnitsOfWork * s.configuration.QueuedMessagesFactor * unitOfWorkByteSize)
	metricValues.AggregatedClientAverageTxByteRate += (metricValues.AggregatedClientD1QueueUnitsOfWork * s.configuration.QueuedMessagesFactor * unitOfWorkByteSize)

	var metric external_metrics.ExternalMetricValue
	switch {
	//
	case strings.HasSuffix(metricName, aggregatedClientTxByteRateTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.AggregatedClientTxByteRate))
	case strings.HasSuffix(metricName, aggregatedClientAverageTxByteRateTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.AggregatedClientAverageTxByteRate))
	//
	case strings.HasSuffix(metricName, aggregatedClientTxMsgRateTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.AggregatedClientTxMsgRate))
	case strings.HasSuffix(metricName, aggregatedClientAverageTxMsgRateTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.AggregatedClientAverageTxMsgRate))

	default:
		// Should never end up here
		err := fmt.Errorf("unidentified metric: %s", metricName)
		s.logger.Error(err, "returning error to calling app")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	s.logger.V(1).Info(fmt.Sprintf("Metrics: '%s'", metricValues))
	// always return true for activation unless its not needed this needs at least one instace
	return []external_metrics.ExternalMetricValue{metric}, true, nil
}

// Close any resources that need disposing when scaler is no longer used or destroyed
func (s *SolaceDMScaler) Close(_ context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

/************************************************************************************************************/
/************ Additional Internal methods                                       *****************************/
/************************************************************************************************************/
func (s *SolaceDMScaler) getClientStats(ctx context.Context, metricValues *SolaceDMScalerMetricValues) error {
	// get client stats for the clients that have the prefix
	clientStatsReqBody := "<rpc><show><client><name>*" + s.configuration.ClientNamePattern + "*</name><stats/></client></show></rpc>"

	bodyBytes, err := s.GetSolaceDMSempMetrics(ctx, s.httpClient, s.configuration.sempURL, s.configuration.Username, s.configuration.Password, clientStatsReqBody)
	if err != nil {
		return fmt.Errorf("reading client stats failed: %w", err)
	}

	var clientStatsReply CSRpcReply

	err = xml.Unmarshal(bodyBytes, &clientStatsReply)
	if err != nil {
		return fmt.Errorf("unmarshalling the body failed: %w", err)
	}

	clients := clientStatsReply.RPC.Show.Client.PrimaryVirtualRouter.Clients

	// make sure they are clean before the agg
	metricValues.AggregatedClientTxByteRate = 0
	metricValues.AggregatedClientAverageTxByteRate = 0
	metricValues.AggregatedClientTxMsgRate = 0
	metricValues.AggregatedClientAverageTxMsgRate = 0
	var numClients int64

	for i := 0; i < len(clients); i++ {
		client := clients[i]

		// only consider the configured vpn
		if client.MessageVpn == s.configuration.MessageVpn {
			numClients++
			s.logger.V(1).Info(fmt.Sprintf("    Client[%d] - ByteRatePerSecond: '%d', AvgByteRatePerMinute: '%d', MsgRatePerSecond: '%d', AvgMsgRatePerMinute: '%d'", i, client.Stats.ByteRatePerSecond, client.Stats.AvgByteRatePerMinute, client.Stats.MsgRatePerSecond, client.Stats.AvgMsgRatePerMinute))
			metricValues.AggregatedClientTxByteRate += client.Stats.ByteRatePerSecond
			metricValues.AggregatedClientAverageTxByteRate += client.Stats.AvgByteRatePerMinute
			metricValues.AggregatedClientTxMsgRate += client.Stats.MsgRatePerSecond
			metricValues.AggregatedClientAverageTxMsgRate += client.Stats.AvgMsgRatePerMinute
		}
	}

	s.logger.V(1).Info(fmt.Sprintf("   MetricValues - ByteRatePerSecond: '%d', AvgByteRatePerMinute: '%d', MsgRatePerSecond: '%d', AvgMsgRatePerMinute: '%d'", metricValues.AggregatedClientTxByteRate, metricValues.AggregatedClientAverageTxByteRate, metricValues.AggregatedClientTxMsgRate, metricValues.AggregatedClientAverageTxMsgRate))
	// no error
	return nil
}

func (s *SolaceDMScaler) getClientStatQueues(ctx context.Context, metricValues *SolaceDMScalerMetricValues) error {
	// get client stats for the clients that have the prefix
	clientStatQueuesReqBody := fmt.Sprintf("<rpc><show><client><name>*%s*</name><stats></stats><queues></queues></client></show></rpc>", s.configuration.ClientNamePattern)

	bodyBytes, err := s.GetSolaceDMSempMetrics(ctx, s.httpClient, s.configuration.sempURL, s.configuration.Username, s.configuration.Password, clientStatQueuesReqBody)
	if err != nil {
		return fmt.Errorf("reading client stats queue failed: %w", err)
	}

	var clientStatsReply CSQRpcReply

	err = xml.Unmarshal(bodyBytes, &clientStatsReply)
	if err != nil {
		return fmt.Errorf("unmarshalling the body failed: %w", err)
	}

	clients := clientStatsReply.RPC.Show.Client.PrimaryVirtualRouter.Clients

	// make sure they are clean before the agg
	metricValues.AggregatedClientD1QueueMsgCount = 0
	metricValues.AggregatedClientD1QueueUnitsOfWork = 0
	metricValues.AggregatedClientD1QueueUnitsOfWorkRatio = 0

	var aggregatedMaxUnitOfWork int64
	var aggregatedUsedUnitsOfWork int64
	var numClients int32

	for i := 0; i < len(clients); i++ {
		client := clients[i]

		// only consider the configured vpn
		if client.MessageVpn == s.configuration.MessageVpn {
			clientQueues := client.ClientQueues

			for j := 0; j < len(clientQueues); j++ {
				clientQueue := clientQueues[j]

				if clientQueue.QueuePriority == d1PriorityQueue {
					s.logger.V(1).Info(fmt.Sprintf("    Client[%d]Q[%s] - LengthMsgs: '%d', MaxWork: '%d', LengthWork: '%d'", i, clientQueue.QueuePriority, clientQueue.LengthMsgs, clientQueue.MaxWork, clientQueue.LengthWork))

					numClients++
					metricValues.AggregatedClientD1QueueMsgCount += clientQueue.LengthMsgs
					metricValues.AggregatedClientD1QueueUnitsOfWork += clientQueue.LengthWork
					aggregatedMaxUnitOfWork += clientQueue.MaxWork
					aggregatedUsedUnitsOfWork += clientQueue.LengthWork
				}
			}
		}
	}

	s.logger.V(1).Info(fmt.Sprintf("   MetricValues - AggregatedClientD1QueueMsgCount: '%d'", metricValues.AggregatedClientD1QueueMsgCount))
	s.logger.V(1).Info(fmt.Sprintf("   MetricValues - AggregatedClientD1QueueUnitsOfWork: '%d'", metricValues.AggregatedClientD1QueueUnitsOfWork))
	// no error
	return nil
}

func (s *SolaceDMScaler) GetSolaceDMSempMetrics(ctx context.Context, httpClient *http.Client, sempUrls []string, username string, password string, requestBody string) ([]byte, error) {
	for _, sempURL := range sempUrls {
		bytes, err := s.GetSolaceDMSempMetricsFromHost(ctx, httpClient, sempURL, username, password, requestBody)

		if err == nil {
			return bytes, nil
		}
		s.logger.Info(fmt.Sprintf("Warning: getting metrics from url: '%s' failed, using next url in list. err: '%s'", sempURL, err))
	}
	// should not reach this code
	return []byte{}, errors.New("no URL returned any data")
}

func (s *SolaceDMScaler) GetSolaceDMSempMetricsFromHost(ctx context.Context, httpClient *http.Client, sempURL string, username string, password string, requestBody string) ([]byte, error) {
	//	Retrieve metrics from Solace SEMP v1
	//	Define HTTP Request
	request, err := http.NewRequestWithContext(ctx, "POST", sempURL, bytes.NewBuffer([]byte(requestBody)))
	if err != nil {
		return []byte{}, fmt.Errorf("failed attempting request to solace semp api: %w", err)
	}

	//	Add HTTP Auth and Headers
	request.SetBasicAuth(username, password)
	request.Header.Set("Content-Type", "application/xml")

	//	Call Solace SEMP API
	response, err := httpClient.Do(request)
	if err != nil {
		return []byte{}, fmt.Errorf("call to solace semp api failed: %w", err)
	}
	// close the response body  reader when the func returns
	defer response.Body.Close()

	// Check HTTP Status Code
	if response.StatusCode < 200 || response.StatusCode > 299 {
		sempError := fmt.Errorf("semp request http status code: %s - %s", strconv.Itoa(response.StatusCode), response.Status)
		return []byte{}, sempError
	}

	responseBodyValue, err := io.ReadAll(response.Body)
	// check errors
	if err != nil {
		return []byte{}, fmt.Errorf("reading response body failed: %w", err)
	}

	return responseBodyValue, nil
}
