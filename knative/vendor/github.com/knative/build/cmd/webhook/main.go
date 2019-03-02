/*
Copyright 2017 Google Inc. All Rights Reserved.
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
	"flag"
	"log"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/knative/pkg/configmap"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"
	"github.com/knative/pkg/signals"
	"github.com/knative/pkg/webhook"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/knative/pkg/system"
)

const (
	logLevelKey = "webhook"
)

func main() {

	flag.Parse()
	cm, err := configmap.Load("/etc/config-logging")
	if err != nil {
		log.Fatalf("Error loading logging configuration %v", err)
	}

	config, err := logging.NewConfigFromMap(cm)
	if err != nil {
		log.Fatalf("Error parsing logging configuration: %v", err)
	}
	logger, _ := logging.NewLoggerFromConfig(config, logLevelKey)
	defer logger.Sync()
	logger = logger.With(zap.String(logkey.ControllerType, "webhook"))

	logger.Info("Starting the Configuration Webhook")

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := rest.InClusterConfig()
	if err != nil {
		logger.Fatal("Failed to get in cluster config", zap.Error(err))
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Fatal("Failed to get the client set", zap.Error(err))
	}

	pkgoptions := webhook.ControllerOptions{
		ServiceName:    "build-webhook",
		DeploymentName: "build-webhook",
		Namespace:      system.Namespace(),
		Port:           443,
		SecretName:     "build-webhook-certs",
		WebhookName:    "webhook.build.knative.dev",
	}

	pkgcontroller := webhook.AdmissionController{
		Client:  kubeClient,
		Options: pkgoptions,
		Handlers: map[schema.GroupVersionKind]webhook.GenericCRD{
			v1alpha1.SchemeGroupVersion.WithKind("Build"):                &v1alpha1.Build{},
			v1alpha1.SchemeGroupVersion.WithKind("ClusterBuildTemplate"): &v1alpha1.ClusterBuildTemplate{},
			v1alpha1.SchemeGroupVersion.WithKind("BuildTemplate"):        &v1alpha1.BuildTemplate{},
		},
		Logger: logger,
	}

	pkgcontroller.Run(stopCh)
}
