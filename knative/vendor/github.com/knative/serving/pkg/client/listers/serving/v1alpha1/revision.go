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
	v1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// RevisionLister helps list Revisions.
type RevisionLister interface {
	// List lists all Revisions in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.Revision, err error)
	// Revisions returns an object that can list and get Revisions.
	Revisions(namespace string) RevisionNamespaceLister
	RevisionListerExpansion
}

// revisionLister implements the RevisionLister interface.
type revisionLister struct {
	indexer cache.Indexer
}

// NewRevisionLister returns a new RevisionLister.
func NewRevisionLister(indexer cache.Indexer) RevisionLister {
	return &revisionLister{indexer: indexer}
}

// List lists all Revisions in the indexer.
func (s *revisionLister) List(selector labels.Selector) (ret []*v1alpha1.Revision, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Revision))
	})
	return ret, err
}

// Revisions returns an object that can list and get Revisions.
func (s *revisionLister) Revisions(namespace string) RevisionNamespaceLister {
	return revisionNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// RevisionNamespaceLister helps list and get Revisions.
type RevisionNamespaceLister interface {
	// List lists all Revisions in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.Revision, err error)
	// Get retrieves the Revision from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.Revision, error)
	RevisionNamespaceListerExpansion
}

// revisionNamespaceLister implements the RevisionNamespaceLister
// interface.
type revisionNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Revisions in the indexer for a given namespace.
func (s revisionNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Revision, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Revision))
	})
	return ret, err
}

// Get retrieves the Revision from the indexer for a given namespace and name.
func (s revisionNamespaceLister) Get(name string) (*v1alpha1.Revision, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("revision"), name)
	}
	return obj.(*v1alpha1.Revision), nil
}
