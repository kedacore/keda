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

	serving_v1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	versioned "github.com/knative/serving/pkg/client/clientset/versioned"
	internalinterfaces "github.com/knative/serving/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/knative/serving/pkg/client/listers/serving/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ServiceInformer provides access to a shared informer and lister for
// Services.
type ServiceInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ServiceLister
}

type serviceInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewServiceInformer constructs a new informer for Service type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewServiceInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredServiceInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredServiceInformer constructs a new informer for Service type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredServiceInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ServingV1alpha1().Services(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ServingV1alpha1().Services(namespace).Watch(options)
			},
		},
		&serving_v1alpha1.Service{},
		resyncPeriod,
		indexers,
	)
}

func (f *serviceInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredServiceInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *serviceInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&serving_v1alpha1.Service{}, f.defaultInformer)
}

func (f *serviceInformer) Lister() v1alpha1.ServiceLister {
	return v1alpha1.NewServiceLister(f.Informer().GetIndexer())
}
