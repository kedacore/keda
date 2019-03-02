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

package activator

import (
	"fmt"
	"net/http"
	"time"

	"github.com/knative/pkg/logging/logkey"
	"github.com/knative/serving/pkg/apis/serving"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	clientset "github.com/knative/serving/pkg/client/clientset/versioned"
	revisionresources "github.com/knative/serving/pkg/reconciler/v1alpha1/revision/resources"
	revisionresourcenames "github.com/knative/serving/pkg/reconciler/v1alpha1/revision/resources/names"
	"github.com/knative/serving/pkg/utils"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ Activator = (*revisionActivator)(nil)

type revisionActivator struct {
	readyTimout time.Duration // For testing.
	kubeClient  kubernetes.Interface
	knaClient   clientset.Interface
	logger      *zap.SugaredLogger
}

// NewRevisionActivator creates an Activator that changes revision
// serving status to active if necessary, then returns the endpoint
// once the revision is ready to serve traffic.
func NewRevisionActivator(kubeClient kubernetes.Interface, servingClient clientset.Interface, logger *zap.SugaredLogger) Activator {
	return &revisionActivator{
		readyTimout: 60 * time.Second,
		kubeClient:  kubeClient,
		knaClient:   servingClient,
		logger:      logger,
	}
}

func (r *revisionActivator) Shutdown() {
	// Nothing to do.
}

func (r *revisionActivator) activateRevision(namespace, name, key string) (*v1alpha1.Revision, error) {
	logger := r.logger.With(zap.String(logkey.Key, key))
	rev := RevisionID{
		Namespace: namespace,
		Name:      name,
	}

	// Get the current revision serving state
	revisionClient := r.knaClient.ServingV1alpha1().Revisions(rev.Namespace)
	revision, err := revisionClient.Get(rev.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get the revision")
	}

	// Wait for the revision to not require activation.
	if revision.Status.IsActivationRequired() {
		wi, err := r.knaClient.ServingV1alpha1().Revisions(rev.Namespace).Watch(metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", rev.Name),
		})
		if err != nil {
			return nil, errors.New("failed to watch the revision")
		}
		defer wi.Stop()
		ch := wi.ResultChan()
	RevisionActive:
		for {
			select {
			case <-time.After(r.readyTimout):
				// last chance to check
				if !revision.Status.IsActivationRequired() {
					break RevisionActive
				}
				return nil, errors.New("timeout waiting for the revision to become ready")
			case event := <-ch:
				if revision, ok := event.Object.(*v1alpha1.Revision); ok {
					if revision.Status.IsActivationRequired() {
						logger.Infof("Revision %s is not yet ready", name)
						continue
					} else {
						logger.Infof("Revision %s is ready", name)
					}
					break RevisionActive
				} else {
					return nil, fmt.Errorf("unexpected result type for the revision: %v", event)
				}
			}
		}
	}
	return revision, nil
}

func (r *revisionActivator) revisionEndpoint(revision *v1alpha1.Revision) (end Endpoint, err error) {
	services := r.kubeClient.CoreV1().Services(revision.GetNamespace())
	serviceName := revisionresourcenames.K8sService(revision)
	svc, err := services.Get(serviceName, metav1.GetOptions{})
	if err != nil {
		return end, errors.Wrapf(err, "unable to get service %s for revision", serviceName)
	}

	fqdn := fmt.Sprintf("%s.%s.svc.%s", serviceName, revision.Namespace, utils.GetClusterDomainName())

	// Search for the correct port in all the service ports.
	port := int32(-1)
	for _, p := range svc.Spec.Ports {
		if p.Name == revisionresources.ServicePortName(revision) {
			port = p.Port
			break
		}
	}
	if port == -1 {
		return end, errors.New("revision needs external HTTP port")
	}

	return Endpoint{
		FQDN: fqdn,
		Port: port,
	}, nil
}

// ActiveEndpoint activates the revision `name` and returnts the result.
func (r *revisionActivator) ActiveEndpoint(namespace, name string) ActivationResult {
	key := fmt.Sprintf("%s/%s", namespace, name)
	logger := r.logger.With(zap.String(logkey.Key, key))
	revision, err := r.activateRevision(namespace, name, key)
	if err != nil {
		logger.Errorw("Failed to activate the revision.", zap.Error(err))
		return ActivationResult{
			Status: http.StatusInternalServerError,
			Error:  err,
		}
	}

	serviceName, configurationName := getServiceAndConfigurationLabels(revision)
	endpoint, err := r.revisionEndpoint(revision)
	if err != nil {
		logger.Errorw("Failed to get revision endpoint.", zap.Error(err))
		return ActivationResult{
			Status:            http.StatusInternalServerError,
			ServiceName:       serviceName,
			ConfigurationName: configurationName,
			Error:             err,
		}
	}

	return ActivationResult{
		Status:            http.StatusOK,
		Endpoint:          endpoint,
		ServiceName:       serviceName,
		ConfigurationName: configurationName,
		Error:             nil,
	}
}

func getServiceAndConfigurationLabels(rev *v1alpha1.Revision) (string, string) {
	if rev.Labels == nil {
		return "", ""
	}
	return rev.Labels[serving.ServiceLabelKey], rev.Labels[serving.ConfigurationLabelKey]
}
