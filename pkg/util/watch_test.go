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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
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

func TestGetWatchLabelSelector(t *testing.T) {
	tests := []struct {
		name      string
		set       bool
		value     string
		wantNil   bool
		wantErr   bool
		wantMatch map[string]string
	}{
		{name: "unset returns nil", set: false, wantNil: true},
		{name: "empty string returns nil", set: true, value: "", wantNil: true},
		{name: "quoted empty string returns nil", set: true, value: `""`, wantNil: true},
		{
			name:      "simple equality",
			set:       true,
			value:     "environment=production",
			wantMatch: map[string]string{"environment": "production"},
		},
		{
			name:      "multiple labels",
			set:       true,
			value:     "environment=production,tier=critical",
			wantMatch: map[string]string{"environment": "production", "tier": "critical"},
		},
		{
			name:    "invalid selector",
			set:     true,
			value:   "==broken==",
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withEnv(t, tc.set, tc.value)

			got, err := GetWatchLabelSelector()
			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			if tc.wantNil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.wantMatch, got.MatchLabels)
		})
	}
}

func TestWatchLabelSelectorPredicate(t *testing.T) {
	matching := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "match",
			Labels: map[string]string{"environment": "production"},
		},
	}
	notMatching := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "no-match",
			Labels: map[string]string{"environment": "staging"},
		},
	}
	unlabelled := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "unlabelled"}}

	t.Run("unset accepts every event", func(t *testing.T) {
		withEnv(t, false, "")
		p, err := WatchLabelSelectorPredicate()
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.True(t, p.Create(event.CreateEvent{Object: matching}))
		assert.True(t, p.Create(event.CreateEvent{Object: notMatching}))
		assert.True(t, p.Update(event.UpdateEvent{ObjectNew: unlabelled}))
		assert.True(t, p.Delete(event.DeleteEvent{Object: notMatching}))
		assert.True(t, p.Generic(event.GenericEvent{Object: unlabelled}))
	})

	t.Run("set filters by label", func(t *testing.T) {
		withEnv(t, true, "environment=production")
		p, err := WatchLabelSelectorPredicate()
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.True(t, p.Create(event.CreateEvent{Object: matching}))
		assert.False(t, p.Create(event.CreateEvent{Object: notMatching}))
		assert.False(t, p.Create(event.CreateEvent{Object: unlabelled}))
		assert.True(t, p.Update(event.UpdateEvent{ObjectNew: matching}))
		assert.False(t, p.Update(event.UpdateEvent{ObjectNew: notMatching}))
		assert.True(t, p.Delete(event.DeleteEvent{Object: matching}))
		assert.False(t, p.Generic(event.GenericEvent{Object: notMatching}))
	})

	t.Run("set notation (absence) selects unlabelled objects", func(t *testing.T) {
		withEnv(t, true, "!environment")
		p, err := WatchLabelSelectorPredicate()
		require.NoError(t, err)
		assert.False(t, p.Create(event.CreateEvent{Object: matching}))
		assert.False(t, p.Create(event.CreateEvent{Object: notMatching}))
		assert.True(t, p.Create(event.CreateEvent{Object: unlabelled}))
	})

	t.Run("invalid selector returns error", func(t *testing.T) {
		withEnv(t, true, "==broken==")
		p, err := WatchLabelSelectorPredicate()
		assert.Error(t, err)
		assert.Nil(t, p)
	})
}

func TestWatchLabelSelectorByObject(t *testing.T) {
	so := &corev1.ConfigMap{}
	hpa := &corev1.Secret{}

	t.Run("unset returns nil map (no filtering)", func(t *testing.T) {
		withEnv(t, false, "")
		out, err := WatchLabelSelectorByObject(so, hpa)
		require.NoError(t, err)
		assert.Nil(t, out)
	})

	t.Run("set applies same selector to every object type", func(t *testing.T) {
		withEnv(t, true, "environment=production")
		out, err := WatchLabelSelectorByObject(so, hpa)
		require.NoError(t, err)
		require.Len(t, out, 2)

		matchA := labels.Set{"environment": "production"}
		matchB := labels.Set{"environment": "staging"}

		soEntry, ok := out[client.Object(so)]
		require.True(t, ok, "ScaledObject entry missing")
		require.NotNil(t, soEntry.Label)
		assert.True(t, soEntry.Label.Matches(matchA))
		assert.False(t, soEntry.Label.Matches(matchB))

		hpaEntry, ok := out[client.Object(hpa)]
		require.True(t, ok, "HPA entry missing")
		assert.True(t, hpaEntry.Label.Matches(matchA))
	})

	t.Run("invalid selector propagates error", func(t *testing.T) {
		withEnv(t, true, "==broken==")
		out, err := WatchLabelSelectorByObject(so)
		assert.Error(t, err)
		assert.Nil(t, out)
	})

	t.Run("no objects with active filter returns empty map", func(t *testing.T) {
		withEnv(t, true, "environment=production")
		out, err := WatchLabelSelectorByObject()
		require.NoError(t, err)
		assert.NotNil(t, out)
		assert.Empty(t, out)
	})
}

// withEnv either sets WATCH_LABEL_SELECTOR=value (when set==true) or unsets it,
// restoring the prior state at test end. Keeps tests hermetic when neighbours
// touch the same env var.
func withEnv(t *testing.T, set bool, value string) {
	t.Helper()
	prev, had := os.LookupEnv(WatchLabelSelectorEnvVar)
	if set {
		t.Setenv(WatchLabelSelectorEnvVar, value)
	} else {
		if err := os.Unsetenv(WatchLabelSelectorEnvVar); err != nil {
			t.Fatalf("unsetenv %s: %v", WatchLabelSelectorEnvVar, err)
		}
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(WatchLabelSelectorEnvVar, prev)
		} else {
			_ = os.Unsetenv(WatchLabelSelectorEnvVar)
		}
	})
}
