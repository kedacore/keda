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

package keda

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/controllers/keda/util"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	"github.com/kedacore/keda/v2/pkg/mock/mock_eventemitter"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scaling"
)

func TestScaledJobEnsureFinalizer_AddsWhenAbsent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)

	sj := &kedav1alpha1.ScaledJob{
		ObjectMeta: metav1.ObjectMeta{Name: "sj", Namespace: "ns", ResourceVersion: "1"},
	}

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "sj", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ client.ObjectKey, fresh *kedav1alpha1.ScaledJob, _ ...client.GetOption) error {
			*fresh = *sj.DeepCopy()
			return nil
		})
	mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, patched *kedav1alpha1.ScaledJob, _ client.Patch, _ ...client.PatchOption) error {
			if !util.Contains(patched.GetFinalizers(), scaledJobFinalizer) {
				t.Errorf("expected finalizer added, got %v", patched.GetFinalizers())
			}
			return nil
		})

	r := &ScaledJobReconciler{Client: mockClient}

	if err := r.ensureFinalizer(context.Background(), logr.Discard(), sj); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !util.Contains(sj.GetFinalizers(), scaledJobFinalizer) {
		t.Errorf("expected caller's ScaledJob synced with finalizer, got %v", sj.GetFinalizers())
	}
}

func TestFinalizeScaledJob_RemovesFinalizerAndStopsScaleLoop(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	mockScaleHandler := mock_scaling.NewMockScaleHandler(ctrl)
	mockEventHandler := mock_eventemitter.NewMockEventHandler(ctrl)

	sj := &kedav1alpha1.ScaledJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "sj",
			Namespace:       "ns",
			Finalizers:      []string{scaledJobFinalizer},
			ResourceVersion: "1",
		},
	}

	mockScaleHandler.EXPECT().DeleteScalableObject(gomock.Any(), gomock.Any()).Return(nil)
	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "sj", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ client.ObjectKey, fresh *kedav1alpha1.ScaledJob, _ ...client.GetOption) error {
			*fresh = *sj.DeepCopy()
			return nil
		})
	mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, patched *kedav1alpha1.ScaledJob, _ client.Patch, _ ...client.PatchOption) error {
			if util.Contains(patched.GetFinalizers(), scaledJobFinalizer) {
				t.Errorf("expected finalizer removed, got %v", patched.GetFinalizers())
			}
			return nil
		})
	mockEventHandler.EXPECT().Emit(gomock.Any(), "ns/sj", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	r := &ScaledJobReconciler{Client: mockClient, EventEmitter: mockEventHandler, scaleHandler: mockScaleHandler}

	if err := r.finalizeScaledJob(context.Background(), logr.Discard(), sj, "ns/sj"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if util.Contains(sj.GetFinalizers(), scaledJobFinalizer) {
		t.Errorf("expected caller's ScaledJob synced without finalizer, got %v", sj.GetFinalizers())
	}
}
