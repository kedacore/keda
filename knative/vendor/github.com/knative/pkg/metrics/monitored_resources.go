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

package metrics

import (
	"github.com/knative/pkg/metrics/metricskey"
)

type gcpMetadata struct {
	project  string
	location string
	cluster  string
}

type KnativeRevision struct {
	Project           string
	Location          string
	ClusterName       string
	NamespaceName     string
	ServiceName       string
	ConfigurationName string
	RevisionName      string
}

func (kr *KnativeRevision) MonitoredResource() (resType string, labels map[string]string) {
	labels = map[string]string{
		metricskey.LabelProject:           kr.Project,
		metricskey.LabelLocation:          kr.Location,
		metricskey.LabelClusterName:       kr.ClusterName,
		metricskey.LabelNamespaceName:     kr.NamespaceName,
		metricskey.LabelServiceName:       kr.ServiceName,
		metricskey.LabelConfigurationName: kr.ConfigurationName,
		metricskey.LabelRevisionName:      kr.RevisionName,
	}
	return "knative_revision", labels
}

type Global struct{}

func (g *Global) MonitoredResource() (resType string, labels map[string]string) {
	return "global", nil
}
