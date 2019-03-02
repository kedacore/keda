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

package testing

import (
	cachingv1alpha1 "github.com/knative/caching/pkg/apis/caching/v1alpha1"
	fakecachingclientset "github.com/knative/caching/pkg/client/clientset/versioned/fake"
	cachinglisters "github.com/knative/caching/pkg/client/listers/caching/v1alpha1"
	istiov1alpha3 "github.com/knative/pkg/apis/istio/v1alpha3"
	fakesharedclientset "github.com/knative/pkg/client/clientset/versioned/fake"
	istiolisters "github.com/knative/pkg/client/listers/istio/v1alpha3"
	kpa "github.com/knative/serving/pkg/apis/autoscaling/v1alpha1"
	networking "github.com/knative/serving/pkg/apis/networking/v1alpha1"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	fakeservingclientset "github.com/knative/serving/pkg/client/clientset/versioned/fake"
	kpalisters "github.com/knative/serving/pkg/client/listers/autoscaling/v1alpha1"
	networkinglisters "github.com/knative/serving/pkg/client/listers/networking/v1alpha1"
	servinglisters "github.com/knative/serving/pkg/client/listers/serving/v1alpha1"
	"github.com/knative/serving/pkg/reconciler/testing"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	autoscalingv1listers "k8s.io/client-go/listers/autoscaling/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

var buildAddToScheme = func(scheme *runtime.Scheme) {
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: "build.knative.dev", Version: "v1alpha1", Kind: "Build"}, &unstructured.Unstructured{})
}

var clientSetSchemes = []func(*runtime.Scheme){
	fakekubeclientset.AddToScheme,
	fakesharedclientset.AddToScheme,
	fakeservingclientset.AddToScheme,
	fakecachingclientset.AddToScheme,
	buildAddToScheme,
}

type Listers struct {
	sorter testing.ObjectSorter
}

func NewListers(objs []runtime.Object) Listers {
	scheme := runtime.NewScheme()

	for _, addTo := range clientSetSchemes {
		addTo(scheme)
	}

	ls := Listers{
		sorter: testing.NewObjectSorter(scheme),
	}

	ls.sorter.AddObjects(objs...)

	return ls
}

func (l *Listers) indexerFor(obj runtime.Object) cache.Indexer {
	return l.sorter.IndexerForObjectType(obj)
}

func (l *Listers) GetKubeObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakekubeclientset.AddToScheme)
}

func (l *Listers) GetCachingObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakecachingclientset.AddToScheme)
}

func (l *Listers) GetServingObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakeservingclientset.AddToScheme)
}

func (l *Listers) GetBuildObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(buildAddToScheme)
}

func (l *Listers) GetSharedObjects() []runtime.Object {
	return l.sorter.ObjectsForSchemeFunc(fakesharedclientset.AddToScheme)
}

func (l *Listers) GetServiceLister() servinglisters.ServiceLister {
	return servinglisters.NewServiceLister(l.indexerFor(&v1alpha1.Service{}))
}

func (l *Listers) GetRouteLister() servinglisters.RouteLister {
	return servinglisters.NewRouteLister(l.indexerFor(&v1alpha1.Route{}))
}

func (l *Listers) GetConfigurationLister() servinglisters.ConfigurationLister {
	return servinglisters.NewConfigurationLister(l.indexerFor(&v1alpha1.Configuration{}))
}

func (l *Listers) GetRevisionLister() servinglisters.RevisionLister {
	return servinglisters.NewRevisionLister(l.indexerFor(&v1alpha1.Revision{}))
}

func (l *Listers) GetPodAutoscalerLister() kpalisters.PodAutoscalerLister {
	return kpalisters.NewPodAutoscalerLister(l.indexerFor(&kpa.PodAutoscaler{}))
}

func (l *Listers) GetHorizontalPodAutoscalerLister() autoscalingv1listers.HorizontalPodAutoscalerLister {
	return autoscalingv1listers.NewHorizontalPodAutoscalerLister(l.indexerFor(&autoscalingv1.HorizontalPodAutoscaler{}))
}

// GetClusterIngressLister get lister for ClusterIngress resource.
func (l *Listers) GetClusterIngressLister() networkinglisters.ClusterIngressLister {
	return networkinglisters.NewClusterIngressLister(l.indexerFor(&networking.ClusterIngress{}))
}

func (l *Listers) GetVirtualServiceLister() istiolisters.VirtualServiceLister {
	return istiolisters.NewVirtualServiceLister(l.indexerFor(&istiov1alpha3.VirtualService{}))
}

func (l *Listers) GetImageLister() cachinglisters.ImageLister {
	return cachinglisters.NewImageLister(l.indexerFor(&cachingv1alpha1.Image{}))
}

func (l *Listers) GetDeploymentLister() appsv1listers.DeploymentLister {
	return appsv1listers.NewDeploymentLister(l.indexerFor(&appsv1.Deployment{}))
}

func (l *Listers) GetK8sServiceLister() corev1listers.ServiceLister {
	return corev1listers.NewServiceLister(l.indexerFor(&corev1.Service{}))
}

func (l *Listers) GetEndpointsLister() corev1listers.EndpointsLister {
	return corev1listers.NewEndpointsLister(l.indexerFor(&corev1.Endpoints{}))
}

func (l *Listers) GetConfigMapLister() corev1listers.ConfigMapLister {
	return corev1listers.NewConfigMapLister(l.indexerFor(&corev1.ConfigMap{}))
}
