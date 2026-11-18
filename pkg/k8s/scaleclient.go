/*
Copyright 2022 The KEDA Authors

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

package k8s

import (
	"context"
	"fmt"
	"time"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/scale"
	ctrl "sigs.k8s.io/controller-runtime"

	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

var log = ctrl.Log.WithName("scaleclient")

// ScalesGetter provides access to the scale subresources in a namespace.
type ScalesGetter interface {
	Scales(namespace string) ScaleInterface
}

// ScaleInterface addresses the exact resource resolved from scaleTargetRef.
type ScaleInterface interface {
	Get(ctx context.Context, target schema.GroupVersionResource, name string, opts metav1.GetOptions) (*autoscalingv1.Scale, error)
	Update(ctx context.Context, target schema.GroupVersionResource, value *autoscalingv1.Scale, opts metav1.UpdateOptions) (*autoscalingv1.Scale, error)
}

type versionedScaleClient struct {
	client    dynamic.Interface
	kinds     scale.ScaleKindResolver
	converter *scale.ScaleConverter
}

type namespacedScales struct {
	*versionedScaleClient
	namespace string
}

func (c *versionedScaleClient) Scales(namespace string) ScaleInterface {
	return &namespacedScales{versionedScaleClient: c, namespace: namespace}
}

func (c *namespacedScales) Get(ctx context.Context, target schema.GroupVersionResource, name string, opts metav1.GetOptions) (*autoscalingv1.Scale, error) {
	obj, err := c.client.Resource(target).Namespace(c.namespace).Get(ctx, name, opts, "scale")
	if err != nil {
		return nil, err
	}
	return c.toScale(obj)
}

func (c *versionedScaleClient) toScale(obj *unstructured.Unstructured) (*autoscalingv1.Scale, error) {
	converted, err := c.converter.ConvertToVersion(obj, autoscalingv1.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	value, ok := converted.(*autoscalingv1.Scale)
	if !ok {
		return nil, fmt.Errorf("expected Scale, got %T", converted)
	}
	return value, nil
}

func (c *namespacedScales) Update(ctx context.Context, target schema.GroupVersionResource, value *autoscalingv1.Scale, opts metav1.UpdateOptions) (*autoscalingv1.Scale, error) {
	// The Scale body can have a different API version from its parent resource.
	kind, err := c.kinds.ScaleForResource(target)
	if err != nil {
		return nil, err
	}
	converted, err := c.converter.ConvertToVersion(value, kind.GroupVersion())
	if err != nil {
		return nil, err
	}
	body, err := runtime.DefaultUnstructuredConverter.ToUnstructured(converted)
	if err != nil {
		return nil, err
	}
	obj, err := c.client.Resource(target).Namespace(c.namespace).Update(ctx, &unstructured.Unstructured{Object: body}, opts, "scale")
	if err != nil {
		return nil, err
	}
	return c.toScale(obj)
}

// InitScaleClient initializes scale client and returns k8s version
func InitScaleClient(mgr ctrl.Manager) (ScalesGetter, kedautil.K8sVersion, error) {
	kubeVersion := kedautil.K8sVersion{}

	// create Discovery clientset
	// TODO If we need to increase the QPS of scaling API calls, copy and tweak this RESTConfig.
	clientset, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		log.Error(err, "not able to create Discovery clientset")
		return nil, kubeVersion, err
	}

	// Find out Kubernetes version
	version, err := clientset.ServerVersion()
	if err == nil {
		kubeVersion = kedautil.NewK8sVersion(version)
	} else {
		log.Error(err, "not able to get Kubernetes version")
		return nil, kubeVersion, err
	}

	client, err := newScaleResourceClient(mgr.GetConfig())
	if err != nil {
		return nil, kubeVersion, err
	}
	return &versionedScaleClient{
		client:    client,
		kinds:     scale.NewDiscoveryScaleKindResolver(clientset),
		converter: scale.NewScaleConverter(),
	}, kubeVersion, nil
}

func newScaleResourceClient(config *rest.Config) (*dynamic.DynamicClient, error) {
	config = rest.CopyConfig(config)
	if config.Timeout == 0 {
		// Preserve the discovery REST client's default deadline for /scale requests.
		// see also: https://github.com/kubernetes/client-go/blob/v0.36.6/discovery/discovery_client.go#L55-L57
		config.Timeout = 32 * time.Second
	}
	return dynamic.NewForConfig(config)
}
