package scalers

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type kubernetesResourceMetadataTestData struct {
	metadata  map[string]string
	namespace string
	isError   bool
	errorMsg  string
}

var parseKubernetesResourceMetadataTestDataset = []kubernetesResourceMetadataTestData{
	// Valid ConfigMap
	{map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "threshold", "targetValue": "10"}, "test", false, ""},
	// Valid Secret
	{map[string]string{"resourceKind": "Secret", "resourceName": "my-secret", "key": "threshold", "targetValue": "10"}, "test", false, ""},
	// Valid with all optional fields
	{map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "data", "valueLocation": "metrics.threshold", "targetValue": "10", "activationTargetValue": "5", "format": "json", "valueType": "float64"}, "test", false, ""},
	// Missing resourceKind
	{map[string]string{"resourceName": "my-config", "key": "threshold", "targetValue": "10"}, "test", true, "resourceKind"},
	// Missing resourceName
	{map[string]string{"resourceKind": "ConfigMap", "key": "threshold", "targetValue": "10"}, "test", true, "resourceName"},
	// Missing key
	{map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "targetValue": "10"}, "test", true, "key"},
	// Missing targetValue
	{map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "threshold"}, "test", true, "targetValue"},
	// Invalid resourceKind
	{map[string]string{"resourceKind": "Pod", "resourceName": "my-pod", "key": "threshold", "targetValue": "10"}, "test", true, ""},
	// Invalid format
	{map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "threshold", "targetValue": "10", "format": "xml"}, "test", true, ""},
	// Invalid valueType
	{map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "threshold", "targetValue": "10", "valueType": "string"}, "test", true, ""},
	// valueLocation with number format (should fail validation)
	{map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "threshold", "targetValue": "10", "format": "number", "valueLocation": "some.path"}, "test", true, "valueLocation"},
}

func TestParseKubernetesResourceMetadata(t *testing.T) {
	for idx, testData := range parseKubernetesResourceMetadataTestDataset {
		t.Run(fmt.Sprintf("Test %d", idx), func(t *testing.T) {
			_, err := NewKubernetesResourceScaler(
				fake.NewClientBuilder().Build(),
				&scalersconfig.ScalerConfig{
					TriggerMetadata:         testData.metadata,
					ScalableObjectNamespace: testData.namespace,
				},
			)
			if testData.isError {
				if err == nil {
					t.Errorf("Expected error but got success")
				}
			} else {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
			}
		})
	}
}

type kubernetesResourceIsActiveTestData struct {
	metadata       map[string]string
	namespace      string
	configMapData  map[string]string
	secretData     map[string][]byte
	expectedActive bool
	expectedValue  float64
}

var isActiveKubernetesResourceTestDataset = []kubernetesResourceIsActiveTestData{
	// ConfigMap with simple number - inactive
	{
		metadata:       map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "threshold", "targetValue": "10", "activationTargetValue": "5"},
		namespace:      "test",
		configMapData:  map[string]string{"threshold": "3"},
		expectedActive: false,
		expectedValue:  3,
	},
	// ConfigMap with simple number - active
	{
		metadata:       map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "threshold", "targetValue": "10", "activationTargetValue": "5"},
		namespace:      "test",
		configMapData:  map[string]string{"threshold": "8"},
		expectedActive: true,
		expectedValue:  8,
	},
	// ConfigMap with JSON format
	{
		metadata:       map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "data", "valueLocation": "metrics.threshold", "targetValue": "10", "format": "json"},
		namespace:      "test",
		configMapData:  map[string]string{"data": `{"metrics": {"threshold": 15}}`},
		expectedActive: true,
		expectedValue:  15,
	},
	// ConfigMap with YAML format
	{
		metadata:       map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "data", "valueLocation": "metrics.threshold", "targetValue": "10", "format": "yaml"},
		namespace:      "test",
		configMapData:  map[string]string{"data": "metrics:\n  threshold: 20\n"},
		expectedActive: true,
		expectedValue:  20,
	},
	// ConfigMap with quantity type
	{
		metadata:       map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "threshold", "targetValue": "10", "valueType": "quantity"},
		namespace:      "test",
		configMapData:  map[string]string{"threshold": "500m"},
		expectedActive: true, // 0.5 > 0 (default activationValue)
		expectedValue:  0.5,
	},
	// ConfigMap with int64 type
	{
		metadata:       map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "threshold", "targetValue": "10", "valueType": "int64"},
		namespace:      "test",
		configMapData:  map[string]string{"threshold": "42"},
		expectedActive: true,
		expectedValue:  42,
	},
	// Secret with simple number
	{
		metadata:       map[string]string{"resourceKind": "Secret", "resourceName": "my-secret", "key": "threshold", "targetValue": "10"},
		namespace:      "test",
		secretData:     map[string][]byte{"threshold": []byte("25")},
		expectedActive: true,
		expectedValue:  25,
	},
	// Secret with JSON format
	{
		metadata:       map[string]string{"resourceKind": "Secret", "resourceName": "my-secret", "key": "data", "valueLocation": "limit", "targetValue": "100", "format": "json"},
		namespace:      "test",
		secretData:     map[string][]byte{"data": []byte(`{"limit": 150}`)},
		expectedActive: true,
		expectedValue:  150,
	},
}

func TestKubernetesResourceIsActive(t *testing.T) {
	for idx, testData := range isActiveKubernetesResourceTestDataset {
		t.Run(fmt.Sprintf("Test %d", idx), func(t *testing.T) {
			clientBuilder := fake.NewClientBuilder()

			if testData.configMapData != nil {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testData.metadata["resourceName"],
						Namespace: testData.namespace,
					},
					Data: testData.configMapData,
				}
				clientBuilder = clientBuilder.WithRuntimeObjects(cm)
			}

			if testData.secretData != nil {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testData.metadata["resourceName"],
						Namespace: testData.namespace,
					},
					Data: testData.secretData,
				}
				clientBuilder = clientBuilder.WithRuntimeObjects(secret)
			}

			s, err := NewKubernetesResourceScaler(
				clientBuilder.Build(),
				&scalersconfig.ScalerConfig{
					TriggerMetadata:         testData.metadata,
					AuthParams:              map[string]string{},
					GlobalHTTPTimeout:       1000 * time.Millisecond,
					ScalableObjectNamespace: testData.namespace,
				},
			)
			if err != nil {
				t.Errorf("Error creating scaler: %v", err)
				return
			}

			metrics, isActive, err := s.GetMetricsAndActivity(context.TODO(), "test-metric")
			if err != nil {
				t.Errorf("Error getting metrics: %v", err)
				return
			}

			if testData.expectedActive && !isActive {
				t.Error("Expected active but got inactive")
			}
			if !testData.expectedActive && isActive {
				t.Error("Expected inactive but got active")
			}

			// Check metric value
			if len(metrics) != 1 {
				t.Errorf("Expected 1 metric, got %d", len(metrics))
				return
			}

			// AsApproximateFloat64() returns the value as base units
			// GenerateMetricInMili stores value*1000 as milli units
			// So AsApproximateFloat64() gives us back the original value
			actualValue := metrics[0].Value.AsApproximateFloat64()
			if actualValue != testData.expectedValue {
				t.Errorf("Expected metric value %f, got %f", testData.expectedValue, actualValue)
			}
		})
	}
}

type kubernetesResourceGetMetricSpecTestData struct {
	metadata     map[string]string
	namespace    string
	triggerIndex int
	expectedName string
}

var getMetricSpecKubernetesResourceTestDataset = []kubernetesResourceGetMetricSpecTestData{
	{
		metadata:     map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "threshold", "targetValue": "10"},
		namespace:    "test",
		triggerIndex: 0,
		expectedName: "s0-configmap-my-config-threshold",
	},
	{
		metadata:     map[string]string{"resourceKind": "Secret", "resourceName": "my-secret", "key": "limit", "targetValue": "100"},
		namespace:    "default",
		triggerIndex: 1,
		expectedName: "s1-secret-my-secret-limit",
	},
	{
		metadata:     map[string]string{"resourceKind": "ConfigMap", "resourceName": "scaling-config", "key": "queue_length", "targetValue": "50"},
		namespace:    "production",
		triggerIndex: 2,
		expectedName: "s2-configmap-scaling-config-queue_length",
	},
}

func TestKubernetesResourceGetMetricSpecForScaling(t *testing.T) {
	for idx, testData := range getMetricSpecKubernetesResourceTestDataset {
		t.Run(fmt.Sprintf("Test %d", idx), func(t *testing.T) {
			s, err := NewKubernetesResourceScaler(
				fake.NewClientBuilder().Build(),
				&scalersconfig.ScalerConfig{
					TriggerMetadata:         testData.metadata,
					AuthParams:              map[string]string{},
					GlobalHTTPTimeout:       1000 * time.Millisecond,
					ScalableObjectNamespace: testData.namespace,
					TriggerIndex:            testData.triggerIndex,
				},
			)
			if err != nil {
				t.Errorf("Error creating scaler: %v", err)
				return
			}

			metricSpecs := s.GetMetricSpecForScaling(context.Background())

			if len(metricSpecs) != 1 {
				t.Errorf("Expected 1 metric spec, got %d", len(metricSpecs))
				return
			}

			if metricSpecs[0].External.Metric.Name != testData.expectedName {
				t.Errorf("Expected metric name '%s', got '%s'", testData.expectedName, metricSpecs[0].External.Metric.Name)
			}
		})
	}
}

func TestKubernetesResourceErrorCases(t *testing.T) {
	testCases := []struct {
		name          string
		metadata      map[string]string
		configMapData map[string]string
		secretData    map[string][]byte
		expectError   bool
		errorContains string
	}{
		{
			name:          "ConfigMap not found",
			metadata:      map[string]string{"resourceKind": "ConfigMap", "resourceName": "missing-config", "key": "threshold", "targetValue": "10"},
			configMapData: nil,
			expectError:   true,
			errorContains: "not found",
		},
		{
			name:          "Key not found in ConfigMap",
			metadata:      map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "missing-key", "targetValue": "10"},
			configMapData: map[string]string{"threshold": "10"},
			expectError:   true,
			errorContains: "key missing-key not found",
		},
		{
			name:          "Secret not found",
			metadata:      map[string]string{"resourceKind": "Secret", "resourceName": "missing-secret", "key": "threshold", "targetValue": "10"},
			secretData:    nil,
			expectError:   true,
			errorContains: "not found",
		},
		{
			name:          "Invalid number format",
			metadata:      map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "threshold", "targetValue": "10"},
			configMapData: map[string]string{"threshold": "not-a-number"},
			expectError:   true,
			errorContains: "parsing",
		},
		{
			name:          "Invalid JSON",
			metadata:      map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "data", "valueLocation": "metrics.threshold", "targetValue": "10", "format": "json"},
			configMapData: map[string]string{"data": `{invalid json}`},
			expectError:   true,
			errorContains: "not found",
		},
		{
			name:          "JSON path not found",
			metadata:      map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "data", "valueLocation": "missing.path", "targetValue": "10", "format": "json"},
			configMapData: map[string]string{"data": `{"metrics": {"threshold": 10}}`},
			expectError:   true,
			errorContains: "not found",
		},
		{
			name:          "Invalid YAML",
			metadata:      map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "data", "valueLocation": "metrics.threshold", "targetValue": "10", "format": "yaml"},
			configMapData: map[string]string{"data": "invalid:\n  yaml\n    bad indent"},
			expectError:   true,
			errorContains: "parsing YAML",
		},
		{
			name:          "Invalid quantity",
			metadata:      map[string]string{"resourceKind": "ConfigMap", "resourceName": "my-config", "key": "threshold", "targetValue": "10", "valueType": "quantity"},
			configMapData: map[string]string{"threshold": "invalid-quantity"},
			expectError:   true,
			errorContains: "quantity",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clientBuilder := fake.NewClientBuilder()

			if tc.configMapData != nil {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tc.metadata["resourceName"],
						Namespace: "test",
					},
					Data: tc.configMapData,
				}
				clientBuilder = clientBuilder.WithRuntimeObjects(cm)
			}

			if tc.secretData != nil {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tc.metadata["resourceName"],
						Namespace: "test",
					},
					Data: tc.secretData,
				}
				clientBuilder = clientBuilder.WithRuntimeObjects(secret)
			}

			s, err := NewKubernetesResourceScaler(
				clientBuilder.Build(),
				&scalersconfig.ScalerConfig{
					TriggerMetadata:         tc.metadata,
					AuthParams:              map[string]string{},
					GlobalHTTPTimeout:       1000 * time.Millisecond,
					ScalableObjectNamespace: "test",
				},
			)
			if err != nil {
				// Some errors are expected during scaler creation
				if !tc.expectError {
					t.Errorf("Unexpected error creating scaler: %v", err)
				}
				return
			}

			_, _, err = s.GetMetricsAndActivity(context.TODO(), "test-metric")
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s' but got success", tc.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
			}
		})
	}
}
