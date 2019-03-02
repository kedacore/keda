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
package v1alpha1

import (
	time "time"

	networking_v1alpha1 "github.com/knative/serving/pkg/apis/networking/v1alpha1"
	versioned "github.com/knative/serving/pkg/client/clientset/versioned"
	internalinterfaces "github.com/knative/serving/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/knative/serving/pkg/client/listers/networking/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ClusterIngressInformer provides access to a shared informer and lister for
// ClusterIngresses.
type ClusterIngressInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ClusterIngressLister
}

type clusterIngressInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewClusterIngressInformer constructs a new informer for ClusterIngress type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewClusterIngressInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredClusterIngressInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredClusterIngressInformer constructs a new informer for ClusterIngress type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredClusterIngressInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NetworkingV1alpha1().ClusterIngresses().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.NetworkingV1alpha1().ClusterIngresses().Watch(options)
			},
		},
		&networking_v1alpha1.ClusterIngress{},
		resyncPeriod,
		indexers,
	)
}

func (f *clusterIngressInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredClusterIngressInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *clusterIngressInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&networking_v1alpha1.ClusterIngress{}, f.defaultInformer)
}

func (f *clusterIngressInformer) Lister() v1alpha1.ClusterIngressLister {
	return v1alpha1.NewClusterIngressLister(f.Informer().GetIndexer())
}
