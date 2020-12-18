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
	blobCountMetricName    = "blobCount"
	defaultTargetBlobCount = 5
	defaultBlobDelimiter   = "/"
	defaultBlobPrefix      = ""
)

type azureBlobScaler struct {
	metadata    *azureBlobMetadata
	podIdentity kedav1alpha1.PodIdentityProvider
	httpClient  *http.Client
}

type azureBlobMetadata struct {
	targetBlobCount   int
	blobContainerName string
	blobDelimiter     string
	blobPrefix        string
	connection        string
	accountName       string
}

var azureBlobLog = logf.Log.WithName("azure_blob_scaler")

// NewAzureBlobScaler creates a new azureBlobScaler
func NewAzureBlobScaler(config *ScalerConfig) (Scaler, error) {
	meta, podIdentity, err := parseAzureBlobMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure blob metadata: %s", err)
	}

	return &azureBlobScaler{
		metadata:    meta,
		podIdentity: podIdentity,
		httpClient:  kedautil.CreateHTTPClient(config.GlobalHTTPTimeout),
	}, nil
}

func parseAzureBlobMetadata(config *ScalerConfig) (*azureBlobMetadata, kedav1alpha1.PodIdentityProvider, error) {
	meta := azureBlobMetadata{}
	meta.targetBlobCount = defaultTargetBlobCount
	meta.blobDelimiter = defaultBlobDelimiter
	meta.blobPrefix = defaultBlobPrefix

	if val, ok := config.TriggerMetadata[blobCountMetricName]; ok {
		blobCount, err := strconv.Atoi(val)
		if err != nil {
			azureBlobLog.Error(err, "Error parsing azure blob metadata", "blobCountMetricName", blobCountMetricName)
			return nil, "", fmt.Errorf("error parsing azure blob metadata %s: %s", blobCountMetricName, err.Error())
		}

		meta.targetBlobCount = blobCount
	}

	if val, ok := config.TriggerMetadata["blobContainerName"]; ok && val != "" {
		meta.blobContainerName = val
	} else {
		return nil, "", fmt.Errorf("no blobContainerName given")
	}

	if val, ok := config.TriggerMetadata["blobDelimiter"]; ok && val != "" {
		meta.blobDelimiter = val
	}

	if val, ok := config.TriggerMetadata["blobPrefix"]; ok && val != "" {
		meta.blobPrefix = val + meta.blobDelimiter
	}

	// before triggerAuthentication CRD, pod identity was configured using this property
	if val, ok := config.TriggerMetadata["useAAdPodIdentity"]; ok && config.PodIdentity == "" && val == "true" {
		config.PodIdentity = kedav1alpha1.PodIdentityProviderAzure
	}

	// If the Use AAD Pod Identity is not present, or set to "none"
	// then check for connection string
	switch config.PodIdentity {
	case "", kedav1alpha1.PodIdentityProviderNone:
		// Azure Blob Scaler expects a "connection" parameter in the metadata
		// of the scaler or in a TriggerAuthentication object
		if config.AuthParams["connection"] != "" {
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
		return nil, "", fmt.Errorf("pod identity %s not supported for azure storage blobs", config.PodIdentity)
	}

	return &meta, config.PodIdentity, nil
}

// GetScaleDecision is a func
func (s *azureBlobScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := azure.GetAzureBlobListLength(
		ctx,
		s.httpClient,
		s.podIdentity,
		s.metadata.connection,
		s.metadata.blobContainerName,
		s.metadata.accountName,
		s.metadata.blobDelimiter,
		s.metadata.blobPrefix,
	)

	if err != nil {
		azureBlobLog.Error(err, "error)")
		return false, err
	}

	return length > 0, nil
}

func (s *azureBlobScaler) Close() error {
	return nil
}

func (s *azureBlobScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	targetBlobCount := resource.NewQuantity(int64(s.metadata.targetBlobCount), resource.DecimalSI)
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: kedautil.NormalizeString(fmt.Sprintf("%s-%s", "azure-blob", s.metadata.blobContainerName)),
		},
		Target: v2beta2.MetricTarget{
			Type:         v2beta2.AverageValueMetricType,
			AverageValue: targetBlobCount,
		},
	}
	metricSpec := v2beta2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta2.MetricSpec{metricSpec}
}

// GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureBlobScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	bloblen, err := azure.GetAzureBlobListLength(
		ctx,
		s.httpClient,
		s.podIdentity,
		s.metadata.connection,
		s.metadata.blobContainerName,
		s.metadata.accountName,
		s.metadata.blobDelimiter,
		s.metadata.blobPrefix,
	)

	if err != nil {
		azureBlobLog.Error(err, "error getting blob list length")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(int64(bloblen), resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
