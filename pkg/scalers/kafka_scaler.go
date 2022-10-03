package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type kafkaScaler struct {
	metricType v2.MetricTargetType
	metadata   kafkaMetadata
	client     sarama.Client
	admin      sarama.ClusterAdmin
	logger     logr.Logger
}

type kafkaMetadata struct {
	bootstrapServers       []string
	group                  string
	topic                  string
	lagThreshold           int64
	activationLagThreshold int64
	offsetResetPolicy      offsetResetPolicy
	allowIdleConsumers     bool
	version                sarama.KafkaVersion

	// If an invalid offset is found, whether to scale to 1 (false - the default) so consumption can
	// occur or scale to 0 (true). See discussion in https://github.com/kedacore/keda/issues/2612
	scaleToZeroOnInvalidOffset bool

	// SASL
	saslType kafkaSaslType
	username string
	password string

	// TLS
	enableTLS   bool
	cert        string
	key         string
	keyPassword string
	ca          string

	scalerIndex int
}

type offsetResetPolicy string

const (
	latest   offsetResetPolicy = "latest"
	earliest offsetResetPolicy = "earliest"
)

type kafkaSaslType string

// supported SASL types
const (
	KafkaSASLTypeNone        kafkaSaslType = "none"
	KafkaSASLTypePlaintext   kafkaSaslType = "plaintext"
	KafkaSASLTypeSCRAMSHA256 kafkaSaslType = "scram_sha256"
	KafkaSASLTypeSCRAMSHA512 kafkaSaslType = "scram_sha512"
)

const (
	lagThresholdMetricName             = "lagThreshold"
	activationLagThresholdMetricName   = "activationLagThreshold"
	kafkaMetricType                    = "External"
	defaultKafkaLagThreshold           = 10
	defaultKafkaActivationLagThreshold = 0
	defaultOffsetResetPolicy           = latest
	invalidOffset                      = -1
)

// NewKafkaScaler creates a new kafkaScaler
func NewKafkaScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	logger := InitializeLogger(config, "kafka_scaler")

	kafkaMetadata, err := parseKafkaMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing kafka metadata: %s", err)
	}

	client, admin, err := getKafkaClients(kafkaMetadata)
	if err != nil {
		return nil, err
	}

	return &kafkaScaler{
		client:     client,
		admin:      admin,
		metricType: metricType,
		metadata:   kafkaMetadata,
		logger:     logger,
	}, nil
}

func parseKafkaAuthParams(config *ScalerConfig, meta *kafkaMetadata) error {
	meta.saslType = KafkaSASLTypeNone
	if val, ok := config.AuthParams["sasl"]; ok {
		val = strings.TrimSpace(val)
		mode := kafkaSaslType(val)

		if mode == KafkaSASLTypePlaintext || mode == KafkaSASLTypeSCRAMSHA256 || mode == KafkaSASLTypeSCRAMSHA512 {
			if config.AuthParams["username"] == "" {
				return errors.New("no username given")
			}
			meta.username = strings.TrimSpace(config.AuthParams["username"])

			if config.AuthParams["password"] == "" {
				return errors.New("no password given")
			}
			meta.password = strings.TrimSpace(config.AuthParams["password"])
			meta.saslType = mode
		} else {
			return fmt.Errorf("err SASL mode %s given", mode)
		}
	}

	meta.enableTLS = false
	if val, ok := config.AuthParams["tls"]; ok {
		val = strings.TrimSpace(val)

		if val == "enable" {
			certGiven := config.AuthParams["cert"] != ""
			keyGiven := config.AuthParams["key"] != ""
			if certGiven && !keyGiven {
				return errors.New("key must be provided with cert")
			}
			if keyGiven && !certGiven {
				return errors.New("cert must be provided with key")
			}
			meta.ca = config.AuthParams["ca"]
			meta.cert = config.AuthParams["cert"]
			meta.key = config.AuthParams["key"]
			if value, found := config.AuthParams["keyPassword"]; found {
				meta.keyPassword = value
			} else {
				meta.keyPassword = ""
			}
			meta.enableTLS = true
		} else if val != "disable" {
			return fmt.Errorf("err incorrect value for TLS given: %s", val)
		}
	}

	return nil
}

func parseKafkaMetadata(config *ScalerConfig, logger logr.Logger) (kafkaMetadata, error) {
	meta := kafkaMetadata{}
	switch {
	case config.TriggerMetadata["bootstrapServersFromEnv"] != "":
		meta.bootstrapServers = strings.Split(config.ResolvedEnv[config.TriggerMetadata["bootstrapServersFromEnv"]], ",")
	case config.TriggerMetadata["bootstrapServers"] != "":
		meta.bootstrapServers = strings.Split(config.TriggerMetadata["bootstrapServers"], ",")
	default:
		return meta, errors.New("no bootstrapServers given")
	}

	switch {
	case config.TriggerMetadata["consumerGroupFromEnv"] != "":
		meta.group = config.ResolvedEnv[config.TriggerMetadata["consumerGroupFromEnv"]]
	case config.TriggerMetadata["consumerGroup"] != "":
		meta.group = config.TriggerMetadata["consumerGroup"]
	default:
		return meta, errors.New("no consumer group given")
	}

	switch {
	case config.TriggerMetadata["topicFromEnv"] != "":
		meta.topic = config.ResolvedEnv[config.TriggerMetadata["topicFromEnv"]]
	case config.TriggerMetadata["topic"] != "":
		meta.topic = config.TriggerMetadata["topic"]
	default:
		meta.topic = ""
		logger.V(1).Info(fmt.Sprintf("consumer group %q has no topic specified, "+
			"will use all topics subscribed by the consumer group for scaling", meta.group))
	}

	meta.offsetResetPolicy = defaultOffsetResetPolicy

	if config.TriggerMetadata["offsetResetPolicy"] != "" {
		policy := offsetResetPolicy(config.TriggerMetadata["offsetResetPolicy"])
		if policy != earliest && policy != latest {
			return meta, fmt.Errorf("err offsetResetPolicy policy %q given", policy)
		}
		meta.offsetResetPolicy = policy
	}

	meta.lagThreshold = defaultKafkaLagThreshold

	if val, ok := config.TriggerMetadata[lagThresholdMetricName]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("error parsing %q: %s", lagThresholdMetricName, err)
		}
		if t <= 0 {
			return meta, fmt.Errorf("%q must be positive number", lagThresholdMetricName)
		}
		meta.lagThreshold = t
	}

	meta.activationLagThreshold = defaultKafkaActivationLagThreshold

	if val, ok := config.TriggerMetadata[activationLagThresholdMetricName]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("error parsing %q: %s", activationLagThresholdMetricName, err)
		}
		if t <= 0 {
			return meta, fmt.Errorf("%q must be positive number", activationLagThresholdMetricName)
		}
		meta.activationLagThreshold = t
	}

	if err := parseKafkaAuthParams(config, &meta); err != nil {
		return meta, err
	}

	meta.allowIdleConsumers = false
	if val, ok := config.TriggerMetadata["allowIdleConsumers"]; ok {
		t, err := strconv.ParseBool(val)
		if err != nil {
			return meta, fmt.Errorf("error parsing allowIdleConsumers: %s", err)
		}
		meta.allowIdleConsumers = t
	}

	meta.scaleToZeroOnInvalidOffset = false
	if val, ok := config.TriggerMetadata["scaleToZeroOnInvalidOffset"]; ok {
		t, err := strconv.ParseBool(val)
		if err != nil {
			return meta, fmt.Errorf("error parsing scaleToZeroOnInvalidOffset: %s", err)
		}
		meta.scaleToZeroOnInvalidOffset = t
	}

	meta.version = sarama.V1_0_0_0
	if val, ok := config.TriggerMetadata["version"]; ok {
		val = strings.TrimSpace(val)
		version, err := sarama.ParseKafkaVersion(val)
		if err != nil {
			return meta, fmt.Errorf("error parsing kafka version: %s", err)
		}
		meta.version = version
	}
	meta.scalerIndex = config.ScalerIndex
	return meta, nil
}

// IsActive determines if we need to scale from zero
func (s *kafkaScaler) IsActive(ctx context.Context) (bool, error) {
	totalLag, err := s.getTotalLag()
	if err != nil {
		return false, err
	}

	return totalLag > s.metadata.activationLagThreshold, nil
}

func getKafkaClients(metadata kafkaMetadata) (sarama.Client, sarama.ClusterAdmin, error) {
	config := sarama.NewConfig()
	config.Version = metadata.version

	if metadata.saslType != KafkaSASLTypeNone {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = metadata.username
		config.Net.SASL.Password = metadata.password
	}

	if metadata.enableTLS {
		config.Net.TLS.Enable = true
		tlsConfig, err := kedautil.NewTLSConfigWithPassword(metadata.cert, metadata.key, metadata.keyPassword, metadata.ca)
		if err != nil {
			return nil, nil, err
		}
		config.Net.TLS.Config = tlsConfig
	}

	if metadata.saslType == KafkaSASLTypePlaintext {
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}

	if metadata.saslType == KafkaSASLTypeSCRAMSHA256 {
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA256} }
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
	}

	if metadata.saslType == KafkaSASLTypeSCRAMSHA512 {
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
		config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
	}

	client, err := sarama.NewClient(metadata.bootstrapServers, config)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating kafka client: %s", err)
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		if !client.Closed() {
			client.Close()
		}
		return nil, nil, fmt.Errorf("error creating kafka admin: %s", err)
	}

	return client, admin, nil
}

func (s *kafkaScaler) getTopicPartitions() (map[string][]int32, error) {
	var topicsToDescribe = make([]string, 0)

	// when no topic is specified, query to cg group to fetch all subscribed topics
	if s.metadata.topic == "" {
		listCGOffsetResponse, err := s.admin.ListConsumerGroupOffsets(s.metadata.group, nil)
		if err != nil {
			return nil, fmt.Errorf("error listing cg offset: %s", err)
		}

		if listCGOffsetResponse.Err > 0 {
			errMsg := fmt.Errorf("error listing cg offset: %s", listCGOffsetResponse.Err.Error())
			s.logger.Error(errMsg, "")
		}

		for topicName := range listCGOffsetResponse.Blocks {
			topicsToDescribe = append(topicsToDescribe, topicName)
		}
	} else {
		topicsToDescribe = []string{s.metadata.topic}
	}

	topicsMetadata, err := s.admin.DescribeTopics(topicsToDescribe)
	if err != nil {
		return nil, fmt.Errorf("error describing topics: %s", err)
	}

	if s.metadata.topic != "" && len(topicsMetadata) != 1 {
		return nil, fmt.Errorf("expected only 1 topic metadata, got %d", len(topicsMetadata))
	}

	topicPartitions := make(map[string][]int32, len(topicsMetadata))
	for _, topicMetadata := range topicsMetadata {
		if topicMetadata.Err > 0 {
			errMsg := fmt.Errorf("error describing topics: %s", topicMetadata.Err.Error())
			s.logger.Error(errMsg, "")
		}
		partitionMetadata := topicMetadata.Partitions
		partitions := make([]int32, len(partitionMetadata))
		for i, p := range partitionMetadata {
			partitions[i] = p.ID
		}
		topicPartitions[topicMetadata.Name] = partitions
	}
	return topicPartitions, nil
}

func (s *kafkaScaler) getConsumerOffsets(topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	offsets, err := s.admin.ListConsumerGroupOffsets(s.metadata.group, topicPartitions)
	if err != nil {
		return nil, fmt.Errorf("error listing consumer group offsets: %s", err)
	}
	if offsets.Err > 0 {
		errMsg := fmt.Errorf("error listing consumer group offsets: %s", offsets.Err.Error())
		s.logger.Error(errMsg, "")
	}
	return offsets, nil
}

func (s *kafkaScaler) getLagForPartition(topic string, partitionID int32, offsets *sarama.OffsetFetchResponse, topicPartitionOffsets map[string]map[int32]int64) (int64, error) {
	block := offsets.GetBlock(topic, partitionID)
	if block == nil {
		errMsg := fmt.Errorf("error finding offset block for topic %s and partition %d", topic, partitionID)
		s.logger.Error(errMsg, "")
		return 0, errMsg
	}
	if block.Err > 0 {
		errMsg := fmt.Errorf("error finding offset block for topic %s and partition %d: %s", topic, partitionID, offsets.Err.Error())
		s.logger.Error(errMsg, "")
	}

	consumerOffset := block.Offset
	if consumerOffset == invalidOffset && s.metadata.offsetResetPolicy == latest {
		retVal := int64(1)
		if s.metadata.scaleToZeroOnInvalidOffset {
			retVal = 0
		}
		msg := fmt.Sprintf(
			"invalid offset found for topic %s in group %s and partition %d, probably no offset is committed yet. Returning with lag of %d",
			topic, s.metadata.group, partitionID, retVal)
		s.logger.V(0).Info(msg)
		return retVal, nil
	}

	if _, found := topicPartitionOffsets[topic]; !found {
		return 0, fmt.Errorf("error finding partition offset for topic %s", topic)
	}
	latestOffset := topicPartitionOffsets[topic][partitionID]
	if consumerOffset == invalidOffset && s.metadata.offsetResetPolicy == earliest {
		return latestOffset, nil
	}
	return latestOffset - consumerOffset, nil
}

// Close closes the kafka admin and client
func (s *kafkaScaler) Close(context.Context) error {
	// underlying client will also be closed on admin's Close() call
	err := s.admin.Close()
	if err != nil {
		return err
	}

	return nil
}

func (s *kafkaScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	var metricName string
	if s.metadata.topic != "" {
		metricName = fmt.Sprintf("kafka-%s", s.metadata.topic)
	} else {
		metricName = fmt.Sprintf("kafka-%s-topics", s.metadata.group)
	}

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(metricName)),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.lagThreshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: kafkaMetricType}
	return []v2.MetricSpec{metricSpec}
}

type consumerOffsetResult struct {
	consumerOffsets *sarama.OffsetFetchResponse
	err             error
}

type producerOffsetResult struct {
	producerOffsets map[string]map[int32]int64
	err             error
}

func (s *kafkaScaler) getConsumerAndProducerOffsets(topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, map[string]map[int32]int64, error) {
	consumerChan := make(chan consumerOffsetResult, 1)
	go func() {
		consumerOffsets, err := s.getConsumerOffsets(topicPartitions)
		consumerChan <- consumerOffsetResult{consumerOffsets, err}
	}()

	producerChan := make(chan producerOffsetResult, 1)
	go func() {
		producerOffsets, err := s.getProducerOffsets(topicPartitions)
		producerChan <- producerOffsetResult{producerOffsets, err}
	}()

	consumerRes := <-consumerChan
	if consumerRes.err != nil {
		return nil, nil, consumerRes.err
	}

	producerRes := <-producerChan
	if producerRes.err != nil {
		return nil, nil, producerRes.err
	}

	return consumerRes.consumerOffsets, producerRes.producerOffsets, nil
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *kafkaScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	totalLag, err := s.getTotalLag()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, err
	}
	metric := GenerateMetricInMili(metricName, float64(totalLag))

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *kafkaScaler) getTotalLag() (int64, error) {
	topicPartitions, err := s.getTopicPartitions()
	if err != nil {
		return 0, err
	}

	consumerOffsets, producerOffsets, err := s.getConsumerAndProducerOffsets(topicPartitions)
	if err != nil {
		return 0, err
	}

	totalLag := int64(0)
	totalTopicPartitions := int64(0)

	for topic, partitionsOffsets := range producerOffsets {
		for partition := range partitionsOffsets {
			lag, _ := s.getLagForPartition(topic, partition, consumerOffsets, producerOffsets)
			totalLag += lag
		}
		totalTopicPartitions += (int64)(len(partitionsOffsets))
	}
	s.logger.V(1).Info(fmt.Sprintf("Kafka scaler: Providing metrics based on totalLag %v, topicPartitions %v, threshold %v", totalLag, len(topicPartitions), s.metadata.lagThreshold))

	if !s.metadata.allowIdleConsumers {
		// don't scale out beyond the number of topicPartitions
		if (totalLag / s.metadata.lagThreshold) > totalTopicPartitions {
			totalLag = totalTopicPartitions * s.metadata.lagThreshold
		}
	}
	return totalLag, nil
}

type brokerOffsetResult struct {
	offsetResp *sarama.OffsetResponse
	err        error
}

func (s *kafkaScaler) getProducerOffsets(topicPartitions map[string][]int32) (map[string]map[int32]int64, error) {
	version := int16(0)
	if s.client.Config().Version.IsAtLeast(sarama.V0_10_1_0) {
		version = 1
	}

	// Step 1: build one OffsetRequest instance per broker.
	requests := make(map[*sarama.Broker]*sarama.OffsetRequest)

	for topic, partitions := range topicPartitions {
		for _, partitionID := range partitions {
			broker, err := s.client.Leader(topic, partitionID)
			if err != nil {
				return nil, err
			}
			request, ok := requests[broker]
			if !ok {
				request = &sarama.OffsetRequest{Version: version}
				requests[broker] = request
			}
			request.AddBlock(topic, partitionID, sarama.OffsetNewest, 1)
		}
	}

	// Step 2: send requests, one per broker, and collect topicPartitionsOffsets
	resultCh := make(chan brokerOffsetResult, len(requests))
	var wg sync.WaitGroup
	wg.Add(len(requests))
	for broker, request := range requests {
		go func(brCopy *sarama.Broker, reqCopy *sarama.OffsetRequest) {
			defer wg.Done()
			response, err := brCopy.GetAvailableOffsets(reqCopy)
			resultCh <- brokerOffsetResult{response, err}
		}(broker, request)
	}

	wg.Wait()
	close(resultCh)

	topicPartitionsOffsets := make(map[string]map[int32]int64)
	for brokerOffsetRes := range resultCh {
		if brokerOffsetRes.err != nil {
			return nil, brokerOffsetRes.err
		}

		for topic, blocks := range brokerOffsetRes.offsetResp.Blocks {
			if _, found := topicPartitionsOffsets[topic]; !found {
				topicPartitionsOffsets[topic] = make(map[int32]int64)
			}
			for partitionID, block := range blocks {
				if block.Err != sarama.ErrNoError {
					return nil, block.Err
				}
				topicPartitionsOffsets[topic][partitionID] = block.Offset
			}
		}
	}

	return topicPartitionsOffsets, nil
}
