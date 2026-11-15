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

package keda

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
)

func TestFindHPAForScaledObject(t *testing.T) {
	const (
		soName        = "my-so"
		namespace     = "default"
		previousName  = "hpa-a"
		configuredHPA = "hpa-b"
	)

	newSO := func(statusHpaName string) *kedav1alpha1.ScaledObject {
		return &kedav1alpha1.ScaledObject{
			ObjectMeta: metav1.ObjectMeta{Name: soName, Namespace: namespace},
			Spec: kedav1alpha1.ScaledObjectSpec{
				ScaleTargetRef: &kedav1alpha1.ScaleTarget{Name: "cart"},
				Advanced: &kedav1alpha1.AdvancedConfig{
					HorizontalPodAutoscalerConfig: &kedav1alpha1.HorizontalPodAutoscalerConfig{Name: configuredHPA},
				},
			},
			Status: kedav1alpha1.ScaledObjectStatus{HpaName: statusHpaName},
		}
	}
	newHPA := func(name, target string, owned bool) *autoscalingv2.HorizontalPodAutoscaler {
		hpa := &autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: target},
				MaxReplicas:    3,
			},
		}
		if owned {
			hpa.OwnerReferences = []metav1.OwnerReference{{APIVersion: "keda.sh/v1alpha1", Kind: "ScaledObject", Name: soName}}
		}
		return hpa
	}

	tests := []struct {
		name          string
		statusHpaName string
		existing      []client.Object
		expectedName  string
	}{
		{
			name:          "no HPA exists",
			statusHpaName: "",
			expectedName:  "",
		},
		{
			name:          "HPA under the configured name is returned, even when foreign",
			statusHpaName: "",
			existing:      []client.Object{newHPA(configuredHPA, "checkout", false)},
			expectedName:  configuredHPA,
		},
		{
			name:          "own HPA under the previous name is returned for renaming",
			statusHpaName: previousName,
			existing:      []client.Object{newHPA(previousName, "cart", true)},
			expectedName:  previousName,
		},
		{
			name:          "foreign HPA under the previous name is skipped when the configured HPA does not exist",
			statusHpaName: previousName,
			existing:      []client.Object{newHPA(previousName, "checkout", false)},
			expectedName:  "",
		},
		{
			name:          "foreign HPA under the previous name is skipped in favour of the configured HPA",
			statusHpaName: previousName,
			existing:      []client.Object{newHPA(previousName, "checkout", false), newHPA(configuredHPA, "cart", true)},
			expectedName:  configuredHPA,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, autoscalingv2.AddToScheme(scheme))
			require.NoError(t, kedav1alpha1.AddToScheme(scheme))
			r := &ScaledObjectReconciler{
				Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(test.existing...).Build(),
			}

			hpa, err := r.findHPAForScaledObject(context.Background(), logr.Discard(), newSO(test.statusHpaName))
			require.NoError(t, err)
			if test.expectedName == "" {
				require.Nil(t, hpa)
				return
			}
			require.NotNil(t, hpa)
			require.Equal(t, test.expectedName, hpa.Name)
		})
	}
}
