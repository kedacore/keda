package scalers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Shopify/sarama"
	log "github.com/Sirupsen/logrus"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
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
	brokers []string
	group   string
	topic   string
}

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

	return meta, nil
}

// IsActive determines if we need to scale from zero
func (s *kafkaScaler) IsActive(ctx context.Context) (bool, error) {
	lag, err := s.getKafkaOffsetLag()
	if err != nil {
		log.Errorf("error getting offset: %s", err)
		return false, err
	}

	return lag > 0, nil
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

func (s *kafkaScaler) getKafkaOffsetLag() (int32, error) {
	topicsMetadata, err := s.admin.DescribeTopics([]string{s.metadata.topic})
	if err != nil {
		log.Errorf("error describing topics: %s\n", err)
		return -1, err
	}
	if len(topicsMetadata) != 1 {
		log.Errorf("expected only 1 topic metadata, got %d\n", len(topicsMetadata))
		return -1, err
	}
	partitionMetadata := topicsMetadata[0].Partitions
	partitions := make([]int32, len(partitionMetadata))
	for i, p := range partitionMetadata {
		partitions[i] = p.ID
	}

	offsets, err := s.admin.ListConsumerGroupOffsets(s.metadata.group, map[string][]int32{
		s.metadata.topic: partitions,
	})
	if err != nil {
		log.Errorf("error listing consumer group offsets: %s\n", err)
		return -1, err
	}

	totalLag := int64(0)
	for _, partition := range partitions {
		block := offsets.GetBlock(s.metadata.topic, partition)
		consumerOffset := block.Offset
		latestOffset, err := s.client.GetOffset(s.metadata.topic, partition, sarama.OffsetNewest)
		if err != nil {
			log.Errorf("error finding latest offset for topic %s and partition %d\n", s.metadata.topic, partition)
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
		totalLag += lag

		log.Debugf("Group %s has a lag of %d for topic %s and partition %d\n", s.metadata.group, lag, s.metadata.topic, partition)

		// TODO: Should we break out as soon as we detect a non-zero
		// lag? When we scale to more than 1, we may want the total
		// lag and not just whether it's zero or non-zero.
	}

	if totalLag > 0 {
		return 1, nil
	}
	return 0, nil
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
	// TODO: a real metric spec
	return []v2beta1.MetricSpec{}
}

//GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *kafkaScaler) GetMetrics(ctx context.Context, merticName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	// TODO: actual metric values
	return []external_metrics.ExternalMetricValue{}, nil
}
