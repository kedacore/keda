/*
Copyright 2021 The Knative Authors

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

package v1

import (
	"context"

	"knative.dev/pkg/apis"
)

// PodSpecValidator is a callback to validate a PodSpecable.
type PodSpecValidator func(context.Context, *WithPod) *apis.FieldError

// Validate implements apis.Validatable
func (wp *WithPod) Validate(ctx context.Context) *apis.FieldError {
	if psv := GetPodSpecValidator(ctx); psv != nil {
		return psv(ctx, wp)
	}
	return nil
}

// psvKey is used for associating a PodSpecValidator with a context.Context
type psvKey struct{}

func WithPodSpecValidator(ctx context.Context, psv PodSpecValidator) context.Context {
	return context.WithValue(ctx, psvKey{}, psv)
}

// GetPodSpecValidator extracts the PodSpecValidator from the context.
func GetPodSpecValidator(ctx context.Context) PodSpecValidator {
	untyped := ctx.Value(psvKey{})
	if untyped == nil {
		return nil
	}
	return untyped.(PodSpecValidator)
}

// PodValidator is a callback to validate Pods.
type PodValidator func(context.Context, *Pod) *apis.FieldError

// Validate implements apis.Validatable
func (p *Pod) Validate(ctx context.Context) *apis.FieldError {
	if pv := GetPodValidator(ctx); pv != nil {
		return pv(ctx, p)
	}
	return nil
}

// pvKey is used for associating a PodValidator with a context.Context
type pvKey struct{}

func WithPodValidator(ctx context.Context, pv PodValidator) context.Context {
	return context.WithValue(ctx, pvKey{}, pv)
}

// GetPodValidator extracts the PodValidator from the context.
func GetPodValidator(ctx context.Context) PodValidator {
	untyped := ctx.Value(pvKey{})
	if untyped == nil {
		return nil
	}
	return untyped.(PodValidator)
}
