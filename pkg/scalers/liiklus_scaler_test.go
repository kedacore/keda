package scalers

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"go.uber.org/mock/gomock"

	"github.com/kedacore/keda/v2/pkg/scalers/liiklus"
	mock_liiklus "github.com/kedacore/keda/v2/pkg/scalers/liiklus/mocks"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseLiiklusMetadataTestData struct {
	name             string
	metadata         map[string]string
	ExpectedErr      error
	ExpectedMetadata *liiklusMetadata
}

type liiklusMetricIdentifier struct {
	metadataTestData *parseLiiklusMetadataTestData
	triggerIndex     int
	name             string
}

var parseLiiklusMetadataTestDataset = []parseLiiklusMetadataTestData{
	{
		name:     "Empty metadata",
		metadata: map[string]string{},
		ExpectedErr: fmt.Errorf("error parsing liiklus metadata: " +
			"missing required parameter \"address\" in [triggerMetadata]\n" +
			"missing required parameter \"topic\" in [triggerMetadata]\n" +
			"missing required parameter \"group\" in [triggerMetadata]"),
		ExpectedMetadata: nil,
	},
	{
		name:     "Empty address",
		metadata: map[string]string{"topic": "foo"},
		ExpectedErr: fmt.Errorf("error parsing liiklus metadata: " +
			"missing required parameter \"address\" in [triggerMetadata]\n" +
			"missing required parameter \"group\" in [triggerMetadata]"),
		ExpectedMetadata: nil,
	},
	{
		name:     "Empty group",
		metadata: map[string]string{"topic": "foo", "address": "using-mock"},
		ExpectedErr: fmt.Errorf("error parsing liiklus metadata: " +
			"missing required parameter \"group\" in [triggerMetadata]"),
		ExpectedMetadata: nil,
	},
	{
		name:        "Valid",
		metadata:    map[string]string{"topic": "foo", "address": "using-mock", "group": "mygroup"},
		ExpectedErr: nil,
		ExpectedMetadata: &liiklusMetadata{
			LagThreshold:           10,
			ActivationLagThreshold: 0,
			Address:                "using-mock",
			Topic:                  "foo",
			Group:                  "mygroup",
			GroupVersion:           0,
			triggerIndex:           0,
		},
	},
	{
		name:             "Invalid activationLagThreshold",
		metadata:         map[string]string{"topic": "foo", "address": "using-mock", "group": "mygroup", "activationLagThreshold": "invalid"},
		ExpectedErr:      fmt.Errorf("error parsing liiklus metadata: unable to set param \"activationLagThreshold\" value \"invalid\": unable to unmarshal to field type int64: invalid character 'i' looking for beginning of value"),
		ExpectedMetadata: nil,
	},
	{
		name:        "Custom lagThreshold",
		metadata:    map[string]string{"topic": "foo", "address": "using-mock", "group": "mygroup", "lagThreshold": "20"},
		ExpectedErr: nil,
		ExpectedMetadata: &liiklusMetadata{
			LagThreshold:           20,
			ActivationLagThreshold: 0,
			Address:                "using-mock",
			Topic:                  "foo",
			Group:                  "mygroup",
			GroupVersion:           0,
			triggerIndex:           0,
		},
	},
}

var liiklusMetricIdentifiers = []liiklusMetricIdentifier{
	{&parseLiiklusMetadataTestDataset[5], 0, "s0-liiklus-foo"},
	{&parseLiiklusMetadataTestDataset[5], 1, "s1-liiklus-foo"},
}

func TestLiiklusParseMetadata(t *testing.T) {
	for _, testData := range parseLiiklusMetadataTestDataset {
		t.Run(testData.name, func(t *testing.T) {
			meta, err := parseLiiklusMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata})

			// error cases
			if testData.ExpectedErr != nil {
				if err == nil {
					t.Errorf("Expected error %v but got success", testData.ExpectedErr)
				} else if err.Error() != testData.ExpectedErr.Error() {
					t.Errorf("Expected error %v but got %v", testData.ExpectedErr, err)
				}
				return // Skip the rest of the checks for error cases
			}

			// success cases
			if err != nil {
				t.Errorf("Expected success but got error %v", err)
			}
			if testData.ExpectedMetadata != nil {
				if testData.ExpectedMetadata.Address != meta.Address {
					t.Errorf("Expected address %q but got %q", testData.ExpectedMetadata.Address, meta.Address)
				}
				if meta.Group != testData.ExpectedMetadata.Group {
					t.Errorf("Expected group %q but got %q", testData.ExpectedMetadata.Group, meta.Group)
				}
				if meta.Topic != testData.ExpectedMetadata.Topic {
					t.Errorf("Expected topic %q but got %q", testData.ExpectedMetadata.Topic, meta.Topic)
				}
				if meta.LagThreshold != testData.ExpectedMetadata.LagThreshold {
					t.Errorf("Expected threshold %d but got %d", testData.ExpectedMetadata.LagThreshold, meta.LagThreshold)
				}
				if meta.ActivationLagThreshold != testData.ExpectedMetadata.ActivationLagThreshold {
					t.Errorf("Expected activation threshold %d but got %d", testData.ExpectedMetadata.ActivationLagThreshold, meta.ActivationLagThreshold)
				}
				if meta.GroupVersion != testData.ExpectedMetadata.GroupVersion {
					t.Errorf("Expected group version %d but got %d", testData.ExpectedMetadata.GroupVersion, meta.GroupVersion)
				}
			}
		})
	}
}

func TestLiiklusScalerActiveBehavior(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	lm, _ := parseLiiklusMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: map[string]string{"topic": "foo", "address": "using-mock", "group": "mygroup"}})
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

	_, active, err := scaler.GetMetricsAndActivity(context.Background(), "m")
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

	_, active, err = scaler.GetMetricsAndActivity(context.Background(), "m")
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

	lm, _ := parseLiiklusMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: map[string]string{"topic": "foo", "address": "using-mock", "group": "mygroup"}})
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

	values, _, err := scaler.GetMetricsAndActivity(context.Background(), "m")
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
	values, _, err = scaler.GetMetricsAndActivity(context.Background(), "m")
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
		t.Run(testData.name, func(t *testing.T) {
			meta, err := parseLiiklusMetadata(&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadataTestData.metadata, TriggerIndex: testData.triggerIndex})
			if err != nil {
				t.Fatal("Could not parse metadata:", err)
			}
			mockLiiklusScaler := liiklusScaler{"", meta, nil, nil, logr.Discard()}
			metricSpec := mockLiiklusScaler.GetMetricSpecForScaling(context.Background())
			if metricSpec[0].External.Metric.Name != testData.name {
				t.Errorf("Wrong External metric source name: %s", metricSpec[0].External.Metric.Name)
			}
		})
	}
}
