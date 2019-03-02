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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/knative/pkg/kmeta"
	"github.com/knative/serving/pkg/apis/serving"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
)

// MakeBuild creates an Unstructured Build object from the passed in Configuration and fills
// in metadata and references based on the Configuration.
func MakeBuild(config *v1alpha1.Configuration) *unstructured.Unstructured {
	if config.Spec.Build == nil {
		return nil
	}

	u := GetBuild(&config.Spec)

	// Compute the hash of the current build's spec.
	sum := sha256.Sum256(config.Spec.Build.Raw)
	h := hex.EncodeToString(sum[:])

	// Put it into a label for later lookups.
	l := u.GetLabels()
	if l == nil {
		l = make(map[string]string)
	}
	l[serving.BuildHashLabelKey] = h[:63] // Labels can only be 63 characters.
	u.SetLabels(l)

	// Clear the name if it's been explicitly set
	// We want the build to have a generated name
	//
	// Note: K8s apimachinery >=v1.12 calling SetName will 'remove' the name
	//       <v1.12 will set the name to an empty string
	u.SetName("")
	u.SetNamespace(config.Namespace)
	u.SetGenerateName(config.Name + "-")
	u.SetOwnerReferences([]metav1.OwnerReference{*kmeta.NewControllerRef(config)})
	return u
}

// GetBuild extracts an Unstructured Build object from the passed in ConfigurationSpec.
func GetBuild(configSpec *v1alpha1.ConfigurationSpec) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	if err := configSpec.Build.As(u); err != nil {
		b := &buildv1alpha1.Build{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "build.knative.dev/v1alpha1",
				Kind:       "Build",
			},
		}
		if err := configSpec.Build.As(&b.Spec); err != nil {
			// This is validated by the webhook.
			panic(err.Error())
		}

		u = MustToUnstructured(b)
	}
	// After calling `As()` we can be sure that `.Raw` is populated.

	return u
}

func MustToUnstructured(build *buildv1alpha1.Build) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}

	b, err := json.Marshal(build)
	if err != nil {
		panic(err.Error())
	}

	if err := json.Unmarshal(b, u); err != nil {
		panic(err.Error())
	}

	return u
}

func UnstructuredWithContent(content map[string]interface{}) *unstructured.Unstructured {
	if content == nil {
		return nil
	}
	u := &unstructured.Unstructured{}
	u.SetUnstructuredContent(content)
	return u.DeepCopy()
}
