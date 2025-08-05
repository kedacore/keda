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
	"fmt"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/go-logr/logr"
	"github.com/gobwas/glob"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/keda-scalers/azure"
	"github.com/kedacore/keda/v2/keda-scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	blobCountMetricName           = "blobCount"
	activationBlobCountMetricName = "activationBlobCount"
	defaultTargetBlobCount        = 5
	defaultBlobDelimiter          = "/"
	defaultBlobPrefix             = ""
)

type azureBlobScaler struct {
	metricType  v2.MetricTargetType
	metadata    *azure.BlobMetadata
	podIdentity kedav1alpha1.AuthPodIdentity
	blobClient  *azblob.Client
	logger      logr.Logger
}

// NewAzureBlobScaler creates a new azureBlobScaler
func NewAzureBlobScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "azure_blob_scaler")

	meta, podIdentity, err := parseAzureBlobMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure blob metadata: %w", err)
	}

	blobClient, err := azure.GetStorageBlobClient(logger, podIdentity, meta.Connection, meta.AccountName, meta.EndpointSuffix, config.GlobalHTTPTimeout)
	if err != nil {
		return nil, fmt.Errorf("error creating azure blob client: %w", err)
	}

	return &azureBlobScaler{
		metricType:  metricType,
		metadata:    meta,
		blobClient:  blobClient,
		podIdentity: podIdentity,
		logger:      logger,
	}, nil
}

func parseAzureBlobMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*azure.BlobMetadata, kedav1alpha1.AuthPodIdentity, error) {
	meta := azure.BlobMetadata{}
	meta.TargetBlobCount = defaultTargetBlobCount
	meta.BlobDelimiter = defaultBlobDelimiter

	if val, ok := config.TriggerMetadata[blobCountMetricName]; ok {
		blobCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			logger.Error(err, "Error parsing azure blob metadata", "blobCountMetricName", blobCountMetricName)
			return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("error parsing azure blob metadata %s: %w", blobCountMetricName, err)
		}

		meta.TargetBlobCount = blobCount
	}

	meta.ActivationTargetBlobCount = 0
	if val, ok := config.TriggerMetadata[activationBlobCountMetricName]; ok {
		activationBlobCount, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			logger.Error(err, "Error parsing azure blob metadata", activationBlobCountMetricName, activationBlobCountMetricName)
			return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("error parsing azure blob metadata %s: %w", activationBlobCountMetricName, err)
		}

		meta.ActivationTargetBlobCount = activationBlobCount
	}

	if val, ok := config.TriggerMetadata["blobContainerName"]; ok && val != "" {
		meta.BlobContainerName = val
	} else {
		return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("no blobContainerName given")
	}

	if val, ok := config.TriggerMetadata["blobDelimiter"]; ok && val != "" {
		meta.BlobDelimiter = val
	}

	if val, ok := config.TriggerMetadata["recursive"]; ok && val != "" {
		recursive, err := strconv.ParseBool(val)
		if err != nil {
			return nil, kedav1alpha1.AuthPodIdentity{}, err
		}

		if recursive {
			meta.BlobDelimiter = ""
		}
	}

	if val, ok := config.TriggerMetadata["globPattern"]; ok && val != "" {
		glob, err := glob.Compile(val)
		if err != nil {
			return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("invalid glob pattern - %w", err)
		}
		meta.GlobPattern = &glob
	}

	if val, ok := config.TriggerMetadata["blobPrefix"]; ok && val != "" {
		prefix := val + meta.BlobDelimiter
		meta.BlobPrefix = &prefix
	}

	endpointSuffix, err := azure.ParseAzureStorageEndpointSuffix(config.TriggerMetadata, azure.BlobEndpoint)
	if err != nil {
		return nil, kedav1alpha1.AuthPodIdentity{}, err
	}

	meta.EndpointSuffix = endpointSuffix

	// If the Use AAD Pod Identity is not present, or set to "none"
	// then check for connection string
	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		// Azure Blob Scaler expects a "connection" parameter in the metadata
		// of the scaler or in a TriggerAuthentication object
		if config.AuthParams["connection"] != "" {
			meta.Connection = config.AuthParams["connection"]
		} else if config.TriggerMetadata["connectionFromEnv"] != "" {
			meta.Connection = config.ResolvedEnv[config.TriggerMetadata["connectionFromEnv"]]
		}

		if len(meta.Connection) == 0 {
			return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("no connection setting given")
		}
	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		// If the Use AAD Pod Identity / Workload Identity is present then check account name
		if val, ok := config.TriggerMetadata["accountName"]; ok && val != "" {
			meta.AccountName = val
		} else {
			return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("no accountName given")
		}
	default:
		return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("pod identity %s not supported for azure storage blobs", config.PodIdentity.Provider)
	}

	meta.TriggerIndex = config.TriggerIndex

	return &meta, config.PodIdentity, nil
}

func (s *azureBlobScaler) Close(context.Context) error {
	return nil
}

func (s *azureBlobScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.TriggerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-blob-%s", s.metadata.BlobContainerName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetBlobCount),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureBlobScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	bloblen, err := azure.GetAzureBlobListLength(
		ctx,
		s.blobClient,
		s.metadata,
	)

	if err != nil {
		s.logger.Error(err, "error getting blob list length")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(bloblen))

	return []external_metrics.ExternalMetricValue{metric}, bloblen > s.metadata.ActivationTargetBlobCount, nil
}
