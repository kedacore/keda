// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resolve

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	yaml "gopkg.in/yaml.v2"

	"golang.org/x/sync/errgroup"

	"github.com/google/go-containerregistry/pkg/ko/build"
	"github.com/google/go-containerregistry/pkg/ko/publish"
)

// ImageReferences resolves supported references to images within the input yaml
// to published image digests.
func ImageReferences(input []byte, builder build.Interface, publisher publish.Interface) ([]byte, error) {
	// First, walk the input objects and collect a list of supported references
	refs := make(map[string]struct{})
	// The loop is to support multi-document yaml files.
	// This is handled by using a yaml.Decoder and reading objects until io.EOF, see:
	// https://github.com/go-yaml/yaml/blob/v2.2.1/yaml.go#L124
	decoder := yaml.NewDecoder(bytes.NewBuffer(input))
	for {
		var obj interface{}
		if err := decoder.Decode(&obj); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		// This simply returns the replaced object, which we discard during the gathering phase.
		if _, err := replaceRecursive(obj, func(ref string) (string, error) {
			if builder.IsSupportedReference(ref) {
				refs[ref] = struct{}{}
			}
			return ref, nil
		}); err != nil {
			return nil, err
		}
	}

	// Next, perform parallel builds for each of the supported references.
	var sm sync.Map
	var errg errgroup.Group
	for ref := range refs {
		ref := ref
		errg.Go(func() error {
			img, err := builder.Build(ref)
			if err != nil {
				return err
			}
			digest, err := publisher.Publish(img, ref)
			if err != nil {
				return err
			}
			sm.Store(ref, digest.String())
			return nil
		})
	}
	if err := errg.Wait(); err != nil {
		return nil, err
	}

	// Last, walk the inputs again and replace the supported references with their published images.
	decoder = yaml.NewDecoder(bytes.NewBuffer(input))
	buf := bytes.NewBuffer(nil)
	encoder := yaml.NewEncoder(buf)
	for {
		var obj interface{}
		if err := decoder.Decode(&obj); err != nil {
			if err == io.EOF {
				return buf.Bytes(), nil
			}
			return nil, err
		}
		// Recursively walk input, replacing supported reference with our computed digests.
		obj2, err := replaceRecursive(obj, func(ref string) (string, error) {
			if !builder.IsSupportedReference(ref) {
				return ref, nil
			}
			if val, ok := sm.Load(ref); ok {
				return val.(string), nil
			}
			return "", fmt.Errorf("resolved reference to %q not found", ref)
		})
		if err != nil {
			return nil, err
		}

		if err := encoder.Encode(obj2); err != nil {
			return nil, err
		}
	}
}

type replaceString func(string) (string, error)

// replaceRecursive walks the provided untyped object recursively by switching
// on the type of the object at each level. It supports walking through the
// keys and values of maps, and the elements of an array. When a leaf of type
// string is encountered, this will call the provided replaceString function on
// it. This function does not support walking through struct types, but also
// should not need to as the input is expected to be the result of parsing yaml
// or json into an interface{}, which should only produce primitives, maps and
// arrays. This function will return a copy of the object rebuilt by the walk
// with the replacements made.
func replaceRecursive(obj interface{}, rs replaceString) (interface{}, error) {
	switch typed := obj.(type) {
	case map[interface{}]interface{}:
		m2 := make(map[interface{}]interface{}, len(typed))
		for k, v := range typed {
			k2, err := replaceRecursive(k, rs)
			if err != nil {
				return nil, err
			}
			v2, err := replaceRecursive(v, rs)
			if err != nil {
				return nil, err
			}
			m2[k2] = v2
		}
		return m2, nil

	case []interface{}:
		a2 := make([]interface{}, len(typed))
		for idx, v := range typed {
			v2, err := replaceRecursive(v, rs)
			if err != nil {
				return nil, err
			}
			a2[idx] = v2
		}
		return a2, nil

	case string:
		// call our replaceString on this string leaf.
		return rs(typed)

	default:
		// leave other leaves alone.
		return typed, nil
	}
}
