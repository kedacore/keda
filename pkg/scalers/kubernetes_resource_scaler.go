package scalers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v3"
	v2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type kubernetesResourceScaler struct {
	metricType v2.MetricTargetType
	metadata   *kubernetesResourceMetadata
	kubeClient client.Client
	logger     logr.Logger
}

type kubernetesResourceMetadata struct {
	// Resource identification
	ResourceKind resourceKind `keda:"name=resourceKind,order=triggerMetadata,enum=ConfigMap;Secret"`
	ResourceName string       `keda:"name=resourceName,order=triggerMetadata"`
	Key          string       `keda:"name=key,order=triggerMetadata"`

	// Value extraction
	Format        resourceFormat `keda:"name=format,order=triggerMetadata,default=number,enum=number;json;yaml"`
	ValueLocation string         `keda:"name=valueLocation,order=triggerMetadata,optional"`
	ValueType     valueType      `keda:"name=valueType,order=triggerMetadata,default=float64,enum=float64;int64;quantity"`

	// Scaling thresholds
	TargetValue           float64 `keda:"name=targetValue,order=triggerMetadata"`
	ActivationTargetValue float64 `keda:"name=activationTargetValue,order=triggerMetadata,default=0"`

	// Internal fields
	namespace      string
	triggerIndex   int
	asMetricSource bool
}

type resourceKind string

const (
	configMapKind resourceKind = "ConfigMap"
	secretKind    resourceKind = "Secret"
)

type resourceFormat string

const (
	numberFormat resourceFormat = "number"
	jsonFormat   resourceFormat = "json"
	yamlFormat   resourceFormat = "yaml"
)

type valueType string

const (
	float64Type  valueType = "float64"
	int64Type    valueType = "int64"
	quantityType valueType = "quantity"
)

const (
	kubernetesResourceMetricType = "External"
	valueTypeErrorMsg            = "valueLocation must point to value of type number or a string representing a Quantity, got: '%s'"
	valueLocationRequiredMsg     = "valueLocation is required for %s format"
	resourceKindUnsupportedMsg   = "unsupported resourceKind: %s"
	keyNotFoundMsg               = "key %s not found in %s %s/%s"
	pathNotFoundMsg              = "path %s not found in %s"
)

func (m *kubernetesResourceMetadata) Validate() error {
	if m.TargetValue <= 0 && !m.asMetricSource {
		return fmt.Errorf("targetValue must be a float greater than 0")
	}

	if m.Format == numberFormat && m.ValueLocation != "" {
		return fmt.Errorf("valueLocation is only supported with json or yaml format")
	}

	if (m.Format == jsonFormat || m.Format == yamlFormat) && m.ValueLocation == "" {
		return fmt.Errorf(valueLocationRequiredMsg, m.Format)
	}

	return nil
}

// NewKubernetesResourceScaler creates a new kubernetesResourceScaler
func NewKubernetesResourceScaler(kubeClient client.Client, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	meta, err := parseKubernetesResourceMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing kubernetes resource metadata: %w", err)
	}

	return &kubernetesResourceScaler{
		metricType: metricType,
		metadata:   meta,
		kubeClient: kubeClient,
		logger:     InitializeLogger(config, "kubernetes_resource_scaler"),
	}, nil
}

func parseKubernetesResourceMetadata(config *scalersconfig.ScalerConfig) (*kubernetesResourceMetadata, error) {
	meta := &kubernetesResourceMetadata{}
	meta.namespace = config.ScalableObjectNamespace
	meta.triggerIndex = config.TriggerIndex
	meta.asMetricSource = config.AsMetricSource

	err := config.TypedConfig(meta)
	if err != nil {
		return nil, fmt.Errorf("error parsing kubernetes resource metadata: %w", err)
	}

	return meta, nil
}

func (s *kubernetesResourceScaler) Close(context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *kubernetesResourceScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	metricName := kedautil.NormalizeString(fmt.Sprintf("%s-%s-%s", strings.ToLower(string(s.metadata.ResourceKind)), s.metadata.ResourceName, s.metadata.Key))
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, metricName),
		},
		Target: GetMetricTargetMili(s.metricType, s.metadata.TargetValue),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: kubernetesResourceMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric
func (s *kubernetesResourceScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	value, err := s.getMetricValue(ctx)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, fmt.Errorf("error getting metric value from %s: %w", s.metadata.ResourceKind, err)
	}

	metric := GenerateMetricInMili(metricName, value)

	return []external_metrics.ExternalMetricValue{metric}, value > s.metadata.ActivationTargetValue, nil
}

func (s *kubernetesResourceScaler) getMetricValue(ctx context.Context) (float64, error) {
	switch s.metadata.ResourceKind {
	case configMapKind:
		return s.getValueFromConfigMap(ctx)
	case secretKind:
		return s.getValueFromSecret(ctx)
	default:
		return 0, fmt.Errorf(resourceKindUnsupportedMsg, s.metadata.ResourceKind)
	}
}

func (s *kubernetesResourceScaler) getValueFromConfigMap(ctx context.Context) (float64, error) {
	configMap := &corev1.ConfigMap{}
	err := s.kubeClient.Get(ctx, client.ObjectKey{
		Name:      s.metadata.ResourceName,
		Namespace: s.metadata.namespace,
	}, configMap)
	if err != nil {
		return 0, fmt.Errorf("error getting ConfigMap %s/%s: %w", s.metadata.namespace, s.metadata.ResourceName, err)
	}

	value, exists := configMap.Data[s.metadata.Key]
	if !exists {
		return 0, fmt.Errorf(keyNotFoundMsg, s.metadata.Key, configMapKind, s.metadata.namespace, s.metadata.ResourceName)
	}

	return s.parseValue(value)
}

func (s *kubernetesResourceScaler) getValueFromSecret(ctx context.Context) (float64, error) {
	secret := &corev1.Secret{}
	err := s.kubeClient.Get(ctx, client.ObjectKey{
		Name:      s.metadata.ResourceName,
		Namespace: s.metadata.namespace,
	}, secret)
	if err != nil {
		return 0, fmt.Errorf("error getting Secret %s/%s: %w", s.metadata.namespace, s.metadata.ResourceName, err)
	}

	valueBytes, exists := secret.Data[s.metadata.Key]
	if !exists {
		return 0, fmt.Errorf(keyNotFoundMsg, s.metadata.Key, secretKind, s.metadata.namespace, s.metadata.ResourceName)
	}

	return s.parseValue(string(valueBytes))
}

func (s *kubernetesResourceScaler) parseValue(rawValue string) (float64, error) {
	switch s.metadata.Format {
	case numberFormat:
		return s.parseNumericValue(rawValue)
	case jsonFormat:
		return s.parseJSONValue(rawValue)
	case yamlFormat:
		return s.parseYAMLValue(rawValue)
	default:
		return 0, fmt.Errorf("unsupported format: %s", s.metadata.Format)
	}
}

func (s *kubernetesResourceScaler) parseNumericValue(value string) (float64, error) {
	switch s.metadata.ValueType {
	case float64Type:
		var floatVal float64
		_, err := fmt.Sscanf(value, "%f", &floatVal)
		if err != nil {
			return 0, fmt.Errorf("error parsing value as float64: %w", err)
		}
		return floatVal, nil
	case int64Type:
		var intVal int64
		_, err := fmt.Sscanf(value, "%d", &intVal)
		if err != nil {
			return 0, fmt.Errorf("error parsing value as int64: %w", err)
		}
		return float64(intVal), nil
	case quantityType:
		return s.parseQuantity(value)
	default:
		return 0, fmt.Errorf("unsupported value type: %s", s.metadata.ValueType)
	}
}

func (s *kubernetesResourceScaler) parseJSONValue(value string) (float64, error) {
	result := gjson.Get(value, s.metadata.ValueLocation)
	if !result.Exists() {
		return 0, fmt.Errorf(pathNotFoundMsg, s.metadata.ValueLocation, jsonFormat)
	}

	return s.convertGjsonResultToFloat(result)
}

func (s *kubernetesResourceScaler) convertGjsonResultToFloat(result gjson.Result) (float64, error) {
	switch s.metadata.ValueType {
	case float64Type:
		if result.Type == gjson.Number {
			return result.Float(), nil
		}
		if result.Type == gjson.String {
			return s.parseQuantity(result.String())
		}
		return 0, fmt.Errorf(valueTypeErrorMsg, result.Type.String())
	case int64Type:
		if result.Type == gjson.Number {
			return float64(result.Int()), nil
		}
		return 0, fmt.Errorf(valueTypeErrorMsg, result.Type.String())
	case quantityType:
		if result.Type == gjson.String {
			return s.parseQuantity(result.String())
		}
		if result.Type == gjson.Number {
			return result.Float(), nil
		}
		return 0, fmt.Errorf(valueTypeErrorMsg, result.Type.String())
	default:
		return 0, fmt.Errorf("unsupported value type: %s", s.metadata.ValueType)
	}
}

func (s *kubernetesResourceScaler) parseYAMLValue(value string) (float64, error) {
	var yamlMap map[string]interface{}
	err := yaml.Unmarshal([]byte(value), &yamlMap)
	if err != nil {
		return 0, fmt.Errorf("error parsing YAML: %w", err)
	}

	// Use kedautil.GetValueByPath like metrics-api scaler does
	pathValue, err := kedautil.GetValueByPath(yamlMap, s.metadata.ValueLocation)
	if err != nil {
		return 0, fmt.Errorf(pathNotFoundMsg, s.metadata.ValueLocation, yamlFormat)
	}

	return s.convertValueToFloat(pathValue)
}

// convertValueToFloat converts various types to float64, similar to metrics-api scaler
func (s *kubernetesResourceScaler) convertValueToFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		if s.metadata.ValueType == quantityType {
			return s.parseQuantity(v)
		}
		// Try to parse as number
		var floatVal float64
		_, err := fmt.Sscanf(v, "%f", &floatVal)
		if err != nil {
			return 0, fmt.Errorf(valueTypeErrorMsg, v)
		}
		return floatVal, nil
	default:
		return 0, fmt.Errorf(valueTypeErrorMsg, v)
	}
}

// parseQuantity is a helper function to parse quantity strings
func (s *kubernetesResourceScaler) parseQuantity(value string) (float64, error) {
	quantity, err := resource.ParseQuantity(value)
	if err != nil {
		return 0, fmt.Errorf("error parsing value as quantity: %w", err)
	}
	return quantity.AsApproximateFloat64(), nil
}
