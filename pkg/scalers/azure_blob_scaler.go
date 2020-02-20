package scalers

import (
	"context"
	"fmt"
	"strconv"

	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	blobCountMetricName     	 = "blobCount"
	defaultTargetBlobCount  	 = 5
	defaultBlobDelimiter     	 = "/"
	defaultBlobPrefix        	 = ""
	defaultBlobConnectionSetting = "AzureWebJobsStorage"
)

type azureBlobScaler struct {
	metadata *azureBlobMetadata
	podIdentity string
}

type azureBlobMetadata struct {
	targetBlobCount   int
	blobContainerName string
	blobDelimiter     string
	blobPrefix 		  string
	connection        string
	useAAdPodIdentity bool
	accountName       string
}

var azureBlobLog = logf.Log.WithName("azure_blob_scaler")

// NewAzureBlobScaler creates a new azureBlobScaler
func NewAzureBlobScaler(resolvedEnv, metadata, authParams map[string]string, podIdentity string) (Scaler, error) {
	meta, podIdentity, err := parseAzureBlobMetadata(metadata, resolvedEnv, authParams, podIdentity)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure blob metadata: %s", err)
	}

	return &azureBlobScaler{
		metadata: meta,
		podIdentity: podIdentity,
	}, nil
}

func parseAzureBlobMetadata(metadata, resolvedEnv, authParams map[string]string, podAuth string) (*azureBlobMetadata, string, error) {
	meta := azureBlobMetadata{}
	meta.targetBlobCount = defaultTargetBlobCount
	meta.blobDelimiter = defaultBlobDelimiter
	meta.blobPrefix = defaultBlobPrefix

	if val, ok := metadata[blobCountMetricName]; ok {
		blobCount, err := strconv.Atoi(val)
		if err != nil {
			azureBlobLog.Error(err, "Error parsing azure blob metadata", "blobCountMetricName", blobCountMetricName)
			return nil, "", fmt.Errorf("Error parsing azure blob metadata %s: %s", blobCountMetricName, err.Error())
		}

		meta.targetBlobCount = blobCount
	}

	if val, ok := metadata["blobContainerName"]; ok && val != "" {
		meta.blobContainerName = val
	} else {
		return nil, "", fmt.Errorf("no blobContainerName given")
	}

	if val, ok := metadata["blobDelimiter"]; ok {
		if val != "" {
			meta.blobDelimiter = val
		}
	}

	if val, ok := metadata["blobPrefix"]; ok {
		if val != "" {
			meta.blobPrefix = val + meta.blobDelimiter
		}
	}
	// before triggerAuthentication CRD, pod identity was configured using this property
	if val, ok := metadata["useAAdPodIdentity"]; ok && podAuth == "" {
		if val == "true" {
			podAuth = "azure"
		}
	}

	// If the Use AAD Pod Identity is not present, or set to "none"
	// then check for connection string
	if podAuth == "" || podAuth == "none" {
		// Azure Blob Scaler expects a "connection" parameter in the metadata
		// of the scaler or in a TriggerAuthentication object
		connection := authParams["connection"]
		if connection != "" {
			// Found the connection in a parameter from TriggerAuthentication
			meta.connection = connection
		} else {
		connectionSetting := defaultBlobConnectionSetting
		if val, ok := metadata["connection"]; ok && val != "" {
			connectionSetting = val
		}

		if val, ok := resolvedEnv[connectionSetting]; ok {
			meta.connection = val
		} else {
			return nil, "", fmt.Errorf("no connection setting given")
		}
	}
	} else if podAuth == "azure" {
		// If the Use AAD Pod Identity is present then check account name
		if val, ok := metadata["accountName"]; ok && val != "" {
			meta.accountName = val
		} else {
			return nil, "", fmt.Errorf("no accountName given")
		}
	} else {
		return nil, "", fmt.Errorf("pod identity %s not supported for azure storage blobs", podAuth)
	}

	return &meta, podAuth, nil
}

// GetScaleDecision is a func
func (s *azureBlobScaler) IsActive(ctx context.Context) (bool, error) {
	length, err := GetAzureBlobListLength(
		ctx,
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

func (s *azureBlobScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetBlobCount := resource.NewQuantity(int64(s.metadata.targetBlobCount), resource.DecimalSI)
	externalMetric := &v2beta1.ExternalMetricSource{MetricName: blobCountMetricName, TargetAverageValue: targetBlobCount}
	metricSpec := v2beta1.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2beta1.MetricSpec{metricSpec}
}

func (s *azureBlobScaler) GetMetricSpecForScalingJob() []v2beta1.MetricSpec {
	return s.GetMetricSpecForScaling()
}

//GetMetrics returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureBlobScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	bloblen, err := GetAzureBlobListLength(
		ctx,
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
