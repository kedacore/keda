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

	build_v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	versioned "github.com/knative/build/pkg/client/clientset/versioned"
	internalinterfaces "github.com/knative/build/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/knative/build/pkg/client/listers/build/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// BuildInformer provides access to a shared informer and lister for
// Builds.
type BuildInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.BuildLister
}

type buildInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewBuildInformer constructs a new informer for Build type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewBuildInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredBuildInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredBuildInformer constructs a new informer for Build type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredBuildInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BuildV1alpha1().Builds(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BuildV1alpha1().Builds(namespace).Watch(options)
			},
		},
		&build_v1alpha1.Build{},
		resyncPeriod,
		indexers,
	)
}

func (f *buildInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredBuildInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *buildInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&build_v1alpha1.Build{}, f.defaultInformer)
}

func (f *buildInformer) Lister() v1alpha1.BuildLister {
	return v1alpha1.NewBuildLister(f.Informer().GetIndexer())
}
