/*
Copyright 2018 The Knative Authors
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

package metricskey

import "k8s.io/apimachinery/pkg/util/sets"

const (
	// ResourceTypeKnativeRevision is the Stackdriver resource type for Knative revision
	ResourceTypeKnativeRevision = "knative_revision"

	// LabelProject is the label for project (e.g. GCP GAIA ID, AWS project name)
	LabelProject = "project_id"

	// LabelLocation is the label for location (e.g. GCE zone, AWS region) where the service is deployed
	LabelLocation = "location"

	// LabelClusterName is the label for immutable name of the cluster
	LabelClusterName = "cluster_name"

	// LabelNamespaceName is the label for immutable name of the namespace that the service is deployed
	LabelNamespaceName = "namespace_name"

	// LabelServiceName is the label for the deployed service name
	LabelServiceName = "service_name"

	// LabelRouteName is the label for immutable name of the route that receives the request
	LabelRouteName = "route_name"

	// LabelConfigurationName is the label for the configuration which created the monitored revision
	LabelConfigurationName = "configuration_name"

	// LabelRevisionName is the label for the monitored revision
	LabelRevisionName = "revision_name"

	// ValueUnknown is the default value if the field is unknown, e.g. project will be unknown if Knative
	// is not running on GKE.
	ValueUnknown = "unknown"
)

var (
	// KnativeRevisionLabels stores the set of resource labels for resource type knative_revision.
	// LabelRouteName is added as extra label since it is optional, not in this map.
	KnativeRevisionLabels = sets.NewString(
		LabelProject,
		LabelLocation,
		LabelClusterName,
		LabelNamespaceName,
		LabelServiceName,
		LabelConfigurationName,
		LabelRevisionName,
	)

	// KnativeRevisionMetrics stores a set of metric types which are supported
	// by resource type knative_revision.
	KnativeRevisionMetrics = sets.NewString(
		"knative.dev/serving/activator/request_count",
		"knative.dev/serving/activator/request_latencies",
		"knative.dev/serving/autoscaler/desired_pods",
		"knative.dev/serving/autoscaler/requested_pods",
		"knative.dev/serving/autoscaler/actual_pods",
		"knative.dev/serving/autoscaler/stable_request_concurrency",
		"knative.dev/serving/autoscaler/panic_request_concurrency",
		"knative.dev/serving/autoscaler/target_concurrency_per_pod",
	)
)
