package scalers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	azureMonitorMetricName       = "metricName"
	targetValueName              = "targetValue"
	defaultClientIDSetting       = ""
	defaultClientPasswordSetting = ""
)

type azureMonitorScaler struct {
	metadata *azureMonitorMetadata
}

type azureMonitorMetadata struct {
	resourceURI         string
	tenantID            string
	subscriptionID      string
	resourceGroupName   string
	name                string
	filter              string
	aggregationInterval string
	aggregationType     string
	clientID            string
	clientPassword      string
	targetValue         int
}

var azureMonitorLog = logf.Log.WithName("azure_monitor_scaler")

// NewAzureMonitorScaler creates a new AzureMonitorScaler
func NewAzureMonitorScaler(resolvedEnv, metadata, authParams map[string]string) (Scaler, error) {
	meta, err := parseAzureMonitorMetadata(metadata, resolvedEnv, authParams)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure monitor metadata: %s", err)
	}

	return &azureMonitorScaler{
		metadata: meta,
	}, nil
}

func parseAzureMonitorMetadata(metadata, resolvedEnv, authParams map[string]string) (*azureMonitorMetadata, error) {
	meta := azureMonitorMetadata{}

	if val, ok := metadata[targetValueName]; ok && val != "" {
		targetValue, err := strconv.Atoi(val)
		if err != nil {
			azureMonitorLog.Error(err, "Error parsing azure monitor metadata", "targetValue", targetValueName)
			return nil, fmt.Errorf("Error parsing azure monitor metadata %s: %s", targetValueName, err.Error())
		}
		meta.targetValue = targetValue
	} else {
		return nil, fmt.Errorf("no targetValue given")
	}

	if val, ok := metadata["resourceURI"]; ok && val != "" {
		resourceURI := strings.Split(val, "/")
		if len(resourceURI) != 3 {
			return nil, fmt.Errorf("resourceURI not in the correct format. Should be namespace/resource_type/resource_name")
		}
		meta.resourceURI = val
	} else {
		return nil, fmt.Errorf("no resourceURI given")
	}

	if val, ok := metadata["resourceGroupName"]; ok && val != "" {
		meta.resourceGroupName = val
	} else {
		return nil, fmt.Errorf("no resourceGroupName given")
	}

	if val, ok := metadata[azureMonitorMetricName]; ok && val != "" {
		meta.name = val
	} else {
		return nil, fmt.Errorf("no metricName given")
	}

	if val, ok := metadata["metricAggregationType"]; ok && val != "" {
		meta.aggregationType = val
	} else {
		return nil, fmt.Errorf("no metricAggregationType given")
	}

	if val, ok := metadata["metricFilter"]; ok && val != "" {
		meta.filter = val
	}

	if val, ok := metadata["metricAggregationInterval"]; ok && val != "" {
		aggregationInterval := strings.Split(val, ":")
		if len(aggregationInterval) != 3 {
			return nil, fmt.Errorf("metricAggregationInterval not in the correct format. Should be hh:mm:ss")
		}
		meta.aggregationInterval = val
	}

	// Required authentication parameters below

	if val, ok := metadata["subscriptionId"]; ok && val != "" {
		meta.subscriptionID = val
	} else {
		return nil, fmt.Errorf("no subscriptionId given")
	}

	if val, ok := metadata["tenantId"]; ok && val != "" {
		meta.tenantID = val
	} else {
		return nil, fmt.Errorf("no tenantId given")
	}

	if val, ok := authParams["activeDirectoryClientId"]; ok && val != "" {
		meta.clientID = val
	} else {
		clientIDSetting := defaultClientIDSetting
		if val, ok := metadata["activeDirectoryClientId"]; ok && val != "" {
			clientIDSetting = val
		}

		if val, ok := resolvedEnv[clientIDSetting]; ok {
			meta.clientID = val
		} else {
			return nil, fmt.Errorf("no activeDirectoryClientId given")
		}
	}

	if val, ok := authParams["activeDirectoryClientPassword"]; ok && val != "" {
		meta.clientPassword = val
	} else {
		clientPasswordSetting := defaultClientPasswordSetting
		if val, ok := metadata["activeDirectoryClientPassword"]; ok && val != "" {
			clientPasswordSetting = val
		}

		if val, ok := resolvedEnv[clientPasswordSetting]; ok {
			meta.clientPassword = val
		} else {
			return nil, fmt.Errorf("no activeDirectoryClientPassword given")
		}
	}

	return &meta, nil
}

// Returns true if the Azure Monitor metric value is greater than zero
func (s *azureMonitorScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := GetAzureMetricValue(ctx, s.metadata)
	if err != nil {
		azureMonitorLog.Error(err, "error getting azure monitor metric")
		return false, err
	}

	return val > 0, nil
}

func (s *azureMonitorScaler) Close() error {
	return nil
}

func (s *azureMonitorScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetMetricVal := resource.NewQuantity(int64(s.metadata.targetValue), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: azureMonitorMetricName, TargetAverageValue: targetMetricVal}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

func (s *azureMonitorScaler) GetMetricSpecForScalingJob() []v2beta1.MetricSpec {
	return s.GetMetricSpecForScaling()
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureMonitorScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := GetAzureMetricValue(ctx, s.metadata)
	if err != nil {
		azureMonitorLog.Error(err, "error getting azure monitor metric")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(val), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
