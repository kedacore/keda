package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Shopify/sarama"
	log "github.com/Sirupsen/logrus"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type kafkaScaler struct {
	resolvedSecrets map[string]string
	metadata        kafkaMetadata
	client          sarama.Client
	admin           sarama.ClusterAdmin
}

type kafkaMetadata struct {
	brokers      []string
	group        string
	topic        string
	lagThreshold int64
}

const (
	lagThresholdMetricName = "lagThreshold"
	kafkaMetricType        = "External"
)

// NewKafkaScaler creates a new kafkaScaler
func NewKafkaScaler(resolvedSecrets, metadata map[string]string) (Scaler, error) {
	kafkaMetadata, err := parseKafkaMetadata(metadata)
	if err != nil {
		return nil, fmt.Errorf("error parsing kafka metadata: %s", err)
	}

	client, admin, err := getKafkaClients(kafkaMetadata)
	if err != nil {
		return nil, err
	}

	return &kafkaScaler{
		client:          client,
		admin:           admin,
		metadata:        kafkaMetadata,
		resolvedSecrets: resolvedSecrets,
	}, nil
}

func parseKafkaMetadata(metadata map[string]string) (kafkaMetadata, error) {
	meta := kafkaMetadata{}

	if metadata["brokers"] == "" {
		return meta, errors.New("no brokers given")
	}
	meta.brokers = strings.Split(metadata["brokers"], ",")

	if metadata["groupName"] == "" {
		return meta, errors.New("no group name given")
	}
	meta.group = metadata["groupName"]

	if metadata["topicName"] == "" {
		return meta, errors.New("no topic name given")
	}
	meta.topic = metadata["topicName"]

	if metadata[lagThresholdMetricName] == "" {
		return meta, errors.New("no lag threshold given")
	}

	t, err := strconv.ParseInt(metadata[lagThresholdMetricName], 10, 64)
	if err == nil {
		return meta, fmt.Errorf("couldn't parse %s", lagThresholdMetricName)
	}
	meta.lagThreshold = t

	return meta, nil
}

// IsActive determines if we need to scale from zero
func (s *kafkaScaler) IsActive(ctx context.Context) (bool, error) {
	partitions, err := s.getPartitions()
	if err != nil {
		return false, err
	}

	offsets, err := s.getOffsets(partitions)
	if err != nil {
		return false, err
	}

	for _, partition := range partitions {
		lag := s.getLagForPartition(partition, offsets)
		log.Debugf("Group %s has a lag of %d for topic %s and partition %d\n", s.metadata.group, lag, s.metadata.topic, partition)

		// Return as soon as a lag was detected for any partition
		if lag > 0 {
			return true, nil
		}
	}

	return false, nil
}

func getKafkaClients(metadata kafkaMetadata) (sarama.Client, sarama.ClusterAdmin, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V1_0_0_0

	client, err := sarama.NewClient(metadata.brokers, config)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating kafka client: %s", err)
	}

	admin, err := sarama.NewClusterAdmin(metadata.brokers, config)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating kafka admin: %s", err)
	}

	return client, admin, nil
}

func (s *kafkaScaler) getPartitions() ([]int32, error) {
	topicsMetadata, err := s.admin.DescribeTopics([]string{s.metadata.topic})
	if err != nil {
		return nil, fmt.Errorf("error describing topics: %s", err)
	}
	if len(topicsMetadata) != 1 {
		return nil, fmt.Errorf("expected only 1 topic metadata, got %d", len(topicsMetadata))
	}

	partitionMetadata := topicsMetadata[0].Partitions
	partitions := make([]int32, len(partitionMetadata))
	for i, p := range partitionMetadata {
		partitions[i] = p.ID
	}

	return partitions, nil
}

func (s *kafkaScaler) getOffsets(partitions []int32) (*sarama.OffsetFetchResponse, error) {
	offsets, err := s.admin.ListConsumerGroupOffsets(s.metadata.group, map[string][]int32{
		s.metadata.topic: partitions,
	})

	if err != nil {
		return nil, fmt.Errorf("error listing consumer group offsets: %s", err)
	}

	return offsets, nil
}

func (s *kafkaScaler) getLagForPartition(partition int32, offsets *sarama.OffsetFetchResponse) int64 {
	block := offsets.GetBlock(s.metadata.topic, partition)
	consumerOffset := block.Offset
	latestOffset, err := s.client.GetOffset(s.metadata.topic, partition, sarama.OffsetNewest)
	if err != nil {
		log.Errorf("error finding latest offset for topic %s and partition %d\n", s.metadata.topic, partition)
		return 0
	}

	var lag int64
	// For now, assume a consumer group that has no committed
	// offset will read all messages from the topic. This may be
	// something we want to allow users to configure.
	if consumerOffset == sarama.OffsetNewest || consumerOffset == sarama.OffsetOldest {
		lag = latestOffset
	} else {
		lag = latestOffset - consumerOffset
	}

	return lag
}

func (s *kafkaScaler) Close() error {
	err := s.client.Close()
	if err != nil {
		return err
	}
	err = s.admin.Close()
	if err != nil {
		return err
	}

	return nil
}

func (s *kafkaScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	return []v2beta1.MetricSpec{
		{
			External: &v2beta1.ExternalMetricSource{
				MetricName:         lagThresholdMetricName,
				TargetAverageValue: resource.NewQuantity(s.metadata.lagThreshold, resource.DecimalSI),
			},
			Type: kafkaMetricType,
		},
	}
}

//GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *kafkaScaler) GetMetrics(ctx context.Context, merticName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	partitions, err := s.getPartitions()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, err
	}

	offsets, err := s.getOffsets(partitions)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, err
	}

	totalLag := int64(0)
	for _, partition := range partitions {
		lag := s.getLagForPartition(partition, offsets)
		log.Debugf("Group %s has a lag of %d for topic %s and partition %d\n", s.metadata.group, lag, s.metadata.topic, partition)

		totalLag += lag
	}

	// We should find a way to not scale beyond the number of partitions
	metric := external_metrics.ExternalMetricValue{
		MetricName: merticName,
		Value:      *resource.NewQuantity(int64(totalLag), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
