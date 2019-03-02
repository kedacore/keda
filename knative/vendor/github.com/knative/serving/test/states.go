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

package test

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/knative/serving/pkg/apis/serving"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
)

// states contains functions for asserting against the state of Knative Serving
// crds to see if they have achieved the states specified in the spec
// (https://github.com/knative/serving/blob/master/docs/spec/spec.md).

// AllRouteTrafficAtRevision will check the revision that route r is routing
// traffic to and return true if 100% of the traffic is routing to revisionName.
func AllRouteTrafficAtRevision(names ResourceNames) func(r *v1alpha1.Route) (bool, error) {
	return func(r *v1alpha1.Route) (bool, error) {
		for _, tt := range r.Status.Traffic {
			if tt.Percent == 100 {
				if tt.RevisionName != names.Revision {
					return true, fmt.Errorf("Expected traffic revision name to be %s but actually is %s", names.Revision, tt.RevisionName)
				}

				if tt.Name != names.TrafficTarget {
					return true, fmt.Errorf("Expected traffic target name to be %s but actually is %s", names.TrafficTarget, tt.Name)
				}

				return true, nil
			}
		}
		return false, nil
	}
}

// RouteTrafficToRevisionWithInClusterDNS will check the revision that route r is routing
// traffic using in cluster DNS and return true if the revision received the request.
func TODO_RouteTrafficToRevisionWithInClusterDNS(r *v1alpha1.Route) (bool, error) {
	if r.Status.Address == nil {
		return false, fmt.Errorf("Expected route %s to implement Addressable, missing .status.address", r.Name)
	}
	if r.Status.Address.Hostname == "" {
		return false, fmt.Errorf("Expected route %s to have in cluster dns status set", r.Name)
	}
	// TODO make a curl request from inside the cluster using
	// r.Status.Address.Hostname to validate DNS is set correctly
	return true, nil
}

// ServiceTrafficToRevisionWithInClusterDNS will check the revision that route r is routing
// traffic using in cluster DNS and return true if the revision received the request.
func TODO_ServiceTrafficToRevisionWithInClusterDNS(s *v1alpha1.Service) (bool, error) {
	if s.Status.Address == nil {
		return false, fmt.Errorf("Expected service %s to implement Addressable, missing .status.address", s.Name)
	}
	if s.Status.Address.Hostname == "" {
		return false, fmt.Errorf("Expected service %s to have in cluster dns status set", s.Name)
	}
	// TODO make a curl request from inside the cluster using
	// s.Status.Address.Hostname to validate DNS is set correctly
	return true, nil
}

// IsRevisionReady will check the status conditions of the revision and return true if the revision is
// ready to serve traffic. It will return false if the status indicates a state other than deploying
// or being ready. It will also return false if the type of the condition is unexpected.
func IsRevisionReady(r *v1alpha1.Revision) (bool, error) {
	return r.Generation == r.Status.ObservedGeneration && r.Status.IsReady(), nil
}

// IsServiceReady will check the status conditions of the service and return true if the service is
// ready. This means that its configurations and routes have all reported ready.
func IsServiceReady(s *v1alpha1.Service) (bool, error) {
	return s.Generation == s.Status.ObservedGeneration && s.Status.IsReady(), nil
}

// IsRouteReady will check the status conditions of the route and return true if the route is
// ready.
func IsRouteReady(r *v1alpha1.Route) (bool, error) {
	return r.Generation == r.Status.ObservedGeneration && r.Status.IsReady(), nil
}

// ConfigurationHasCreatedRevision returns whether the Configuration has created a Revision.
func ConfigurationHasCreatedRevision(c *v1alpha1.Configuration) (bool, error) {
	return c.Status.LatestCreatedRevisionName != "", nil
}

// IsRevisionBuildFailed will check the status conditions of the revision and
// return true if the revision's build failed.
func IsRevisionBuildFailed(r *v1alpha1.Revision) (bool, error) {
	if cond := r.Status.GetCondition(v1alpha1.RevisionConditionBuildSucceeded); cond != nil {
		return cond.Status == corev1.ConditionFalse, nil
	}
	return false, nil
}

// IsConfigRevisionCreationFailed will check the status conditions of the
// configuration and return true if the configuration's revision failed to
// create.
func IsConfigRevisionCreationFailed(c *v1alpha1.Configuration) (bool, error) {
	if cond := c.Status.GetCondition(v1alpha1.ConfigurationConditionReady); cond != nil {
		return cond.Status == corev1.ConditionFalse && cond.Reason == "RevisionFailed", nil
	}
	return false, nil
}

// IsRevisionAtExpectedGeneration returns a function that will check if the annotations
// on the revision include an annotation for the generation and that the annotation is
// set to the expected value.
func IsRevisionAtExpectedGeneration(expectedGeneration string) func(r *v1alpha1.Revision) (bool, error) {
	return func(r *v1alpha1.Revision) (bool, error) {
		if a, ok := r.Labels[serving.ConfigurationGenerationLabelKey]; ok {
			if a != expectedGeneration {
				return true, fmt.Errorf("Expected Revision %s to be labeled with generation %s but was %s instead", r.Name, expectedGeneration, a)
			}
			return true, nil
		}
		return true, fmt.Errorf("Expected Revision %s to be labeled with generation %s but there was no label", r.Name, expectedGeneration)
	}
}
