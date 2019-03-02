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

// BuildTemplateInformer provides access to a shared informer and lister for
// BuildTemplates.
type BuildTemplateInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.BuildTemplateLister
}

type buildTemplateInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewBuildTemplateInformer constructs a new informer for BuildTemplate type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewBuildTemplateInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredBuildTemplateInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredBuildTemplateInformer constructs a new informer for BuildTemplate type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredBuildTemplateInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BuildV1alpha1().BuildTemplates(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BuildV1alpha1().BuildTemplates(namespace).Watch(options)
			},
		},
		&build_v1alpha1.BuildTemplate{},
		resyncPeriod,
		indexers,
	)
}

func (f *buildTemplateInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredBuildTemplateInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *buildTemplateInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&build_v1alpha1.BuildTemplate{}, f.defaultInformer)
}

func (f *buildTemplateInformer) Lister() v1alpha1.BuildTemplateLister {
	return v1alpha1.NewBuildTemplateLister(f.Informer().GetIndexer())
}
