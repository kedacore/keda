package scalers

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	//Metric Type
	solaceDMExternalMetricType = "External"
	//Scaler ID
	solaceDMScalerID = "solaceDirectMessaging"
	// Target Client TxByteRate
	aggregateClientTxByteRateTargetMetricName = "aggregateClientTxByteRateTarget"
	// Target Client AverageTxByteRate
	aggregateClientAverageTxByteRateTargetMetricName = "aggregateClientAverageTxByteRateTarget"
	// Target Client TxMsgRate
	aggregateClientTxMsgRateTargetMetricName = "aggregateClientTxMsgRateTarget"
	// Target Client AverageTxMsgRate
	aggregateClientAverageTxMsgRateTargetMetricName = "aggregateClientAverageTxMsgRateTarget"
	// Target D1 Queue backlog
	aggregateClientD1QueueBacklogTargetMetricName = "aggregateClientD1QueueBacklogTarget"
	//
	d1QueueBacklogMetricBaseValue int64 = 100
	//SEMP v1 URL Pattern
	sempUrl = "http://%s/SEMP"
	//D-1 Queue
	d1PriorityQueue = "D-1"
)

type SolaceDMScalerConfiguration struct {
	// Scaler index
	triggerIndex int

	//Host
	host string `keda:"name=host,  order=triggerMetadata"`
	// Basic Auth Username
	username string `keda:"name=username, order=authParams;triggerMetadata;resolvedEnv"`
	// Basic Auth Password
	password string `keda:"name=password, order=authParams;triggerMetadata;resolvedEnv"`

	// Message VPN
	messageVpn string `keda:"name=messageVpn,   order=triggerMetadata"`
	// Client Name Prefix
	clientNamePrefix string `keda:"name=clientNamePrefix,   order=triggerMetadata"`

	// Target Client TxByteRate
	aggregateClientTxByteRateTarget int64 `keda:"name=aggregateClientTxByteRateTarget,       order=triggerMetadata, optional"`
	// Target Client AverageTxByteRate
	aggregateClientAverageTxByteRateTarget int64 `keda:"name=aggregateClientAverageTxByteRateTarget,       order=triggerMetadata, optional"`
	// Target Client TxMsgRate
	aggregateClientTxMsgRateTarget int64 `keda:"name=aggregateClientTxMsgRateTarget,       order=triggerMetadata, optional"`
	// Target Client AverageTxMsgRate
	aggregateClientAverageTxMsgRateTarget int64 `keda:"name=aggregateClientAverageTxMsgRateTarget,       order=triggerMetadata, optional"`

	// Activation Client TxByteRate
	activationAggregateClientTxByteRateTarget int `keda:"name=activationAggregateClientTxByteRateTarget,       order=triggerMetadata, default=0"`
	// Activation Target Average TxByteRate
	activationAggregateClientAverageTxByteRateTarget int `keda:"name=activationAggregateClientAverageTxByteRateTarget,       order=triggerMetadata, default=0"`
	// Activation Client TxMsgRate
	activationAggregateClientTxMsgRateTarget int `keda:"name=activationAggregateClientTxMsgRateTarget,       order=triggerMetadata, default=0"`
	// Activation Target Average TxMsgRate
	activationAggregateClientAverageTxMsgRateTarget int `keda:"name=activationAggregateClientAverageTxMsgRateTarget,       order=triggerMetadata, default=0"`

	// Full SEMP URLs to get stats
	sempUrl string
}

// SolaceMetricValues is the struct for Observed Metric Values
type SolaceDMScalerMetricValues struct {
	//	Observed clientTxByteRate
	aggregateClientTxByteRate int64
	//	Observed clientAverageTxByteRate
	aggregateClientAverageTxByteRate int64
	//	Observed clientTxMsgRate
	aggregateClientTxMsgRate int64
	//	Observed clientAverageTxMsgRate
	aggregateClientAverageTxMsgRate int64
	//	Observed Client queued messages count
	aggregateClientD1QueueMsgCount int64
	//	Observed Client queue message count
	aggregateClientD1QueueUnitsOfWorkRatio int64
}

type SolaceDMScaler struct {
	metricType    v2.MetricTargetType
	configuration *SolaceDMScalerConfiguration
	httpClient    *http.Client
	logger        logr.Logger
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
		return "error"
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
		return "error"
	}

	return fmt.Sprint(string(out))
}

/*****/

// Constructor for SolaceDMScaler
func NewSolaceDMScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	// Create HTTP Client
	httpClient := kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false)

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, solaceDMScalerID+"_scaler")

	// Parse Solace Metadata
	scalerConfig, err := parseSolaceDMConfiguration(config)
	if err != nil {
		logger.Error(err, "Error parsing Solace Trigger Metadata or missing values")
		return nil, err
	}

	return &SolaceDMScaler{
		metricType:    metricType,
		configuration: scalerConfig,
		httpClient:    httpClient,
		logger:        logger,
	}, nil
}

func parseSolaceDMConfiguration(scalerConfig *scalersconfig.ScalerConfig) (*SolaceDMScalerConfiguration, error) {
	meta := &SolaceDMScalerConfiguration{}
	if err := scalerConfig.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing metadata: %w", err)
	}
	meta.triggerIndex = scalerConfig.TriggerIndex
	meta.sempUrl = fmt.Sprintf(sempUrl, meta.host)
	return meta, nil
}

// Interface required method!!
// GetMetricSpecForScaling returns the metrics based on which this scaler determines that the ScaleTarget scales. This is used to construct the HPA spec that is created for
// this scaled object. The labels used should match the selectors used in GetMetrics
func (s *SolaceDMScaler) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	var metricSpecList []v2.MetricSpec
	var triggerIndex = s.configuration.triggerIndex
	var clientNamePattern = s.configuration.clientNamePrefix

	// Target Client TxByteRate
	if s.configuration.aggregateClientTxByteRateTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-dm-%s-%s", clientNamePattern, aggregateClientTxByteRateTargetMetricName))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(triggerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.configuration.aggregateClientTxByteRateTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceDMExternalMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	//
	// Target Client AverageTxByteRate
	if s.configuration.aggregateClientAverageTxByteRateTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-dm-%s-%s", clientNamePattern, aggregateClientAverageTxByteRateTargetMetricName))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(triggerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.configuration.aggregateClientAverageTxByteRateTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceDMExternalMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	//
	// Target Client TxMsgRate
	if s.configuration.aggregateClientTxMsgRateTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-dm-%s-%s", clientNamePattern, aggregateClientTxMsgRateTargetMetricName))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(triggerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.configuration.aggregateClientTxMsgRateTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceDMExternalMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	//
	// Target Client AverageTxMsgRate
	if s.configuration.aggregateClientAverageTxMsgRateTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-dm-%s-%s", clientNamePattern, aggregateClientAverageTxMsgRateTargetMetricName))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(triggerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.configuration.aggregateClientAverageTxMsgRateTarget),
		}
		metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceDMExternalMetricType}
		metricSpecList = append(metricSpecList, metricSpec)
	}
	//
	// D-1 Queue Backlog
	//this metric will always report!
	// if the TxRate is >= TxRateTarget this metric will report the same value as the target value - '100' meaning the consumers are on track
	// if the TxRate is <  TxRateTarget could be because of slow consumer or issues, so
	// metric value = base value + ((length-work/max-work)*100) of D-1 queue
	// HPA always takes the higher metric for the scaling so this metric will be relevant only in that scenario
	metricName := kedautil.NormalizeString(fmt.Sprintf("solace-dm-%s-%s", clientNamePattern, aggregateClientD1QueueBacklogTargetMetricName))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(triggerIndex, metricName),
		},
		Target: GetMetricTarget(s.metricType, d1QueueBacklogMetricBaseValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: solaceDMExternalMetricType}
	metricSpecList = append(metricSpecList, metricSpec)

	return metricSpecList

}

// Interface required method!!
// GetMetricsAndActivity returns the metric values and activity for a metric Name
func (s *SolaceDMScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	metricValues, err := &SolaceDMScalerMetricValues{}, error(nil)

	err = s.getClientStats(ctx, metricValues)

	if err != nil {
		log.Fatal(err, "-", "Error getting semp metrics")
	}
	//
	err = s.getClientStatQueues(ctx, metricValues)

	if err != nil {
		log.Fatal(err, "-", "Error getting semp metrics")
	}

	var metric external_metrics.ExternalMetricValue
	switch {
	//
	case strings.HasSuffix(metricName, aggregateClientTxByteRateTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.aggregateClientTxByteRate))
	case strings.HasSuffix(metricName, aggregateClientAverageTxByteRateTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.aggregateClientAverageTxByteRate))
	//
	case strings.HasSuffix(metricName, aggregateClientTxMsgRateTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.aggregateClientTxMsgRate))
	case strings.HasSuffix(metricName, aggregateClientAverageTxMsgRateTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.aggregateClientAverageTxMsgRate))
	//
	case strings.HasSuffix(metricName, aggregateClientD1QueueBacklogTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.aggregateClientD1QueueUnitsOfWorkRatio))

	default:
		// Should never end up here
		err := fmt.Errorf("unidentified metric: %s", metricName)
		s.logger.Error(err, "returning error to calling app")
		return []external_metrics.ExternalMetricValue{}, false, err
	}
	//always return true for activation unless its not needed this needs at least one instace
	return []external_metrics.ExternalMetricValue{metric}, true, nil
}

// Interface required method!!
// Close any resources that need disposing when scaler is no longer used or destroyed
func (s *SolaceDMScaler) Close(ctx context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

/************************************************************************************************************/
/************ Additional Internal methods                                       *****************************/
/************************************************************************************************************/
func (s *SolaceDMScaler) getClientStats(ctx context.Context, metricValues *SolaceDMScalerMetricValues) error {
	//get client stats for the clients that have the prefix
	clientStatsReqBody := "<rpc><show><client><name>" + s.configuration.clientNamePrefix + "*</name><stats/></client></show></rpc>"

	bodyBytes, err := s.getSEMPMetrics(ctx, clientStatsReqBody)
	if err != nil {
		return fmt.Errorf("reading client stats failed: %w", err)
	}

	var clientStatsReply CSRpcReply

	err = xml.Unmarshal(bodyBytes, &clientStatsReply)
	if err != nil {
		return fmt.Errorf("unmarshalling the body failed: %w", err)
	}

	clients := clientStatsReply.RPC.Show.Client.PrimaryVirtualRouter.Clients

	//make sure they are clean before the agg
	metricValues.aggregateClientTxByteRate = 0
	metricValues.aggregateClientAverageTxByteRate = 0
	metricValues.aggregateClientTxMsgRate = 0
	metricValues.aggregateClientAverageTxMsgRate = 0

	for i := 0; i < len(clients); i++ {
		client := clients[i]

		//only consider the configured vpn
		if client.MessageVpn == s.configuration.messageVpn {
			metricValues.aggregateClientTxByteRate += client.Stats.ByteRatePerSecond
			metricValues.aggregateClientAverageTxByteRate += client.Stats.AvgByteRatePerMinute
			metricValues.aggregateClientTxMsgRate += client.Stats.MsgRatePerSecond
			metricValues.aggregateClientAverageTxMsgRate += client.Stats.AvgMsgRatePerMinute
		}
	}
	//no error
	return nil
}

func (s *SolaceDMScaler) getClientStatQueues(ctx context.Context, metricValues *SolaceDMScalerMetricValues) error {
	//get client stats for the clients that have the prefix
	clientStatQueuesReqBody := fmt.Sprintf("<rpc><show><client><name>%s*</name><stats></stats><queues></queues></client></show></rpc>", s.configuration.clientNamePrefix)

	bodyBytes, err := s.getSEMPMetrics(ctx, clientStatQueuesReqBody)
	if err != nil {
		return fmt.Errorf("reading client stats failed: %w", err)
	}

	var clientStatsReply CSQRpcReply

	err = xml.Unmarshal(bodyBytes, &clientStatsReply)
	if err != nil {
		return fmt.Errorf("unmarshalling the body failed: %w", err)
	}

	clients := clientStatsReply.RPC.Show.Client.PrimaryVirtualRouter.Clients

	//make sure they are clean before the agg
	metricValues.aggregateClientD1QueueMsgCount = 0

	var aggregatedMaxUnitOfWork int64 = 0
	var aggregatedUsedUnitsOfWork int64 = 0

	for i := 0; i < len(clients); i++ {
		client := clients[i]

		//only consider the configured vpn
		if client.MessageVpn == s.configuration.messageVpn {
			clientQueues := client.ClientQueues

			for j := 0; j < len(clientQueues); j++ {
				clientQueue := clientQueues[j]

				if clientQueue.QueuePriority == d1PriorityQueue {
					metricValues.aggregateClientD1QueueMsgCount += clientQueue.LengthMsgs
					aggregatedMaxUnitOfWork += clientQueue.MaxWork
					aggregatedUsedUnitsOfWork += clientQueue.LengthWork
				}
			}
		}

		calculatedV := d1QueueBacklogMetricBaseValue + ((aggregatedUsedUnitsOfWork / aggregatedMaxUnitOfWork) * 100)
		metricValues.aggregateClientD1QueueUnitsOfWorkRatio = int64(calculatedV)
	}
	//no error
	return nil
}

func (s *SolaceDMScaler) getSEMPMetrics(ctx context.Context, requestBody string) ([]byte, error) {
	var sempUrl = s.configuration.sempUrl
	var httpClient = s.httpClient

	//	Retrieve metrics from Solace SEMP v1
	//	Define HTTP Request
	request, err := http.NewRequestWithContext(ctx, "POST", sempUrl, bytes.NewBuffer([]byte(requestBody)))
	if err != nil {
		return []byte{}, fmt.Errorf("failed attempting request to solace semp api: %w", err)
	}

	//	Add HTTP Auth and Headers
	request.SetBasicAuth(s.configuration.username, s.configuration.password)
	request.Header.Set("Content-Type", "application/xml")

	//	Call Solace SEMP API
	response, err := httpClient.Do(request)
	if err != nil {
		return []byte{}, fmt.Errorf("call to solace semp api failed: %w", err)
	}
	// close the response body  reader when the func returns
	defer response.Body.Close()

	s.logger.Info("Response", "Header", response)
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

	s.logger.Info("Response", "Body", string(responseBodyValue))

	return responseBodyValue, nil
}
