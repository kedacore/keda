/*
Copyright 2023 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Please note that this is an experimental scaler based on the kafka-go library.

package scalers

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/aws_msk_iam_v2"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	awsutils "github.com/kedacore/keda/v2/pkg/scalers/aws"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type apacheKafkaScaler struct {
	metricType      v2.MetricTargetType
	metadata        apacheKafkaMetadata
	client          *kafka.Client
	logger          logr.Logger
	previousOffsets map[string]map[int]int64
}

type apacheKafkaMetadata struct {
	bootstrapServers       []string
	group                  string
	topic                  []string
	partitionLimitation    []int32
	lagThreshold           int64
	activationLagThreshold int64
	offsetResetPolicy      offsetResetPolicy
	allowIdleConsumers     bool
	excludePersistentLag   bool

	// If an invalid offset is found, whether to scale to 1 (false - the default) so consumption can
	// occur or scale to 0 (true). See discussion in https://github.com/kedacore/keda/issues/2612
	scaleToZeroOnInvalidOffset bool
	limitToPartitionsWithLag   bool

	// SASL
	saslType kafkaSaslType
	username string
	password string

	// MSK
	awsRegion        string
	awsEndpoint      string
	awsAuthorization awsutils.AuthorizationMetadata

	// TLS
	enableTLS   bool
	cert        string
	key         string
	keyPassword string
	ca          string

	triggerIndex int
}

const (
	KafkaSASLTypeMskIam = "aws_msk_iam"
)

// NewApacheKafkaScaler creates a new apacheKafkaScaler
func NewApacheKafkaScaler(ctx context.Context, config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "apache_kafka_scaler")

	kafkaMetadata, err := parseApacheKafkaMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing kafka metadata: %w", err)
	}

	client, err := getApacheKafkaClient(ctx, kafkaMetadata, logger)
	if err != nil {
		return nil, err
	}

	previousOffsets := make(map[string]map[int]int64)

	return &apacheKafkaScaler{
		client:          client,
		metricType:      metricType,
		metadata:        kafkaMetadata,
		logger:          logger,
		previousOffsets: previousOffsets,
	}, nil
}

func parseApacheKafkaAuthParams(config *ScalerConfig, meta *apacheKafkaMetadata) error {
	meta.enableTLS = false
	var enableTLS bool
	tlsString, err := getParameterFromConfigV2(config, "tls", true, true, false, true, "disable")
	if err != nil {
		return fmt.Errorf("error incorrect TLS value given. %w", err)
	}
	tlsString = strings.TrimSpace(tlsString)
	switch tlsString {
	case stringEnable:
		enableTLS = true
	case stringDisable:
		enableTLS = false
	default:
		return fmt.Errorf("error incorrect TLS value given, got %s", tlsString)
	}

	if enableTLS {
		certGiven, err := getParameterFromConfigV2(config, "cert", false, true, false, true, "")
		if err != nil {
			return err
		}
		keyGiven, err := getParameterFromConfigV2(config, "key", false, true, false, true, "")
		if err != nil {
			return err
		}
		if certGiven == "" && keyGiven != "" {
			return errors.New("key must be provided with cert")
		}
		if keyGiven == "" && certGiven != "" {
			return errors.New("cert must be provided with key")
		}
		ca, err := getParameterFromConfigV2(config, "ca", false, true, false, true, "")
		if err != nil {
			return err
		}
		meta.ca = ca
		meta.cert = certGiven
		meta.key = keyGiven
		keyPassword, err := getParameterFromConfigV2(config, "keyPassword", false, true, false, true, "")
		if err != nil {
			return err
		}
		meta.keyPassword = keyPassword
		meta.enableTLS = true
	}

	meta.saslType = KafkaSASLTypeNone
	saslAuthType, err := getParameterFromConfigV2(config, "sasl", true, true, false, true, "")
	if err != nil {
		return err
	}

	if saslAuthType != "" {
		saslAuthType = strings.TrimSpace(saslAuthType)
		switch mode := kafkaSaslType(saslAuthType); mode {
		case KafkaSASLTypeMskIam:
			meta.saslType = mode
			awsEndpoint, err := getParameterFromConfigV2(config, "awsEndpoint", true, false, false, true, "")
			if err != nil {
				return err
			}
			if awsEndpoint != "" {
				meta.awsEndpoint = awsEndpoint
			}
			if !meta.enableTLS {
				return errors.New("TLS is required for MSK")
			}
			awsRegion, err := getParameterFromConfigV2(config, "awsRegion", true, false, false, false, "")
			if err != nil {
				return fmt.Errorf("%w. No awsRegion given", err)
			}
			meta.awsRegion = awsRegion
			auth, err := awsutils.GetAwsAuthorization(config.TriggerUniqueKey, config.PodIdentity, config.TriggerMetadata, config.AuthParams, config.ResolvedEnv)
			if err != nil {
				return err
			}
			meta.awsAuthorization = auth
		case KafkaSASLTypePlaintext:
			fallthrough
		case KafkaSASLTypeSCRAMSHA256:
			fallthrough
		case KafkaSASLTypeSCRAMSHA512:
			username, err := getParameterFromConfigV2(config, "username", false, true, false, false, "")
			if err != nil {
				return fmt.Errorf("%w. No username given", err)
			}
			meta.username = strings.TrimSpace(username)

			password, err := getParameterFromConfigV2(config, "password", false, true, false, false, "")
			if err != nil {
				return fmt.Errorf("%w. No password given", err)
			}
			meta.password = strings.TrimSpace(password)
		case KafkaSASLTypeOAuthbearer:
			return errors.New("SASL/OAUTHBEARER is not implemented yet")
		default:
			return fmt.Errorf("err sasl type %q given", mode)
		}
	}

	return nil
}

func parseApacheKafkaMetadata(config *ScalerConfig, logger logr.Logger) (apacheKafkaMetadata, error) {
	meta := apacheKafkaMetadata{}
	bootstrapServers, err := getParameterFromConfigV2(config, "bootstrapServers", true, false, true, false, "")
	if err != nil {
		return meta, fmt.Errorf("no bootstrapServers given. %w", err)
	}
	meta.bootstrapServers = strings.Split(bootstrapServers, ",")

	consumerGroup, err := getParameterFromConfigV2(config, "consumerGroup", true, false, true, false, "")
	if err != nil {
		return meta, fmt.Errorf("no consumer group given. %w", err)
	}
	meta.group = consumerGroup

	topic, err := getParameterFromConfigV2(config, "topic", true, false, true, true, "")
	if err != nil {
		return meta, err
	}
	if topic == "" {
		meta.topic = []string{}
		logger.V(1).Info(fmt.Sprintf("consumer group %q has no topics specified, "+
			"will use all topics subscribed by the consumer group for scaling", meta.group))
	} else {
		meta.topic = strings.Split(topic, ",")
	}

	meta.partitionLimitation = nil
	partitionLimitationMetadata, err := getParameterFromConfigV2(config, "partitionLimitation", true, false, false, true, "")
	if err != nil {
		return meta, err
	}
	if partitionLimitationMetadata != "" {
		if meta.topic == nil || len(meta.topic) == 0 {
			logger.V(1).Info("no specific topics set, ignoring partitionLimitation setting")
		} else {
			pattern := strings.TrimSpace(partitionLimitationMetadata)
			parsed, err := kedautil.ParseInt32List(pattern)
			if err != nil {
				return meta, fmt.Errorf("error parsing in partitionLimitation '%s': %w", pattern, err)
			}
			meta.partitionLimitation = parsed
			logger.V(0).Info(fmt.Sprintf("partition limit active '%s'", pattern))
		}
	}

	meta.offsetResetPolicy = defaultOffsetResetPolicy
	offsetResetPolicyRaw, err := getParameterFromConfigV2(config, "offsetResetPolicy", true, false, false, true, "")
	if err != nil {
		return meta, err
	}
	if offsetResetPolicyRaw != "" {
		policy := offsetResetPolicy(offsetResetPolicyRaw)
		if policy != earliest && policy != latest {
			return meta, fmt.Errorf("err offsetResetPolicy policy %q given", offsetResetPolicyRaw)
		}
		meta.offsetResetPolicy = policy
	}

	meta.lagThreshold = defaultKafkaLagThreshold
	lagThreshold, err := getParameterFromConfigV2(config, lagThresholdMetricName, true, false, false, true, defaultKafkaLagThreshold)
	if err != nil {
		return meta, err
	}
	if lagThreshold <= 0 {
		return meta, fmt.Errorf("%q must be positive number", lagThresholdMetricName)
	}
	meta.lagThreshold = lagThreshold

	meta.activationLagThreshold = defaultKafkaActivationLagThreshold
	activationLagThreshold, err := getParameterFromConfigV2(config, activationLagThresholdMetricName, true, false, false, true, defaultKafkaActivationLagThreshold)
	if err != nil {
		return meta, err
	}
	if activationLagThreshold < 0 {
		return meta, fmt.Errorf("%q must be positive number", activationLagThresholdMetricName)
	}

	if err := parseApacheKafkaAuthParams(config, &meta); err != nil {
		return meta, err
	}

	allowIDConsumers, err := getParameterFromConfigV2(config, "allowIdleConsumers", true, false, false, true, false)
	if err != nil {
		return meta, fmt.Errorf("error parsing allowIdleConsumers: %w", err)
	}
	meta.allowIdleConsumers = allowIDConsumers

	excludePersistentLag, err := getParameterFromConfigV2(config, "excludePersistentLag", true, false, false, true, false)
	if err != nil {
		return meta, fmt.Errorf("error parsing excludePersistentLag: %w", err)
	}
	meta.excludePersistentLag = excludePersistentLag

	scaleToZeroOnInvalidOffset, err := getParameterFromConfigV2(config, "scaleToZeroOnInvalidOffset", true, false, false, true, false)
	if err != nil {
		return meta, fmt.Errorf("error parsing scaleToZeroOnInvalidOffset: %w", err)
	}
	meta.scaleToZeroOnInvalidOffset = scaleToZeroOnInvalidOffset

	limitToPartitionsWithLag, err := getParameterFromConfigV2(config, "limitToPartitionsWithLag", true, false, false, true, false)
	if err != nil {
		return meta, err
	}
	meta.limitToPartitionsWithLag = limitToPartitionsWithLag
	if meta.allowIdleConsumers && meta.limitToPartitionsWithLag {
		return meta, fmt.Errorf("allowIdleConsumers and limitToPartitionsWithLag cannot be set simultaneously")
	}
	if len(meta.topic) == 0 && meta.limitToPartitionsWithLag {
		return meta, fmt.Errorf("topic must be specified when using limitToPartitionsWithLag")
	}

	meta.triggerIndex = config.TriggerIndex
	return meta, nil
}

func getApacheKafkaClient(ctx context.Context, metadata apacheKafkaMetadata, logger logr.Logger) (*kafka.Client, error) {
	var saslMechanism sasl.Mechanism
	var tlsConfig *tls.Config
	var err error

	logger.V(4).Info(fmt.Sprintf("Kafka SASL type %s", metadata.saslType))
	if metadata.enableTLS {
		tlsConfig, err = kedautil.NewTLSConfigWithPassword(metadata.cert, metadata.key, metadata.keyPassword, metadata.ca, false)
		if err != nil {
			return nil, err
		}
	}

	switch metadata.saslType {
	case KafkaSASLTypeNone:
		saslMechanism = nil
	case KafkaSASLTypePlaintext:
		saslMechanism = plain.Mechanism{
			Username: metadata.username,
			Password: metadata.password,
		}
	case KafkaSASLTypeSCRAMSHA256:
		saslMechanism, err = scram.Mechanism(scram.SHA256, metadata.username, metadata.password)
		if err != nil {
			return nil, err
		}
	case KafkaSASLTypeSCRAMSHA512:
		saslMechanism, err = scram.Mechanism(scram.SHA512, metadata.username, metadata.password)
		if err != nil {
			return nil, err
		}
	case KafkaSASLTypeOAuthbearer:
		return nil, errors.New("SASL/OAUTHBEARER is not implemented yet")
	case KafkaSASLTypeMskIam:
		cfg, err := awsutils.GetAwsConfig(ctx, metadata.awsRegion, metadata.awsAuthorization)
		if err != nil {
			return nil, err
		}

		saslMechanism = aws_msk_iam_v2.NewMechanism(*cfg)
	default:
		return nil, fmt.Errorf("err sasl type %q given", metadata.saslType)
	}

	transport := &kafka.Transport{
		TLS:  tlsConfig,
		SASL: saslMechanism,
	}
	client := kafka.Client{
		Addr:      kafka.TCP(metadata.bootstrapServers...),
		Transport: transport,
	}
	if err != nil {
		return nil, fmt.Errorf("error creating kafka client: %w", err)
	}

	return &client, nil
}

func (s *apacheKafkaScaler) getTopicPartitions(ctx context.Context) (map[string][]int, error) {
	metadata, err := s.client.Metadata(ctx, &kafka.MetadataRequest{
		Addr: s.client.Addr,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting metadata: %w", err)
	}
	s.logger.V(1).Info(fmt.Sprintf("Listed topics %v", metadata.Topics))

	if len(s.metadata.topic) == 0 {
		// in case of empty topic name, we will get all topics that the consumer group is subscribed to
		describeGrpReq := &kafka.DescribeGroupsRequest{
			Addr: s.client.Addr,
			GroupIDs: []string{
				s.metadata.group,
			},
		}
		describeGrp, err := s.client.DescribeGroups(ctx, describeGrpReq)
		if err != nil {
			return nil, fmt.Errorf("error describing group: %w", err)
		}
		if len(describeGrp.Groups[0].Members) == 0 {
			return nil, fmt.Errorf("no active members in group %s, group-state is %s", s.metadata.group, describeGrp.Groups[0].GroupState)
		}
		s.logger.V(4).Info(fmt.Sprintf("Described group %s with response %v", s.metadata.group, describeGrp))

		result := make(map[string][]int)
		for _, topic := range metadata.Topics {
			partitions := make([]int, 0)
			for _, partition := range topic.Partitions {
				// if no partitions limitatitions are specified, all partitions are considered
				if (len(s.metadata.partitionLimitation) == 0) ||
					(len(s.metadata.partitionLimitation) > 0 && kedautil.Contains(s.metadata.partitionLimitation, int32(partition.ID))) {
					partitions = append(partitions, partition.ID)
				}
			}
			result[topic.Name] = partitions
		}
		return result, nil
	}
	result := make(map[string][]int)
	for _, topic := range metadata.Topics {
		partitions := make([]int, 0)
		if kedautil.Contains(s.metadata.topic, topic.Name) {
			for _, partition := range topic.Partitions {
				if (len(s.metadata.partitionLimitation) == 0) ||
					(len(s.metadata.partitionLimitation) > 0 && kedautil.Contains(s.metadata.partitionLimitation, int32(partition.ID))) {
					partitions = append(partitions, partition.ID)
				}
			}
		}
		result[topic.Name] = partitions
	}
	return result, nil
}

func (s *apacheKafkaScaler) getConsumerOffsets(ctx context.Context, topicPartitions map[string][]int) (map[string]map[int]int64, error) {
	response, err := s.client.OffsetFetch(
		ctx,
		&kafka.OffsetFetchRequest{
			GroupID: s.metadata.group,
			Topics:  topicPartitions,
		},
	)
	if err != nil || response.Error != nil {
		return nil, fmt.Errorf("error listing consumer group offset: %w", err)
	}
	consumerOffset := make(map[string]map[int]int64)
	for topic, partitionsOffset := range response.Topics {
		consumerOffset[topic] = make(map[int]int64)
		for _, partition := range partitionsOffset {
			consumerOffset[topic][partition.Partition] = partition.CommittedOffset
		}
	}
	return consumerOffset, nil
}

/*
getLagForPartition returns (lag, lagWithPersistent, error)

When excludePersistentLag is set to `false` (default), lag will always be equal to lagWithPersistent
When excludePersistentLag is set to `true`, if partition is deemed to have persistent lag, lag will be set to 0 and lagWithPersistent will be latestOffset - consumerOffset
These return values will allow proper scaling from 0 -> 1 replicas by the IsActive func.
*/
func (s *apacheKafkaScaler) getLagForPartition(topic string, partitionID int, consumerOffsets map[string]map[int]int64, producerOffsets map[string]map[int]int64) (int64, int64, error) {
	if len(consumerOffsets) == 0 {
		return 0, 0, fmt.Errorf("consumerOffsets is empty")
	}
	if len(producerOffsets) == 0 {
		return 0, 0, fmt.Errorf("producerOffsets is empty")
	}

	consumerOffset := consumerOffsets[topic][partitionID]
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

	if _, found := producerOffsets[topic]; !found {
		return 0, 0, fmt.Errorf("error finding partition offset for topic %s", topic)
	}
	producerOffset := producerOffsets[topic][partitionID]
	if consumerOffset == invalidOffset && s.metadata.offsetResetPolicy == earliest {
		return producerOffset, producerOffset, nil
	}

	// This code block tries to prevent KEDA Kafka trigger from scaling the scale target based on erroneous events
	if s.metadata.excludePersistentLag {
		switch previousOffset, found := s.previousOffsets[topic][partitionID]; {
		case !found:
			// No record of previous offset, so store current consumer offset
			// Allow this consumer lag to be considered in scaling
			if _, topicFound := s.previousOffsets[topic]; !topicFound {
				s.previousOffsets[topic] = map[int]int64{partitionID: consumerOffset}
			} else {
				s.previousOffsets[topic][partitionID] = consumerOffset
			}
		case previousOffset == consumerOffset:
			// Indicates consumer is still on the same offset as the previous polling cycle, there may be some issue with consuming this offset.
			// return 0, so this consumer lag is not considered for scaling
			return 0, producerOffset - consumerOffset, nil
		default:
			// Successfully Consumed some messages, proceed to change the previous offset
			s.previousOffsets[topic][partitionID] = consumerOffset
		}
	}

	s.logger.V(4).Info(fmt.Sprintf("Consumer offset for topic %s in group %s and partition %d is %d", topic, s.metadata.group, partitionID, consumerOffset))
	s.logger.V(4).Info(fmt.Sprintf("Producer offset for topic %s in group %s and partition %d is %d", topic, s.metadata.group, partitionID, producerOffset))

	return producerOffset - consumerOffset, producerOffset - consumerOffset, nil
}

// Close closes the kafka client
func (s *apacheKafkaScaler) Close(context.Context) error {
	if s.client == nil {
		return nil
	}
	transport := s.client.Transport.(*kafka.Transport)
	if transport != nil {
		transport.CloseIdleConnections()
	}
	return nil
}

func (s *apacheKafkaScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	var metricName string
	if s.metadata.topic != nil && len(s.metadata.topic) > 0 {
		metricName = fmt.Sprintf("kafka-%s", strings.Join(s.metadata.topic, ","))
	} else {
		metricName = fmt.Sprintf("kafka-%s-topics", s.metadata.group)
	}

	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, kedautil.NormalizeString(metricName)),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.lagThreshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: kafkaMetricType}
	return []v2.MetricSpec{metricSpec}
}

type apacheKafkaConsumerOffsetResult struct {
	consumerOffsets map[string]map[int]int64
	err             error
}

type apacheKafkaProducerOffsetResult struct {
	producerOffsets map[string]map[int]int64
	err             error
}

// getConsumerAndProducerOffsets returns (consumerOffsets, producerOffsets, error)
func (s *apacheKafkaScaler) getConsumerAndProducerOffsets(ctx context.Context, topicPartitions map[string][]int) (map[string]map[int]int64, map[string]map[int]int64, error) {
	consumerChan := make(chan apacheKafkaConsumerOffsetResult, 1)
	go func() {
		consumerOffsets, err := s.getConsumerOffsets(ctx, topicPartitions)
		consumerChan <- apacheKafkaConsumerOffsetResult{consumerOffsets, err}
	}()

	producerChan := make(chan apacheKafkaProducerOffsetResult, 1)
	go func() {
		producerOffsets, err := s.getProducerOffsets(ctx, topicPartitions)
		producerChan <- apacheKafkaProducerOffsetResult{producerOffsets, err}
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
func (s *apacheKafkaScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	totalLag, totalLagWithPersistent, err := s.getTotalLag(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}
	metric := GenerateMetricInMili(metricName, float64(totalLag))

	return []external_metrics.ExternalMetricValue{metric}, totalLagWithPersistent > s.metadata.activationLagThreshold, nil
}

// getTotalLag returns totalLag, totalLagWithPersistent, error
// totalLag and totalLagWithPersistent are the summations of lag and lagWithPersistent returned by getLagForPartition function respectively.
// totalLag maybe less than totalLagWithPersistent when excludePersistentLag is set to `true` due to some partitions deemed as having persistent lag
func (s *apacheKafkaScaler) getTotalLag(ctx context.Context) (int64, int64, error) {
	topicPartitions, err := s.getTopicPartitions(ctx)
	if err != nil {
		return 0, 0, err
	}
	s.logger.V(4).Info(fmt.Sprintf("Kafka scaler: Topic partitions %v", topicPartitions))

	consumerOffsets, producerOffsets, err := s.getConsumerAndProducerOffsets(ctx, topicPartitions)
	s.logger.V(4).Info(fmt.Sprintf("Kafka scaler: Consumer offsets %v, producer offsets %v", consumerOffsets, producerOffsets))
	if err != nil {
		return 0, 0, err
	}

	totalLag := int64(0)
	totalLagWithPersistent := int64(0)
	totalTopicPartitions := int64(0)
	partitionsWithLag := int64(0)

	for topic, partitionsOffsets := range producerOffsets {
		for partition := range partitionsOffsets {
			lag, lagWithPersistent, err := s.getLagForPartition(topic, partition, consumerOffsets, producerOffsets)
			if err != nil {
				return 0, 0, err
			}
			totalLag += lag
			totalLagWithPersistent += lagWithPersistent

			if lag > 0 {
				partitionsWithLag++
			}
		}
		totalTopicPartitions += (int64)(len(partitionsOffsets))
	}
	s.logger.V(1).Info(fmt.Sprintf("Kafka scaler: Providing metrics based on totalLag %v, topicPartitions %v, threshold %v", totalLag, topicPartitions, s.metadata.lagThreshold))

	s.logger.V(1).Info(fmt.Sprintf("Kafka scaler: Consumer offsets %v, producer offsets %v", consumerOffsets, producerOffsets))

	if !s.metadata.allowIdleConsumers || s.metadata.limitToPartitionsWithLag {
		// don't scale out beyond the number of topicPartitions or partitionsWithLag depending on settings
		upperBound := totalTopicPartitions
		if s.metadata.limitToPartitionsWithLag {
			upperBound = partitionsWithLag
		}

		if (totalLag / s.metadata.lagThreshold) > upperBound {
			totalLag = upperBound * s.metadata.lagThreshold
		}
	}
	return totalLag, totalLagWithPersistent, nil
}

// getProducerOffsets returns the latest offsets for the given topic partitions
func (s *apacheKafkaScaler) getProducerOffsets(ctx context.Context, topicPartitions map[string][]int) (map[string]map[int]int64, error) {
	// Step 1: build one OffsetRequest
	offsetRequest := make(map[string][]kafka.OffsetRequest)

	for topic, partitions := range topicPartitions {
		for _, partitionID := range partitions {
			offsetRequest[topic] = append(offsetRequest[topic], kafka.FirstOffsetOf(partitionID), kafka.LastOffsetOf(partitionID))
		}
	}

	// Step 2: send request
	res, err := s.client.ListOffsets(ctx, &kafka.ListOffsetsRequest{
		Addr:   s.client.Addr,
		Topics: offsetRequest,
	})
	if err != nil {
		return nil, err
	}

	// Step 3: parse response and return
	producerOffsets := make(map[string]map[int]int64)
	for topic, partitionOffset := range res.Topics {
		producerOffsets[topic] = make(map[int]int64)
		for _, partition := range partitionOffset {
			producerOffsets[topic][partition.Partition] = partition.LastOffset
		}
	}

	return producerOffsets, nil
}
