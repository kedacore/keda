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

// CronJobDefaulter is a callback to validate a CronJob.
type CronJobDefaulter func(context.Context, *CronJob)

// SetDefaults implements apis.Defaultable
func (c *CronJob) SetDefaults(ctx context.Context) {
	if cd := GetCronJobDefaulter(ctx); cd != nil {
		cd(ctx, c)
	}
}

// cdKey is used for associating a CronJobDefaulter with a context.Context
type cdKey struct{}

func WithCronJobDefaulter(ctx context.Context, cd CronJobDefaulter) context.Context {
	return context.WithValue(ctx, cdKey{}, cd)
}

// GetCronJobDefaulter extracts the CronJobDefaulter from the context.
func GetCronJobDefaulter(ctx context.Context) CronJobDefaulter {
	untyped := ctx.Value(cdKey{})
	if untyped == nil {
		return nil
	}
	return untyped.(CronJobDefaulter)
}
