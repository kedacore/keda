/*
Copyright 2021 The KEDA Authors

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
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	metrics "github.com/rcrowley/go-metrics"
	cast "github.com/spf13/cast"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
)

func init() {
	// Disable metrics for kafka client (sarama)
	// https://github.com/IBM/sarama/issues/1321
	metrics.UseNilMetrics = true
}

// Scaler interface
type Scaler interface {
	// GetMetricsAndActivity returns the metric values and activity for a metric Name
	GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error)

	// GetMetricSpecForScaling returns the metrics based on which this scaler determines that the ScaleTarget scales. This is used to construct the HPA spec that is created for
	// this scaled object. The labels used should match the selectors used in GetMetrics
	GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec

	// Close any resources that need disposing when scaler is no longer used or destroyed
	Close(ctx context.Context) error
}

// PushScaler interface
type PushScaler interface {
	Scaler

	// Run is the only writer to the active channel and must close it once done.
	Run(ctx context.Context, active chan<- bool)
}

var (
	// ErrScalerUnsupportedUtilizationMetricType is returned when v2.UtilizationMetricType
	// is provided as the metric target type for scaler.
	ErrScalerUnsupportedUtilizationMetricType = errors.New("'Utilization' metric type is unsupported for external metrics, allowed values are 'Value' or 'AverageValue'")

	// ErrScalerConfigMissingField is returned when a required field is missing from the scaler config.
	ErrScalerConfigMissingField = errors.New("missing required field in scaler config")
)

// GetFromAuthOrMeta helps to get a field from Auth or Meta sections
func GetFromAuthOrMeta(config *scalersconfig.ScalerConfig, field string) (string, error) {
	var result string
	var err error
	if config.AuthParams[field] != "" {
		result = config.AuthParams[field]
	} else if config.TriggerMetadata[field] != "" {
		result = config.TriggerMetadata[field]
	}
	if result == "" {
		err = fmt.Errorf("%w: no %s given", ErrScalerConfigMissingField, field)
	}
	return result, err
}

// GenerateMetricNameWithIndex helps to add the index prefix to the metric name
func GenerateMetricNameWithIndex(triggerIndex int, metricName string) string {
	return fmt.Sprintf("s%d-%s", triggerIndex, metricName)
}

// RemoveIndexFromMetricName removes the index prefix from the metric name
func RemoveIndexFromMetricName(triggerIndex int, metricName string) (string, error) {
	metricNameSplit := strings.SplitN(metricName, "-", 2)
	if len(metricNameSplit) != 2 {
		return "", fmt.Errorf("metric name without index prefix")
	}

	indexPrefix, metricNameWithoutIndex := metricNameSplit[0], metricNameSplit[1]
	if indexPrefix != fmt.Sprintf("s%d", triggerIndex) {
		return "", fmt.Errorf("metric name contains incorrect index prefix")
	}

	return metricNameWithoutIndex, nil
}

func InitializeLogger(config *scalersconfig.ScalerConfig, scalerName string) logr.Logger {
	return logf.Log.WithName(scalerName).WithValues("type", config.ScalableObjectType, "namespace", config.ScalableObjectNamespace, "name", config.ScalableObjectName)
}

// GetMetricTargetType helps get the metric target type of the scaler
func GetMetricTargetType(config *scalersconfig.ScalerConfig) (v2.MetricTargetType, error) {
	switch config.MetricType {
	case v2.UtilizationMetricType:
		return "", ErrScalerUnsupportedUtilizationMetricType
	case "":
		// Use AverageValue if no metric type was provided
		return v2.AverageValueMetricType, nil
	default:
		return config.MetricType, nil
	}
}

// GetMetricTarget returns a metric target for a valid given metric target type (Value or AverageValue) and value
func GetMetricTarget(metricType v2.MetricTargetType, metricValue int64) v2.MetricTarget {
	target := v2.MetricTarget{
		Type: metricType,
	}

	// Construct the target size as a quantity
	targetQty := resource.NewQuantity(metricValue, resource.DecimalSI)
	if metricType == v2.AverageValueMetricType {
		target.AverageValue = targetQty
	} else {
		target.Value = targetQty
	}

	return target
}

// GetMetricTargetMili returns a metric target for a valid given metric target type (Value or AverageValue) and value in mili scale
func GetMetricTargetMili(metricType v2.MetricTargetType, metricValue float64) v2.MetricTarget {
	target := v2.MetricTarget{
		Type: metricType,
	}

	// Construct the target size as a quantity
	metricValueMili := int64(metricValue * 1000)
	targetQty := resource.NewMilliQuantity(metricValueMili, resource.DecimalSI)
	if metricType == v2.AverageValueMetricType {
		target.AverageValue = targetQty
	} else {
		target.Value = targetQty
	}

	return target
}

// GenerateMetricInMili returns a externalMetricValue with mili as metric scale
func GenerateMetricInMili(metricName string, value float64) external_metrics.ExternalMetricValue {
	valueMili := int64(value * 1000)
	return external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewMilliQuantity(valueMili, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}
}

// Option represents a function type that modifies a configOptions instance.
type Option func(*configOptions)

type configOptions struct {
	useMetadata       bool        // Indicates whether to use metadata.
	useAuthentication bool        // Indicates whether to use authentication.
	useResolvedEnv    bool        // Indicates whether to use resolved environment variables.
	isOptional        bool        // Indicates whether the configuration is optional.
	defaultVal        interface{} // Default value for the configuration.
}

// UseMetadata is an Option function that sets the useMetadata field of configOptions.
func UseMetadata(metadata bool) Option {
	return func(opt *configOptions) {
		opt.useMetadata = metadata
	}
}

// UseAuthentication is an Option function that sets the useAuthentication field of configOptions.
func UseAuthentication(auth bool) Option {
	return func(opt *configOptions) {
		opt.useAuthentication = auth
	}
}

// UseResolvedEnv is an Option function that sets the useResolvedEnv field of configOptions.
func UseResolvedEnv(resolvedEnv bool) Option {
	return func(opt *configOptions) {
		opt.useResolvedEnv = resolvedEnv
	}
}

// IsOptional is an Option function that sets the isOptional field of configOptions.
func IsOptional(optional bool) Option {
	return func(opt *configOptions) {
		opt.isOptional = optional
	}
}

// WithDefaultVal is an Option function that sets the defaultVal field of configOptions.
func WithDefaultVal(defaultVal interface{}) Option {
	return func(opt *configOptions) {
		opt.defaultVal = defaultVal
	}
}

// getParameterFromConfigV2 retrieves a parameter value from the provided ScalerConfig object based on the specified parameter name, target type, and optional configuration options.
//
// This method searches for the parameter value in different places within the ScalerConfig object, such as authentication parameters, trigger metadata, and resolved environment variables, based on the provided options.
// It then attempts to convert the found value to the specified target type and returns it.
//
// Parameters:
//
//	config: A pointer to a ScalerConfig object from which to retrieve the parameter value.
//	parameter: A string representing the name of the parameter to retrieve.
//	targetType: A reflect.Type representing the target type to which the parameter value should be converted.
//	options: An optional variadic parameter that allows configuring the behavior of the method through Option functions.
//
// Returns:
//   - An interface{} representing the retrieved parameter value, converted to the specified target type.
//   - An error, if any occurred during the retrieval or conversion process.
//
// Example Usage:
//
//	To retrieve a parameter value from a ScalerConfig object, you can call this function with the necessary parameters and options
//
//	```
//	val, err := getParameterFromConfigV2(scalerConfig, "parameterName", reflect.TypeOf(int64(0)), UseMetadata(true), UseAuthentication(true))
//	if err != nil {
//	    // Handle error
//	}
func getParameterFromConfigV2(config *scalersconfig.ScalerConfig, parameter string, targetType reflect.Type, options ...Option) (interface{}, error) {
	opt := &configOptions{defaultVal: ""}
	for _, option := range options {
		option(opt)
	}

	foundCount := 0
	var foundVal string
	var convertedVal interface{}
	var foundErr error

	if val, ok := config.AuthParams[parameter]; ok && val != "" {
		foundCount++
		if opt.useAuthentication {
			foundVal = val
		}
	}
	if val, ok := config.TriggerMetadata[parameter]; ok && val != "" {
		foundCount++
		if opt.useMetadata {
			foundVal = val
		}
	}
	if envFromVal, envFromOk := config.TriggerMetadata[fmt.Sprintf("%sFromEnv", parameter)]; envFromOk {
		if val, ok := config.ResolvedEnv[envFromVal]; ok && val != "" {
			foundCount++
			if opt.useResolvedEnv {
				foundVal = val
			}
		}
	}

	convertedVal, foundErr = convertToType(foundVal, targetType)
	switch {
	case foundCount > 1:
		return opt.defaultVal, fmt.Errorf("value for parameter '%s' found in more than one place", parameter)
	case foundCount == 1:
		if foundErr != nil {
			return opt.defaultVal, foundErr
		}
		return convertedVal, nil
	case opt.isOptional:
		return opt.defaultVal, nil
	default:
		return opt.defaultVal, fmt.Errorf("key not found. Either set the correct key or set isOptional to true and set defaultVal")
	}
}

func convertToType(input interface{}, targetType reflect.Type) (interface{}, error) {
	switch targetType.Kind() {
	case reflect.String:
		return fmt.Sprintf("%v", input), nil
	case reflect.Int:
		val, err := cast.ToIntE(input)
		if err != nil {
			return nil, err
		}
		return val, nil
	case reflect.Int8:
		val, err := cast.ToInt8E(input)
		if err != nil {
			return nil, err
		}
		return val, nil
	case reflect.Int16:
		val, err := cast.ToInt16E(input)
		if err != nil {
			return nil, err
		}
		return val, nil
	case reflect.Int32:
		val, err := cast.ToInt32E(input)
		if err != nil {
			return nil, err
		}
		return val, nil
	case reflect.Int64:
		val, err := cast.ToInt64E(input)
		if err != nil {
			return nil, err
		}
		return val, nil
	case reflect.Uint:
		val, err := cast.ToUintE(input)
		if err != nil {
			return nil, err
		}
		return val, nil
	case reflect.Uint8:
		val, err := cast.ToUint8E(input)
		if err != nil {
			return nil, err
		}
		return val, nil
	case reflect.Uint16:
		val, err := cast.ToUint16E(input)
		if err != nil {
			return nil, err
		}
		return val, nil
	case reflect.Uint32:
		val, err := cast.ToUint32E(input)
		if err != nil {
			return nil, err
		}
		return val, nil
	case reflect.Uint64:
		val, err := cast.ToUint64E(input)
		if err != nil {
			return nil, err
		}
		return val, nil
	case reflect.Float32:
		val, err := cast.ToFloat32E(input)
		if err != nil {
			return nil, err
		}
		return val, nil
	case reflect.Float64:
		val, err := cast.ToFloat64E(input)
		if err != nil {
			return nil, err
		}
		return val, nil
	case reflect.Bool:
		val, err := cast.ToBoolE(input)
		if err != nil {
			return nil, err
		}
		return val, nil
	default:
		return nil, fmt.Errorf("unsupported target type: %v", targetType)
	}
}
