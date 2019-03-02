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

// configuration.go provides methods to perform actions on the configuration object.

package test

import (
	"testing"

	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	rtesting "github.com/knative/serving/pkg/reconciler/v1alpha1/testing"
)

// Options are test setup parameters.
type Options struct {
	EnvVars                []corev1.EnvVar
	ContainerPorts         []corev1.ContainerPort
	ContainerConcurrency   int
	RevisionTimeoutSeconds int64
	ContainerResources     corev1.ResourceRequirements
	ReadinessProbe         *corev1.Probe
}

// CreateConfiguration create a configuration resource in namespace with the name names.Config
// that uses the image specified by names.Image.
func CreateConfiguration(t *testing.T, clients *Clients, names ResourceNames, options *Options, fopt ...rtesting.ConfigOption) (*v1alpha1.Configuration, error) {
	config := Configuration(ServingNamespace, names, options, fopt...)
	LogResourceObject(t, ResourceObjects{Config: config})
	return clients.ServingClient.Configs.Create(config)
}

// PatchConfigImage patches the existing config passed in with a new imagePath. Returns the latest Configuration object
func PatchConfigImage(clients *Clients, cfg *v1alpha1.Configuration, imagePath string) (*v1alpha1.Configuration, error) {
	newCfg := cfg.DeepCopy()
	newCfg.Spec.RevisionTemplate.Spec.Container.Image = imagePath
	patchBytes, err := createPatch(cfg, newCfg)
	if err != nil {
		return nil, err
	}
	return clients.ServingClient.Configs.Patch(cfg.ObjectMeta.Name, types.JSONPatchType, patchBytes, "")
}

// WaitForConfigLatestRevision takes a revision in through names and compares it to the current state of LatestCreatedRevisionName in Configuration.
// Once an update is detected in the LatestCreatedRevisionName, the function waits for the created revision to be set in LatestReadyRevisionName
// before returning the name of the revision.
func WaitForConfigLatestRevision(clients *Clients, names ResourceNames) (string, error) {
	var revisionName string
	err := WaitForConfigurationState(clients.ServingClient, names.Config, func(c *v1alpha1.Configuration) (bool, error) {
		if c.Status.LatestCreatedRevisionName != names.Revision {
			revisionName = c.Status.LatestCreatedRevisionName
			return true, nil
		}
		return false, nil
	}, "ConfigurationUpdatedWithRevision")
	if err != nil {
		return "", err
	}
	err = WaitForConfigurationState(clients.ServingClient, names.Config, func(c *v1alpha1.Configuration) (bool, error) {
		return (c.Status.LatestReadyRevisionName == revisionName), nil
	}, "ConfigurationReadyWithRevision")

	return revisionName, err
}
