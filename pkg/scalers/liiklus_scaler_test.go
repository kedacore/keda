package scalers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/kedacore/keda/v2/pkg/scalers/liiklus"
	mock_liiklus "github.com/kedacore/keda/v2/pkg/scalers/liiklus/mocks"
	"github.com/pkg/errors"
)

type parseLiiklusMetadataTestData struct {
	metadata       map[string]string
	err            error
	liiklusAddress string
	group          string
	topic          string
	threshold      int64
}

type liiklusMetricIdentifier struct {
	metadataTestData *parseLiiklusMetadataTestData
	name             string
}

var parseLiiklusMetadataTestDataset = []parseLiiklusMetadataTestData{
	{map[string]string{}, errors.New("no topic provided"), "", "", "", 0},
	{map[string]string{"topic": "foo"}, errors.New("no liiklus API address provided"), "", "", "", 0},
	{map[string]string{"topic": "foo", "address": "bar:6565"}, errors.New("no consumer group provided"), "", "", "", 0},
	{map[string]string{"topic": "foo", "address": "bar:6565", "group": "mygroup"}, nil, "bar:6565", "mygroup", "foo", 10},
	{map[string]string{"topic": "foo", "address": "bar:6565", "group": "mygroup", "lagThreshold": "15"}, nil, "bar:6565", "mygroup", "foo", 15},
}

var liiklusMetricIdentifiers = []liiklusMetricIdentifier{
	{&parseLiiklusMetadataTestDataset[4], "liiklus-foo-mygroup"},
}

func TestLiiklusParseMetadata(t *testing.T) {
	for _, testData := range parseLiiklusMetadataTestDataset {
		meta, err := parseLiiklusMetadata(&ScalerConfig{TriggerMetadata: testData.metadata})
		if err != nil && testData.err == nil {
			t.Error("Expected success but got error", err)
			continue
		}
		if testData.err != nil && err == nil {
			t.Error("Expected error but got success")
			continue
		}
		if testData.err != nil && err != nil && testData.err.Error() != err.Error() {
			t.Errorf("Expected error %v but got %v", testData.err, err)
			continue
		}
		if err != nil {
			continue
		}
		if testData.liiklusAddress != meta.address {
			t.Errorf("Expected address %q but got %q\n", testData.liiklusAddress, meta.address)
			continue
		}
		if meta.group != testData.group {
			t.Errorf("Expected group %q but got %q\n", testData.group, meta.group)
			continue
		}
		if meta.topic != testData.topic {
			t.Errorf("Expected topic %q but got %q\n", testData.topic, meta.topic)
			continue
		}
		if meta.lagThreshold != testData.threshold {
			t.Errorf("Expected threshold %d but got %d\n", testData.threshold, meta.lagThreshold)
			continue
		}
	}
}

func TestLiiklusScalerActiveBehavior(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lm, _ := parseLiiklusMetadata(&ScalerConfig{TriggerMetadata: map[string]string{"topic": "foo", "address": "using-mock", "group": "mygroup"}})
	mockClient := mock_liiklus.NewMockLiiklusServiceClient(ctrl)
	scaler := &liiklusScaler{
		metadata: lm,
		client:   mockClient,
	}

	mockClient.EXPECT().
		GetOffsets(gomock.Any(), gomock.Any()).
		Return(&liiklus.GetOffsetsReply{Offsets: map[uint32]uint64{0: 1}}, nil)
	mockClient.EXPECT().
		GetEndOffsets(gomock.Any(), gomock.Any()).
		Return(&liiklus.GetEndOffsetsReply{Offsets: map[uint32]uint64{0: 2}}, nil)

	active, err := scaler.IsActive(context.Background())
	if err != nil {
		t.Errorf("error calling IsActive: %v", err)
		return
	}
	if !active {
		t.Error("expected IsActive to return true")
	}

	mockClient.EXPECT().
		GetOffsets(gomock.Any(), gomock.Any()).
		Return(&liiklus.GetOffsetsReply{Offsets: map[uint32]uint64{0: 2}}, nil)
	mockClient.EXPECT().
		GetEndOffsets(gomock.Any(), gomock.Any()).
		Return(&liiklus.GetEndOffsetsReply{Offsets: map[uint32]uint64{0: 2}}, nil)

	active, err = scaler.IsActive(context.Background())
	if err != nil {
		t.Errorf("error calling IsActive: %v", err)
		return
	}
	if active {
		t.Error("expected IsActive to return false")
	}
}

func TestLiiklusScalerGetMetricsBehavior(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lm, _ := parseLiiklusMetadata(&ScalerConfig{TriggerMetadata: map[string]string{"topic": "foo", "address": "using-mock", "group": "mygroup"}})
	mockClient := mock_liiklus.NewMockLiiklusServiceClient(ctrl)
	scaler := &liiklusScaler{
		metadata: lm,
		client:   mockClient,
	}

	mockClient.EXPECT().
		GetOffsets(gomock.Any(), gomock.Any()).
		Return(&liiklus.GetOffsetsReply{Offsets: map[uint32]uint64{0: 18, 1: 25}}, nil)
	mockClient.EXPECT().
		GetEndOffsets(gomock.Any(), gomock.Any()).
		Return(&liiklus.GetEndOffsetsReply{Offsets: map[uint32]uint64{0: 20, 1: 30}}, nil)

	values, err := scaler.GetMetrics(context.Background(), "m", nil)
	if err != nil {
		t.Errorf("error calling IsActive: %v", err)
		return
	}

	if values[0].Value.Value() != (20-18)+(30-25) {
		t.Errorf("got wrong metric values: %v", values)
	}

	// Test metrics capping
	mockClient.EXPECT().
		GetOffsets(gomock.Any(), gomock.Any()).
		Return(&liiklus.GetOffsetsReply{Offsets: map[uint32]uint64{0: 1, 1: 15}}, nil)
	mockClient.EXPECT().
		GetEndOffsets(gomock.Any(), gomock.Any()).
		Return(&liiklus.GetEndOffsetsReply{Offsets: map[uint32]uint64{0: 20, 1: 30}}, nil)
	values, err = scaler.GetMetrics(context.Background(), "m", nil)
	if err != nil {
		t.Errorf("error calling IsActive: %v", err)
		return
	}

	if values[0].Value.Value() != 10+10 {
		t.Errorf("got wrong metric values: %v", values)
	}
}

func TestLiiklusGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range liiklusMetricIdentifiers {
		meta, err := parseLiiklusMetadata(&ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata})
		if err != nil {
			t.Fatal("Could not parse metadata:", err)
		}
		mockLiiklusScaler := liiklusScaler{meta, nil, nil}

		metricSpec := mockLiiklusScaler.GetMetricSpecForScaling()
		metricName := metricSpec[0].External.Metric.Name
		if metricName != testData.name {
			t.Error("Wrong External metric source name:", metricName)
		}
	}
}
