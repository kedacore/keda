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

// service.go provides methods to perform actions on the service resource.

package test

import (
	"fmt"
	"testing"

	"github.com/knative/pkg/apis/duck"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	serviceresourcenames "github.com/knative/serving/pkg/reconciler/v1alpha1/service/resources/names"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	rtesting "github.com/knative/serving/pkg/reconciler/v1alpha1/testing"
)

// TODO(dangerd): Move function to duck.CreateBytePatch
func createPatch(cur, desired interface{}) ([]byte, error) {
	patch, err := duck.CreatePatch(cur, desired)
	if err != nil {
		return nil, err
	}
	return patch.MarshalJSON()
}

func validateCreatedServiceStatus(clients *Clients, names *ResourceNames) error {
	return CheckServiceState(clients.ServingClient, names.Service, func(s *v1alpha1.Service) (bool, error) {
		if s.Status.Domain == "" {
			return false, fmt.Errorf("domain is not present in Service status: %v", s)
		}
		names.Domain = s.Status.Domain
		if s.Status.LatestCreatedRevisionName == "" {
			return false, fmt.Errorf("lastCreatedRevision is not present in Service status: %v", s)
		}
		names.Revision = s.Status.LatestCreatedRevisionName
		if s.Status.DeprecatedDomainInternal == "" {
			return false, fmt.Errorf("domainInternal is not present in Service status: %v", s)
		}
		if s.Status.LatestReadyRevisionName == "" {
			return false, fmt.Errorf("lastReadyRevision is not present in Service status: %v", s)
		}
		if s.Status.LatestReadyRevisionName == "" {
			return false, fmt.Errorf("lastReadyRevision is not present in Service status: %v", s)
		}
		if s.Status.ObservedGeneration != 1 {
			return false, fmt.Errorf("observedGeneration is not 1 in Service status: %v", s)
		}
		return true, nil
	})
}

// GetResourceObjects obtains the services resources from the k8s API server.
func GetResourceObjects(clients *Clients, names ResourceNames) (*ResourceObjects, error) {
	routeObject, err := clients.ServingClient.Routes.Get(names.Route, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	serviceObject, err := clients.ServingClient.Services.Get(names.Service, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	configObject, err := clients.ServingClient.Configs.Get(names.Config, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	revisionObject, err := clients.ServingClient.Revisions.Get(names.Revision, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return &ResourceObjects{
		Route:    routeObject,
		Service:  serviceObject,
		Config:   configObject,
		Revision: revisionObject,
	}, nil
}

// CreateReleaseServiceWithLatest creates a `Release` service using `@latest`
// as the only revision.
// This function expects `Service` and `Image` name passed in through `names`.
// Names is updated with the `Route` and `Configuration` created by the Service
// and `ResourceObjects` is returned with the `Service`, `Route`, and `Configuration` objects.
// Returns an error if the service does not come up correctly.
func CreateReleaseServiceWithLatest(
	t *testing.T, clients *Clients,
	names *ResourceNames, options *Options) (*ResourceObjects, error) {
	if names.Service == "" || names.Image == "" {
		return nil, fmt.Errorf("expected non-empty Service and Image name; got Service = %s, Image = %s", names.Service, names.Image)
	}

	t.Log("Creating a new Service as Release with @latest.")
	svc, err := CreateReleaseService(t, clients, *names, options)
	if err != nil {
		return nil, err
	}

	// Populate Route and Configuration Objects with name
	names.Route = serviceresourcenames.Route(svc)
	names.Config = serviceresourcenames.Configuration(svc)

	t.Log("Waiting for Service to transition to Ready.")
	if err := WaitForServiceState(clients.ServingClient, names.Service, IsServiceReady, "ServiceIsReady"); err != nil {
		return nil, err
	}

	t.Log("Checking to ensure Service Status is populated for Ready service.")
	err = validateCreatedServiceStatus(clients, names)
	if err != nil {
		return nil, err
	}

	t.Log("Getting latest objects Created by Service.")
	return GetResourceObjects(clients, *names)
}

// CreateRunLatestServiceReady creates a new RunLatest Service in state 'Ready'. This function expects Service and Image name passed in through 'names'.
// Names is updated with the Route and Configuration created by the Service and ResourceObjects is returned with the Service, Route, and Configuration objects.
// Returns error if the service does not come up correctly.
func CreateRunLatestServiceReady(t *testing.T, clients *Clients, names *ResourceNames, options *Options, fopt ...rtesting.ServiceOption) (*ResourceObjects, error) {
	if names.Service == "" || names.Image == "" {
		return nil, fmt.Errorf("expected non-empty Service and Image name; got Service=%v, Image=%v", names.Service, names.Image)
	}

	t.Logf("Creating a new Service %s as RunLatest.", names.Service)
	svc, err := CreateLatestService(t, clients, *names, options, fopt...)
	if err != nil {
		return nil, err
	}

	// Populate Route and Configuration Objects with name
	names.Route = serviceresourcenames.Route(svc)
	names.Config = serviceresourcenames.Configuration(svc)

	t.Log("Waiting for Service to transition to Ready.")
	if err := WaitForServiceState(clients.ServingClient, names.Service, IsServiceReady, "ServiceIsReady"); err != nil {
		return nil, err
	}

	t.Log("Checking to ensure Service Status is populated for Ready service.")
	err = validateCreatedServiceStatus(clients, names)
	if err != nil {
		return nil, err
	}

	t.Log("Getting latest objects Created by Service.")
	resources, err := GetResourceObjects(clients, *names)
	if err == nil {
		t.Logf("Successfully created Service %s.", names.Domain)
	}
	return resources, err
}

// CreateReleaseService creates a service in namespace with the name names.Service and names.Image,
// configured with `@latest` revision.
func CreateReleaseService(t *testing.T, clients *Clients, names ResourceNames, options *Options, fopt ...rtesting.ServiceOption) (*v1alpha1.Service, error) {
	service := ReleaseLatestService(ServingNamespace, names, options, fopt...)
	LogResourceObject(t, ResourceObjects{Service: service})
	return clients.ServingClient.Services.Create(service)
}

// CreateLatestService creates a service in namespace with the name names.Service and names.Image
func CreateLatestService(t *testing.T, clients *Clients, names ResourceNames, options *Options, fopt ...rtesting.ServiceOption) (*v1alpha1.Service, error) {
	service := LatestService(ServingNamespace, names, options, fopt...)
	LogResourceObject(t, ResourceObjects{Service: service})
	svc, err := clients.ServingClient.Services.Create(service)
	return svc, err
}

// PatchReleaseService patches an existing service in namespace with the name names.Service
func PatchReleaseService(t *testing.T, clients *Clients, svc *v1alpha1.Service, revisions []string, rolloutPercent int) (*v1alpha1.Service, error) {
	newSvc := ReleaseService(svc, revisions, rolloutPercent)
	LogResourceObject(t, ResourceObjects{Service: newSvc})
	patchBytes, err := createPatch(svc, newSvc)
	if err != nil {
		return nil, err
	}
	return clients.ServingClient.Services.Patch(svc.ObjectMeta.Name, types.JSONPatchType, patchBytes, "")
}

// PatchManualService patches an existing service in namespace with the name names.Service
func PatchManualService(t *testing.T, clients *Clients, svc *v1alpha1.Service) (*v1alpha1.Service, error) {
	newSvc := ManualService(svc)
	LogResourceObject(t, ResourceObjects{Service: newSvc})
	patchBytes, err := createPatch(svc, newSvc)
	if err != nil {
		return nil, err
	}
	return clients.ServingClient.Services.Patch(svc.ObjectMeta.Name, types.JSONPatchType, patchBytes, "")
}

// PatchServiceImage patches the existing service passed in with a new imagePath. Returns the latest service object
func PatchServiceImage(t *testing.T, clients *Clients, svc *v1alpha1.Service, imagePath string) (*v1alpha1.Service, error) {
	newSvc := svc.DeepCopy()
	if svc.Spec.RunLatest != nil {
		newSvc.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.Image = imagePath
	} else if svc.Spec.Release != nil {
		newSvc.Spec.Release.Configuration.RevisionTemplate.Spec.Container.Image = imagePath
	} else if svc.Spec.DeprecatedPinned != nil {
		newSvc.Spec.DeprecatedPinned.Configuration.RevisionTemplate.Spec.Container.Image = imagePath
	} else {
		return nil, fmt.Errorf("UpdateImageService(%v): unable to determine service type", svc)
	}
	LogResourceObject(t, ResourceObjects{Service: newSvc})
	patchBytes, err := createPatch(svc, newSvc)
	if err != nil {
		return nil, err
	}
	return clients.ServingClient.Services.Patch(svc.ObjectMeta.Name, types.JSONPatchType, patchBytes, "")
}

// PatchService creates and applies a patch from the diff between curSvc and desiredSvc. Returns the latest service object.
func PatchService(t *testing.T, clients *Clients, curSvc *v1alpha1.Service, desiredSvc *v1alpha1.Service) (*v1alpha1.Service, error) {
	LogResourceObject(t, ResourceObjects{Service: desiredSvc})
	patchBytes, err := createPatch(curSvc, desiredSvc)
	if err != nil {
		return nil, err
	}
	return clients.ServingClient.Services.Patch(curSvc.ObjectMeta.Name, types.JSONPatchType, patchBytes, "")
}

// PatchServiceRevisionTemplateMetadata patches an existing service by adding metadata to the service's RevisionTemplateSpec.
func PatchServiceRevisionTemplateMetadata(t *testing.T, clients *Clients, svc *v1alpha1.Service, metadata metav1.ObjectMeta) (*v1alpha1.Service, error) {
	newSvc := svc.DeepCopy()
	if svc.Spec.RunLatest != nil {
		newSvc.Spec.RunLatest.Configuration.RevisionTemplate.ObjectMeta = metadata
	} else if svc.Spec.Release != nil {
		newSvc.Spec.Release.Configuration.RevisionTemplate.ObjectMeta = metadata
	} else if svc.Spec.DeprecatedPinned != nil {
		newSvc.Spec.DeprecatedPinned.Configuration.RevisionTemplate.ObjectMeta = metadata
	} else {
		return nil, fmt.Errorf("UpdateServiceRevisionTemplateMetadata(%v): unable to determine service type", svc)
	}
	LogResourceObject(t, ResourceObjects{Service: newSvc})
	patchBytes, err := createPatch(svc, newSvc)
	if err != nil {
		return nil, err
	}
	return clients.ServingClient.Services.Patch(svc.ObjectMeta.Name, types.JSONPatchType, patchBytes, "")
}

// WaitForServiceLatestRevision takes a revision in through names and compares it to the current state of LatestCreatedRevisionName in Service.
// Once an update is detected in the LatestCreatedRevisionName, the function waits for the created revision to be set in LatestReadyRevisionName
// before returning the name of the revision.
func WaitForServiceLatestRevision(clients *Clients, names ResourceNames) (string, error) {
	var revisionName string
	err := WaitForServiceState(clients.ServingClient, names.Service, func(s *v1alpha1.Service) (bool, error) {
		if s.Status.LatestCreatedRevisionName != names.Revision {
			revisionName = s.Status.LatestCreatedRevisionName
			return true, nil
		}
		return false, nil
	}, "ServiceUpdatedWithRevision")
	if err != nil {
		return "", err
	}
	err = WaitForServiceState(clients.ServingClient, names.Service, func(s *v1alpha1.Service) (bool, error) {
		return (s.Status.LatestReadyRevisionName == revisionName), nil
	}, "ServiceReadyWithRevision")

	return revisionName, err
}
