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

package main

import (
	"context"
	"flag"
	"time"

	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/knative/pkg/signals"
	clientset "github.com/knative/serving/test/client/clientset/versioned"
	informers "github.com/knative/serving/test/client/informers/externalversions"
	"github.com/knative/serving/test/reconciler/build"
	"go.uber.org/zap"
)

const (
	threadsPerController = 2
	logLevelKey          = "controller"
)

var (
	masterURL  = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	kubeconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
)

func main() {
	flag.Parse()
	logger := logging.FromContext(context.TODO()).Named("controller")

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfig)
	if err != nil {
		logger.Fatalw("Error building kubeconfig", zap.Error(err))
	}

	testingClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		logger.Fatalw("Error building testing clientset", zap.Error(err))
	}

	testingInformerFactory := informers.NewSharedInformerFactory(testingClient, time.Second*30)

	buildInformer := testingInformerFactory.Testing().V1alpha1().Builds()

	// Build all of our controllers, with the clients constructed above.
	// Add new controllers to this array.
	controllers := []*controller.Impl{
		build.NewController(
			logger,
			testingClient,
			buildInformer,
		),
	}

	// These are non-blocking.
	testingInformerFactory.Start(stopCh)

	// Wait for the caches to be synced before starting controllers.
	logger.Info("Waiting for informer caches to sync")
	for i, synced := range []cache.InformerSynced{
		buildInformer.Informer().HasSynced,
	} {
		if ok := cache.WaitForCacheSync(stopCh, synced); !ok {
			logger.Fatalf("Failed to wait for cache at index %d to sync", i)
		}
	}

	// Start all of the controllers.
	for _, ctrlr := range controllers {
		go func(ctrlr *controller.Impl) {
			// We don't expect this to return until stop is called,
			// but if it does, propagate it back.
			if runErr := ctrlr.Run(threadsPerController, stopCh); runErr != nil {
				logger.Fatalw("Error running controller", zap.Error(runErr))
			}
		}(ctrlr)
	}

	<-stopCh
}
