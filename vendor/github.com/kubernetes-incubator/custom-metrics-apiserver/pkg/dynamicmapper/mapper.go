package dynamicmapper

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/restmapper"
)

// RengeneratingDiscoveryRESTMapper is a RESTMapper which Regenerates its cache of mappings periodically.
// It functions by recreating a normal discovery RESTMapper at the specified interval.
// We don't refresh automatically on cache misses, since we get called on every label, plenty of which will
// be unrelated to Kubernetes resources.
type RegeneratingDiscoveryRESTMapper struct {
	discoveryClient discovery.DiscoveryInterface

	refreshInterval time.Duration

	mu sync.RWMutex

	delegate meta.RESTMapper
}

func NewRESTMapper(discoveryClient discovery.DiscoveryInterface, refreshInterval time.Duration) (*RegeneratingDiscoveryRESTMapper, error) {
	mapper := &RegeneratingDiscoveryRESTMapper{
		discoveryClient: discoveryClient,
		refreshInterval: refreshInterval,
	}
	if err := mapper.RegenerateMappings(); err != nil {
		return nil, fmt.Errorf("unable to populate initial set of REST mappings: %v", err)
	}

	return mapper, nil
}

// RunUtil runs the mapping refresher until the given stop channel is closed.
func (m *RegeneratingDiscoveryRESTMapper) RunUntil(stop <-chan struct{}) {
	go wait.Until(func() {
		if err := m.RegenerateMappings(); err != nil {
			glog.Errorf("error regenerating REST mappings from discovery: %v", err)
		}
	}, m.refreshInterval, stop)
}

func (m *RegeneratingDiscoveryRESTMapper) RegenerateMappings() error {
	resources, err := restmapper.GetAPIGroupResources(m.discoveryClient)
	if err != nil {
		return err
	}
	newDelegate := restmapper.NewDiscoveryRESTMapper(resources)

	// don't lock until we're ready to replace
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delegate = newDelegate

	return nil
}

func (m *RegeneratingDiscoveryRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.delegate.KindFor(resource)
}

func (m *RegeneratingDiscoveryRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.delegate.KindsFor(resource)

}
func (m *RegeneratingDiscoveryRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.delegate.ResourceFor(input)

}
func (m *RegeneratingDiscoveryRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.delegate.ResourcesFor(input)

}
func (m *RegeneratingDiscoveryRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.delegate.RESTMapping(gk, versions...)

}
func (m *RegeneratingDiscoveryRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.delegate.RESTMappings(gk, versions...)
}
func (m *RegeneratingDiscoveryRESTMapper) ResourceSingularizer(resource string) (singular string, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.delegate.ResourceSingularizer(resource)
}
