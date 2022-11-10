/*
Copyright 2018 The Knative Authors.

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

package apis

import (
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VolatileTime wraps metav1.Time
//
// Unlike metav1.Time, VolatileTimes are considered semantically equal when
// using kubernetes semantic equality checks.
// Thus differing VolatileTime values are not considered different.
// Note, go-cmp will still return inequality, see unit test if you
// need this behavior for go-cmp.
//
// +kubebuilder:validation:Type=string
type VolatileTime struct {
	Inner metav1.Time `json:",inline"`
}

// MarshalJSON implements the json.Marshaler interface.
func (t VolatileTime) MarshalJSON() ([]byte, error) {
	return t.Inner.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (t *VolatileTime) UnmarshalJSON(b []byte) error {
	return t.Inner.UnmarshalJSON(b)
}

func init() {
	equality.Semantic.AddFunc(
		// Always treat VolatileTime fields as equivalent.
		func(VolatileTime, VolatileTime) bool {
			return true
		},
	)
}
