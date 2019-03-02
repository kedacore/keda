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

package resources

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/knative/build/pkg/reconciler/buildtemplate/resources/names"
	caching "github.com/knative/caching/pkg/apis/caching/v1alpha1"
	"github.com/knative/pkg/kmeta"
)

// Note: namespace is passed separately because this may be used for
// cluster-scoped stuff as well.
func MakeImageCachesFromSpec(
	namespace string,
	bt names.ImageCacheable,
) []caching.Image {
	var caches []caching.Image

	// Avoid duplicates.
	images := sets.NewString()

	for index, container := range bt.TemplateSpec().Steps {
		// TODO(mattmoor): Consider substituting default values to
		// get more caching when substitutions are used in image names.
		if strings.Contains(container.Image, "$") {
			// Skip image names containing substitutions.
			continue
		}
		if images.Has(container.Image) {
			continue
		}
		images.Insert(container.Image)

		caches = append(caches, caching.Image{
			ObjectMeta: metav1.ObjectMeta{
				Name:            names.ImageCache(bt, index),
				Namespace:       namespace,
				Labels:          kmeta.MakeVersionLabels(bt),
				OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(bt)},
			},
			Spec: caching.ImageSpec{
				Image: container.Image,
			},
		})
	}
	return caches
}

func MakeImageCaches(bt *v1alpha1.BuildTemplate) []caching.Image {
	return MakeImageCachesFromSpec(bt.Namespace, bt)
}
