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

// ClusterBuildTemplateInformer provides access to a shared informer and lister for
// ClusterBuildTemplates.
type ClusterBuildTemplateInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ClusterBuildTemplateLister
}

type clusterBuildTemplateInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewClusterBuildTemplateInformer constructs a new informer for ClusterBuildTemplate type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewClusterBuildTemplateInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredClusterBuildTemplateInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredClusterBuildTemplateInformer constructs a new informer for ClusterBuildTemplate type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredClusterBuildTemplateInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BuildV1alpha1().ClusterBuildTemplates().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BuildV1alpha1().ClusterBuildTemplates().Watch(options)
			},
		},
		&build_v1alpha1.ClusterBuildTemplate{},
		resyncPeriod,
		indexers,
	)
}

func (f *clusterBuildTemplateInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredClusterBuildTemplateInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *clusterBuildTemplateInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&build_v1alpha1.ClusterBuildTemplate{}, f.defaultInformer)
}

func (f *clusterBuildTemplateInformer) Lister() v1alpha1.ClusterBuildTemplateLister {
	return v1alpha1.NewClusterBuildTemplateLister(f.Informer().GetIndexer())
}
