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

package credentials

import (
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// VolumePath is the path where build secrets are written.
// It is mutable and exported for testing.
var VolumePath = "/var/build-secrets"

// Builder is the interface for a credential initializer of any type.
type Builder interface {
	// MatchingAnnotations extracts flags for the credential
	// helper from the supplied secret and returns a slice (of
	// length 0 or greater) of applicable domains.
	MatchingAnnotations(*corev1.Secret) []string

	// Write writes the credentials to the correct location.
	Write() error
}

// VolumeName returns the full path to the secret, inside the VolumePath.
func VolumeName(secretName string) string {
	return fmt.Sprintf("%s/%s", VolumePath, secretName)
}

// SortAnnotations return sorted array of strings which has annotationPrefix
// as the prefix in secrets key
func SortAnnotations(secrets map[string]string, annotationPrefix string) []string {
	var mk []string
	for k, v := range secrets {
		if strings.HasPrefix(k, annotationPrefix) {
			mk = append(mk, v)
		}
	}
	sort.Strings(mk)
	return mk
}
