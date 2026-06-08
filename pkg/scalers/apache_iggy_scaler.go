/*
Copyright 2026 The KEDA Authors

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

package scalers

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/apache/iggy/foreign/go/client/tcp"
	iggcon "github.com/apache/iggy/foreign/go/contracts"
	ierror "github.com/apache/iggy/foreign/go/errors"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const apacheIggyMetricType = "External"

type apacheIggyScaler struct {
	metricType      v2.MetricTargetType
	metadata        *apacheIggyMetadata
	client          iggcon.Client
	logger          logr.Logger
	previousOffsets map[uint32]int64
	streamID        iggcon.Identifier
	topicID         iggcon.Identifier
	consumer        iggcon.Consumer
	// newClient builds (and authenticates) a fresh client. It is a field so the
	// scaler can rebuild the connection on failure and so tests can inject a mock.
	newClient func() (iggcon.Client, error)
}

type apacheIggyMetadata struct {
	ServerAddress                      string `keda:"name=serverAddress,          order=triggerMetadata;authParams;resolvedEnv"`
	StreamID                           string `keda:"name=streamId,               order=triggerMetadata"`
	TopicID                            string `keda:"name=topicId,                order=triggerMetadata"`
	ConsumerGroupID                    string `keda:"name=consumerGroupId,        order=triggerMetadata"`
	LagThreshold                       int64  `keda:"name=lagThreshold,           order=triggerMetadata, default=10"`
	ActivationLagThreshold             int64  `keda:"name=activationLagThreshold, order=triggerMetadata, default=0"`
	PartitionLimitation                []int  `keda:"name=partitionLimitation,      order=triggerMetadata, optional, range"`
	LimitToPartitionsWithLag           bool   `keda:"name=limitToPartitionsWithLag, order=triggerMetadata, optional"`
	ExcludePersistentLag               bool   `keda:"name=excludePersistentLag,     order=triggerMetadata, optional"`
	ScaleToZeroOnInvalidOffset         bool   `keda:"name=scaleToZeroOnInvalidOffset,         order=triggerMetadata, optional"`
	AllowIdleConsumers                 bool   `keda:"name=allowIdleConsumers,                 order=triggerMetadata, optional"`
	EnsureEvenDistributionOfPartitions bool   `keda:"name=ensureEvenDistributionOfPartitions, order=triggerMetadata, optional"`

	// Auth - username/password
	Username string `keda:"name=username, order=authParams;resolvedEnv, optional"`
	Password string `keda:"name=password, order=authParams;resolvedEnv, optional"`

	// Auth - Personal Access Token
	AccessToken string `keda:"name=accessToken, order=authParams;resolvedEnv, optional"`

	TriggerIndex int
}

func (m *apacheIggyMetadata) Validate() error {
	if m.LagThreshold <= 0 {
		return fmt.Errorf("lagThreshold must be a positive number")
	}
	if m.ActivationLagThreshold < 0 {
		return fmt.Errorf("activationLagThreshold must be a positive number or zero")
	}
	if m.AllowIdleConsumers && m.LimitToPartitionsWithLag {
		return fmt.Errorf("allowIdleConsumers and limitToPartitionsWithLag cannot be set simultaneously")
	}

	// Partition IDs are 1-indexed in Iggy; reject values that can never match a partition.
	for _, p := range m.PartitionLimitation {
		if p < 1 {
			return fmt.Errorf("partitionLimitation values must be >= 1, got %d", p)
		}
	}

	hasUserPass := m.Username != "" || m.Password != ""
	hasPAT := m.AccessToken != ""

	if hasUserPass && hasPAT {
		return fmt.Errorf("username/password and accessToken are mutually exclusive")
	}
	if !hasUserPass && !hasPAT {
		return fmt.Errorf("one of username/password or accessToken must be provided")
	}
	if hasUserPass && (m.Username == "" || m.Password == "") {
		return fmt.Errorf("both username and password must be provided together")
	}

	return nil
}

func parseApacheIggyMetadata(config *scalersconfig.ScalerConfig) (*apacheIggyMetadata, error) {
	meta := &apacheIggyMetadata{TriggerIndex: config.TriggerIndex}
	if err := config.TypedConfig(meta); err != nil {
		return nil, err
	}
	return meta, nil
}

// NewApacheIggyScaler creates a new Apache Iggy scaler.
func NewApacheIggyScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	s := &apacheIggyScaler{}

	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}
	s.metricType = metricType
	s.logger = InitializeLogger(config, "apache_iggy_scaler")

	meta, err := parseApacheIggyMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing apache iggy metadata: %w", err)
	}
	s.metadata = meta

	// Cache identifiers — these are immutable value types derived from constant metadata
	streamID, err := newIggyIdentifier(meta.StreamID)
	if err != nil {
		return nil, fmt.Errorf("error creating stream identifier: %w", err)
	}
	s.streamID = streamID

	topicID, err := newIggyIdentifier(meta.TopicID)
	if err != nil {
		return nil, fmt.Errorf("error creating topic identifier: %w", err)
	}
	s.topicID = topicID

	groupID, err := newIggyIdentifier(meta.ConsumerGroupID)
	if err != nil {
		return nil, fmt.Errorf("error creating consumer group identifier: %w", err)
	}
	s.consumer = iggcon.NewGroupConsumer(groupID)

	// Build directly on the raw TCP client rather than iggycli.NewIggyClient: the
	// latter spawns a per-client heartbeat goroutine that logs via the stdlib log
	// package, which KEDA cannot route through its own logger. The scaler manages
	// reconnection itself instead (see getTopic).
	s.newClient = func() (iggcon.Client, error) {
		client, err := tcp.NewIggyTcpClient(
			tcp.WithServerAddress(meta.ServerAddress),
		)
		if err != nil {
			return nil, fmt.Errorf("error creating iggy client: %w", err)
		}

		if meta.AccessToken != "" {
			_, err = client.LoginWithPersonalAccessToken(meta.AccessToken)
		} else {
			_, err = client.LoginUser(meta.Username, meta.Password)
		}
		if err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("error authenticating with iggy: %w", err)
		}
		return client, nil
	}

	if err := s.connect(); err != nil {
		return nil, err
	}
	s.previousOffsets = make(map[uint32]int64)
	return s, nil
}

// newIggyIdentifier builds an Iggy identifier from a metadata value, treating a
// fully-numeric value as a numeric resource ID and anything else as a resource
// name, matching the SDK's "unique ID or name" contract. A resource literally
// named with a pure-number string can therefore only be addressed by its ID.
func newIggyIdentifier(value string) (iggcon.Identifier, error) {
	if n, err := strconv.ParseUint(value, 10, 32); err == nil {
		return iggcon.NewIdentifier(uint32(n))
	}
	return iggcon.NewIdentifier(value)
}

// connect builds and stores a fresh, authenticated client.
func (s *apacheIggyScaler) connect() error {
	client, err := s.newClient()
	if err != nil {
		return err
	}
	s.client = client
	return nil
}

// reconnect tears down the current client and establishes a new one. The Iggy
// SDK's TCP client does not reconnect on its own once a connection drops, so the
// scaler rebuilds it explicitly.
func (s *apacheIggyScaler) reconnect() error {
	if s.client != nil {
		_ = s.client.Close()
		s.client = nil
	}
	return s.connect()
}

// isIggyServerError reports whether err is a typed Iggy server response. If it
// is, the server replied (the connection is healthy); if not, err indicates a
// transport/connection problem worth reconnecting for.
func isIggyServerError(err error) bool {
	var ie ierror.IggyError
	return errors.As(err, &ie)
}

func (s *apacheIggyScaler) GetMetricSpecForScaling(_ context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("iggy-%s-%s-%s", s.metadata.StreamID, s.metadata.TopicID, s.metadata.ConsumerGroupID))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.LagThreshold),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: apacheIggyMetricType}
	return []v2.MetricSpec{metricSpec}
}

func (s *apacheIggyScaler) GetMetricsAndActivity(_ context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	topic, err := s.getTopic()
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	partitionCount := topic.PartitionsCount
	partitionLags := make([]int64, 0, partitionCount)
	partitionLagsWithPersistent := make([]int64, 0, partitionCount)

	for i := uint32(1); i <= partitionCount; i++ {
		// Skip partitions not in the limitation list (if specified)
		if len(s.metadata.PartitionLimitation) > 0 && !kedautil.Contains(s.metadata.PartitionLimitation, int(i)) {
			continue
		}

		partitionID := i
		offset, err := s.client.GetConsumerOffset(s.consumer, s.streamID, s.topicID, &partitionID)
		// A typed ConsumerOffsetNotFound means the consumer group simply has no
		// committed offset for this partition yet (expected for fresh consumers); the
		// SDK likewise returns a nil offset with a nil error when the server responds
		// with an empty payload. Both cases mean "no committed offset". Any other error
		// (network, auth, server) is a genuine failure and must not be silently
		// converted into a lag metric, so surface it instead.
		var offsetNotFound ierror.ConsumerOffsetNotFound
		if err != nil && !errors.As(err, &offsetNotFound) {
			return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error fetching consumer offset for partition %d: %w", partitionID, err)
		}
		if err != nil || offset == nil {
			s.logger.V(1).Info("No committed offset for partition, treating as no committed offset",
				"partition", i)
			retVal := int64(1)
			if s.metadata.ScaleToZeroOnInvalidOffset {
				retVal = iggyPartitionHighWatermark(topic, partitionID)
			}
			partitionLags = append(partitionLags, retVal)
			partitionLagsWithPersistent = append(partitionLagsWithPersistent, retVal)
			continue
		}

		fullLag := max(int64(offset.CurrentOffset)-int64(offset.StoredOffset), 0)
		lag := fullLag

		if s.metadata.ExcludePersistentLag {
			storedOffset := int64(offset.StoredOffset)
			previousOffset, found := s.previousOffsets[partitionID]
			switch {
			case !found:
				s.previousOffsets[partitionID] = storedOffset
			case previousOffset == storedOffset:
				lag = 0
			default:
				s.previousOffsets[partitionID] = storedOffset
			}
		}

		partitionLags = append(partitionLags, lag)
		partitionLagsWithPersistent = append(partitionLagsWithPersistent, fullLag)
	}

	totalLag, totalLagWithPersistent := calculateIggyLag(partitionLags, partitionLagsWithPersistent, s.metadata.LagThreshold, s.metadata.AllowIdleConsumers, s.metadata.LimitToPartitionsWithLag, s.metadata.EnsureEvenDistributionOfPartitions)
	isActive := totalLagWithPersistent > s.metadata.ActivationLagThreshold

	s.logger.V(1).Info("Found iggy consumer group lag",
		"stream", s.metadata.StreamID,
		"topic", s.metadata.TopicID,
		"consumerGroup", s.metadata.ConsumerGroupID,
		"totalLag", totalLag,
		"totalLagWithPersistent", totalLagWithPersistent,
		"partitionCount", partitionCount)

	metric := GenerateMetricInMili(metricName, float64(totalLag))
	return []external_metrics.ExternalMetricValue{metric}, isActive, nil
}

func (s *apacheIggyScaler) Close(_ context.Context) error {
	if s.client != nil {
		if err := s.client.LogoutUser(); err != nil {
			s.logger.V(1).Info("Error logging out from iggy", "error", err)
		}
		// Close releases the TCP connection.
		if err := s.client.Close(); err != nil {
			s.logger.V(1).Info("Error closing iggy client", "error", err)
		}
	}
	return nil
}

// getTopic fetches the topic, transparently rebuilding the client once if the
// failure looks like a dropped connection. The Iggy SDK's TCP client does not
// reconnect on its own, so the scaler owns reconnection; typed server errors
// (e.g. topic not found) are returned as-is without a reconnect attempt.
func (s *apacheIggyScaler) getTopic() (*iggcon.TopicDetails, error) {
	if s.client == nil {
		if err := s.connect(); err != nil {
			return nil, err
		}
	}

	topic, err := s.client.GetTopic(s.streamID, s.topicID)
	if err != nil && !isIggyServerError(err) {
		s.logger.V(1).Info("iggy request failed, reconnecting", "error", err)
		if rcErr := s.reconnect(); rcErr != nil {
			return nil, fmt.Errorf("error reconnecting to iggy: %w", rcErr)
		}
		topic, err = s.client.GetTopic(s.streamID, s.topicID)
	}
	if err != nil {
		return nil, fmt.Errorf("error getting topic: %w", err)
	}
	return topic, nil
}

// iggyPartitionHighWatermark returns the high watermark (latest offset) for a
// partition from the topic details. Returns 0 if the partition is not found.
func iggyPartitionHighWatermark(topic *iggcon.TopicDetails, partitionID uint32) int64 {
	for _, p := range topic.Partitions {
		if p.Id == partitionID {
			return int64(p.CurrentOffset)
		}
	}
	return 0
}

// calculateIggyLag computes total lag from per-partition lags.
// partitionLags may exclude persistent lag; partitionLagsWithPersistent always includes all lag.
// Returns (totalLag, totalLagWithPersistent) where totalLag is capped for scaling.
func calculateIggyLag(partitionLags, partitionLagsWithPersistent []int64, lagThreshold int64, allowIdleConsumers, limitToPartitionsWithLag, ensureEvenDistribution bool) (int64, int64) {
	var totalLag int64
	var partitionsWithLag int64
	for _, lag := range partitionLags {
		if lag > 0 {
			totalLag += lag
			partitionsWithLag++
		}
	}

	var totalLagWithPersistent int64
	for _, lag := range partitionLagsWithPersistent {
		if lag > 0 {
			totalLagWithPersistent += lag
		}
	}

	totalPartitions := int64(len(partitionLags))

	if !allowIdleConsumers || limitToPartitionsWithLag || ensureEvenDistribution {
		upperBound := totalPartitions

		if ensureEvenDistribution {
			nextFactor := getNextFactorThatBalancesConsumersToTopicPartitions(totalLag, totalPartitions, lagThreshold)
			totalLag = nextFactor * lagThreshold
		}

		if limitToPartitionsWithLag {
			upperBound = partitionsWithLag
		}

		if lagThreshold > 0 && upperBound > 0 && (totalLag/lagThreshold) > upperBound {
			totalLag = upperBound * lagThreshold
		}
	}

	return totalLag, totalLagWithPersistent
}
