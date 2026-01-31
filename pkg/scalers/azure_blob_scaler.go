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

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/go-logr/logr"
	"github.com/gobwas/glob"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

type azureBlobScaler struct {
	metricType  v2.MetricTargetType
	metadata    *azureBlobMetadata
	podIdentity kedav1alpha1.AuthPodIdentity
	blobClient  *azblob.Client
	logger      logr.Logger
}

type azureBlobMetadata struct {
	triggerIndex int

	BlobContainerName         string `keda:"name=blobContainerName,   order=triggerMetadata"`
	TargetBlobCount           int64  `keda:"name=blobCount,           order=triggerMetadata, default=5"`
	ActivationTargetBlobCount int64  `keda:"name=activationBlobCount, order=triggerMetadata, default=0"`
	BlobDelimiter             string `keda:"name=blobDelimiter,       order=triggerMetadata, default=/"`
	BlobPrefix                string `keda:"name=blobPrefix,          order=triggerMetadata, optional"`
	Recursive                 bool   `keda:"name=recursive,           order=triggerMetadata, default=false"`
	GlobPattern               string `keda:"name=globPattern,         order=triggerMetadata, optional"`
	EndpointSuffix            string `keda:"name=endpointSuffix,      order=triggerMetadata, optional"`

	Connection  string `keda:"name=connection;connectionFromEnv, order=authParams;resolvedEnv;triggerMetadata, optional"`
	AccountName string `keda:"name=accountName,                  order=triggerMetadata,                        optional"`

	// Internal fields
	compiledGlobPattern glob.Glob
}

func (m *azureBlobMetadata) Validate() error {
	// Handle recursive flag, when true, disable delimiter
	if m.Recursive {
		m.BlobDelimiter = ""
	}

	// Compile glob pattern if provided
	if m.GlobPattern != "" {
		compiled, err := glob.Compile(m.GlobPattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %q: %w", m.GlobPattern, err)
		}
		m.compiledGlobPattern = compiled
	}

	// Add delimiter to prefix, if prefix is set and we have a delimiter
	if m.BlobPrefix != "" && m.BlobDelimiter != "" {
		m.BlobPrefix += m.BlobDelimiter
	}

	return nil
}

// NewAzureBlobScaler creates a new azureBlobScaler
func NewAzureBlobScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %w", err)
	}

	logger := InitializeLogger(config, "azure_blob_scaler")

	meta, err := parseAzureBlobMetadata(config, logger)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure blob metadata: %w", err)
	}

	blobClient, err := azure.GetStorageBlobClient(
		logger,
		config.PodIdentity,
		meta.Connection,
		meta.AccountName,
		meta.EndpointSuffix,
		config.GlobalHTTPTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating azure blob client: %w", err)
	}

	return &azureBlobScaler{
		metricType:  metricType,
		metadata:    meta,
		blobClient:  blobClient,
		podIdentity: config.PodIdentity,
		logger:      logger,
	}, nil
}

func parseAzureBlobMetadata(config *scalersconfig.ScalerConfig, logger logr.Logger) (*azureBlobMetadata, error) {
	meta := &azureBlobMetadata{}
	meta.triggerIndex = config.TriggerIndex

	if err := config.TypedConfig(meta); err != nil {
		return nil, fmt.Errorf("error parsing azure blob metadata: %w", err)
	}

	endpointSuffix, err := azure.ParseAzureStorageEndpointSuffix(config.TriggerMetadata, azure.BlobEndpoint)
	if err != nil {
		return nil, err
	}
	meta.EndpointSuffix = endpointSuffix

	switch config.PodIdentity.Provider {
	case "", kedav1alpha1.PodIdentityProviderNone:
		if meta.Connection == "" {
			return nil, fmt.Errorf("connection string is required when not using pod identity)")
		}
		if meta.AccountName != "" {
			logger.Info("accountName is ignored when using connection string authentication")
		}

	case kedav1alpha1.PodIdentityProviderAzureWorkload:
		if meta.AccountName == "" {
			return nil, fmt.Errorf("accountName is required when using Azure Workload Identity")
		}
		if meta.Connection != "" {
			logger.Info("connection string is ignored when using Azure Workload Identity")
		}

	default:
		return nil, fmt.Errorf("pod identity provider %q is not supported for Azure Storage Blobs (supported: none, azure-workload)", config.PodIdentity.Provider)
	}

	return meta, nil
}

func (s *azureBlobScaler) Close(context.Context) error {
	return nil
}

// getAzureBlobListLength returns the count of the blobs in blob container
func (s *azureBlobScaler) getAzureBlobListLength(ctx context.Context) (int64, error) {
	containerClient := s.blobClient.ServiceClient().NewContainerClient(s.metadata.BlobContainerName)

	// If glob pattern is specified, use flat listing to match against all blob names
	if s.metadata.compiledGlobPattern != nil {
		return s.countBlobsWithGlobPattern(ctx, containerClient)
	}

	// Otherwise use hierarchical listing with delimiter support
	return s.countBlobsHierarchical(ctx, containerClient)
}

// countBlobsWithGlobPattern counts blobs matching a glob pattern using flat listing
func (s *azureBlobScaler) countBlobsWithGlobPattern(ctx context.Context, containerClient *container.Client) (int64, error) {
	var count int64

	var prefix *string
	if s.metadata.BlobPrefix != "" {
		prefix = &s.metadata.BlobPrefix
	}

	flatPager := containerClient.NewListBlobsFlatPager(&azblob.ListBlobsFlatOptions{
		Prefix: prefix,
	})

	for flatPager.More() {
		resp, err := flatPager.NextPage(ctx)
		if err != nil {
			return -1, fmt.Errorf("error listing blobs: %w", err)
		}

		for _, blobItem := range resp.Segment.BlobItems {
			if blobItem.Name != nil && s.metadata.compiledGlobPattern.Match(*blobItem.Name) {
				count++
			}
		}
	}

	return count, nil
}

// countBlobsHierarchical counts blobs using hierarchical listing with delimiter support
func (s *azureBlobScaler) countBlobsHierarchical(ctx context.Context, containerClient *container.Client) (int64, error) {
	var count int64

	var prefix *string
	if s.metadata.BlobPrefix != "" {
		prefix = &s.metadata.BlobPrefix
	}

	hierarchyPager := containerClient.NewListBlobsHierarchyPager(
		s.metadata.BlobDelimiter,
		&container.ListBlobsHierarchyOptions{
			Prefix: prefix,
		},
	)

	for hierarchyPager.More() {
		resp, err := hierarchyPager.NextPage(ctx)
		if err != nil {
			return -1, fmt.Errorf("error listing blobs hierarchically: %w", err)
		}
		count += int64(len(resp.Segment.BlobItems))
	}

	return count, nil
}

func (s *azureBlobScaler) GetMetricSpecForScaling(context.Context) []v2.MetricSpec {
	externalMetric := &v2.ExternalMetricSource{
		Metric: v2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(
				s.metadata.triggerIndex,
				kedautil.NormalizeString(fmt.Sprintf("azure-blob-%s", s.metadata.BlobContainerName)),
			),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.TargetBlobCount),
	}
	metricSpec := v2.MetricSpec{External: externalMetric, Type: externalMetricType}
	return []v2.MetricSpec{metricSpec}
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *azureBlobScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	blobCount, err := s.getAzureBlobListLength(ctx)
	if err != nil {
		s.logger.Error(err, "error getting blob list length")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	metric := GenerateMetricInMili(metricName, float64(blobCount))

	return []external_metrics.ExternalMetricValue{metric}, blobCount > s.metadata.ActivationTargetBlobCount, nil
}
