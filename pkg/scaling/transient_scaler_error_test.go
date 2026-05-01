/*
Copyright 2021 The KEDA Authors

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

package scaling

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsTransientScalerCacheRebuildError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "redis closed", err: errors.New(`redis: client is closed`), want: true},
		{name: "wrapped redis closed", err: fmt.Errorf("outer: %w", errors.New(`redis: client is closed`)), want: true},
		{name: "scaler len zero", err: errors.New(`scaler with id 0 not found. Len = 0`), want: true},
		{name: "wrapped scaler len zero", err: fmt.Errorf("cache: %w", errors.New(`scaler with id 2 not found. Len = 0`)), want: true},
		{name: "unrelated", err: errors.New("connection refused"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTransientScalerCacheRebuildError(tt.err); got != tt.want {
				t.Fatalf("IsTransientScalerCacheRebuildError() = %v, want %v", got, tt.want)
			}
		})
	}
}
