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
	Host string `keda:"name=host, order=triggerMetadata"`
	// Basic Auth Username
	Username string `keda:"name=username, order=authParams;triggerMetadata;resolvedEnv"`
	// Basic Auth Password
	Password string `keda:"name=password, order=authParams;triggerMetadata;resolvedEnv"`

	// Message VPN
	MessageVpn string `keda:"name=messageVpn, order=triggerMetadata"`
	// Client Name Prefix
	ClientNamePrefix string `keda:"name=clientNamePrefix, order=triggerMetadata"`

	// Target Client TxByteRate
	AggregatedClientTxByteRateTarget int64 `keda:"name=aggregatedClientTxByteRateTarget, order=triggerMetadata, optional=true, default=0"`
	// Target Client AverageTxByteRate
	AggregatedClientAverageTxByteRateTarget int64 `keda:"name=aggregatedClientAverageTxByteRateTarget, order=triggerMetadata, optional=true, default=0"`
	// Target Client TxMsgRate
	AggregatedClientTxMsgRateTarget int64 `keda:"name=aggregatedClientTxMsgRateTarget, order=triggerMetadata, optional=true, default=0"`
	// Target Client AverageTxMsgRate
	AggregatedClientAverageTxMsgRateTarget int64 `keda:"name=aggregatedClientAverageTxMsgRateTarget, order=triggerMetadata, optional=true, default=0"`

	// Activation Client TxByteRate
	ActivationAggregatedClientTxByteRateTarget int `keda:"name=activationAggregatedClientTxByteRateTarget, order=triggerMetadata, optional=true, default=0"`
	// Activation Target Average TxByteRate
	ActivationAggregatedClientAverageTxByteRateTarget int `keda:"name=activationAggregatedClientAverageTxByteRateTarget, order=triggerMetadata=true, optional, default=0"`
	// Activation Client TxMsgRate
	ActivationAggregatedClientTxMsgRateTarget int `keda:"name=activationAggregatedClientTxMsgRateTarget, order=triggerMetadata, optional=true, default=0"`
	// Activation Target Average TxMsgRate
	ActivationAggregatedClientAverageTxMsgRateTarget int `keda:"name=activationAggregatedClientAverageTxMsgRateTarget, order=triggerMetadata, optional=true, default=0"`

	// Full SEMP URLs to get stats
	sempUrl string
}

func (s *SolaceDMScalerConfiguration) Validate() error {
	//	Check that we have at least one positive target value for the scaler
	if s.AggregatedClientTxByteRateTarget < 1 && s.AggregatedClientAverageTxByteRateTarget < 1 && s.AggregatedClientTxMsgRateTarget < 1 && s.AggregatedClientAverageTxMsgRateTarget < 1 {
		return errors.New("no target value found in the scaler configuration")
	}

	return nil
}

// SolaceMetricValues is the struct for Observed Metric Values
type SolaceDMScalerMetricValues struct {
	//	Observed clientTxByteRate
	AggregateClientTxByteRate int64
	//	Observed clientAverageTxByteRate
	AggregateClientAverageTxByteRate int64
	//	Observed clientTxMsgRate
	AggregateClientTxMsgRate int64
	//	Observed clientAverageTxMsgRate
	AggregateClientAverageTxMsgRate int64
	//	Observed Client queued messages count
	AggregateClientD1QueueMsgCount int64
	//	Observed Client queue message count
	AggregateClientD1QueueUnitsOfWorkRatio int64
}

func (c SolaceDMScalerMetricValues) String() string {
	out, err := json.Marshal(c)
	if err != nil {
		return "error"
	}

	return fmt.Sprint(string(out))
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
	meta.sempUrl = fmt.Sprintf(sempUrl, meta.Host)
	return meta, nil
}

// Interface required method!!
// GetMetricSpecForScaling returns the metrics based on which this scaler determines that the ScaleTarget scales. This is used to construct the HPA spec that is created for
// this scaled object. The labels used should match the selectors used in GetMetrics
func (s *SolaceDMScaler) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	var metricSpecList []v2.MetricSpec
	var triggerIndex = s.configuration.triggerIndex
	var clientNamePattern = s.configuration.ClientNamePrefix

	// Target Client TxByteRate
	if s.configuration.AggregatedClientTxByteRateTarget > 0 {
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-dm-%s-%s", clientNamePattern, aggregateClientTxByteRateTargetMetricName))
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
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-dm-%s-%s", clientNamePattern, aggregateClientAverageTxByteRateTargetMetricName))
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
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-dm-%s-%s", clientNamePattern, aggregateClientTxMsgRateTargetMetricName))
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
		metricName := kedautil.NormalizeString(fmt.Sprintf("solace-dm-%s-%s", clientNamePattern, aggregateClientAverageTxMsgRateTargetMetricName))
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(triggerIndex, metricName),
			},
			Target: GetMetricTarget(s.metricType, s.configuration.AggregatedClientAverageTxMsgRateTarget),
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

	s.logger.Info(fmt.Sprintf("* MetricName: '%s'", metricName))

	if strings.HasSuffix(metricName, aggregateClientTxByteRateTargetMetricName) ||
		strings.HasSuffix(metricName, aggregateClientAverageTxByteRateTargetMetricName) ||
		strings.HasSuffix(metricName, aggregateClientTxMsgRateTargetMetricName) ||
		strings.HasSuffix(metricName, aggregateClientAverageTxMsgRateTargetMetricName) {

		err = s.getClientStats(ctx, metricValues)

		if err != nil {
			s.logger.Error(err, "call to semp endpoint (client stats) failed")
			return []external_metrics.ExternalMetricValue{}, false, err
		}
	} else if strings.HasSuffix(metricName, aggregateClientD1QueueBacklogTargetMetricName) {
		//
		err = s.getClientStatQueues(ctx, metricValues)

		if err != nil {
			s.logger.Error(err, "call to semp endpoint (client queues stats) failed")
			return []external_metrics.ExternalMetricValue{}, false, err
		}
	}

	var metric external_metrics.ExternalMetricValue
	switch {
	//
	case strings.HasSuffix(metricName, aggregateClientTxByteRateTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.AggregateClientTxByteRate))
	case strings.HasSuffix(metricName, aggregateClientAverageTxByteRateTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.AggregateClientAverageTxByteRate))
	//
	case strings.HasSuffix(metricName, aggregateClientTxMsgRateTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.AggregateClientTxMsgRate))
	case strings.HasSuffix(metricName, aggregateClientAverageTxMsgRateTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.AggregateClientAverageTxMsgRate))
	//
	case strings.HasSuffix(metricName, aggregateClientD1QueueBacklogTargetMetricName):
		metric = GenerateMetricInMili(metricName, float64(metricValues.AggregateClientD1QueueUnitsOfWorkRatio))

	default:
		// Should never end up here
		err := fmt.Errorf("unidentified metric: %s", metricName)
		s.logger.Error(err, "returning error to calling app")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	s.logger.Info(fmt.Sprintf("Metrics: '%s'", metricValues))
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
	clientStatsReqBody := "<rpc><show><client><name>" + s.configuration.ClientNamePrefix + "*</name><stats/></client></show></rpc>"

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
	metricValues.AggregateClientTxByteRate = 0
	metricValues.AggregateClientAverageTxByteRate = 0
	metricValues.AggregateClientTxMsgRate = 0
	metricValues.AggregateClientAverageTxMsgRate = 0
	var numClients int64 = 0

	for i := 0; i < len(clients); i++ {
		client := clients[i]

		//only consider the configured vpn
		if client.MessageVpn == s.configuration.MessageVpn {
			numClients++
			s.logger.Info(fmt.Sprintf("    Client[%d] - ByteRatePerSecond: '%d', AvgByteRatePerMinute: '%d', MsgRatePerSecond: '%d', AvgMsgRatePerMinute: '%d'", i, client.Stats.ByteRatePerSecond, client.Stats.AvgByteRatePerMinute, client.Stats.MsgRatePerSecond, client.Stats.AvgMsgRatePerMinute))
			metricValues.AggregateClientTxByteRate += client.Stats.ByteRatePerSecond
			metricValues.AggregateClientAverageTxByteRate += client.Stats.AvgByteRatePerMinute
			metricValues.AggregateClientTxMsgRate += client.Stats.MsgRatePerSecond
			metricValues.AggregateClientAverageTxMsgRate += client.Stats.AvgMsgRatePerMinute
		}
	}
	/*
		metricValues.AggregateClientTxByteRate = int64(float64(metricValues.AggregateClientTxByteRate) / float64(numClients))
		metricValues.AggregateClientAverageTxByteRate = int64(float64(metricValues.AggregateClientAverageTxByteRate) / float64(numClients))
		metricValues.AggregateClientTxMsgRate = int64(float64(metricValues.AggregateClientTxMsgRate) / float64(numClients))
		metricValues.AggregateClientAverageTxMsgRate = int64(float64(metricValues.AggregateClientAverageTxMsgRate) / float64(numClients))
	*/

	s.logger.Info(fmt.Sprintf("   MetricValues - ByteRatePerSecond: '%d', AvgByteRatePerMinute: '%d', MsgRatePerSecond: '%d', AvgMsgRatePerMinute: '%d'", metricValues.AggregateClientTxByteRate, metricValues.AggregateClientAverageTxByteRate, metricValues.AggregateClientTxMsgRate, metricValues.AggregateClientAverageTxMsgRate))
	//no error
	return nil
}

func (s *SolaceDMScaler) getClientStatQueues(ctx context.Context, metricValues *SolaceDMScalerMetricValues) error {
	//get client stats for the clients that have the prefix
	clientStatQueuesReqBody := fmt.Sprintf("<rpc><show><client><name>%s*</name><stats></stats><queues></queues></client></show></rpc>", s.configuration.ClientNamePrefix)

	bodyBytes, err := s.getSEMPMetrics(ctx, clientStatQueuesReqBody)
	if err != nil {
		return fmt.Errorf("reading client stats queue failed: %w", err)
	}

	var clientStatsReply CSQRpcReply

	err = xml.Unmarshal(bodyBytes, &clientStatsReply)
	if err != nil {
		return fmt.Errorf("unmarshalling the body failed: %w", err)
	}

	clients := clientStatsReply.RPC.Show.Client.PrimaryVirtualRouter.Clients

	//make sure they are clean before the agg
	metricValues.AggregateClientD1QueueMsgCount = 0
	metricValues.AggregateClientD1QueueUnitsOfWorkRatio = 0

	var aggregatedMaxUnitOfWork int64 = 0
	var aggregatedUsedUnitsOfWork int64 = 0
	var numClients int32 = 0

	for i := 0; i < len(clients); i++ {
		client := clients[i]

		//only consider the configured vpn
		if client.MessageVpn == s.configuration.MessageVpn {
			clientQueues := client.ClientQueues

			for j := 0; j < len(clientQueues); j++ {
				clientQueue := clientQueues[j]

				if clientQueue.QueuePriority == d1PriorityQueue {
					s.logger.Info(fmt.Sprintf("    Client[%d]Q[%s] - LengthMsgs: '%d', MaxWork: '%d', LengthWork: '%d'", i, clientQueue.QueuePriority, clientQueue.LengthMsgs, clientQueue.MaxWork, clientQueue.LengthWork))

					numClients++
					metricValues.AggregateClientD1QueueMsgCount += clientQueue.LengthMsgs
					aggregatedMaxUnitOfWork += clientQueue.MaxWork
					aggregatedUsedUnitsOfWork += clientQueue.LengthWork
				}
			}
		}
	}

	if aggregatedMaxUnitOfWork == 0 {
		metricValues.AggregateClientD1QueueUnitsOfWorkRatio = int64(d1QueueBacklogMetricBaseValue)
	} else {
		var ratio float64 = (float64(aggregatedUsedUnitsOfWork) / float64(aggregatedMaxUnitOfWork))
		var calculatedValue float64 = (float64(1.0) + ratio) * float64(d1QueueBacklogMetricBaseValue)

		metricValues.AggregateClientD1QueueUnitsOfWorkRatio = int64(calculatedValue)
	}

	s.logger.Info(fmt.Sprintf("   MetricValues - d1QueueBacklogMetricBaseValue: '%d', aggregatedUsedUnitsOfWork: '%d', aggregatedMaxUnitOfWork: '%d', AggregateClientD1QueueUnitsOfWorkRatio: '%d'", d1QueueBacklogMetricBaseValue, aggregatedUsedUnitsOfWork, aggregatedMaxUnitOfWork, metricValues.AggregateClientD1QueueUnitsOfWorkRatio))
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
	request.SetBasicAuth(s.configuration.Username, s.configuration.Password)
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

	//s.logger.V(1).Info("Response", "Body", string(responseBodyValue))

	return responseBodyValue, nil
}
