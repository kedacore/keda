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

// crdpolling contains functions which poll Knative Serving CRDs until they
// get into the state desired by the caller or time out.

package test

import (
	"context"
	"fmt"
	"time"

	pkgTest "github.com/knative/pkg/test"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
	apiv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	k8styped "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	interval = 1 * time.Second
	timeout  = 6 * time.Minute
)

// WaitForRouteState polls the status of the Route called name from client every
// interval until inState returns `true` indicating it is done, returns an
// error or timeout. desc will be used to name the metric that is emitted to
// track how long it took for name to get into the state checked by inState.
func WaitForRouteState(client *ServingClients, name string, inState func(r *v1alpha1.Route) (bool, error), desc string) error {
	metricName := fmt.Sprintf("WaitForRouteState/%s/%s", name, desc)
	_, span := trace.StartSpan(context.Background(), metricName)
	defer span.End()

	var lastState *v1alpha1.Route
	waitErr := wait.PollImmediate(interval, timeout, func() (bool, error) {
		var err error
		lastState, err = client.Routes.Get(name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return inState(lastState)
	})

	if waitErr != nil {
		return errors.Wrapf(waitErr, "route %q is not in desired state, got: %+v", name, lastState)
	}
	return nil
}

// CheckRouteState verifies the status of the Route called name from client
// is in a particular state by calling `inState` and expecting `true`.
// This is the non-polling variety of WaitForRouteState
func CheckRouteState(client *ServingClients, name string, inState func(r *v1alpha1.Route) (bool, error)) error {
	r, err := client.Routes.Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if done, err := inState(r); err != nil {
		return err
	} else if !done {
		return fmt.Errorf("route %q is not in desired state, got: %+v", name, r)
	}
	return nil
}

// WaitForConfigurationState polls the status of the Configuration called name
// from client every interval until inState returns `true` indicating it
// is done, returns an error or timeout. desc will be used to name the metric
// that is emitted to track how long it took for name to get into the state checked by inState.
func WaitForConfigurationState(client *ServingClients, name string, inState func(c *v1alpha1.Configuration) (bool, error), desc string) error {
	metricName := fmt.Sprintf("WaitForConfigurationState/%s/%s", name, desc)
	_, span := trace.StartSpan(context.Background(), metricName)
	defer span.End()

	var lastState *v1alpha1.Configuration
	waitErr := wait.PollImmediate(interval, timeout, func() (bool, error) {
		var err error
		lastState, err = client.Configs.Get(name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return inState(lastState)
	})

	if waitErr != nil {
		return errors.Wrapf(waitErr, "configuration %q is not in desired state, got: %+v", name, lastState)
	}
	return nil
}

// CheckConfigurationState verifies the status of the Configuration called name from client
// is in a particular state by calling `inState` and expecting `true`.
// This is the non-polling variety of WaitForConfigurationState
func CheckConfigurationState(client *ServingClients, name string, inState func(r *v1alpha1.Configuration) (bool, error)) error {
	c, err := client.Configs.Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if done, err := inState(c); err != nil {
		return err
	} else if !done {
		return fmt.Errorf("configuration %q is not in desired state, got: %+v", name, c)
	}
	return nil
}

// WaitForRevisionState polls the status of the Revision called name
// from client every `interval` until `inState` returns `true` indicating it
// is done, returns an error or timeout. desc will be used to name the metric
// that is emitted to track how long it took for name to get into the state checked by inState.
func WaitForRevisionState(client *ServingClients, name string, inState func(r *v1alpha1.Revision) (bool, error), desc string) error {
	metricName := fmt.Sprintf("WaitForRevision/%s/%s", name, desc)
	_, span := trace.StartSpan(context.Background(), metricName)
	defer span.End()

	var lastState *v1alpha1.Revision
	waitErr := wait.PollImmediate(interval, timeout, func() (bool, error) {
		var err error
		lastState, err = client.Revisions.Get(name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return inState(lastState)
	})

	if waitErr != nil {
		return errors.Wrapf(waitErr, "revision %q is not in desired state, got: %+v", name, lastState)
	}
	return nil
}

// CheckRevisionState verifies the status of the Revision called name from client
// is in a particular state by calling `inState` and expecting `true`.
// This is the non-polling variety of WaitForRevisionState
func CheckRevisionState(client *ServingClients, name string, inState func(r *v1alpha1.Revision) (bool, error)) error {
	r, err := client.Revisions.Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if done, err := inState(r); err != nil {
		return err
	} else if !done {
		return fmt.Errorf("revision %q is not in desired state, got: %+v", name, r)
	}
	return nil
}

// WaitForServiceState polls the status of the Service called name
// from client every `interval` until `inState` returns `true` indicating it
// is done, returns an error or timeout. desc will be used to name the metric
// that is emitted to track how long it took for name to get into the state checked by inState.
func WaitForServiceState(client *ServingClients, name string, inState func(s *v1alpha1.Service) (bool, error), desc string) error {
	metricName := fmt.Sprintf("WaitForServiceState/%s/%s", name, desc)
	_, span := trace.StartSpan(context.Background(), metricName)
	defer span.End()

	var lastState *v1alpha1.Service
	waitErr := wait.PollImmediate(interval, timeout, func() (bool, error) {
		var err error
		lastState, err = client.Services.Get(name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return inState(lastState)
	})

	if waitErr != nil {
		return errors.Wrapf(waitErr, "service %q is not in desired state, got: %+v", name, lastState)
	}
	return nil
}

// CheckServiceState verifies the status of the Service called name from client
// is in a particular state by calling `inState` and expecting `true`.
// This is the non-polling variety of WaitForServiceState.
func CheckServiceState(client *ServingClients, name string, inState func(s *v1alpha1.Service) (bool, error)) error {
	s, err := client.Services.Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if done, err := inState(s); err != nil {
		return err
	} else if !done {
		return fmt.Errorf("service %q is not in desired state, got: %+v", name, s)
	}
	return nil
}

// GetConfigMap gets the knative serving config map.
func GetConfigMap(client *pkgTest.KubeClient) k8styped.ConfigMapInterface {
	return client.Kube.CoreV1().ConfigMaps("knative-serving")
}

// DeploymentScaledToZeroFunc returns a func that evaluates if a deployment has scaled to 0 pods.
func DeploymentScaledToZeroFunc(d *apiv1beta1.Deployment) (bool, error) {
	return d.Status.ReadyReplicas == 0, nil
}
