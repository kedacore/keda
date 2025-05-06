/*
Copyright 2025 The KEDA Authors

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

package cache

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
)

func TestEmptyScalersCache(t *testing.T) {
	RegisterTestingT(t)

	cache := &ScalersCache{
		Scalers: make([]ScalerBuilder, 0),
	}
	go func() {
		scalers, configs := cache.GetScalers()
		Expect(scalers).To(BeEmpty())
		Expect(configs).To(BeEmpty())
	}()

	go func() {
		pushScalers := cache.GetPushScalers()
		Expect(pushScalers).To(BeEmpty())
	}()

	go func() {
		metrics := cache.GetMetricSpecForScaling(context.Background())
		Expect(metrics).To(BeEmpty())
	}()

	go func() {
		metrics, err := cache.GetMetricSpecForScalingForScaler(context.Background(), 0)
		Expect(err).To(Not(BeNil()))
		Expect(metrics).To(BeNil())
	}()

	go func() {
		cache.Close(context.Background())
	}()
}
