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
	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// BuildTemplateLister helps list BuildTemplates.
type BuildTemplateLister interface {
	// List lists all BuildTemplates in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.BuildTemplate, err error)
	// BuildTemplates returns an object that can list and get BuildTemplates.
	BuildTemplates(namespace string) BuildTemplateNamespaceLister
	BuildTemplateListerExpansion
}

// buildTemplateLister implements the BuildTemplateLister interface.
type buildTemplateLister struct {
	indexer cache.Indexer
}

// NewBuildTemplateLister returns a new BuildTemplateLister.
func NewBuildTemplateLister(indexer cache.Indexer) BuildTemplateLister {
	return &buildTemplateLister{indexer: indexer}
}

// List lists all BuildTemplates in the indexer.
func (s *buildTemplateLister) List(selector labels.Selector) (ret []*v1alpha1.BuildTemplate, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.BuildTemplate))
	})
	return ret, err
}

// BuildTemplates returns an object that can list and get BuildTemplates.
func (s *buildTemplateLister) BuildTemplates(namespace string) BuildTemplateNamespaceLister {
	return buildTemplateNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// BuildTemplateNamespaceLister helps list and get BuildTemplates.
type BuildTemplateNamespaceLister interface {
	// List lists all BuildTemplates in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.BuildTemplate, err error)
	// Get retrieves the BuildTemplate from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.BuildTemplate, error)
	BuildTemplateNamespaceListerExpansion
}

// buildTemplateNamespaceLister implements the BuildTemplateNamespaceLister
// interface.
type buildTemplateNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all BuildTemplates in the indexer for a given namespace.
func (s buildTemplateNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.BuildTemplate, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.BuildTemplate))
	})
	return ret, err
}

// Get retrieves the BuildTemplate from the indexer for a given namespace and name.
func (s buildTemplateNamespaceLister) Get(name string) (*v1alpha1.BuildTemplate, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("buildtemplate"), name)
	}
	return obj.(*v1alpha1.BuildTemplate), nil
}
