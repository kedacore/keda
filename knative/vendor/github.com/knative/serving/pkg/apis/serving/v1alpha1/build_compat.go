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

package v1alpha1

import (
	"bytes"
	"encoding/json"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
)

// RawExtension is modeled after runtime.RawExtension, and should be
// replaced with it (or an alias) once we can stop supporting embedded
// BuildSpecs.
type RawExtension struct {
	// Field order is the precedence for JSON marshaling if multiple
	// fields are set.
	Raw       []byte
	Object    runtime.Object
	BuildSpec *buildv1alpha1.BuildSpec
}

var _ json.Unmarshaler = (*RawExtension)(nil)
var _ json.Marshaler = (*RawExtension)(nil)

func (re *RawExtension) UnmarshalJSON(in []byte) error {
	if re == nil {
		return errors.New("RawExtension: UnmarshalJSON on nil pointer")
	}
	if !bytes.Equal(in, []byte("null")) {
		re.Raw = append(re.Raw[0:0], in...)
	}
	return nil
}

// MarshalJSON may get called on pointers or values, so implement MarshalJSON on value.
// http://stackoverflow.com/questions/21390979/custom-marshaljson-never-gets-called-in-go
func (re RawExtension) MarshalJSON() ([]byte, error) {
	switch {
	case re.Raw != nil:
		return re.Raw, nil

	case re.Object != nil:
		return json.Marshal(re.Object)

	case re.BuildSpec != nil:
		return json.Marshal(re.BuildSpec)

	default:
		return []byte("null"), nil
	}
}

func (re *RawExtension) ensureRaw() (err error) {
	switch {
	case re.Raw != nil:
		// Nothing to do.
	case re.Object != nil, re.BuildSpec != nil:
		re.Raw, err = re.MarshalJSON()
	}
	return
}

// As is a helper to decode the raw object into a particular type.
// The type is expected to exhaustively specify the fields encountered.
func (re *RawExtension) As(x interface{}) error {
	if err := re.ensureRaw(); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewBuffer(re.Raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode(&x)
}

// AsDuck is a helper to decode the raw object into a particular duck type.
// The type may only represent a subset of the fields present.
func (re *RawExtension) AsDuck(x interface{}) error {
	if err := re.ensureRaw(); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewBuffer(re.Raw))
	// Allow unknown fields.
	return decoder.Decode(&x)
}
