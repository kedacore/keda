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
	v1alpha1 "github.com/knative/serving/test/apis/testing/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// BuildLister helps list Builds.
type BuildLister interface {
	// List lists all Builds in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.Build, err error)
	// Builds returns an object that can list and get Builds.
	Builds(namespace string) BuildNamespaceLister
	BuildListerExpansion
}

// buildLister implements the BuildLister interface.
type buildLister struct {
	indexer cache.Indexer
}

// NewBuildLister returns a new BuildLister.
func NewBuildLister(indexer cache.Indexer) BuildLister {
	return &buildLister{indexer: indexer}
}

// List lists all Builds in the indexer.
func (s *buildLister) List(selector labels.Selector) (ret []*v1alpha1.Build, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Build))
	})
	return ret, err
}

// Builds returns an object that can list and get Builds.
func (s *buildLister) Builds(namespace string) BuildNamespaceLister {
	return buildNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// BuildNamespaceLister helps list and get Builds.
type BuildNamespaceLister interface {
	// List lists all Builds in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.Build, err error)
	// Get retrieves the Build from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.Build, error)
	BuildNamespaceListerExpansion
}

// buildNamespaceLister implements the BuildNamespaceLister
// interface.
type buildNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Builds in the indexer for a given namespace.
func (s buildNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Build, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Build))
	})
	return ret, err
}

// Get retrieves the Build from the indexer for a given namespace and name.
func (s buildNamespaceLister) Get(name string) (*v1alpha1.Build, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("build"), name)
	}
	return obj.(*v1alpha1.Build), nil
}
