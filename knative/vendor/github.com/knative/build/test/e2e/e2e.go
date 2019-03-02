// +build e2e

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

package e2e

import (
	"errors"
	"flag"
	"fmt"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	buildversioned "github.com/knative/build/pkg/client/clientset/versioned"
	buildtyped "github.com/knative/build/pkg/client/clientset/versioned/typed/build/v1alpha1"
	"github.com/knative/pkg/test"
	"github.com/knative/pkg/test/logging"
	corev1 "k8s.io/api/core/v1"
	kuberrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

type clients struct {
	kubeClient  *test.KubeClient
	buildClient *buildClient
}

var (
	// Sentinel error from watchBuild when the build failed.
	errBuildFailed = errors.New("build failed")

	// Sentinal error from watchBuild when watch timed out before build finished
	errWatchTimeout = errors.New("watch ended before build finished")
)

func teardownNamespace(clients *clients, buildTestNamespace string, logger *logging.BaseLogger) {
	if clients != nil && clients.kubeClient != nil {
		logger.Infof("Deleting namespace %q", buildTestNamespace)

		if err := clients.kubeClient.Kube.CoreV1().Namespaces().Delete(buildTestNamespace, &metav1.DeleteOptions{}); err != nil && !kuberrors.IsNotFound(err) {
			logger.Fatalf("Error deleting namespace %q: %v", buildTestNamespace, err)
		}
	}
}

func teardownBuild(clients *clients, logger *logging.BaseLogger, buildTestNamespace, name string) {
	if clients != nil && clients.buildClient != nil {
		logger.Infof("Deleting build %q in namespace %q", name, buildTestNamespace)

		if err := clients.buildClient.builds.Delete(name, &metav1.DeleteOptions{}); err != nil && !kuberrors.IsNotFound(err) {
			logger.Fatalf("Error deleting build %q: %v", name, err)
		}
	}
}

func teardownClusterTemplate(clients *clients, logger *logging.BaseLogger, name string) {
	if clients != nil && clients.buildClient != nil {
		logger.Infof("Deleting cluster template %q", name)

		if err := clients.buildClient.clusterTemplates.Delete(name, &metav1.DeleteOptions{}); err != nil && !kuberrors.IsNotFound(err) {
			logger.Fatalf("Error deleting cluster template %q: %v", name, err)
		}
	}
}

func buildClients(buildTestNamespace string, logger *logging.BaseLogger) *clients {
	clients, err := newClients(test.Flags.Kubeconfig, test.Flags.Cluster, buildTestNamespace)
	if err != nil {
		logger.Fatalf("Error creating newClients: %v", err)
	}
	return clients
}

func createTestNamespace(logger *logging.BaseLogger) (string, *clients) {
	buildTestNamespace := AppendRandomString("build-tests")
	clients := buildClients(buildTestNamespace, logger)

	// Ensure the test namespace exists, by trying to create it and ignoring
	// already-exists errors.
	if _, err := clients.kubeClient.Kube.CoreV1().Namespaces().Create(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildTestNamespace,
		},
	}); err == nil {
		logger.Infof("Created namespace %q", buildTestNamespace)
	} else if kuberrors.IsAlreadyExists(err) {
		logger.Infof("Namespace %q already exists", buildTestNamespace)
	} else {
		logger.Fatalf("Error creating namespace %q: %v", buildTestNamespace, err)
	}
	return buildTestNamespace, clients
}

func newClients(configPath string, clusterName string, namespace string) (*clients, error) {
	overrides := clientcmd.ConfigOverrides{}
	// Override the cluster name if provided.
	if clusterName != "" {
		overrides.Context.Cluster = clusterName
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(&clientcmd.ClientConfigLoadingRules{
		ExplicitPath: configPath,
	}, &overrides).ClientConfig()
	if err != nil {
		return nil, err
	}

	kubeClient, err := test.NewKubeClient(configPath, clusterName)
	if err != nil {
		return nil, err
	}

	bcs, err := buildversioned.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	buildClient := &buildClient{
		builds:           bcs.BuildV1alpha1().Builds(namespace),
		buildTemplates:   bcs.BuildV1alpha1().BuildTemplates(namespace),
		clusterTemplates: bcs.BuildV1alpha1().ClusterBuildTemplates(),
	}

	return &clients{
		kubeClient:  kubeClient,
		buildClient: buildClient,
	}, nil
}

type buildClient struct {
	builds           buildtyped.BuildInterface
	buildTemplates   buildtyped.BuildTemplateInterface
	clusterTemplates buildtyped.ClusterBuildTemplateInterface
}

func (c *buildClient) watchBuild(name string) (*v1alpha1.Build, error) {
	ls := metav1.SingleObject(metav1.ObjectMeta{Name: name})
	// TODO: Update watchBuild function to take this as parameter depending on test requirements

	// Any build that takes longer than this timeout will result in
	// errWatchTimeout.
	var timeout int64 = 120
	ls.TimeoutSeconds = &timeout

	w, err := c.builds.Watch(ls)
	if err != nil {
		return nil, err
	}
	var latest *v1alpha1.Build
	for evt := range w.ResultChan() {
		switch evt.Type {
		case watch.Deleted:
			return nil, errors.New("build deleted")
		case watch.Error:
			return nil, fmt.Errorf("error event: %v", evt.Object)
		}

		b, ok := evt.Object.(*v1alpha1.Build)
		if !ok {
			return nil, fmt.Errorf("object was not a Build: %v", err)
		}
		latest = b

		for _, cond := range b.Status.Conditions {
			if cond.Type == v1alpha1.BuildSucceeded {
				switch cond.Status {
				case corev1.ConditionTrue:
					return b, nil
				case corev1.ConditionFalse:
					return b, errBuildFailed
				case corev1.ConditionUnknown:
					continue
				}
			}
		}
	}
	return latest, errWatchTimeout
}

// initialize is responsible for setting up and tearing down the testing environment,
// namely the test namespace.
func initialize(contextName string) (string, *logging.BaseLogger, *clients) {
	flag.Parse()
	logging.InitializeLogger(test.Flags.LogVerbose)
	logger := logging.GetContextLogger("initialize")
	flag.Set("alsologtostderr", "true")
	if test.Flags.EmitMetrics {
		logging.InitializeMetricExporter()
	}

	buildTestNamespace, clients := createTestNamespace(logger)

	// Cleanup namespace
	test.CleanupOnInterrupt(func() { teardownNamespace(clients, buildTestNamespace, logger) }, logger)

	testLogger := logging.GetContextLogger(contextName)
	return buildTestNamespace, testLogger, buildClients(buildTestNamespace, testLogger)
}
