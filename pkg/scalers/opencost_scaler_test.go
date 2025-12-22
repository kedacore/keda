package scalers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type parseOpenCostMetadataTestData struct {
	metadata map[string]string
	isError  bool
	testName string
}

type openCostMetricIdentifier struct {
	metadataTestData *parseOpenCostMetadataTestData
	triggerIndex     int
	name             string
}

var testOpenCostMetadata = []parseOpenCostMetadataTestData{
	// Valid configurations
	{
		metadata: map[string]string{
			"serverAddress": "http://opencost.opencost:9003",
			"costThreshold": "100",
		},
		isError:  false,
		testName: "valid minimal config",
	},
	{
		metadata: map[string]string{
			"serverAddress":           "http://opencost.opencost:9003",
			"costThreshold":           "50",
			"window":                  "24h",
			"aggregate":               "namespace",
			"filter":                  "default",
			"costType":                "cpuCost",
			"activationCostThreshold": "10",
		},
		isError:  false,
		testName: "valid full config",
	},
	{
		metadata: map[string]string{
			"serverAddress": "http://opencost.opencost:9003",
			"costThreshold": "100",
			"costType":      "gpuCost",
		},
		isError:  false,
		testName: "valid GPU cost config",
	},
	// Invalid configurations
	{
		metadata: map[string]string{
			"costThreshold": "100",
		},
		isError:  true,
		testName: "missing serverAddress",
	},
	{
		metadata: map[string]string{
			"serverAddress": "http://opencost.opencost:9003",
		},
		isError:  true,
		testName: "missing costThreshold",
	},
	{
		metadata: map[string]string{
			"serverAddress": "http://opencost.opencost:9003",
			"costThreshold": "100",
			"costType":      "invalidCost",
		},
		isError:  true,
		testName: "invalid costType",
	},
	{
		metadata: map[string]string{
			"serverAddress": "http://opencost.opencost:9003",
			"costThreshold": "100",
			"aggregate":     "invalidAggregate",
		},
		isError:  true,
		testName: "invalid aggregate",
	},
	{
		metadata: map[string]string{
			"serverAddress": "http://opencost.opencost:9003",
			"costThreshold": "-10",
		},
		isError:  true,
		testName: "negative costThreshold",
	},
}

var openCostMetricIdentifiers = []openCostMetricIdentifier{
	{
		metadataTestData: &testOpenCostMetadata[0],
		triggerIndex:     0,
		name:             "s0-opencost-namespace-totalCost",
	},
	{
		metadataTestData: &testOpenCostMetadata[1],
		triggerIndex:     1,
		name:             "s1-opencost-namespace-cpuCost",
	},
}

func TestOpenCostParseMetadata(t *testing.T) {
	for _, testData := range testOpenCostMetadata {
		t.Run(testData.testName, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadata,
				TriggerIndex:    0,
			}
			_, err := parseOpenCostMetadata(config)
			if testData.isError {
				assert.Error(t, err, "expected error but got none")
			} else {
				assert.NoError(t, err, "unexpected error: %v", err)
			}
		})
	}
}

func TestOpenCostGetMetricSpecForScaling(t *testing.T) {
	for _, testData := range openCostMetricIdentifiers {
		t.Run(testData.name, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: testData.metadataTestData.metadata,
				TriggerIndex:    testData.triggerIndex,
			}
			meta, err := parseOpenCostMetadata(config)
			assert.NoError(t, err)

			mockScaler := openCostScaler{
				metadata: meta,
				logger:   logr.Discard(),
			}

			metricSpec := mockScaler.GetMetricSpecForScaling(context.Background())
			metricName := metricSpec[0].External.Metric.Name
			assert.Equal(t, testData.name, metricName)
		})
	}
}

func TestOpenCostGetMetricsAndActivity(t *testing.T) {
	// Create a mock OpenCost server
	mockResponse := openCostAllocationResponse{
		Code:   200,
		Status: "success",
		Data: []map[string]openCostItem{
			{
				"default": {
					Name:        "default",
					TotalCost:   150.50,
					CPUCost:     50.25,
					GPUCost:     0,
					RAMCost:     75.15,
					PVCost:      10.10,
					NetworkCost: 15.00,
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/allocation", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	testCases := []struct {
		name                    string
		costThreshold           string
		costType                string
		expectedActive          bool
		activationCostThreshold string
		expectedCost            float64
	}{
		{
			name:                    "active when cost exceeds threshold",
			costThreshold:           "100",
			costType:                "totalCost",
			expectedActive:          true,
			expectedCost:            150.50,
			activationCostThreshold: "0",
		},
		{
			name:                    "inactive when cost below activation threshold",
			costThreshold:           "200",
			costType:                "totalCost",
			expectedActive:          false,
			expectedCost:            150.50,
			activationCostThreshold: "200",
		},
		{
			name:                    "CPU cost type",
			costThreshold:           "100",
			costType:                "cpuCost",
			expectedActive:          true,
			expectedCost:            50.25,
			activationCostThreshold: "0",
		},
		{
			name:                    "RAM cost type",
			costThreshold:           "100",
			costType:                "ramCost",
			expectedActive:          true,
			expectedCost:            75.15,
			activationCostThreshold: "0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &scalersconfig.ScalerConfig{
				TriggerMetadata: map[string]string{
					"serverAddress":           server.URL,
					"costThreshold":           tc.costThreshold,
					"costType":                tc.costType,
					"activationCostThreshold": tc.activationCostThreshold,
				},
				TriggerIndex: 0,
			}

			scaler, err := NewOpenCostScaler(config)
			assert.NoError(t, err)

			metrics, isActive, err := scaler.GetMetricsAndActivity(context.Background(), "test-metric")
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedActive, isActive)
			assert.Len(t, metrics, 1)
		})
	}
}

func TestOpenCostScalerClose(t *testing.T) {
	config := &scalersconfig.ScalerConfig{
		TriggerMetadata: map[string]string{
			"serverAddress": "http://opencost.opencost:9003",
			"costThreshold": "100",
		},
		TriggerIndex: 0,
	}

	scaler, err := NewOpenCostScaler(config)
	assert.NoError(t, err)

	err = scaler.Close(context.Background())
	assert.NoError(t, err)
}

func TestOpenCostExtractCost(t *testing.T) {
	item := openCostItem{
		TotalCost:   100.0,
		CPUCost:     25.0,
		GPUCost:     10.0,
		RAMCost:     40.0,
		PVCost:      15.0,
		NetworkCost: 10.0,
	}

	testCases := []struct {
		costType     string
		expectedCost float64
	}{
		{"totalCost", 100.0},
		{"cpuCost", 25.0},
		{"gpuCost", 10.0},
		{"ramCost", 40.0},
		{"pvCost", 15.0},
		{"networkCost", 10.0},
	}

	for _, tc := range testCases {
		t.Run(tc.costType, func(t *testing.T) {
			scaler := &openCostScaler{
				metadata: &openCostScalerMetadata{
					CostType: tc.costType,
				},
			}
			cost := scaler.extractCost(item)
			assert.Equal(t, tc.expectedCost, cost)
		})
	}
}
