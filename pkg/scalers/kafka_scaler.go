package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/IBM/sarama"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type kafkaScaler struct {
	metricType      v2.MetricTargetType
	metadata        kafkaMetadata
	client          sarama.Client
	admin           sarama.ClusterAdmin
	logger          logr.Logger
	previousOffsets map[string]map[int32]int64
}

const (
	stringEnable  = "enable"
	stringDisable = "disable"
)

type kafkaMetadata struct {
	bootstrapServers       []string
	group                  string
	topic                  string
	partitionLimitation    []int32
	lagThreshold           int64
	activationLagThreshold int64
	offsetResetPolicy      offsetResetPolicy
	allowIdleConsumers     bool
	excludePersistentLag   bool
	version                sarama.KafkaVersion

	// If an invalid offset is found, whether to scale to 1 (false - the default) so consumption can
	// occur or scale to 0 (true). See discussion in https://github.com/kedacore/keda/issues/2612
	scaleToZeroOnInvalidOffset bool

	// SASL
	saslType kafkaSaslType
	username string
	password string

	// OAUTHBEARER
	scopes                []string
	oauthTokenEndpointURI string
	oauthExtensions       map[string]string

	// TLS
	enableTLS   bool
	cert        string
	key         string
	keyPassword string
	ca          string
	unsafeSsl   bool

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
	KafkaSASLTypeOAuthbearer kafkaSaslType = "oauthbearer"
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
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "kafka_scaler")

	kafkaMetadata, err := parseKafkaMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing kafka metadata: %w", err)
	}

	client, admin, err := getKafkaClients(kafkaMetadata)
	if err != nil {
		return nil, err
	}

	previousOffsets := make(map[string]map[int32]int64)

	return &kafkaScaler{
		client:          client,
		admin:           admin,
		metricType:      metricType,
		metadata:        kafkaMetadata,
		logger:          logger,
		previousOffsets: previousOffsets,
	}, nil
}

func parseKafkaAuthParams(config *ScalerConfig, meta *kafkaMetadata) error {
	meta.saslType = KafkaSASLTypeNone
	var saslAuthType string
	switch {
	case config.TriggerMetadata["sasl"] != "":
		saslAuthType = config.TriggerMetadata["sasl"]
	default:
		saslAuthType = ""
	}
	if val, ok := config.AuthParams["sasl"]; ok {
		if saslAuthType != "" {
			return errors.New("unable to set `sasl` in both ScaledObject and TriggerAuthentication together")
		}
		saslAuthType = val
	}

	if saslAuthType != "" {
		saslAuthType = strings.TrimSpace(saslAuthType)
		mode := kafkaSaslType(saslAuthType)

		if mode == KafkaSASLTypePlaintext || mode == KafkaSASLTypeSCRAMSHA256 || mode == KafkaSASLTypeSCRAMSHA512 || mode == KafkaSASLTypeOAuthbearer {
			if config.AuthParams["username"] == "" {
				return errors.New("no username given")
			}
			meta.username = strings.TrimSpace(config.AuthParams["username"])

			if config.AuthParams["password"] == "" {
				return errors.New("no password given")
			}
			meta.password = strings.TrimSpace(config.AuthParams["password"])
			meta.saslType = mode

			if mode == KafkaSASLTypeOAuthbearer {
				meta.scopes = strings.Split(config.AuthParams["scopes"], ",")

				if config.AuthParams["oauthTokenEndpointUri"] == "" {
					return errors.New("no oauth token endpoint uri given")
				}
				meta.oauthTokenEndpointURI = strings.TrimSpace(config.AuthParams["oauthTokenEndpointUri"])

				meta.oauthExtensions = make(map[string]string)
				oauthExtensionsRaw := config.AuthParams["oauthExtensions"]
				if oauthExtensionsRaw != "" {
					for _, extension := range strings.Split(oauthExtensionsRaw, ",") {
						splittedExtension := strings.Split(extension, "=")
						if len(splittedExtension) != 2 {
							return errors.New("invalid OAuthBearer extension, must be of format key=value")
						}
						meta.oauthExtensions[splittedExtension[0]] = splittedExtension[1]
					}
				}
			}
		} else {
			return fmt.Errorf("err SASL mode %s given", mode)
		}
	}

	meta.enableTLS = false
	enableTLS := false
	if val, ok := config.TriggerMetadata["tls"]; ok {
		switch val {
		case stringEnable:
			enableTLS = true
		case stringDisable:
			enableTLS = false
		default:
			return fmt.Errorf("error incorrect TLS value given, got %s", val)
		}
	}

	if val, ok := config.AuthParams["tls"]; ok {
		val = strings.TrimSpace(val)
		if enableTLS {
			return errors.New("unable to set `tls` in both ScaledObject and TriggerAuthentication together")
		}
		switch val {
		case stringEnable:
			enableTLS = true
		case stringDisable:
			enableTLS = false
		default:
			return fmt.Errorf("error incorrect TLS value given, got %s", val)
		}
	}

	if enableTLS {
		return parseTLS(config, meta)
	}

	return nil
}

func parseTLS(config *ScalerConfig, meta *kafkaMetadata) error {
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
	meta.unsafeSsl = defaultUnsafeSsl

	if val, ok := config.TriggerMetadata["unsafeSsl"]; ok {
		unsafeSsl, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("error parsing unsafeSsl: %w", err)
		}
		meta.unsafeSsl = unsafeSsl
	}

	if value, found := config.AuthParams["keyPassword"]; found {
		meta.keyPassword = value
	} else {
		meta.keyPassword = ""
	}
	meta.enableTLS = true

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

	meta.partitionLimitation = nil
	partitionLimitationMetadata := strings.TrimSpace(config.TriggerMetadata["partitionLimitation"])
	if partitionLimitationMetadata != "" {
		if meta.topic == "" {
			logger.V(1).Info("no specific topic set, ignoring partitionLimitation setting")
		} else {
			pattern := config.TriggerMetadata["partitionLimitation"]
			parsed, err := kedautil.ParseInt32List(pattern)
			if err != nil {
				return meta, fmt.Errorf("error parsing in partitionLimitation '%s': %w", pattern, err)
			}
			meta.partitionLimitation = parsed
			logger.V(0).Info(fmt.Sprintf("partition limit active '%s'", pattern))
		}
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
			return meta, fmt.Errorf("error parsing %q: %w", lagThresholdMetricName, err)
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
			return meta, fmt.Errorf("error parsing %q: %w", activationLagThresholdMetricName, err)
		}
		if t < 0 {
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
			return meta, fmt.Errorf("error parsing allowIdleConsumers: %w", err)
		}
		meta.allowIdleConsumers = t
	}

	meta.excludePersistentLag = false
	if val, ok := config.TriggerMetadata["excludePersistentLag"]; ok {
		t, err := strconv.ParseBool(val)
		if err != nil {
			return meta, fmt.Errorf("error parsing excludePersistentLag: %w", err)
		}
		meta.excludePersistentLag = t
	}

	meta.scaleToZeroOnInvalidOffset = false
	if val, ok := config.TriggerMetadata["scaleToZeroOnInvalidOffset"]; ok {
		t, err := strconv.ParseBool(val)
		if err != nil {
			return meta, fmt.Errorf("error parsing scaleToZeroOnInvalidOffset: %w", err)
		}
		meta.scaleToZeroOnInvalidOffset = t
	}

	meta.version = sarama.V1_0_0_0
	if val, ok := config.TriggerMetadata["version"]; ok {
		val = strings.TrimSpace(val)
		version, err := sarama.ParseKafkaVersion(val)
		if err != nil {
			return meta, fmt.Errorf("error parsing kafka version: %w", err)
		}
		meta.version = version
	}
	meta.scalerIndex = config.ScalerIndex
	return meta, nil
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
		tlsConfig, err := kedautil.NewTLSConfigWithPassword(metadata.cert, metadata.key, metadata.keyPassword, metadata.ca, metadata.unsafeSsl)
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

	if metadata.saslType == KafkaSASLTypeOAuthbearer {
		config.Net.SASL.Mechanism = sarama.SASLTypeOAuth
		config.Net.SASL.TokenProvider = OAuthBearerTokenProvider(metadata.username, metadata.password, metadata.oauthTokenEndpointURI, metadata.scopes, metadata.oauthExtensions)
	}

	client, err := sarama.NewClient(metadata.bootstrapServers, config)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating kafka client: %w", err)
	}

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		if !client.Closed() {
			client.Close()
		}
		return nil, nil, fmt.Errorf("error creating kafka admin: %w", err)
	}

	return client, admin, nil
}

func (s *kafkaScaler) getTopicPartitions() (map[string][]int32, error) {
	var topicsToDescribe = make([]string, 0)

	// when no topic is specified, query to cg group to fetch all subscribed topics
	if s.metadata.topic == "" {
		listCGOffsetResponse, err := s.admin.ListConsumerGroupOffsets(s.metadata.group, nil)
		if err != nil {
			return nil, fmt.Errorf("error listing cg offset: %w", err)
		}

		if listCGOffsetResponse.Err > 0 {
			errMsg := fmt.Errorf("error listing cg offset: %w", listCGOffsetResponse.Err)
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
		return nil, fmt.Errorf("error describing topics: %w", err)
	}

	if s.metadata.topic != "" && len(topicsMetadata) != 1 {
		return nil, fmt.Errorf("expected only 1 topic metadata, got %d", len(topicsMetadata))
	}

	topicPartitions := make(map[string][]int32, len(topicsMetadata))
	for _, topicMetadata := range topicsMetadata {
		if topicMetadata.Err > 0 {
			errMsg := fmt.Errorf("error describing topics: %w", topicMetadata.Err)
			s.logger.Error(errMsg, "")
		}
		partitionMetadata := topicMetadata.Partitions
		var partitions []int32
		for _, p := range partitionMetadata {
			if s.isActivePartition(p.ID) {
				partitions = append(partitions, p.ID)
			}
		}
		if len(partitions) == 0 {
			return nil, fmt.Errorf("expected at least one active partition within the topic '%s'", topicMetadata.Name)
		}

		topicPartitions[topicMetadata.Name] = partitions
	}
	return topicPartitions, nil
}

func (s *kafkaScaler) isActivePartition(pID int32) bool {
	if s.metadata.partitionLimitation == nil {
		return true
	}
	for _, _pID := range s.metadata.partitionLimitation {
		if pID == _pID {
			return true
		}
	}
	return false
}

func (s *kafkaScaler) getConsumerOffsets(topicPartitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	offsets, err := s.admin.ListConsumerGroupOffsets(s.metadata.group, topicPartitions)
	if err != nil {
		return nil, fmt.Errorf("error listing consumer group offsets: %w", err)
	}
	if offsets.Err > 0 {
		errMsg := fmt.Errorf("error listing consumer group offsets: %w", offsets.Err)
		s.logger.Error(errMsg, "")
	}
	return offsets, nil
}

// getLagForPartition returns (lag, lagWithPersistent, error)
// When excludePersistentLag is set to `false` (default), lag will always be equal to lagWithPersistent
// When excludePersistentLag is set to `true`, if partition is deemed to have persistent lag, lag will be set to 0 and lagWithPersistent will be latestOffset - consumerOffset
// These return values will allow proper scaling from 0 -> 1 replicas by the IsActive func.
func (s *kafkaScaler) getLagForPartition(topic string, partitionID int32, offsets *sarama.OffsetFetchResponse, topicPartitionOffsets map[string]map[int32]int64) (int64, int64, error) {
	block := offsets.GetBlock(topic, partitionID)
	if block == nil {
		errMsg := fmt.Errorf("error finding offset block for topic %s and partition %d from offset block: %v", topic, partitionID, offsets.Blocks)
		s.logger.Error(errMsg, "")
		return 0, 0, errMsg
	}
	if block.Err > 0 {
		errMsg := fmt.Errorf("error finding offset block for topic %s and partition %d: %w", topic, partitionID, offsets.Err)
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
		s.logger.V(1).Info(msg)
		return retVal, retVal, nil
	}

	if _, found := topicPartitionOffsets[topic]; !found {
		return 0, 0, fmt.Errorf("error finding partition offset for topic %s", topic)
	}
	latestOffset := topicPartitionOffsets[topic][partitionID]
	if consumerOffset == invalidOffset && s.metadata.offsetResetPolicy == earliest {
		return latestOffset, latestOffset, nil
	}

	// This code block tries to prevent KEDA Kafka trigger from scaling the scale target based on erroneous events
	if s.metadata.excludePersistentLag {
		switch previousOffset, found := s.previousOffsets[topic][partitionID]; {
		case !found:
			// No record of previous offset, so store current consumer offset
			// Allow this consumer lag to be considered in scaling
			if _, topicFound := s.previousOffsets[topic]; !topicFound {
				s.previousOffsets[topic] = map[int32]int64{partitionID: consumerOffset}
			} else {
				s.previousOffsets[topic][partitionID] = consumerOffset
			}
		case previousOffset == consumerOffset:
			// Indicates consumer is still on the same offset as the previous polling cycle, there may be some issue with consuming this offset.
			// return 0, so this consumer lag is not considered for scaling
			return 0, latestOffset - consumerOffset, nil
		default:
			// Successfully Consumed some messages, proceed to change the previous offset
			s.previousOffsets[topic][partitionID] = consumerOffset
		}
	}

	return latestOffset - consumerOffset, latestOffset - consumerOffset, nil
}

// Close closes the kafka admin and client
func (s *kafkaScaler) Close(context.Context) error {
	// underlying client will also be closed on admin's Close() call
	if s.admin == nil {
		return nil
	}
	return s.admin.Close()
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

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *kafkaScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	totalLag, totalLagWithPersistent, err := s.getTotalLag()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}
	metric := GenerateMetricInMili(metricName, float64(totalLag))

	return []external_metrics.ExternalMetricValue{metric}, totalLagWithPersistent > s.metadata.activationLagThreshold, nil
}

// getTotalLag returns totalLag, totalLagWithPersistent, error
// totalLag and totalLagWithPersistent are the summations of lag and lagWithPersistent returned by getLagForPartition function respectively.
// totalLag maybe less than totalLagWithPersistent when excludePersistentLag is set to `true` due to some partitions deemed as having persistent lag
func (s *kafkaScaler) getTotalLag() (int64, int64, error) {
	topicPartitions, err := s.getTopicPartitions()
	if err != nil {
		return 0, 0, err
	}

	consumerOffsets, producerOffsets, err := s.getConsumerAndProducerOffsets(topicPartitions)
	if err != nil {
		return 0, 0, err
	}

	totalLag := int64(0)
	totalLagWithPersistent := int64(0)
	totalTopicPartitions := int64(0)

	for topic, partitionsOffsets := range producerOffsets {
		for partition := range partitionsOffsets {
			lag, lagWithPersistent, err := s.getLagForPartition(topic, partition, consumerOffsets, producerOffsets)
			if err != nil {
				return 0, 0, err
			}
			totalLag += lag
			totalLagWithPersistent += lagWithPersistent
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
	return totalLag, totalLagWithPersistent, nil
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
