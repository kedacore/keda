/*
Copyright 2019 The Knative Authors
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

package metrics

import (
	"path"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/monitoredresource"
	"github.com/knative/pkg/metrics/metricskey"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.uber.org/zap"
)

// customMetricTypeDomain is the metric type prefix for unsupported metrics by
// resource type knative_revision.
// See: https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.metricDescriptors#MetricDescriptor
const customMetricTypeDomain = "custom.googleapis.com/knative.dev"

var (
	// gcpMetadataFunc is the function used to fetch GCP metadata.
	// In product usage, this is always set to function retrieveGCPMetadata.
	// In unit tests this is set to a fake one to avoid calling GCP metadata
	// service.
	gcpMetadataFunc func() *gcpMetadata
)

func init() {
	// Set gcpMetadataFunc to call GCP metadata service.
	gcpMetadataFunc = retrieveGCPMetadata
}

func newStackdriverExporter(config *metricsConfig, logger *zap.SugaredLogger) (view.Exporter, error) {
	gm := gcpMetadataFunc()
	mtf := getMetricTypeFunc(config.domain, config.component)
	e, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID:               config.stackdriverProjectID,
		GetMetricDisplayName:    mtf, // Use metric type for display name for custom metrics. No impact on built-in metrics.
		GetMetricType:           mtf,
		GetMonitoredResource:    getMonitoredResourceFunc(config.domain, config.component, gm),
		DefaultMonitoringLabels: &stackdriver.Labels{},
	})
	if err != nil {
		logger.Errorw("Failed to create the Stackdriver exporter: ", zap.Error(err))
		return nil, err
	}
	logger.Infof("Created Opencensus Stackdriver exporter with config %v", config)
	return e, nil
}

func getMonitoredResourceFunc(domain, component string, gm *gcpMetadata) func(v *view.View, tags []tag.Tag) ([]tag.Tag, monitoredresource.Interface) {
	return func(view *view.View, tags []tag.Tag) ([]tag.Tag, monitoredresource.Interface) {
		metricType := path.Join(domain, component, view.Measure.Name())
		if metricskey.KnativeRevisionMetrics.Has(metricType) {
			return getKnativeRevisionMonitoredResource(view, tags, gm)
		}
		// Unsupported metric by knative_revision, use "global" resource type.
		return getGlobalMonitoredResource(view, tags)
	}
}

func getKnativeRevisionMonitoredResource(
	v *view.View, tags []tag.Tag, gm *gcpMetadata) ([]tag.Tag, monitoredresource.Interface) {
	tagsMap := getTagsMap(tags)
	kr := &KnativeRevision{
		// The first three resource labels are from metadata.
		Project:     gm.project,
		Location:    gm.location,
		ClusterName: gm.cluster,
		// The rest resource labels are from metrics labels.
		NamespaceName:     valueOrUnknown(metricskey.LabelNamespaceName, tagsMap),
		ServiceName:       valueOrUnknown(metricskey.LabelServiceName, tagsMap),
		ConfigurationName: valueOrUnknown(metricskey.LabelConfigurationName, tagsMap),
		RevisionName:      valueOrUnknown(metricskey.LabelRevisionName, tagsMap),
	}

	var newTags []tag.Tag
	for _, t := range tags {
		// Keep the metrics labels that are not resource labels
		if !metricskey.KnativeRevisionLabels.Has(t.Key.Name()) {
			newTags = append(newTags, t)
		}
	}

	return newTags, kr
}

func getTagsMap(tags []tag.Tag) map[string]string {
	tagsMap := map[string]string{}
	for _, t := range tags {
		tagsMap[t.Key.Name()] = t.Value
	}
	return tagsMap
}

func valueOrUnknown(key string, tagsMap map[string]string) string {
	if value, ok := tagsMap[key]; ok {
		return value
	}
	return metricskey.ValueUnknown
}

func getGlobalMonitoredResource(v *view.View, tags []tag.Tag) ([]tag.Tag, monitoredresource.Interface) {
	return tags, &Global{}
}

func getMetricTypeFunc(domain, component string) func(view *view.View) string {
	return func(view *view.View) string {
		metricType := path.Join(domain, component, view.Measure.Name())
		if metricskey.KnativeRevisionMetrics.Has(metricType) {
			return metricType
		}
		// Unsupported metric by knative_revision, use custom domain.
		return path.Join(customMetricTypeDomain, component, view.Measure.Name())
	}
}
