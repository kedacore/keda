package scalers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/kedacore/keda/v2/pkg/scalers/azure"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	queueLengthMetricName    = "queueLength"
	defaultTargetQueueLength = 5
	externalMetricType       = "External"
)

type azureQueueScaler struct {
	metadata    *azureQueueMetadata
	podIdentity kedav1alpha1.PodIdentityProvider
	httpClient  *http.Client
}

type azureQueueMetadata struct {
	targetQueueLength int
	queueName         string
	connection        string
	accountName       string
}

var azureQueueLog = logf.Log.WithName("azure_queue_scaler")

// NewAzureQueueScaler creates a new scaler for queue
func NewAzureQueueScaler(config *ScalerConfig) (Scaler, error) {
	meta, podIdentity, err := parseAzureQueueMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure queue metadata: %s", err)
	}

	return &azureQueueScaler{
		metadata:    meta,
		podIdentity: podIdentity,
		httpClient:  kedautil.CreateHTTPClient(config.GlobalHTTPTimeout),
	}, nil
}

func parseAzureQueueMetadata(config *ScalerConfig) (*azureQueueMetadata, kedav1alpha1.PodIdentityProvider, error) {
	meta := azureQueueMetadata{}
	meta.targetQueueLength = defaultTargetQueueLength

	if val, ok := config.TriggerMetadata[queueLengthMetricName]; ok {
		queueLength, err := strconv.Atoi(val)
		if err != nil {
			azureQueueLog.Error(err, "Error parsing azure queue metadata", "queueLengthMetricName", queueLengthMetricName)
			return nil, "", fmt.Errorf("error parsing azure queue metadata %s: %s", queueLengthMetricName, err.Error())
		}

		meta.targetQueueLength = queueLength
	}

	if val, ok := config.TriggerMetadata["queueName"]; ok && val != "" {
		meta.queueName = val
	} else {
		return nil, "", fmt.Errorf("no queueName given")
	}

	// before triggerAuthentication CRD, pod identity was configured using this property
	if val, ok := config.TriggerMetadata["useAAdPodIdentity"]; ok && config.PodIdentity == "" {
		if val == "true" {
			config.PodIdentity = kedav1alpha1.PodIdentityProviderAzure
		}
	}

	// If the Use AAD Pod Identity is not present, or set to "none"
	// then check for connection string
	switch config.PodIdentity {
	case "", kedav1alpha1.PodIdentityProviderNone:
		// Azure Queue Scaler expects a "connection" parameter in the metadata
		// of the scaler or in a TriggerAuthentication object
		if config.AuthParams["connection"] != "" {
			// Found the connection in a parameter from TriggerAuthentication
			meta.connection = config.AuthParams["connection"]
		} else if config.TriggerMetadata["connectionFromEnv"] != "" {
			meta.connection = config.ResolvedEnv[config.TriggerMetadata["connectionFromEnv"]]
		}

		if len(meta.connection) == 0 {
			return nil, "", fmt.Errorf("no connection setting given")
		}
	case kedav1alpha1.PodIdentityProviderAzure:
		// If the Use AAD Pod Identity is present then check account name
		if val, ok := config.TriggerMetadata["accountName"]; ok && val != "" {
			meta.accountName = val
		} else {
			return nil, "", fmt.Errorf("no accountName given")
		}
	default:
		return nil, "", fmt.Errorf("pod identity %s not supported for azure storage queues", config.PodIdentity)
	}

	return &meta, config.PodIdentity, nil
}

// IsActive determines whether this scaler is currently active
func (s *azureQueueScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := azure.GetAzureQueueLength(
		ctx,
		s.httpClient,
		s.podIdentity,
		s.metadata.connection,
		s.metadata.queueName,
		s.metadata.accountName,
	)

	if err != nil {
		azureQueueLog.Error(err, "error)")
		return false, err
	}

	return length > 0, nil
}

func (s *azureQueueScaler) Close() error {
	return nil
}

func (s *azureQueueScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetQueueLengthQty := resource.NewQuantity(int64(s.metadata.targetQueueLength), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s", "azure-queue", s.metadata.queueName)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetQueueLengthQty,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureQueueScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	queuelen, err := azure.GetAzureQueueLength(
		ctx,
		s.httpClient,
		s.podIdentity,
		s.metadata.connection,
		s.metadata.queueName,
		s.metadata.accountName,
	)

	if err != nil {
		azureQueueLog.Error(err, "error getting queue length")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(queuelen), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
