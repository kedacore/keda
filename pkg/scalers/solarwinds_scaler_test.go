package scalers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseSolarWindsMetadataTestData struct {
	metadata    map[string]string
	authParams  map[string]string
	isError     bool
	description string
}

var testSolarWindsMetadata = []parseSolarWindsMetadataTestData{
	// No metadata
	{metadata: map[string]string{}, authParams: map[string]string{}, isError: true, description: "No metadata"},
	// Valid metadata
	{
		metadata: map[string]string{
			"host":            "https://api.na-01.cloud.solarwinds.com",
			"apiToken":        "testToken",
			"targetValue":     "10",
			"activationValue": "5",
			"metricName":      "testMetric",
			"aggregation":     "AVG",
			"intervalS":       "60",
			"filter":          "testFilter",
		},
		authParams:  map[string]string{},
		isError:     false,
		description: "Valid metadata",
	},
	// Valid metadata without filter (optional)
	{
		metadata: map[string]string{
			"host":            "https://api.na-01.cloud.solarwinds.com",
			"apiToken":        "testToken",
			"targetValue":     "10",
			"activationValue": "5",
			"metricName":      "testMetric",
			"aggregation":     "AVG",
			"intervalS":       "60",
		},
		authParams:  map[string]string{},
		isError:     false,
		description: "Valid metadata without filter (optional)",
	},
	// Invalid host
	{
		metadata: map[string]string{
			"host":            "invalid-url",
			"apiToken":        "testToken",
			"targetValue":     "10",
			"activationValue": "5",
			"metricName":      "testMetric",
			"aggregation":     "AVG",
			"intervalS":       "60",
			"filter":          "testFilter",
		},
		authParams:  map[string]string{},
		isError:     true,
		description: "Invalid host",
	},
}

func TestParseSolarWindsMetadata(t *testing.T) {
	for _, testData := range testSolarWindsMetadata {
		t.Run(testData.description, func(t *testing.T) {
			_, err := parseSolarWindsMetadata(&scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				AuthParams:      testData.authParams,
			})
			if err != nil && !testData.isError {
				t.Errorf("Expected success but got error: %v", err)
			}
			if testData.isError && err == nil {
				t.Errorf("Expected error but got success")
			}
		})
	}
}

func TestSolarWindsScalerGetMetricSpecForScaling(t *testing.T) {
	meta := &solarWindsMetadata{
		MetricName:   "testMetric",
		TargetValue:  3,
		triggerIndex: 0,
	}
	scaler := &solarWindsScaler{
		metricType: v2.AverageValueMetricType,
		metadata:   meta,
	}

	metricSpec := scaler.GetMetricSpecForScaling(context.Background())
	assert.Equal(t, "s0-solarwinds", metricSpec[0].External.Metric.Name)
	assert.Equal(t, int64(3), metricSpec[0].External.Target.AverageValue.Value())
}
