package scalers

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"

	scalersconfig "github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

func TestGetMetricTargetType(t *testing.T) {
	cases := []struct {
		name           string
		config         *scalersconfig.ScalerConfig
		wantmetricType v2.MetricTargetType
		wantErr        error
	}{
		{
			name:           "utilization metric type",
			config:         &scalersconfig.ScalerConfig{MetricType: v2.UtilizationMetricType},
			wantmetricType: "",
			wantErr:        ErrScalerUnsupportedUtilizationMetricType,
		},
		{
			name:           "average value metric type",
			config:         &scalersconfig.ScalerConfig{MetricType: v2.AverageValueMetricType},
			wantmetricType: v2.AverageValueMetricType,
			wantErr:        nil,
		},
		{
			name:           "value metric type",
			config:         &scalersconfig.ScalerConfig{MetricType: v2.ValueMetricType},
			wantmetricType: v2.ValueMetricType,
			wantErr:        nil,
		},
		{
			name:           "no metric type",
			config:         &scalersconfig.ScalerConfig{},
			wantmetricType: v2.AverageValueMetricType,
			wantErr:        nil,
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			metricType, err := GetMetricTargetType(c.config)
			if c.wantErr != nil {
				assert.ErrorIs(t, err, c.wantErr)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, c.wantmetricType, metricType)
		})
	}
}

func TestGetMetricTarget(t *testing.T) {
	cases := []struct {
		name             string
		metricType       v2.MetricTargetType
		metricValue      int64
		wantmetricTarget v2.MetricTarget
	}{
		{
			name:             "average value metric type",
			metricType:       v2.AverageValueMetricType,
			metricValue:      10,
			wantmetricTarget: v2.MetricTarget{Type: v2.AverageValueMetricType, AverageValue: resource.NewQuantity(10, resource.DecimalSI)},
		},
		{
			name:             "value metric type",
			metricType:       v2.ValueMetricType,
			metricValue:      20,
			wantmetricTarget: v2.MetricTarget{Type: v2.ValueMetricType, Value: resource.NewQuantity(20, resource.DecimalSI)},
		},
	}

	for _, testCase := range cases {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			metricTarget := GetMetricTarget(c.metricType, c.metricValue)
			assert.Equal(t, c.wantmetricTarget, metricTarget)
		})
	}
}

func TestRemoveIndexFromMetricName(t *testing.T) {
	cases := []struct {
		triggerIndex                         int
		metricName                           string
		expectedMetricNameWithoutIndexPrefix string
		isError                              bool
	}{
		// Proper input
		{triggerIndex: 0, metricName: "s0-metricName", expectedMetricNameWithoutIndexPrefix: "metricName", isError: false},
		// Proper input with triggerIndex > 9
		{triggerIndex: 123, metricName: "s123-metricName", expectedMetricNameWithoutIndexPrefix: "metricName", isError: false},
		// Incorrect index prefix
		{triggerIndex: 1, metricName: "s0-metricName", expectedMetricNameWithoutIndexPrefix: "", isError: true},
		// Incorrect index prefix
		{triggerIndex: 0, metricName: "0-metricName", expectedMetricNameWithoutIndexPrefix: "", isError: true},
		// No index prefix
		{triggerIndex: 0, metricName: "metricName", expectedMetricNameWithoutIndexPrefix: "", isError: true},
	}

	for _, testCase := range cases {
		metricName, err := RemoveIndexFromMetricName(testCase.triggerIndex, testCase.metricName)
		if err != nil && !testCase.isError {
			t.Error("Expected success but got error", err)
		}

		if testCase.isError && err == nil {
			t.Error("Expected error but got success")
		}

		if err == nil {
			if metricName != testCase.expectedMetricNameWithoutIndexPrefix {
				t.Errorf("Expected - %s, Got - %s", testCase.expectedMetricNameWithoutIndexPrefix, metricName)
			}
		}
	}
}

type getParameterFromConfigTestData struct {
	name              string
	authParams        map[string]string
	metadata          map[string]string
	resolvedEnv       map[string]string
	parameter         string
	useAuthentication bool
	useMetadata       bool
	useResolvedEnv    bool
	isOptional        bool
	defaultVal        string
	targetType        reflect.Type
	expectedResult    interface{}
	isError           bool
	errorMessage      string
}

var getParameterFromConfigTestDataset = []getParameterFromConfigTestData{
	{
		name:              "test_authParam_only",
		authParams:        map[string]string{"key1": "value1"},
		parameter:         "key1",
		useAuthentication: true,
		targetType:        reflect.TypeOf(string("")),
		expectedResult:    "value1",
		isError:           false,
	},
	{
		name:           "test_trigger_metadata_only",
		metadata:       map[string]string{"key1": "value1"},
		parameter:      "key1",
		useMetadata:    true,
		targetType:     reflect.TypeOf(string("")),
		expectedResult: "value1",
		isError:        false,
	},
	{
		name:           "test_resolved_env_only",
		metadata:       map[string]string{"key1FromEnv": "key1"},
		resolvedEnv:    map[string]string{"key1": "value1"},
		parameter:      "key1",
		useResolvedEnv: true,
		targetType:     reflect.TypeOf(string("")),
		expectedResult: "value1",
		isError:        false,
	},
	{
		name:              "test_authParam_and_resolved_env_only",
		authParams:        map[string]string{"key1": "value1"},
		metadata:          map[string]string{"key1FromEnv": "key1"},
		resolvedEnv:       map[string]string{"key1": "value1"},
		parameter:         "key1",
		useAuthentication: true,
		useResolvedEnv:    true,
		targetType:        reflect.TypeOf(string("")),
		expectedResult:    "",
		isError:           true,
		errorMessage:      "value for parameter 'key1' found in more than one place",
	},
	{
		name:              "test_authParam_and_trigger_metadata_only",
		authParams:        map[string]string{"key1": "value1"},
		metadata:          map[string]string{"key1": "value2"},
		parameter:         "key1",
		useMetadata:       true,
		useAuthentication: true,
		targetType:        reflect.TypeOf(string("")),
		expectedResult:    "",
		isError:           true,
		errorMessage:      "value for parameter 'key1' found in more than one place",
	},
	{
		name:           "test_trigger_metadata_and_resolved_env_only",
		metadata:       map[string]string{"key1": "value1", "key1FromEnv": "key1"},
		resolvedEnv:    map[string]string{"key1": "value1"},
		parameter:      "key1",
		useResolvedEnv: true,
		useMetadata:    true,
		targetType:     reflect.TypeOf(string("")),
		expectedResult: "",
		isError:        true,
		errorMessage:   "value for parameter 'key1' found in more than one place",
	},
	{
		name:           "test_isOptional_key_not_found",
		metadata:       map[string]string{"key1": "value1"},
		parameter:      "key2",
		useResolvedEnv: true,
		useMetadata:    true,
		isOptional:     true,
		targetType:     reflect.TypeOf(string("")),
		expectedResult: "", // Should return empty string
		isError:        false,
	},
	{
		name:           "test_default_value_key_not_found",
		metadata:       map[string]string{"key1": "value1"},
		parameter:      "key2",
		useResolvedEnv: true,
		useMetadata:    true,
		isOptional:     true,
		defaultVal:     "default",
		targetType:     reflect.TypeOf(string("")),
		expectedResult: "default",
		isError:        false,
	},
	{
		name:           "test_error",
		metadata:       map[string]string{"key1": "value1"},
		parameter:      "key2",
		useResolvedEnv: true,
		useMetadata:    true,
		targetType:     reflect.TypeOf(string("")),
		expectedResult: "default", // Should return empty string
		isError:        true,
		errorMessage:   "key not found. Either set the correct key or set isOptional to true and set defaultVal",
	},
	{
		name:              "test_authParam_bool",
		authParams:        map[string]string{"key1": "true"},
		parameter:         "key1",
		useAuthentication: true,
		targetType:        reflect.TypeOf(true),
		expectedResult:    true,
	},
}

func TestGetParameterFromConfigV2(t *testing.T) {
	for _, testData := range getParameterFromConfigTestDataset {
		val, err := getParameterFromConfigV2(
			&scalersconfig.ScalerConfig{TriggerMetadata: testData.metadata, AuthParams: testData.authParams, ResolvedEnv: testData.resolvedEnv},
			testData.parameter,
			testData.targetType,
			UseMetadata(testData.useMetadata),
			UseAuthentication(testData.useAuthentication),
			UseResolvedEnv(testData.useResolvedEnv),
			IsOptional(testData.isOptional),
			WithDefaultVal(testData.defaultVal),
		)
		if testData.isError {
			assert.NotNilf(t, err, "test %s: expected error but got success, testData - %+v", testData.name, testData)
			assert.Containsf(t, err.Error(), testData.errorMessage, "test %s: %v", testData.name, err.Error())
		} else {
			assert.Nilf(t, err, "test %s:%v", testData.name, err)
			assert.Equalf(t, testData.expectedResult, val, "test %s: expected %s but got %s", testData.name, testData.expectedResult, val)
		}
	}
}

type convertStringToTypeTestData struct {
	name           string
	input          interface{}
	targetType     reflect.Type
	expectedOutput interface{}
	isError        bool
	errorMessage   string
}

var convertStringToTypeDataset = []convertStringToTypeTestData{
	// int64 source
	{
		name:           "int64 to float64",
		input:          int64(1234),
		targetType:     reflect.TypeOf(float64(1234)),
		expectedOutput: float64(1234),
		isError:        false,
	},
	{
		name:           "int64 to float32",
		input:          int64(1234),
		targetType:     reflect.TypeOf(float32(1234)),
		expectedOutput: float32(1234),
		isError:        false,
	},
	{
		name:           "int64 to uint64",
		input:          int64(1234),
		targetType:     reflect.TypeOf(uint64(1234)),
		expectedOutput: uint64(1234),
		isError:        false,
	},
	{
		name:           "int64 to uint32",
		input:          int64(1234),
		targetType:     reflect.TypeOf(uint32(1234)),
		expectedOutput: uint32(1234),
		isError:        false,
	},
	// int32 source
	{
		name:           "int32 to float64",
		input:          int32(1234),
		targetType:     reflect.TypeOf(float64(1234)),
		expectedOutput: float64(1234),
		isError:        false,
	},
	{
		name:           "int32 to float32",
		input:          int32(1234),
		targetType:     reflect.TypeOf(float32(1234)),
		expectedOutput: float32(1234),
		isError:        false,
	},
	{
		name:           "int32 to uint64",
		input:          int32(1234),
		targetType:     reflect.TypeOf(uint64(1234)),
		expectedOutput: uint64(1234),
		isError:        false,
	},
	{
		name:           "int32 to uint32",
		input:          int32(1234),
		targetType:     reflect.TypeOf(uint32(1234)),
		expectedOutput: uint32(1234),
		isError:        false,
	},
	// float64 source
	{
		name:           "float64 to int64",
		input:          float64(1234),
		targetType:     reflect.TypeOf(int64(1234)),
		expectedOutput: int64(1234),
		isError:        false,
	},
	{
		name:           "float64 to int32",
		input:          float64(1234),
		targetType:     reflect.TypeOf(int32(1234)),
		expectedOutput: int32(1234),
		isError:        false,
	},
	{
		name:           "float64 to uint64",
		input:          float64(1234),
		targetType:     reflect.TypeOf(uint64(1234)),
		expectedOutput: uint64(1234),
		isError:        false,
	},
	{
		name:           "float64 to uint32",
		input:          float64(1234),
		targetType:     reflect.TypeOf(uint32(1234)),
		expectedOutput: uint32(1234),
		isError:        false,
	},
	// float32 source
	{
		name:           "float32 to int64",
		input:          float32(1234),
		targetType:     reflect.TypeOf(int64(1234)),
		expectedOutput: int64(1234),
		isError:        false,
	},
	{
		name:           "float32 to int32",
		input:          float32(1234),
		targetType:     reflect.TypeOf(int32(1234)),
		expectedOutput: int32(1234),
		isError:        false,
	},
	{
		name:           "float32 to uint64",
		input:          float32(1234),
		targetType:     reflect.TypeOf(uint64(1234)),
		expectedOutput: uint64(1234),
		isError:        false,
	},
	{
		name:           "float32 to uint32",
		input:          float32(1234),
		targetType:     reflect.TypeOf(uint32(1234)),
		expectedOutput: uint32(1234),
		isError:        false,
	},
	// string source
	{
		name:           "string to float64",
		input:          "1234",
		targetType:     reflect.TypeOf(float64(1234)),
		expectedOutput: float64(1234),
		isError:        false,
	},
	{
		name:           "string to float32",
		input:          "1234",
		targetType:     reflect.TypeOf(float32(1234)),
		expectedOutput: float32(1234),
		isError:        false,
	},
	{
		name:           "string to int64",
		input:          "1234",
		targetType:     reflect.TypeOf(int64(1234)),
		expectedOutput: int64(1234),
		isError:        false,
	},
	{
		name:           "string to int32",
		input:          "1234",
		targetType:     reflect.TypeOf(int32(1234)),
		expectedOutput: int32(1234),
		isError:        false,
	},
	{
		name:           "string to uint64",
		input:          "1234",
		targetType:     reflect.TypeOf(uint64(1234)),
		expectedOutput: uint64(1234),
		isError:        false,
	},
	{
		name:           "string to uint32",
		input:          "1234",
		targetType:     reflect.TypeOf(uint32(1234)),
		expectedOutput: uint32(1234),
		isError:        false,
	},
	{
		name:           "string to bool",
		input:          "true",
		targetType:     reflect.TypeOf(true),
		expectedOutput: true,
		isError:        false,
	},
	{
		name:           "unsupported type",
		input:          "Unsupported Type",
		targetType:     reflect.TypeOf([]int{}),
		expectedOutput: "error",
		isError:        true,
		errorMessage:   "unsupported target type: []int",
	},
}

func TestConvertStringToType(t *testing.T) {
	for _, testData := range convertStringToTypeDataset {
		val, err := convertToType(testData.input, testData.targetType)

		if testData.isError {
			assert.NotNilf(t, err, "test %s: expected error but got success, testData - %+v", testData.name, testData)
			assert.Containsf(t, err.Error(), testData.errorMessage, "test %s: %s", testData.name, testData.errorMessage)
		} else {
			assert.Nil(t, err)
			assert.Equalf(t, testData.expectedOutput, val, "test %s: expected %s but got %s", testData.name, testData.expectedOutput, val)
		}
	}
}
