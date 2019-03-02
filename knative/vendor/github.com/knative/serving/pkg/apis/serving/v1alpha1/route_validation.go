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
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/api/equality"

	"github.com/knative/pkg/apis"
	"k8s.io/apimachinery/pkg/util/validation"
)

func (r *Route) Validate() *apis.FieldError {
	return ValidateObjectMetadata(r.GetObjectMeta()).ViaField("metadata").
		Also(r.Spec.Validate().ViaField("spec"))
}

func (rs *RouteSpec) Validate() *apis.FieldError {
	if equality.Semantic.DeepEqual(rs, &RouteSpec{}) {
		return apis.ErrMissingField(apis.CurrentField)
	}

	// Where a named traffic target points
	type namedTarget struct {
		r string // revision name
		c string // config name
		i int    // index of first occurrence
	}

	// Track the targets of named TrafficTarget entries (to detect duplicates).
	trafficMap := make(map[string]namedTarget)

	var errs *apis.FieldError
	percentSum := 0
	for i, tt := range rs.Traffic {
		errs = errs.Also(tt.Validate().ViaFieldIndex("traffic", i))

		percentSum += tt.Percent

		if tt.Name == "" {
			// No Name field, so skip the uniqueness check.
			continue
		}
		nt := namedTarget{
			r: tt.RevisionName,
			c: tt.ConfigurationName,
			i: i,
		}
		if ent, ok := trafficMap[tt.Name]; !ok {
			// No entry exists, so add ours
			trafficMap[tt.Name] = nt
		} else {
			// We want only single definition of the route, even if it points
			// to the same config or revision.
			errs = errs.Also(&apis.FieldError{
				Message: fmt.Sprintf("Multiple definitions for %q", tt.Name),
				Paths: []string{
					fmt.Sprintf("traffic[%d].name", ent.i),
					fmt.Sprintf("traffic[%d].name", nt.i),
				},
			})
		}
	}

	if percentSum != 100 {
		errs = errs.Also(&apis.FieldError{
			Message: fmt.Sprintf("Traffic targets sum to %d, want 100", percentSum),
			Paths:   []string{"traffic"},
		})
	}
	return errs
}

// Validate verifies that TrafficTarget is properly configured.
func (tt *TrafficTarget) Validate() *apis.FieldError {
	var errs *apis.FieldError
	switch {
	case tt.RevisionName != "" && tt.ConfigurationName != "":
		errs = apis.ErrMultipleOneOf("revisionName", "configurationName")
	case tt.RevisionName != "":
		if verrs := validation.IsQualifiedName(tt.RevisionName); len(verrs) > 0 {
			errs = apis.ErrInvalidKeyName(tt.RevisionName, "revisionName", verrs...)
		}
	case tt.ConfigurationName != "":
		if verrs := validation.IsQualifiedName(tt.ConfigurationName); len(verrs) > 0 {
			errs = apis.ErrInvalidKeyName(tt.ConfigurationName, "configurationName", verrs...)
		}
	default:
		errs = apis.ErrMissingOneOf("revisionName", "configurationName")
	}
	if tt.Percent < 0 || tt.Percent > 100 {
		errs = errs.Also(apis.ErrOutOfBoundsValue(strconv.Itoa(tt.Percent), "0", "100", "percent"))
	}
	return errs
}
