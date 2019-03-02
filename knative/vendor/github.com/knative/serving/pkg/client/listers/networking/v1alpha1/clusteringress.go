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
	v1alpha1 "github.com/knative/serving/pkg/apis/networking/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ClusterIngressLister helps list ClusterIngresses.
type ClusterIngressLister interface {
	// List lists all ClusterIngresses in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.ClusterIngress, err error)
	// Get retrieves the ClusterIngress from the index for a given name.
	Get(name string) (*v1alpha1.ClusterIngress, error)
	ClusterIngressListerExpansion
}

// clusterIngressLister implements the ClusterIngressLister interface.
type clusterIngressLister struct {
	indexer cache.Indexer
}

// NewClusterIngressLister returns a new ClusterIngressLister.
func NewClusterIngressLister(indexer cache.Indexer) ClusterIngressLister {
	return &clusterIngressLister{indexer: indexer}
}

// List lists all ClusterIngresses in the indexer.
func (s *clusterIngressLister) List(selector labels.Selector) (ret []*v1alpha1.ClusterIngress, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ClusterIngress))
	})
	return ret, err
}

// Get retrieves the ClusterIngress from the index for a given name.
func (s *clusterIngressLister) Get(name string) (*v1alpha1.ClusterIngress, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("clusteringress"), name)
	}
	return obj.(*v1alpha1.ClusterIngress), nil
}
