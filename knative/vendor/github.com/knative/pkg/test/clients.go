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

// This file contains an object which encapsulates k8s clients which are useful for e2e tests.

package test

import (
	"github.com/knative/pkg/test/logging"
	"github.com/knative/pkg/test/spoof"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8styped "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeClient holds instances of interfaces for making requests to kubernetes client.
type KubeClient struct {
	Kube *kubernetes.Clientset
}

// NewSpoofingClient returns a spoofing client to make requests
func NewSpoofingClient(client *KubeClient, logf logging.FormatLogger, domain string, resolvable bool) (*spoof.SpoofingClient, error) {
	return spoof.New(client.Kube, logf, domain, resolvable, Flags.IngressEndpoint)
}

// NewKubeClient instantiates and returns several clientsets required for making request to the
// kube client specified by the combination of clusterName and configPath. Clients can make requests within namespace.
func NewKubeClient(configPath string, clusterName string) (*KubeClient, error) {
	cfg, err := BuildClientConfig(configPath, clusterName)
	if err != nil {
		return nil, err
	}

	k, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &KubeClient{Kube: k}, nil
}

// BuildClientConfig builds the client config specified by the config path and the cluster name
func BuildClientConfig(kubeConfigPath string, clusterName string) (*rest.Config, error) {
	overrides := clientcmd.ConfigOverrides{}
	// Override the cluster name if provided.
	if clusterName != "" {
		overrides.Context.Cluster = clusterName
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		&overrides).ClientConfig()
}

// UpdateConfigMap updates the config map for specified @name with values
func (client *KubeClient) UpdateConfigMap(name string, configName string, values map[string]string) error {
	configMap, err := client.getConfigMap(name).Get(configName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	for key, value := range values {
		configMap.Data[key] = value
	}

	_, err = client.getConfigMap(name).Update(configMap)
	return err
}

// getConfigMap gets the knative serving config map.
func (client *KubeClient) getConfigMap(name string) k8styped.ConfigMapInterface {
	return client.Kube.CoreV1().ConfigMaps(name)
}
