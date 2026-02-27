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
	"fmt"

	iggcon "github.com/apache/iggy/foreign/go/contracts"
	"github.com/apache/iggy/foreign/go/iggycli"
	"github.com/apache/iggy/foreign/go/tcp"
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
	client          iggycli.Client
	logger          logr.Logger
	previousOffsets map[uint32]int64
	streamID        iggcon.Identifier
	topicID         iggcon.Identifier
	consumer        iggcon.Consumer
	cancel          context.CancelFunc
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
	streamID, err := iggcon.NewIdentifier(meta.StreamID)
	if err != nil {
		return nil, fmt.Errorf("error creating stream identifier: %w", err)
	}
	s.streamID = streamID

	topicID, err := iggcon.NewIdentifier(meta.TopicID)
	if err != nil {
		return nil, fmt.Errorf("error creating topic identifier: %w", err)
	}
	s.topicID = topicID

	groupID, err := iggcon.NewIdentifier(meta.ConsumerGroupID)
	if err != nil {
		return nil, fmt.Errorf("error creating consumer group identifier: %w", err)
	}
	s.consumer = iggcon.NewGroupConsumer(groupID)

	// Use a cancellable context so Close() can stop the SDK's heartbeat goroutine
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	client, err := iggycli.NewIggyClient(
		iggycli.WithTcp(
			tcp.WithServerAddress(meta.ServerAddress),
			tcp.WithContext(ctx),
		),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("error creating iggy client: %w", err)
	}

	if meta.AccessToken != "" {
		_, err = client.LoginWithPersonalAccessToken(meta.AccessToken)
	} else {
		_, err = client.LoginUser(meta.Username, meta.Password)
	}
	if err != nil {
		cancel()
		return nil, fmt.Errorf("error authenticating with iggy: %w", err)
	}

	s.client = client
	s.previousOffsets = make(map[uint32]int64)
	return s, nil
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
	topic, err := s.client.GetTopic(s.streamID, s.topicID)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting topic: %w", err)
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
		if err != nil {
			// All GetConsumerOffset errors are treated as "no committed offset" because
			// the Iggy SDK does not expose typed errors to distinguish missing offsets
			// from transient failures (network, auth, etc.) in this path.
			s.logger.V(1).Info("Error fetching consumer offset, treating as no committed offset",
				"partition", i, "error", err)
			retVal := int64(1)
			if s.metadata.ScaleToZeroOnInvalidOffset {
				retVal = 0
			}
			partitionLags = append(partitionLags, retVal)
			partitionLagsWithPersistent = append(partitionLagsWithPersistent, retVal)
			continue
		}

		// The Iggy SDK returns nil offset with nil error when the server
		// responds with an empty payload (e.g., no committed offset yet).
		if offset == nil {
			s.logger.V(1).Info("Nil offset returned for partition, treating as no committed offset",
				"partition", i)
			retVal := int64(1)
			if s.metadata.ScaleToZeroOnInvalidOffset {
				retVal = 0
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
	}
	// Cancel the context to stop the SDK's heartbeat goroutine and release the TCP connection
	if s.cancel != nil {
		s.cancel()
	}
	return nil
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
