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

package testing

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// TODO(mattmoor): Move this into `knative/pkg/configmap`
const exampleKey = "_example"

// ConfigMapFromTestFile creates a v1.ConfigMap from a YAML file
// It loads the YAML file from the testdata folder.
func ConfigMapFromTestFile(t *testing.T, name string, allowed ...string) *corev1.ConfigMap {
	t.Helper()

	cm, _ := ConfigMapsFromTestFile(t, name, allowed...)
	return cm
}

// configMapsFromTestFile creates two corev1.ConfigMap resources from the config
// file read from the testdata directory:
// 1. The raw configmap read in.
// 2. A second version of the configmap augmenting `data:` with what's parsed from the value of `_example:`
func ConfigMapsFromTestFile(t *testing.T, name string, allowed ...string) (*corev1.ConfigMap, *corev1.ConfigMap) {
	t.Helper()

	b, err := ioutil.ReadFile(fmt.Sprintf("testdata/%s.yaml", name))
	if err != nil {
		t.Fatalf("ReadFile() = %v", err)
	}

	var orig corev1.ConfigMap

	// Use github.com/ghodss/yaml since it reads json struct
	// tags so things unmarshal properly
	if err := yaml.Unmarshal(b, &orig); err != nil {
		t.Fatalf("yaml.Unmarshal() = %v", err)
	}

	// We expect each of the allowed keys, and a key holding an example
	// configuration for us to validate.
	allowed = append(allowed, exampleKey)

	if len(orig.Data) != len(allowed) {
		// See here for why we only check in empty ConfigMaps:
		// https://github.com/knative/serving/issues/2668
		t.Errorf("Data = %v, wanted %v", orig.Data, allowed)
	}
	allow := sets.NewString(allowed...)
	for key := range orig.Data {
		if !allow.Has(key) {
			t.Errorf("Encountered key %q in %q that wasn't on the allowed list", key, name)
		}
	}
	// With the length and membership checks, we know that the keyspace matches.

	exampleBody := orig.Data[exampleKey]
	// Check that exampleBody does not have lines that end in a trailing space,
	for i, line := range strings.Split(exampleBody, "\n") {
		if strings.HasSuffix(line, " ") {
			t.Errorf("line %d of %q example contains trailing spaces", i, name)
		}
	}

	// Parse exampleBody into exemplar.Data
	exemplar := orig.DeepCopy()
	if err := yaml.Unmarshal([]byte(exampleBody), &exemplar.Data); err != nil {
		t.Fatalf("yaml.Unmarshal() = %v", err)
	}
	// Augment the sample with actual configuration
	for k, v := range orig.Data {
		if _, ok := exemplar.Data[k]; ok {
			continue
		}
		exemplar.Data[k] = v
	}

	return &orig, exemplar
}
