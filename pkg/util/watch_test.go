/*
Copyright 2026 The KEDA Authors

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

package util

import (
	"testing"
)

func TestGetWatchNamespaces(t *testing.T) {
	tests := []struct {
		name      string
		envValue  string
		envSet    bool
		wantKeys  []string
		wantError bool
	}{
		{
			name:      "env not set returns error",
			envSet:    false,
			wantError: true,
		},
		{
			name:     "empty string returns empty map",
			envValue: "",
			envSet:   true,
			wantKeys: []string{},
		},
		{
			name:     "quoted empty string returns empty map",
			envValue: `""`,
			envSet:   true,
			wantKeys: []string{},
		},
		{
			name:     "single namespace",
			envValue: "default",
			envSet:   true,
			wantKeys: []string{"default"},
		},
		{
			name:     "multiple namespaces",
			envValue: "ns1,ns2,ns3",
			envSet:   true,
			wantKeys: []string{"ns1", "ns2", "ns3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envSet {
				t.Setenv("WATCH_NAMESPACE", tt.envValue)
			}

			got, err := GetWatchNamespaces()

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.wantKeys) {
				t.Fatalf("expected %d keys, got %d", len(tt.wantKeys), len(got))
			}
			for _, key := range tt.wantKeys {
				if _, ok := got[key]; !ok {
					t.Errorf("expected key %q not found in result", key)
				}
			}
		})
	}
}
