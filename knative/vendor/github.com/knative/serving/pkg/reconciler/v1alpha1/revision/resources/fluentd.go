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

package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/knative/pkg/kmeta"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/revision/config"
	"github.com/knative/serving/pkg/reconciler/v1alpha1/revision/resources/names"
)

// fluentdSidecarPreOutputConfig defines source and filter configurations for
// files under /var/log.
const fluentdSidecarPreOutputConfig = `
<source>
	@type tail
	path /var/log/revisions/**/*.*
	pos_file /var/log/varlog.log.pos
	tag raw.*
	format none
	message_key log
	read_from_head true
</source>

<filter raw.var.log.**>
	@type record_transformer
	enable_ruby true
	<record>
	  kubernetes ${ {"container_name": "#{ENV['SERVING_CONTAINER_NAME']}", "namespace_name": "#{ENV['SERVING_NAMESPACE']}", "pod_name": "#{ENV['SERVING_POD_NAME']}", "labels": {"knative_dev/configuration": "#{ENV['SERVING_CONFIGURATION']}", "knative_dev/revision": "#{ENV['SERVING_REVISION']}"} } }
		stream varlog
		# Line breaks may be trimmed when collecting from files. Add them back so that
		# multi line logs are still in multi line after combined by detect_exceptions.
		# Remove this if https://github.com/GoogleCloudPlatform/fluent-plugin-detect-exceptions/pull/10 is released
		log ${ if record["log"].end_with?("\n") then record["log"] else record["log"] + "\n" end }
	</record>
</filter>

<match raw.var.log.**>
	@id raw.var.log
	@type detect_exceptions
	remove_tag_prefix raw
	message log
	stream stream
	multiline_flush_interval 5
	max_bytes 500000
	max_lines 1000
</match>

`

const fluentdConfigMapVolumeName = "configmap"

var (
	fluentdResources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: fluentdContainerCPU,
		},
	}

	fluentdVolumeMounts = []corev1.VolumeMount{{
		Name:      varLogVolumeName,
		MountPath: "/var/log/revisions",
	}, {
		Name:      fluentdConfigMapVolumeName,
		MountPath: "/etc/fluent/config.d",
	}}
)

func MakeFluentdConfigMap(rev *v1alpha1.Revision, observabilityConfig *config.Observability) *corev1.ConfigMap {
	varlogConf := fluentdSidecarPreOutputConfig + observabilityConfig.FluentdSidecarOutputConfig
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            names.FluentdConfigMap(rev),
			Namespace:       rev.Namespace,
			Labels:          makeLabels(rev),
			Annotations:     makeAnnotations(rev),
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(rev)},
		},
		Data: map[string]string{
			"varlog.conf": varlogConf,
		},
	}
}

func makeFluentdConfigMapVolume(rev *v1alpha1.Revision) *corev1.Volume {
	return &corev1.Volume{
		Name: fluentdConfigMapVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: names.FluentdConfigMap(rev),
				},
			},
		},
	}
}

func makeFluentdContainer(rev *v1alpha1.Revision, observabilityConfig *config.Observability) *corev1.Container {
	configName := ""
	if owner := metav1.GetControllerOf(rev); owner != nil && owner.Kind == "Configuration" {
		configName = owner.Name
	}

	return &corev1.Container{
		Name:      FluentdContainerName,
		Image:     observabilityConfig.FluentdSidecarImage,
		Resources: fluentdResources,
		Env: []corev1.EnvVar{{
			Name:  "FLUENTD_ARGS",
			Value: "--no-supervisor -q",
		}, {
			Name:  "SERVING_CONTAINER_NAME",
			Value: UserContainerName,
		}, {
			Name:  "SERVING_CONFIGURATION",
			Value: configName,
		}, {
			Name:  "SERVING_REVISION",
			Value: rev.Name,
		}, {
			Name:  "SERVING_NAMESPACE",
			Value: rev.Namespace,
		}, {
			Name: "SERVING_POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		}},
		VolumeMounts: fluentdVolumeMounts,
	}
}
