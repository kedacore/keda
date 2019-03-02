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
	pipelineversioned "github.com/knative/build-pipeline/pkg/client/clientset/versioned"
	pipelinev1alpha1 "github.com/knative/build-pipeline/pkg/client/clientset/versioned/typed/pipeline/v1alpha1"
	"github.com/knative/pkg/test"
	"github.com/knative/serving/pkg/client/clientset/versioned"
	servingtyped "github.com/knative/serving/pkg/client/clientset/versioned/typed/serving/v1alpha1"
	testbuildtyped "github.com/knative/serving/test/client/clientset/versioned/typed/testing/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Clients holds instances of interfaces for making requests to Knative Serving.
type Clients struct {
	KubeClient     *test.KubeClient
	ServingClient  *ServingClients
	BuildClient    *BuildClient
	PipelineClient *PipelineClient
	Dynamic        dynamic.Interface
}

// BuildClient holds instances of interfaces for making requests to build client.
type BuildClient struct {
	TestBuilds testbuildtyped.BuildInterface
}

// PipelineClient holds instances of interfaces for making requests to pipeline client.
type PipelineClient struct {
	Pipeline         pipelinev1alpha1.PipelineInterface
	Task             pipelinev1alpha1.TaskInterface
	TaskRun          pipelinev1alpha1.TaskRunInterface
	PipelineRun      pipelinev1alpha1.PipelineRunInterface
	PipelineResource pipelinev1alpha1.PipelineResourceInterface
}

// ServingClients holds instances of interfaces for making requests to knative serving clients
type ServingClients struct {
	Routes    servingtyped.RouteInterface
	Configs   servingtyped.ConfigurationInterface
	Revisions servingtyped.RevisionInterface
	Services  servingtyped.ServiceInterface
}

// NewClients instantiates and returns several clientsets required for making request to the
// Knative Serving cluster specified by the combination of clusterName and configPath. Clients can
// make requests within namespace.
func NewClients(configPath string, clusterName string, namespace string) (*Clients, error) {
	clients := &Clients{}
	cfg, err := buildClientConfig(configPath, clusterName)
	if err != nil {
		return nil, err
	}

	// We poll, so set our limits high.
	cfg.QPS = 100
	cfg.Burst = 200

	clients.KubeClient, err = test.NewKubeClient(configPath, clusterName)
	if err != nil {
		return nil, err
	}

	clients.BuildClient, err = newBuildClient(cfg, namespace)
	if err != nil {
		return nil, err
	}

	clients.PipelineClient, err = newPipelineClients(cfg, namespace)
	if err != nil {
		return nil, err
	}

	clients.ServingClient, err = newServingClients(cfg, namespace)
	if err != nil {
		return nil, err
	}

	clients.Dynamic, err = dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return clients, nil
}

// NewBuildclient instantiates and returns several clientsets required for making request to the
// build client specified by the combination of clusterName and configPath. Clients can make requests within namespace.
func newBuildClient(cfg *rest.Config, namespace string) (*BuildClient, error) {
	tcs, err := testbuildtyped.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &BuildClient{
		TestBuilds: tcs.Builds(namespace),
	}, nil
}

// NewPipelineClients instantiates and returns several clientsets required for making request to the
// pipeline client.
func newPipelineClients(cfg *rest.Config, namespace string) (*PipelineClient, error) {
	pcs, err := pipelineversioned.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &PipelineClient{
		Pipeline:         pcs.PipelineV1alpha1().Pipelines(namespace),
		PipelineResource: pcs.PipelineV1alpha1().PipelineResources(namespace),
		PipelineRun:      pcs.PipelineV1alpha1().PipelineRuns(namespace),
		Task:             pcs.PipelineV1alpha1().Tasks(namespace),
		TaskRun:          pcs.PipelineV1alpha1().TaskRuns(namespace),
	}, nil
}

// NewServingClients instantiates and returns the serving clientset required to make requests to the
// knative serving cluster.
func newServingClients(cfg *rest.Config, namespace string) (*ServingClients, error) {
	cs, err := versioned.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &ServingClients{
		Configs:   cs.ServingV1alpha1().Configurations(namespace),
		Revisions: cs.ServingV1alpha1().Revisions(namespace),
		Routes:    cs.ServingV1alpha1().Routes(namespace),
		Services:  cs.ServingV1alpha1().Services(namespace),
	}, nil
}

// Delete will delete all Routes and Configs with the names routes and configs, if clients
// has been successfully initialized.
func (clients *ServingClients) Delete(routes []string, configs []string, services []string) error {
	deletions := []struct {
		client interface {
			Delete(name string, options *v1.DeleteOptions) error
		}
		items []string
	}{
		{clients.Routes, routes},
		{clients.Configs, configs},
		{clients.Services, services},
	}

	propPolicy := v1.DeletePropagationForeground
	dopt := &v1.DeleteOptions{
		PropagationPolicy: &propPolicy,
	}

	for _, deletion := range deletions {
		if deletion.client == nil {
			continue
		}

		for _, item := range deletion.items {
			if item == "" {
				continue
			}

			if err := deletion.client.Delete(item, dopt); err != nil {
				return err
			}
		}
	}

	return nil
}

func buildClientConfig(kubeConfigPath string, clusterName string) (*rest.Config, error) {
	overrides := clientcmd.ConfigOverrides{}
	// Override the cluster name if provided.
	if clusterName != "" {
		overrides.Context.Cluster = clusterName
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		&overrides).ClientConfig()
}
