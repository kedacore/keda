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

package build

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// Interface abstracts different methods for turning a supported importpath
// reference into a v1.Image.
type Interface interface {
	// IsSupportedReference determines whether the given reference is to an importpath reference
	// that Ko supports building.
	// TODO(mattmoor): Verify that some base repo: foo.io/bar can be suffixed with this reference and parsed.
	IsSupportedReference(string) bool

	// Build turns the given importpath reference into a v1.Image containing the Go binary.
	Build(string) (v1.Image, error)
}
