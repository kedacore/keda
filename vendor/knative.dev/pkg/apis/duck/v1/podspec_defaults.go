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
)

// PodSpecDefaulter is a callback to validate a PodSpecable.
type PodSpecDefaulter func(context.Context, *WithPod)

// SetDefaults implements apis.Defaultable
func (wp *WithPod) SetDefaults(ctx context.Context) {
	if psd := GetPodSpecDefaulter(ctx); psd != nil {
		psd(ctx, wp)
	}
}

// psdKey is used for associating a PodSpecDefaulter with a context.Context
type psdKey struct{}

func WithPodSpecDefaulter(ctx context.Context, psd PodSpecDefaulter) context.Context {
	return context.WithValue(ctx, psdKey{}, psd)
}

// GetPodSpecDefaulter extracts the PodSpecDefaulter from the context.
func GetPodSpecDefaulter(ctx context.Context) PodSpecDefaulter {
	untyped := ctx.Value(psdKey{})
	if untyped == nil {
		return nil
	}
	return untyped.(PodSpecDefaulter)
}

// PodDefaulter is a callback to validate a Pod.
type PodDefaulter func(context.Context, *Pod)

// SetDefaults implements apis.Defaultable
func (p *Pod) SetDefaults(ctx context.Context) {
	if pd := GetPodDefaulter(ctx); pd != nil {
		pd(ctx, p)
	}
}

// pdKey is used for associating a PodDefaulter with a context.Context
type pdKey struct{}

func WithPodDefaulter(ctx context.Context, pd PodDefaulter) context.Context {
	return context.WithValue(ctx, pdKey{}, pd)
}

// GetPodDefaulter extracts the PodDefaulter from the context.
func GetPodDefaulter(ctx context.Context) PodDefaulter {
	untyped := ctx.Value(pdKey{})
	if untyped == nil {
		return nil
	}
	return untyped.(PodDefaulter)
}
