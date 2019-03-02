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
	"github.com/knative/build/pkg/apis/build/v1alpha1"
	buildtemplateresources "github.com/knative/build/pkg/reconciler/buildtemplate/resources"
	caching "github.com/knative/caching/pkg/apis/caching/v1alpha1"
	"github.com/knative/pkg/system"
)

func MakeImageCaches(bt *v1alpha1.ClusterBuildTemplate) []caching.Image {
	return buildtemplateresources.MakeImageCachesFromSpec(system.Namespace(), bt)
}
