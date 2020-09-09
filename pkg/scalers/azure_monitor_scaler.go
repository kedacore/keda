package scalers

import (
	"context"
	"fmt"
	"github.com/kedacore/keda/pkg/scalers/azure"
	"strconv"
	"strings"

	kedautil "github.com/kedacore/keda/pkg/util"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	azureMonitorMetricName = "metricName"
	targetValueName        = "targetValue"
)

type azureMonitorScaler struct {
	metadata    *azureMonitorMetadata
	podIdentity string
}

type azureMonitorMetadata struct {
	azureMonitorInfo azure.MonitorInfo
	targetValue      int
}

var azureMonitorLog = logf.Log.WithName("azure_monitor_scaler")

// NewAzureMonitorScaler creates a new AzureMonitorScaler
func NewAzureMonitorScaler(resolvedEnv, metadata, authParams map[string]string, podIdentity string) (Scaler, error) {
	meta, err := parseAzureMonitorMetadata(metadata, resolvedEnv, authParams, podIdentity)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure monitor metadata: %s", err)
	}

	return &azureMonitorScaler{
		metadata:    meta,
		podIdentity: podIdentity,
	}, nil
}

func parseAzureMonitorMetadata(metadata, resolvedEnv, authParams map[string]string, podIdentity string) (*azureMonitorMetadata, error) {
	meta := azureMonitorMetadata{
		azureMonitorInfo: azure.MonitorInfo{},
	}

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
		meta.azureMonitorInfo.ResourceURI = val
	} else {
		return nil, fmt.Errorf("no resourceURI given")
	}

	if val, ok := metadata["resourceGroupName"]; ok && val != "" {
		meta.azureMonitorInfo.ResourceGroupName = val
	} else {
		return nil, fmt.Errorf("no resourceGroupName given")
	}

	if val, ok := metadata[azureMonitorMetricName]; ok && val != "" {
		meta.azureMonitorInfo.Name = val
	} else {
		return nil, fmt.Errorf("no metricName given")
	}

	if val, ok := metadata["metricAggregationType"]; ok && val != "" {
		meta.azureMonitorInfo.AggregationType = val
	} else {
		return nil, fmt.Errorf("no metricAggregationType given")
	}

	if val, ok := metadata["metricFilter"]; ok && val != "" {
		meta.azureMonitorInfo.Filter = val
	}

	if val, ok := metadata["metricAggregationInterval"]; ok && val != "" {
		aggregationInterval := strings.Split(val, ":")
		if len(aggregationInterval) != 3 {
			return nil, fmt.Errorf("metricAggregationInterval not in the correct format. Should be hh:mm:ss")
		}
		meta.azureMonitorInfo.AggregationInterval = val
	}

	// Required authentication parameters below

	if val, ok := metadata["subscriptionId"]; ok && val != "" {
		meta.azureMonitorInfo.SubscriptionID = val
	} else {
		return nil, fmt.Errorf("no subscriptionId given")
	}

	if val, ok := metadata["tenantId"]; ok && val != "" {
		meta.azureMonitorInfo.TenantID = val
	} else {
		return nil, fmt.Errorf("no tenantId given")
	}

	if podIdentity == "" || podIdentity == "none" {
		if authParams["activeDirectoryClientId"] != "" {
			meta.azureMonitorInfo.ClientID = authParams["activeDirectoryClientId"]
		} else if metadata["activeDirectoryClientId"] != "" {
			meta.azureMonitorInfo.ClientID = metadata["activeDirectoryClientId"]
		} else if metadata["activeDirectoryClientIdFromEnv"] != "" {
			meta.azureMonitorInfo.ClientID = resolvedEnv[metadata["activeDirectoryClientIdFromEnv"]]
		}

		if len(meta.azureMonitorInfo.ClientID) == 0 {
			return nil, fmt.Errorf("no activeDirectoryClientId given")
		}

		if authParams["activeDirectoryClientPassword"] != "" {
			meta.azureMonitorInfo.ClientPassword = authParams["activeDirectoryClientPassword"]
		} else if metadata["activeDirectoryClientPassword"] != "" {
			meta.azureMonitorInfo.ClientPassword = metadata["activeDirectoryClientPassword"]
		} else if metadata["activeDirectoryClientPasswordFromEnv"] != "" {
			meta.azureMonitorInfo.ClientPassword = resolvedEnv[metadata["activeDirectoryClientPasswordFromEnv"]]
		}

		if len(meta.azureMonitorInfo.ClientPassword) == 0 {
			return nil, fmt.Errorf("no activeDirectoryClientPassword given")
		}
	} else if podIdentity != "azure" {
		return nil, fmt.Errorf("Azure Monitor doesn't support pod identity %s", podIdentity)
	}

	return &meta, nil
}

// Returns true if the Azure Monitor metric value is greater than zero
func (s *azureMonitorScaler) IsActive(ctx context.Context) (bool, error) {
	val, err := azure.GetAzureMetricValue(ctx, s.metadata.azureMonitorInfo, s.podIdentity)
	if err != nil {
		azureMonitorLog.Error(err, "error getting azure monitor metric")
		return false, err
	}

	return val > 0, nil
}

func (s *azureMonitorScaler) Close() error {
	return nil
}

func (s *azureMonitorScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetMetricVal := resource.NewQuantity(int64(s.metadata.targetValue), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: fmt.Sprintf("%s-%s-%s-%s", "azure-monitor", kedautil.NormalizeString(s.metadata.azureMonitorInfo.ResourceURI), s.metadata.azureMonitorInfo.ResourceGroupName, s.metadata.azureMonitorInfo.Name),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetMetricVal,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureMonitorScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	val, err := azure.GetAzureMetricValue(ctx, s.metadata.azureMonitorInfo, s.podIdentity)
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
