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

package tracker

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/tools/cache"

	"github.com/knative/pkg/kmeta"
)

// New returns an implementation of Interface that lets a Reconciler
// register a particular resource as watching an ObjectReference for
// a particular lease duration.  This watch must be refreshed
// periodically (e.g. by a controller resync) or it will expire.
//
// When OnChanged is called by the informer for a particular
// GroupVersionKind, the provided callback is called with the "key"
// of each object actively watching the changed object.
func New(callback func(string), lease time.Duration) Interface {
	return &impl{
		leaseDuration: lease,
		cb:            callback,
	}
}

type impl struct {
	m sync.Mutex
	// mapping maps from an object reference to the set of
	// keys for objects watching it.
	mapping map[corev1.ObjectReference]set

	// The amount of time that an object may watch another
	// before having to renew the lease.
	leaseDuration time.Duration

	cb func(string)
}

// Check that impl implements Interface.
var _ Interface = (*impl)(nil)

// set is a map from keys to expirations
type set map[string]time.Time

// Track implements Interface.
func (i *impl) Track(ref corev1.ObjectReference, obj interface{}) error {
	invalidFields := map[string][]string{
		"APIVersion": validation.IsQualifiedName(ref.APIVersion),
		"Kind":       validation.IsCIdentifier(ref.Kind),
		"Namespace":  validation.IsDNS1123Label(ref.Namespace),
		"Name":       validation.IsDNS1123Subdomain(ref.Name),
	}
	fieldErrors := []string{}
	for k, v := range invalidFields {
		for _, msg := range v {
			fieldErrors = append(fieldErrors, fmt.Sprintf("%s: %s", k, msg))
		}
	}
	if len(fieldErrors) > 0 {
		sort.Strings(fieldErrors)
		return fmt.Errorf("Invalid ObjectReference:\n%s", strings.Join(fieldErrors, "\n"))
	}

	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return err
	}

	i.m.Lock()
	defer i.m.Unlock()
	if i.mapping == nil {
		i.mapping = make(map[corev1.ObjectReference]set)
	}

	l, ok := i.mapping[ref]
	if !ok {
		l = set{}
	}
	if expiry, ok := l[key]; !ok || isExpired(expiry) {
		// When covering an uncovered key, immediately call the
		// registered callback to ensure that the following pattern
		// doesn't create problems:
		//    foo, err := lister.Get(key)
		//    // Later...
		//    err := tracker.Track(fooRef, parent)
		// In this example, "Later" represents a window where "foo" may
		// have changed or been created while the Track is not active.
		// The simplest way of eliminating such a window is to call the
		// callback to "catch up" immediately following new
		// registrations.
		i.cb(key)
	}
	// Overwrite the key with a new expiration.
	l[key] = time.Now().Add(i.leaseDuration)

	i.mapping[ref] = l
	return nil
}

func objectReference(item kmeta.Accessor) corev1.ObjectReference {
	gvk := item.GroupVersionKind()
	apiVersion, kind := gvk.ToAPIVersionAndKind()
	return corev1.ObjectReference{
		APIVersion: apiVersion,
		Kind:       kind,
		Namespace:  item.GetNamespace(),
		Name:       item.GetName(),
	}
}

func isExpired(expiry time.Time) bool {
	return time.Now().After(expiry)
}

// OnChanged implements Interface.
func (i *impl) OnChanged(obj interface{}) {
	item, err := kmeta.DeletionHandlingAccessor(obj)
	if err != nil {
		// TODO(mattmoor): We should consider logging here.
		return
	}

	or := objectReference(item)

	// TODO(mattmoor): Consider locking the mapping (global) for a
	// smaller scope and leveraging a per-set lock to guard its access.
	i.m.Lock()
	defer i.m.Unlock()
	s, ok := i.mapping[or]
	if !ok {
		// TODO(mattmoor): We should consider logging here.
		return
	}

	for key, expiry := range s {
		// If the expiration has lapsed, then delete the key.
		if isExpired(expiry) {
			delete(s, key)
			continue
		}
		i.cb(key)
	}

	if len(s) == 0 {
		delete(i.mapping, or)
	}
}
