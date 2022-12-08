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

// CronJobValidator is a callback to validate a CronJob.
type CronJobValidator func(context.Context, *CronJob) *apis.FieldError

// Validate implements apis.Validatable
func (c *CronJob) Validate(ctx context.Context) *apis.FieldError {
	if cv := GetCronJobValidator(ctx); cv != nil {
		return cv(ctx, c)
	}
	return nil
}

// cvKey is used for associating a CronJobValidator with a context.Context
type cvKey struct{}

func WithCronJobValidator(ctx context.Context, cv CronJobValidator) context.Context {
	return context.WithValue(ctx, cvKey{}, cv)
}

// GetCronJobValidator extracts the CronJobValidator from the context.
func GetCronJobValidator(ctx context.Context) CronJobValidator {
	untyped := ctx.Value(cvKey{})
	if untyped == nil {
		return nil
	}
	return untyped.(CronJobValidator)
}
