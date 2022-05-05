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
	"net/http"
	"strconv"

	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers/azure"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

const (
	queueLengthMetricName    = "queueLength"
	defaultTargetQueueLength = 5
	externalMetricType       = "External"
)

type azureQueueScaler struct {
	metricType  v2beta2.MetricTargetType
	metadata    *azureQueueMetadata
	podIdentity kedav1alpha1.AuthPodIdentity
	httpClient  *http.Client
}

type azureQueueMetadata struct {
	targetQueueLength int64
	queueName         string
	connection        string
	accountName       string
	endpointSuffix    string
	scalerIndex       int
}

var azureQueueLog = logf.Log.WithName("azure_queue_scaler")

// NewAzureQueueScaler creates a new scaler for queue
func NewAzureQueueScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting scaler metric type: %s", err)
	}

	meta, podIdentity, err := parseAzureQueueMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing azure queue metadata: %s", err)
	}

	return &azureQueueScaler{
		metricType:  metricType,
		metadata:    meta,
		podIdentity: podIdentity,
		httpClient:  kedautil.CreateHTTPClient(config.GlobalHTTPTimeout, false),
	}, nil
}

func parseAzureQueueMetadata(config *ScalerConfig) (*azureQueueMetadata, kedav1alpha1.AuthPodIdentity, error) {
	meta := azureQueueMetadata{}
	meta.targetQueueLength = defaultTargetQueueLength

	if val, ok := config.TriggerMetadata[queueLengthMetricName]; ok {
		queueLength, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			azureQueueLog.Error(err, "Error parsing azure queue metadata", "queueLengthMetricName", queueLengthMetricName)
			return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("error parsing azure queue metadata %s: %s", queueLengthMetricName, err.Error())
		}

		meta.targetQueueLength = queueLength
	}

	endpointSuffix, err := azure.ParseAzureStorageEndpointSuffix(config.TriggerMetadata, azure.QueueEndpoint)
	if err != nil {
		return nil, kedav1alpha1.AuthPodIdentity{}, err
	}

	meta.endpointSuffix = endpointSuffix

	if val, ok := config.TriggerMetadata["queueName"]; ok && val != "" {
		meta.queueName = val
	} else {
		return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("no queueName given")
	}

	// before triggerAuthentication CRD, pod identity was configured using this property
	if val, ok := config.TriggerMetadata["useAAdPodIdentity"]; ok && config.PodIdentity.Provider == "" {
		if val == "true" {
			config.PodIdentity = kedav1alpha1.AuthPodIdentity{Provider: "azure"}
		}
	}

	// If the Use AAD Pod Identity is not present, or set to "none"
	// then check for connection string
	switch config.PodIdentity.Provider {
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
			return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("no connection setting given")
		}
	case kedav1alpha1.PodIdentityProviderAzure:
		// If the Use AAD Pod Identity is present then check account name
		if val, ok := config.TriggerMetadata["accountName"]; ok && val != "" {
			meta.accountName = val
		} else {
			return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("no accountName given")
		}
	default:
		return nil, kedav1alpha1.AuthPodIdentity{}, fmt.Errorf("pod identity %s not supported for azure storage queues", config.PodIdentity)
	}

	meta.scalerIndex = config.ScalerIndex

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
		s.metadata.endpointSuffix,
	)

	if err != nil {
		azureQueueLog.Error(err, "error)")
		return false, err
	}

	return length > 0, nil
}

func (s *azureQueueScaler) Close(context.Context) error {
	return nil
}

func (s *azureQueueScaler) GetMetricSpecForScaling(context.Context) []v2beta2.MetricSpec {
	externalMetric := &v2beta2.ExternalMetricSource{
		Metric: v2beta2.MetricIdentifier{
			Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, kedautil.NormalizeString(fmt.Sprintf("azure-queue-%s", s.metadata.queueName))),
		},
		Target: GetMetricTarget(s.metricType, s.metadata.targetQueueLength),
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
		s.metadata.endpointSuffix,
	)

	if err != nil {
		azureQueueLog.Error(err, "error getting queue length")
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      *resource.NewQuantity(queuelen, resource.DecimalSI),
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}
