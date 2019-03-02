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

// RevisionInformer provides access to a shared informer and lister for
// Revisions.
type RevisionInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.RevisionLister
}

type revisionInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewRevisionInformer constructs a new informer for Revision type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewRevisionInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredRevisionInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredRevisionInformer constructs a new informer for Revision type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredRevisionInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ServingV1alpha1().Revisions(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ServingV1alpha1().Revisions(namespace).Watch(options)
			},
		},
		&serving_v1alpha1.Revision{},
		resyncPeriod,
		indexers,
	)
}

func (f *revisionInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredRevisionInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *revisionInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&serving_v1alpha1.Revision{}, f.defaultInformer)
}

func (f *revisionInformer) Lister() v1alpha1.RevisionLister {
	return v1alpha1.NewRevisionLister(f.Informer().GetIndexer())
}
