package scalers

import (
	"errors"
	"testing"

	iggcon "github.com/apache/iggy/foreign/go/contracts"
	"github.com/go-logr/logr"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

// mockIggyClient implements the subset of iggycli.Client used by apacheIggyScaler.
type mockIggyClient struct {
	topic   *iggcon.TopicDetails
	offsets map[uint32]*iggcon.ConsumerOffsetInfo
	errors  map[uint32]error
}

func (m *mockIggyClient) GetTopic(_, _ iggcon.Identifier) (*iggcon.TopicDetails, error) {
	return m.topic, nil
}

func (m *mockIggyClient) GetConsumerOffset(_ iggcon.Consumer, _, _ iggcon.Identifier, partitionID *uint32) (*iggcon.ConsumerOffsetInfo, error) {
	if err, ok := m.errors[*partitionID]; ok {
		return nil, err
	}
	return m.offsets[*partitionID], nil
}

// Stub methods to satisfy iggycli.Client interface.
func (m *mockIggyClient) GetStream(_ iggcon.Identifier) (*iggcon.StreamDetails, error) {
	return nil, nil
}
func (m *mockIggyClient) GetStreams() ([]iggcon.Stream, error) { return nil, nil }
func (m *mockIggyClient) CreateStream(_ string) (*iggcon.StreamDetails, error) {
	return nil, nil
}
func (m *mockIggyClient) UpdateStream(_ iggcon.Identifier, _ string) error { return nil }
func (m *mockIggyClient) DeleteStream(_ iggcon.Identifier) error           { return nil }
func (m *mockIggyClient) GetTopics(_ iggcon.Identifier) ([]iggcon.Topic, error) {
	return nil, nil
}
func (m *mockIggyClient) CreateTopic(_ iggcon.Identifier, _ string, _ uint32, _ iggcon.CompressionAlgorithm, _ iggcon.Duration, _ uint64, _ *uint8) (*iggcon.TopicDetails, error) {
	return nil, nil
}
func (m *mockIggyClient) UpdateTopic(_ iggcon.Identifier, _ iggcon.Identifier, _ string, _ iggcon.CompressionAlgorithm, _ iggcon.Duration, _ uint64, _ *uint8) error {
	return nil
}
func (m *mockIggyClient) DeleteTopic(_, _ iggcon.Identifier) error { return nil }
func (m *mockIggyClient) SendMessages(_ iggcon.Identifier, _ iggcon.Identifier, _ iggcon.Partitioning, _ []iggcon.IggyMessage) error {
	return nil
}
func (m *mockIggyClient) PollMessages(_ iggcon.Identifier, _ iggcon.Identifier, _ iggcon.Consumer, _ iggcon.PollingStrategy, _ uint32, _ bool, _ *uint32) (*iggcon.PolledMessage, error) {
	return nil, nil
}
func (m *mockIggyClient) StoreConsumerOffset(_ iggcon.Consumer, _ iggcon.Identifier, _ iggcon.Identifier, _ uint64, _ *uint32) error {
	return nil
}
func (m *mockIggyClient) GetConsumerGroups(_ iggcon.Identifier, _ iggcon.Identifier) ([]iggcon.ConsumerGroup, error) {
	return nil, nil
}
func (m *mockIggyClient) DeleteConsumerOffset(_ iggcon.Consumer, _ iggcon.Identifier, _ iggcon.Identifier, _ *uint32) error {
	return nil
}
func (m *mockIggyClient) GetConsumerGroup(_, _, _ iggcon.Identifier) (*iggcon.ConsumerGroupDetails, error) {
	return nil, nil
}
func (m *mockIggyClient) CreateConsumerGroup(_, _ iggcon.Identifier, _ string) (*iggcon.ConsumerGroupDetails, error) {
	return nil, nil
}
func (m *mockIggyClient) DeleteConsumerGroup(_, _, _ iggcon.Identifier) error { return nil }
func (m *mockIggyClient) JoinConsumerGroup(_, _, _ iggcon.Identifier) error   { return nil }
func (m *mockIggyClient) LeaveConsumerGroup(_, _, _ iggcon.Identifier) error  { return nil }
func (m *mockIggyClient) CreatePartitions(_, _ iggcon.Identifier, _ uint32) error {
	return nil
}
func (m *mockIggyClient) DeletePartitions(_, _ iggcon.Identifier, _ uint32) error {
	return nil
}
func (m *mockIggyClient) GetUser(_ iggcon.Identifier) (*iggcon.UserInfoDetails, error) {
	return nil, nil
}
func (m *mockIggyClient) GetUsers() ([]iggcon.UserInfo, error) { return nil, nil }
func (m *mockIggyClient) CreateUser(_ string, _ string, _ iggcon.UserStatus, _ *iggcon.Permissions) (*iggcon.UserInfoDetails, error) {
	return nil, nil
}
func (m *mockIggyClient) UpdateUser(_ iggcon.Identifier, _ *string, _ *iggcon.UserStatus) error {
	return nil
}
func (m *mockIggyClient) UpdatePermissions(_ iggcon.Identifier, _ *iggcon.Permissions) error {
	return nil
}
func (m *mockIggyClient) ChangePassword(_ iggcon.Identifier, _, _ string) error { return nil }
func (m *mockIggyClient) DeleteUser(_ iggcon.Identifier) error                  { return nil }
func (m *mockIggyClient) CreatePersonalAccessToken(_ string, _ uint32) (*iggcon.RawPersonalAccessToken, error) {
	return nil, nil
}
func (m *mockIggyClient) DeletePersonalAccessToken(_ string) error { return nil }
func (m *mockIggyClient) GetPersonalAccessTokens() ([]iggcon.PersonalAccessTokenInfo, error) {
	return nil, nil
}
func (m *mockIggyClient) LoginWithPersonalAccessToken(_ string) (*iggcon.IdentityInfo, error) {
	return nil, nil
}
func (m *mockIggyClient) LoginUser(_, _ string) (*iggcon.IdentityInfo, error) { return nil, nil }
func (m *mockIggyClient) LogoutUser() error                                   { return nil }
func (m *mockIggyClient) GetStats() (*iggcon.Stats, error)                    { return nil, nil }
func (m *mockIggyClient) Ping() error                                         { return nil }
func (m *mockIggyClient) GetClients() ([]iggcon.ClientInfo, error)            { return nil, nil }
func (m *mockIggyClient) GetClient(_ uint32) (*iggcon.ClientInfoDetails, error) {
	return nil, nil
}

type parseApacheIggyMetadataTestData struct {
	metadata   map[string]string
	authParams map[string]string
	isError    bool
}

var validApacheIggyMetadata = map[string]string{
	"serverAddress":   "localhost:8090",
	"streamId":        "test-stream",
	"topicId":         "test-topic",
	"consumerGroupId": "test-group",
}

var parseApacheIggyMetadataTestDataset = []parseApacheIggyMetadataTestData{
	// success - username/password auth
	{validApacheIggyMetadata, map[string]string{"username": "admin", "password": "admin"}, false},
	// success - PAT auth
	{validApacheIggyMetadata, map[string]string{"accessToken": "my-token"}, false},
	// success - custom lagThreshold
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "lagThreshold": "100",
	}, map[string]string{"accessToken": "tok"}, false},
	// success - custom activationLagThreshold
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "activationLagThreshold": "5",
	}, map[string]string{"accessToken": "tok"}, false},
	// failure - missing serverAddress
	{map[string]string{"streamId": "s", "topicId": "t", "consumerGroupId": "g"},
		map[string]string{"accessToken": "tok"}, true},
	// failure - missing streamId
	{map[string]string{"serverAddress": "localhost:8090", "topicId": "t", "consumerGroupId": "g"},
		map[string]string{"accessToken": "tok"}, true},
	// failure - missing topicId
	{map[string]string{"serverAddress": "localhost:8090", "streamId": "s", "consumerGroupId": "g"},
		map[string]string{"accessToken": "tok"}, true},
	// failure - missing consumerGroupId
	{map[string]string{"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t"},
		map[string]string{"accessToken": "tok"}, true},
	// failure - no auth provided
	{validApacheIggyMetadata, map[string]string{}, true},
	// failure - both auth methods provided
	{validApacheIggyMetadata, map[string]string{
		"username": "admin", "password": "admin", "accessToken": "tok",
	}, true},
	// failure - username without password
	{validApacheIggyMetadata, map[string]string{"username": "admin"}, true},
	// failure - password without username
	{validApacheIggyMetadata, map[string]string{"password": "admin"}, true},
	// failure - lagThreshold is 0
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "lagThreshold": "0",
	}, map[string]string{"accessToken": "tok"}, true},
	// failure - lagThreshold is negative
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "lagThreshold": "-1",
	}, map[string]string{"accessToken": "tok"}, true},
	// failure - activationLagThreshold is negative
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "activationLagThreshold": "-1",
	}, map[string]string{"accessToken": "tok"}, true},
	// success - partitionLimitation with list
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "partitionLimitation": "1,2,3",
	}, map[string]string{"accessToken": "tok"}, false},
	// success - partitionLimitation with range
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "partitionLimitation": "1-4,8,10-12",
	}, map[string]string{"accessToken": "tok"}, false},
	// success - offsetResetPolicy=earliest
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "offsetResetPolicy": "earliest",
	}, map[string]string{"accessToken": "tok"}, false},
	// success - offsetResetPolicy=latest
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "offsetResetPolicy": "latest",
	}, map[string]string{"accessToken": "tok"}, false},
	// success - scaleToZeroOnInvalidOffset
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "scaleToZeroOnInvalidOffset": "true",
	}, map[string]string{"accessToken": "tok"}, false},
	// failure - allowIdleConsumers and limitToPartitionsWithLag conflict
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "allowIdleConsumers": "true", "limitToPartitionsWithLag": "true",
	}, map[string]string{"accessToken": "tok"}, true},
	// failure - invalid offsetResetPolicy
	{map[string]string{
		"serverAddress": "localhost:8090", "streamId": "s", "topicId": "t",
		"consumerGroupId": "g", "offsetResetPolicy": "invalid",
	}, map[string]string{"accessToken": "tok"}, true},
}

func TestApacheIggyParseMetadata(t *testing.T) {
	for idx, testData := range parseApacheIggyMetadataTestDataset {
		config := &scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadata,
			AuthParams:      testData.authParams,
		}
		_, err := parseApacheIggyMetadata(config)
		if err != nil && !testData.isError {
			t.Errorf("test index %d: expected success but got error: %v", idx, err)
		}
		if err == nil && testData.isError {
			t.Errorf("test index %d: expected error but got success", idx)
		}
	}
}

func TestApacheIggyDefaultValues(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: validApacheIggyMetadata,
		AuthParams:      map[string]string{"accessToken": "tok"},
	}
	meta, err := parseApacheIggyMetadata(config)
	if err != nil {
		t.Fatalf("expected success but got error: %v", err)
	}
	if meta.LagThreshold != 10 {
		t.Errorf("expected default lagThreshold 10, got %d", meta.LagThreshold)
	}
	if meta.ActivationLagThreshold != 0 {
		t.Errorf("expected default activationLagThreshold 0, got %d", meta.ActivationLagThreshold)
	}
	if meta.OffsetResetPolicy != latest {
		t.Errorf("expected default offsetResetPolicy 'latest', got %q", meta.OffsetResetPolicy)
	}
	if meta.ScaleToZeroOnInvalidOffset {
		t.Errorf("expected default scaleToZeroOnInvalidOffset false, got true")
	}
}

func TestApacheIggyPartitionLimitation(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"serverAddress":       "localhost:8090",
			"streamId":            "s",
			"topicId":             "t",
			"consumerGroupId":     "g",
			"partitionLimitation": "1,3,5-7",
		},
		AuthParams: map[string]string{"accessToken": "tok"},
	}
	meta, err := parseApacheIggyMetadata(config)
	if err != nil {
		t.Fatalf("expected success but got error: %v", err)
	}
	expected := []int{1, 3, 5, 6, 7}
	if len(meta.PartitionLimitation) != len(expected) {
		t.Fatalf("expected %d partitions, got %d", len(expected), len(meta.PartitionLimitation))
	}
	for i, v := range expected {
		if meta.PartitionLimitation[i] != v {
			t.Errorf("expected partition %d at index %d, got %d", v, i, meta.PartitionLimitation[i])
		}
	}
}

type apacheIggyMetricIdentifier struct {
	metadataTestData *parseApacheIggyMetadataTestData
	triggerIndex     int
	name             string
}

var apacheIggyMetricIdentifiers = []apacheIggyMetricIdentifier{
	{&parseApacheIggyMetadataTestDataset[0], 0, "s0-iggy-test-stream-test-topic-test-group"},
	{&parseApacheIggyMetadataTestDataset[0], 1, "s1-iggy-test-stream-test-topic-test-group"},
	{&parseApacheIggyMetadataTestDataset[1], 0, "s0-iggy-test-stream-test-topic-test-group"},
}

func TestApacheIggyGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range apacheIggyMetricIdentifiers {
		config := &scalersconfig.ScalerConfig{
			TriggerMetadata: testData.metadataTestData.metadata,
			AuthParams:      testData.metadataTestData.authParams,
			TriggerIndex:    testData.triggerIndex,
		}
		meta, err := parseApacheIggyMetadata(config)
		if err != nil {
			t.Fatal("could not parse metadata:", err)
		}

		mockScaler := apacheIggyScaler{
			metadata:   meta,
			metricType: v2.AverageValueMetricType,
		}

		metricSpec := mockScaler.GetMetricSpecForScaling(t.Context())
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Errorf("expected %s, got %s", testData.name, metricName)
		}
	}
}

type apacheIggyLagTestData struct {
	description                 string
	partitionLags               []int64
	partitionLagsWithPersistent []int64 // nil means same as partitionLags
	lagThreshold                int64
	activationLagThreshold      int64
	allowIdleConsumers          bool
	limitToPartitionsWithLag    bool
	ensureEvenDistribution      bool
	expectedMetric              int64
	expectedLagWithPersistent   int64
	expectedActive              bool
}

var apacheIggyLagTestDataset = []apacheIggyLagTestData{
	{
		description:               "no lag, inactive",
		partitionLags:             []int64{0, 0, 0},
		lagThreshold:              10,
		activationLagThreshold:    0,
		expectedMetric:            0,
		expectedLagWithPersistent: 0,
		expectedActive:            false,
	},
	{
		description:               "some lag, active",
		partitionLags:             []int64{5, 3, 2},
		lagThreshold:              10,
		activationLagThreshold:    0,
		expectedMetric:            10,
		expectedLagWithPersistent: 10,
		expectedActive:            true,
	},
	{
		description:               "lag below activation threshold, inactive",
		partitionLags:             []int64{2, 1, 0},
		lagThreshold:              10,
		activationLagThreshold:    5,
		expectedMetric:            3,
		expectedLagWithPersistent: 3,
		expectedActive:            false,
	},
	{
		description:               "lag exceeds partition cap",
		partitionLags:             []int64{50, 50, 50},
		lagThreshold:              10,
		activationLagThreshold:    0,
		expectedMetric:            30,
		expectedLagWithPersistent: 150,
		expectedActive:            true,
	},
	{
		description:               "single partition with lag",
		partitionLags:             []int64{7},
		lagThreshold:              10,
		activationLagThreshold:    0,
		expectedMetric:            7,
		expectedLagWithPersistent: 7,
		expectedActive:            true,
	},
	{
		description:               "zero partitions",
		partitionLags:             []int64{},
		lagThreshold:              10,
		activationLagThreshold:    0,
		expectedMetric:            0,
		expectedLagWithPersistent: 0,
		expectedActive:            false,
	},
	{
		description:               "limitToPartitionsWithLag caps to partitions with lag",
		partitionLags:             []int64{50, 0, 50, 0, 0},
		lagThreshold:              10,
		activationLagThreshold:    0,
		limitToPartitionsWithLag:  true,
		expectedMetric:            20, // 2 partitions with lag * 10 threshold
		expectedLagWithPersistent: 100,
		expectedActive:            true,
	},
	{
		description:               "limitToPartitionsWithLag no lag",
		partitionLags:             []int64{0, 0, 0},
		lagThreshold:              10,
		activationLagThreshold:    0,
		limitToPartitionsWithLag:  true,
		expectedMetric:            0,
		expectedLagWithPersistent: 0,
		expectedActive:            false,
	},
	{
		description:                 "excludePersistentLag - persistent partition excluded from scaling but counts for activation",
		partitionLags:               []int64{0, 5, 3},  // lag=0 for stuck partition
		partitionLagsWithPersistent: []int64{10, 5, 3}, // full lag includes stuck partition
		lagThreshold:                10,
		activationLagThreshold:      0,
		expectedMetric:              8,
		expectedLagWithPersistent:   18,
		expectedActive:              true,
	},
	{
		description:                 "excludePersistentLag - all partitions persistent, scale to zero but still active",
		partitionLags:               []int64{0, 0, 0},
		partitionLagsWithPersistent: []int64{10, 20, 30},
		lagThreshold:                10,
		activationLagThreshold:      0,
		expectedMetric:              0,
		expectedLagWithPersistent:   60,
		expectedActive:              true,
	},
	{
		description:               "allowIdleConsumers removes partition cap",
		partitionLags:             []int64{50, 50, 50},
		lagThreshold:              10,
		activationLagThreshold:    0,
		allowIdleConsumers:        true,
		expectedMetric:            150, // no cap applied
		expectedLagWithPersistent: 150,
		expectedActive:            true,
	},
	{
		description:               "ensureEvenDistribution rounds to factor of partitions",
		partitionLags:             []int64{15, 15, 15, 15, 15, 15}, // 6 partitions, total=90
		lagThreshold:              10,
		activationLagThreshold:    0,
		ensureEvenDistribution:    true,
		expectedMetric:            60, // 90/10=9 replicas, but nearest factor of 6 is 6, so 6*10=60
		expectedLagWithPersistent: 90,
		expectedActive:            true,
	},
	{
		description:               "ensureEvenDistribution with 12 partitions",
		partitionLags:             []int64{5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}, // 12 partitions, total=60
		lagThreshold:              10,
		activationLagThreshold:    0,
		ensureEvenDistribution:    true,
		expectedMetric:            60, // 60/10=6 replicas, 6 is a factor of 12, so 6*10=60
		expectedLagWithPersistent: 60,
		expectedActive:            true,
	},
}

func TestApacheIggyCalculateLag(t *testing.T) {
	for _, testData := range apacheIggyLagTestDataset {
		t.Run(testData.description, func(t *testing.T) {
			lagsWithPersistent := testData.partitionLagsWithPersistent
			if lagsWithPersistent == nil {
				lagsWithPersistent = testData.partitionLags
			}
			metric, lagWithPersistent := calculateIggyLag(
				testData.partitionLags,
				lagsWithPersistent,
				testData.lagThreshold,
				testData.allowIdleConsumers,
				testData.limitToPartitionsWithLag,
				testData.ensureEvenDistribution,
			)
			if metric != testData.expectedMetric {
				t.Errorf("expected metric %d, got %d", testData.expectedMetric, metric)
			}
			if lagWithPersistent != testData.expectedLagWithPersistent {
				t.Errorf("expected lagWithPersistent %d, got %d", testData.expectedLagWithPersistent, lagWithPersistent)
			}
			isActive := lagWithPersistent > testData.activationLagThreshold
			if isActive != testData.expectedActive {
				t.Errorf("expected active %v, got %v", testData.expectedActive, isActive)
			}
		})
	}
}

func newTestIggyScaler(client *mockIggyClient, meta *apacheIggyMetadata) *apacheIggyScaler {
	streamID, _ := iggcon.NewIdentifier(meta.StreamID)
	topicID, _ := iggcon.NewIdentifier(meta.TopicID)
	groupID, _ := iggcon.NewIdentifier(meta.ConsumerGroupID)
	return &apacheIggyScaler{
		metricType:      v2.AverageValueMetricType,
		metadata:        meta,
		client:          client,
		logger:          logr.Discard(),
		previousOffsets: make(map[uint32]int64),
		streamID:        streamID,
		topicID:         topicID,
		consumer:        iggcon.NewGroupConsumer(groupID),
	}
}

func TestApacheIggyGetMetricsAndActivity_NilOffset(t *testing.T) {
	client := &mockIggyClient{
		topic: &iggcon.TopicDetails{
			Topic: iggcon.Topic{PartitionsCount: 2},
		},
		offsets: map[uint32]*iggcon.ConsumerOffsetInfo{
			1: nil, // nil offset, nil error
			2: {PartitionId: 2, CurrentOffset: 10, StoredOffset: 5},
		},
	}
	meta := &apacheIggyMetadata{
		StreamID:        "test-stream",
		TopicID:         "test-topic",
		ConsumerGroupID: "test-group",
		LagThreshold:    10,
	}
	scaler := newTestIggyScaler(client, meta)

	metrics, isActive, err := scaler.GetMetricsAndActivity(t.Context(), "s0-iggy-test-stream-test-topic-test-group")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !isActive {
		t.Error("expected active, got inactive")
	}
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	// partition 1: nil offset → lag=1, partition 2: lag=max(10-5,0)=5, total=6
	expectedLag := int64(6)
	gotLag := metrics[0].Value.MilliValue() / 1000
	if gotLag != expectedLag {
		t.Errorf("expected lag %d, got %d", expectedLag, gotLag)
	}
}

func TestApacheIggyGetMetricsAndActivity_NilOffsetScaleToZero(t *testing.T) {
	client := &mockIggyClient{
		topic: &iggcon.TopicDetails{
			Topic: iggcon.Topic{PartitionsCount: 1},
		},
		offsets: map[uint32]*iggcon.ConsumerOffsetInfo{
			1: nil,
		},
	}
	meta := &apacheIggyMetadata{
		StreamID:                   "test-stream",
		TopicID:                    "test-topic",
		ConsumerGroupID:            "test-group",
		LagThreshold:               10,
		ScaleToZeroOnInvalidOffset: true,
	}
	scaler := newTestIggyScaler(client, meta)

	metrics, isActive, err := scaler.GetMetricsAndActivity(t.Context(), "s0-iggy-test-stream-test-topic-test-group")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if isActive {
		t.Error("expected inactive, got active")
	}
	gotLag := metrics[0].Value.MilliValue() / 1000
	if gotLag != 0 {
		t.Errorf("expected lag 0, got %d", gotLag)
	}
}

func TestApacheIggyGetMetricsAndActivity_AllNilOffsets(t *testing.T) {
	client := &mockIggyClient{
		topic: &iggcon.TopicDetails{
			Topic: iggcon.Topic{PartitionsCount: 3},
		},
		offsets: map[uint32]*iggcon.ConsumerOffsetInfo{
			1: nil,
			2: nil,
			3: nil,
		},
	}
	meta := &apacheIggyMetadata{
		StreamID:        "test-stream",
		TopicID:         "test-topic",
		ConsumerGroupID: "test-group",
		LagThreshold:    10,
	}
	scaler := newTestIggyScaler(client, meta)

	metrics, isActive, err := scaler.GetMetricsAndActivity(t.Context(), "s0-iggy-test-stream-test-topic-test-group")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !isActive {
		t.Error("expected active, got inactive")
	}
	// 3 partitions each with lag=1 → total=3
	gotLag := metrics[0].Value.MilliValue() / 1000
	if gotLag != 3 {
		t.Errorf("expected lag 3, got %d", gotLag)
	}
}

func TestApacheIggyGetMetricsAndActivity_ErrorOffset(t *testing.T) {
	client := &mockIggyClient{
		topic: &iggcon.TopicDetails{
			Topic: iggcon.Topic{PartitionsCount: 2},
		},
		offsets: map[uint32]*iggcon.ConsumerOffsetInfo{
			2: {PartitionId: 2, CurrentOffset: 20, StoredOffset: 10},
		},
		errors: map[uint32]error{
			1: errors.New("connection refused"),
		},
	}
	meta := &apacheIggyMetadata{
		StreamID:        "test-stream",
		TopicID:         "test-topic",
		ConsumerGroupID: "test-group",
		LagThreshold:    10,
	}
	scaler := newTestIggyScaler(client, meta)

	metrics, isActive, err := scaler.GetMetricsAndActivity(t.Context(), "s0-iggy-test-stream-test-topic-test-group")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !isActive {
		t.Error("expected active, got inactive")
	}
	// partition 1: error → lag=1, partition 2: lag=max(20-10,0)=10, total=11
	gotLag := metrics[0].Value.MilliValue() / 1000
	if gotLag != 11 {
		t.Errorf("expected lag 11, got %d", gotLag)
	}
}

func TestApacheIggyGetMetricsAndActivity_NormalLag(t *testing.T) {
	client := &mockIggyClient{
		topic: &iggcon.TopicDetails{
			Topic: iggcon.Topic{PartitionsCount: 3},
		},
		offsets: map[uint32]*iggcon.ConsumerOffsetInfo{
			1: {PartitionId: 1, CurrentOffset: 100, StoredOffset: 95},
			2: {PartitionId: 2, CurrentOffset: 100, StoredOffset: 100},
			3: {PartitionId: 3, CurrentOffset: 100, StoredOffset: 90},
		},
	}
	meta := &apacheIggyMetadata{
		StreamID:        "test-stream",
		TopicID:         "test-topic",
		ConsumerGroupID: "test-group",
		LagThreshold:    10,
	}
	scaler := newTestIggyScaler(client, meta)

	metrics, isActive, err := scaler.GetMetricsAndActivity(t.Context(), "s0-iggy-test-stream-test-topic-test-group")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !isActive {
		t.Error("expected active, got inactive")
	}
	// partition 1: lag=5, partition 2: lag=0, partition 3: lag=10, total=15
	gotLag := metrics[0].Value.MilliValue() / 1000
	if gotLag != 15 {
		t.Errorf("expected lag 15, got %d", gotLag)
	}
}

func TestApacheIggyGetMetricsAndActivity_ExcludePersistentLag(t *testing.T) {
	client := &mockIggyClient{
		topic: &iggcon.TopicDetails{
			Topic: iggcon.Topic{PartitionsCount: 2},
		},
		offsets: map[uint32]*iggcon.ConsumerOffsetInfo{
			1: {PartitionId: 1, CurrentOffset: 100, StoredOffset: 50},
			2: {PartitionId: 2, CurrentOffset: 100, StoredOffset: 90},
		},
	}
	meta := &apacheIggyMetadata{
		StreamID:             "test-stream",
		TopicID:              "test-topic",
		ConsumerGroupID:      "test-group",
		LagThreshold:         10,
		ExcludePersistentLag: true,
	}
	scaler := newTestIggyScaler(client, meta)

	// First call: offsets are recorded, no previous to compare
	metrics1, _, err := scaler.GetMetricsAndActivity(t.Context(), "s0-iggy-test-stream-test-topic-test-group")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// partition 1: lag=50, partition 2: lag=10, total=20 (capped to 2 partitions * 10)
	gotLag1 := metrics1[0].Value.MilliValue() / 1000
	if gotLag1 != 20 {
		t.Errorf("first call: expected lag 20, got %d", gotLag1)
	}

	// Second call with same offsets: persistent lag excluded
	metrics2, isActive, err := scaler.GetMetricsAndActivity(t.Context(), "s0-iggy-test-stream-test-topic-test-group")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// StoredOffset unchanged → persistent lag, both partitions get lag=0 for scaling
	// But totalLagWithPersistent=60 so still active
	if !isActive {
		t.Error("second call: expected active, got inactive")
	}
	gotLag2 := metrics2[0].Value.MilliValue() / 1000
	if gotLag2 != 0 {
		t.Errorf("second call: expected lag 0, got %d", gotLag2)
	}
}
