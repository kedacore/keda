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

// WithBaseImages is a functional option for overriding the base images
// that are used for different images.
func WithBaseImages(gb GetBase) Option {
	return func(gbo *gobuildOpener) error {
		gbo.getBase = gb
		return nil
	}
}

// WithCreationTime is a functional option for overriding the creation
// time given to images.
func WithCreationTime(t v1.Time) Option {
	return func(gbo *gobuildOpener) error {
		gbo.creationTime = t
		return nil
	}
}

// withBuilder is a functional option for overriding the way go binaries
// are built.  This is exposed for testing.
func withBuilder(b builder) Option {
	return func(gbo *gobuildOpener) error {
		gbo.build = b
		return nil
	}
}
