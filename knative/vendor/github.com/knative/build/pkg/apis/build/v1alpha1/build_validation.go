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

package v1alpha1

import (
	"time"

	"github.com/knative/pkg/apis"
)

// Validate Build
func (b *Build) Validate() *apis.FieldError {
	return validateObjectMetadata(b.GetObjectMeta()).ViaField("metadata").Also(b.Spec.Validate().ViaField("spec"))
}

// Validate for build spec
func (bs *BuildSpec) Validate() *apis.FieldError {
	if bs.Template == nil && len(bs.Steps) == 0 {
		return apis.ErrMissingOneOf("template", "steps")
	}
	if bs.Template != nil && len(bs.Steps) > 0 {
		return apis.ErrMultipleOneOf("template", "steps")
	}

	// If a build specifies a template, all the template's parameters without
	// defaults must be satisfied by the build's parameters.
	if bs.Template != nil {
		return bs.Template.Validate().ViaField("template")
	}

	// Below method potentially has a bug:
	// It does not Validate if only a "Source" has been set, it only validates if multiple sources have been set
	return bs.validateSources().
		Also(ValidateVolumes(bs.Volumes).ViaField("volumes")).
		Also(bs.validateTimeout()).
		Also(validateSteps(bs.Steps).ViaField("steps"))
}

// Validate template
func (b *TemplateInstantiationSpec) Validate() *apis.FieldError {
	if b == nil {
		return nil
	}
	if b.Name == "" {
		return apis.ErrMissingField("name")
	}
	if b.Kind != "" {
		switch b.Kind {
		case ClusterBuildTemplateKind,
			BuildTemplateKind:
			return nil
		default:
			return apis.ErrInvalidValue(string(b.Kind), "kind")
		}
	}
	return nil
}

// Validate build timeout
func (bs *BuildSpec) validateTimeout() *apis.FieldError {
	if bs.Timeout == nil {
		return nil
	}
	maxTimeout := time.Duration(24 * time.Hour)

	if bs.Timeout.Duration > maxTimeout || bs.Timeout.Duration < 0 {
		return apis.ErrOutOfBoundsValue(bs.Timeout.Duration.String(), "0", "24", "timeout")
	}
	return nil
}

// Validate source
func (bs BuildSpec) validateSources() *apis.FieldError {
	var subPathExists bool
	var emptyTargetPath bool
	names := map[string]string{}
	pathtree := pathTree{
		nodeMap: map[string]map[string]string{},
	}

	// Both source and sources cannot be defined in build
	if len(bs.Sources) > 0 && bs.Source != nil {
		return apis.ErrMultipleOneOf("source", "sources")
	}
	for _, source := range bs.Sources {
		// Check all source have unique names
		if _, ok := names[source.Name]; ok {
			return apis.ErrMultipleOneOf("name").ViaField("sources")
		}
		// Multiple sources cannot have subpath defined
		if source.SubPath != "" {
			if subPathExists {
				return apis.ErrMultipleOneOf("subpath").ViaField("sources")
			}
			subPathExists = true
		}
		names[source.Name] = ""

		if source.TargetPath == "" {
			if source.Custom != nil {
				continue
			}
			if emptyTargetPath {
				return apis.ErrInvalidValue("Empty Target Path", "targetPath").ViaField("sources")
			}
			emptyTargetPath = true
		} else {
			if source.Custom != nil {
				return apis.ErrInvalidValue(source.TargetPath, "targetPath").ViaField("sources")
			}
			if err := insertNode(source.TargetPath, pathtree).ViaField("sources"); err != nil {
				return err
			}
		}
	}
	return nil
}
