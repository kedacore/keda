package scalers

import (
	"context"
	"errors"
	"strings"

	"github.com/Shopify/sarama"
	log "github.com/Sirupsen/logrus"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type kafkaScaler struct {
	resolvedSecrets, metadata map[string]string
}

type kafkaMetadata struct {
	brokers []string
	group   string
	topic   string
}

// NewKafkaScaler creates a new kafkaScaler
func NewKafkaScaler(resolvedSecrets, metadata map[string]string) Scaler {
	return &kafkaScaler{
		metadata:        metadata,
		resolvedSecrets: resolvedSecrets,
	}
}

func (s *kafkaScaler) parseKafkaMetadata() (kafkaMetadata, error) {
	meta := kafkaMetadata{}

	if s.metadata["brokers"] == "" {
		return meta, errors.New("no brokers given")
	}
	meta.brokers = strings.Split(s.metadata["brokers"], ",")

	if s.metadata["groupName"] == "" {
		return meta, errors.New("no group name given")
	}
	meta.group = s.metadata["groupName"]

	if s.metadata["topicName"] == "" {
		return meta, errors.New("no topic name given")
	}
	meta.topic = s.metadata["topicName"]

	return meta, nil
}

// IsActive determines if we need to scale from zero
func (s *kafkaScaler) IsActive(ctx context.Context) (bool, error) {
	lag, err := s.getKafkaOffsetLag()
	if err != nil {
		log.Errorf("error %s", err)
		return false, err
	}

	return lag > 0, nil
}

func (s *kafkaScaler) getKafkaOffsetLag() (int32, error) {
	meta, err := s.parseKafkaMetadata()
	if err != nil {
		log.Errorf("erring parsing kafka metadata: %s\n", err)
		return -1, err
	}

	config := sarama.NewConfig()
	config.Version = sarama.V1_0_0_0

	client, err := sarama.NewClient(meta.brokers, config)
	if err != nil {
		log.Errorf("error creating kafka client: %s\n", err)
		return -1, err
	}
	defer client.Close()

	admin, err := sarama.NewClusterAdmin(meta.brokers, config)
	if err != nil {
		log.Errorf("error creating kafka admin: %s\n", err)
		return -1, err
	}
	defer admin.Close()

	topicsMetadata, err := admin.DescribeTopics([]string{meta.topic})
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

	offsets, err := admin.ListConsumerGroupOffsets(meta.group, map[string][]int32{
		meta.topic: partitions,
	})
	if err != nil {
		log.Errorf("error listing consumer group offsets: %s\n", err)
		return -1, err
	}

	totalLag := int64(0)
	for _, partition := range partitions {
		block := offsets.GetBlock(meta.topic, partition)
		consumerOffset := block.Offset
		latestOffset, err := client.GetOffset(meta.topic, partition, sarama.OffsetNewest)
		if err != nil {
			log.Errorf("error finding latest offset for topic %s and partition %d\n", meta.topic, partition)
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

		log.Debugf("Group %s has a lag of %d for topic %s and partition %d\n", meta.group, lag, meta.topic, partition)

		// TODO: Should we break out as soon as we detect a non-zero
		// lag? When we scale to more than 1, we may want the total
		// lag and not just whether it's zero or non-zero.
	}

	if totalLag > 0 {
		return 1, nil
	}
	return 0, nil
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
