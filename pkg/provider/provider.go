package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/Kore/pkg/handler"
	"github.com/Azure/azure-storage-queue-go/azqueue"
	"github.com/golang/glog"
	"github.com/kubernetes-incubator/custom-metrics-apiserver/pkg/provider"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type KoreProvider struct {
	client          dynamic.Interface
	mapper          apimeta.RESTMapper
	values          map[provider.CustomMetricInfo]int64
	externalMetrics []externalMetric
	scaleHandler    *handler.ScaleHandler
}
type externalMetric struct {
	info   provider.ExternalMetricInfo
	labels map[string]string
	value  external_metrics.ExternalMetricValue
}

// NewProvider returns an instance of KoreProvider
func NewProvider(client dynamic.Interface, mapper apimeta.RESTMapper, handler *handler.ScaleHandler) provider.MetricsProvider {
	provider := &KoreProvider{
		client:          client,
		mapper:          mapper,
		values:          make(map[provider.CustomMetricInfo]int64),
		externalMetrics: make([]externalMetric, 2, 10),
		scaleHandler:    handler,
	}
	return provider
}

// GetExternalMetric retrieves metrics from the scalers
// Metric is normally identified by a name and a set of labels/tags. It is up to a specific
// implementation how to translate metricSelector to a filter for metric values.
// Namespace can be used by the implementation for metric identification, access control or ignored.
func (p *KoreProvider) GetExternalMetric(namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	// Note:
	//		metric name and namespace is used to lookup for the CRD which contains configuration to call azure
	// 		if not found then ignored and label selector is parsed for all the metrics
	glog.V(0).Infof("Received request for namespace: %s, metric name: %s, metric selectors: %s", namespace, info.Metric, metricSelector.String())

	// _, selectable := metricSelector.Requirements()
	// if !selectable {
	// 	return nil, errors.NewBadRequest("label is set to not selectable. this should not happen")
	// }

	// azMetricRequest, err := p.getMetricRequest(namespace, info.Metric, metricSelector)
	// if err != nil {
	// 	return nil, errors.NewBadRequest(err.Error())
	// }

	// metricValue, err := p.monitorClient.GetAzureMetric(azMetricRequest)
	// if err != nil {
	// 	glog.Errorf("bad request: %v", err)
	// 	return nil, errors.NewBadRequest(err.Error())
	// }

	// queuelen, err := getQueueLength("DefaultEndpointsProtocol=https;AccountName=aarthiskkore;AccountKey=1zDWHlH4spQrvbiMetXetaauSAzYNV33jYw4v2mWvDiF8O/u5z7se7O+OmEaCpqVMl5CWtlT7o7l2UsfRChZaw==;EndpointSuffix=core.windows.net", "testqueue1")
	// if err != nil {
	// 	glog.Errorf("Error when getting Queue length " + err.Error())
	// }

	externalmetrics, error := p.scaleHandler.GetScaledObjectMetrics(namespace, metricSelector, info.Metric)
	if error != nil {
		return nil, error
	}

	matchingMetrics := []external_metrics.ExternalMetricValue{}
	matchingMetrics = append(matchingMetrics, externalmetrics...)

	return &external_metrics.ExternalMetricValueList{
		Items: matchingMetrics,
	}, nil

}

func (p *KoreProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	externalMetricsInfo := []provider.ExternalMetricInfo{}

	// not implemented yet

	// TODO
	// iterate over all of the resources we have access
	// build metric info from https://docs.microsoft.com/en-us/azure/monitoring-and-diagnostics/monitoring-rest-api-walkthrough#retrieve-metric-definitions-multi-dimensional-api
	// important to remember to cache this and only get it at given interval

	return externalMetricsInfo
}

// GetMetricByName fetches a particular metric for a particular object.
// The namespace will be empty if the metric is root-scoped.
func (p *KoreProvider) GetMetricByName(name types.NamespacedName, info provider.CustomMetricInfo) (*custom_metrics.MetricValue, error) {
	// not implemented yet
	return nil, apiErrors.NewServiceUnavailable("not implemented yet")
}

// GetMetricBySelector fetches a particular metric for a set of objects matching
// the given label selector.  The namespace will be empty if the metric is root-scoped.
func (p *KoreProvider) GetMetricBySelector(namespace string, selector labels.Selector, info provider.CustomMetricInfo) (*custom_metrics.MetricValueList, error) {
	glog.V(0).Infof("Received request for custom metric: groupresource: %s, namespace: %s, metric name: %s, selectors: %s", info.GroupResource.String(), namespace, info.Metric, selector.String())
	return nil, apiErrors.NewServiceUnavailable("not implemented yet")
}

// ListAllMetrics provides a list of all available metrics at
// the current time.  Note that this is not allowed to return
// an error, so it is reccomended that implementors cache and
// periodically update this list, instead of querying every time.
func (p *KoreProvider) ListAllMetrics() []provider.CustomMetricInfo {
	// not implemented yet
	return []provider.CustomMetricInfo{}
}

func getQueueLength(connectionString, queueName string) (int32, error) {
	// From the Azure portal, get your Storage account's name and account key.
	accountName, accountKey, err := accountInfo(connectionString)

	if err != nil {
		return -1, err
	}

	// Use your Storage account's name and key to create a credential object; this is used to access your account.
	credential, err := azqueue.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return -1, err
	}

	// Create a request pipeline that is used to process HTTP(S) requests and responses. It requires
	// your account credentials. In more advanced scenarios, you can configure telemetry, retry policies,
	// logging, and other options. Also, you can configure multiple request pipelines for different scenarios.
	p := azqueue.NewPipeline(credential, azqueue.PipelineOptions{})

	// From the Azure portal, get your Storage account queue service URL endpoint.
	// The URL typically looks like this:
	// https throws in aks, investigate
	u, _ := url.Parse(fmt.Sprintf("http://%s.queue.core.windows.net", accountName))

	// Create an ServiceURL object that wraps the service URL and a request pipeline.
	serviceURL := azqueue.NewServiceURL(*u, p)

	// Now, you can use the serviceURL to perform various queue operations.

	// All HTTP operations allow you to specify a Go context.Context object to control cancellation/timeout.
	ctx := context.TODO() // This example uses a never-expiring context.

	// Create a URL that references a queue in your Azure Storage account.
	// This returns a QueueURL object that wraps the queue's URL and a request pipeline (inherited from serviceURL)
	queueURL := serviceURL.NewQueueURL(queueName) // Queue names require lowercase

	// The code below shows how a client or server can determine the approximate count of messages in the queue:
	props, err := queueURL.GetProperties(ctx)
	if err != nil {
		return -1, err
	}

	return props.ApproximateMessagesCount(), nil
}

func accountInfo(connectionString string) (string, string, error) {
	parts := strings.Split(connectionString, ";")

	var name, key string
	for _, v := range parts {
		if strings.HasPrefix(v, "AccountName") {
			accountParts := strings.SplitN(v, "=", 2)
			if len(accountParts) == 2 {
				name = accountParts[1]
			}
		} else if strings.HasPrefix(v, "AccountKey") {
			keyParts := strings.SplitN(v, "=", 2)
			if len(keyParts) == 2 {
				key = keyParts[1]
			}
		}
	}
	if name == "" || key == "" {
		return "", "", errors.New("Can't parse connection string")
	}

	return name, key, nil
}
