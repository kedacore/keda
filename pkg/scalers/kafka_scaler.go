package scalers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Shopify/sarama"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type kafkaScaler struct {
	metadata kafkaMetadata
	client   sarama.Client
	admin    sarama.ClusterAdmin
}

type kafkaMetadata struct {
	brokers      []string
	group        string
	topic        string
	lagThreshold int64

	// auth
	authMode kafkaAuthMode
	username string
	password string

	// ssl
	cert string
	key  string
	ca   string
}

type kafkaAuthMode string

const (
	kafkaAuthModeForNone            kafkaAuthMode = "none"
	kafkaAuthModeForSaslPlaintext   kafkaAuthMode = "sasl_plaintext"
	kafkaAuthModeForSaslScramSha256 kafkaAuthMode = "sasl_scram_sha256"
	kafkaAuthModeForSaslScramSha512 kafkaAuthMode = "sasl_scram_sha512"
	kafkaAuthModeForSaslSSL         kafkaAuthMode = "sasl_ssl"
)

const (
	lagThresholdMetricName   = "lagThreshold"
	kafkaMetricType          = "External"
	defaultKafkaLagThreshold = 10
)

var kafkaLog = logf.Log.WithName("kafka_scaler")

// NewKafkaScaler creates a new kafkaScaler
func NewKafkaScaler(resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	kafkaMetadata, err := parseKafkaMetadata(resolvedEnv, metadata, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing kafka metadata: %s", err)
	}

	client, admin, err := getKafkaClients(kafkaMetadata)
	if err != nil {
		return nil, err
	}

	return &kafkaScaler{
		client:   client,
		admin:    admin,
		metadata: kafkaMetadata,
	}, nil
}

func parseKafkaMetadata(resolvedEnv, metadata, authParams map[string]string) (kafkaMetadata, error) {
	meta := kafkaMetadata{}

	if metadata["brokerList"] == "" {
		return meta, errors.New("no brokerList given")
	}
	meta.brokers = strings.Split(metadata["brokerList"], ",")

	if metadata["consumerGroup"] == "" {
		return meta, errors.New("no consumer group given")
	}
	meta.group = metadata["consumerGroup"]

	if metadata["topic"] == "" {
		return meta, errors.New("no topic given")
	}
	meta.topic = metadata["topic"]

	meta.lagThreshold = defaultKafkaLagThreshold

	if val, ok := metadata[lagThresholdMetricName]; ok {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return meta, fmt.Errorf("error parsing %s: %s", lagThresholdMetricName, err)
		}
		meta.lagThreshold = t
	}

	meta.authMode = kafkaAuthModeForNone
	if val, ok := authParams["authMode"]; ok {
		mode := kafkaAuthMode(val)
		if mode != kafkaAuthModeForNone && mode != kafkaAuthModeForSaslPlaintext && mode != kafkaAuthModeForSaslSSL && mode != kafkaAuthModeForSaslScramSha256 && mode != kafkaAuthModeForSaslScramSha512 {
			return meta, fmt.Errorf("err auth mode %s given", mode)
		}

		meta.authMode = mode
	}

	if meta.authMode != kafkaAuthModeForNone {
		if authParams["username"] == "" {
			return meta, errors.New("no username given")
		}
		meta.username = authParams["username"]

		if authParams["password"] == "" {
			return meta, errors.New("no password given")
		}
		meta.password = authParams["password"]
	}

	if meta.authMode == kafkaAuthModeForSaslSSL {
		if authParams["ca"] == "" {
			return meta, errors.New("no ca given")
		}
		meta.ca = authParams["ca"]

		if authParams["cert"] == "" {
			return meta, errors.New("no cert given")
		}
		meta.cert = authParams["cert"]

		if authParams["key"] == "" {
			return meta, errors.New("no key given")
		}
		meta.key = authParams["key"]
	}

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
		kafkaLog.V(1).Info(fmt.Sprintf("Group %s has a lag of %d for topic %s and partition %d\n", s.metadata.group, lag, s.metadata.topic, partition))

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

	if ok := metadata.authMode == kafkaAuthModeForSaslPlaintext || metadata.authMode == kafkaAuthModeForSaslSSL || metadata.authMode == kafkaAuthModeForSaslScramSha256 || metadata.authMode == kafkaAuthModeForSaslScramSha512; ok {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = metadata.username
		config.Net.SASL.Password = metadata.password
	}

	if metadata.authMode == kafkaAuthModeForSaslSSL {
		cert, err := tls.X509KeyPair([]byte(metadata.cert), []byte(metadata.key))
		if err != nil {
			return nil, nil, fmt.Errorf("error parse X509KeyPair: %s", err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(metadata.ca))

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		}

		config.Net.TLS.Enable = true
		config.Net.TLS.Config = tlsConfig
	}

	if metadata.authMode == kafkaAuthModeForSaslScramSha256 {
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA256} }
		config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA256)
	}

	if metadata.authMode == kafkaAuthModeForSaslScramSha512 {
		config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
		config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA512)
	}

	if metadata.authMode == kafkaAuthModeForSaslPlaintext {
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		config.Net.TLS.Enable = true
	}

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
		kafkaLog.Error(err, fmt.Sprintf("error finding latest offset for topic %s and partition %d\n", s.metadata.topic, partition))
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

// Close closes the kafka admin and client
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
func (s *kafkaScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
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
		totalLag += lag
	}

	kafkaLog.V(1).Info(fmt.Sprintf("Kafka scaler: Providing metrics based on totalLag %v, partitions %v, threshold %v", totalLag, len(partitions), s.metadata.lagThreshold))

	// don't scale out beyond the number of partitions
	if (totalLag / s.metadata.lagThreshold) > int64(len(partitions)) {
		totalLag = int64(len(partitions)) * s.metadata.lagThreshold
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(totalLag), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
