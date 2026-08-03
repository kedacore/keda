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

package util

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"go.uber.org/mock/gomock"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventemitter"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	"github.com/kedacore/keda/v2/pkg/mock/mock_eventemitter"
)

const testFinalizerName = "finalizer.keda.sh"

func newFinalizerTestScaledObject(finalizers []string, resourceVersion string) *kedav1alpha1.ScaledObject {
	return &kedav1alpha1.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test",
			Namespace:       "ns",
			Finalizers:      finalizers,
			ResourceVersion: resourceVersion,
		},
	}
}

func TestAddFinalizer_AddsWhenAbsent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	obj := newFinalizerTestScaledObject(nil, "1")

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ client.ObjectKey, fresh *kedav1alpha1.ScaledObject, _ ...client.GetOption) error {
			*fresh = *newFinalizerTestScaledObject(nil, "1")
			return nil
		})
	mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, patched *kedav1alpha1.ScaledObject, _ client.Patch, _ ...client.PatchOption) error {
			if !Contains(patched.GetFinalizers(), testFinalizerName) {
				t.Errorf("expected patched object to contain finalizer, got %v", patched.GetFinalizers())
			}
			patched.ResourceVersion = "2"
			return nil
		})

	if err := AddFinalizer(context.Background(), mockClient, obj, testFinalizerName); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Contains(obj.GetFinalizers(), testFinalizerName) {
		t.Errorf("expected caller's obj to be synced with finalizer, got %v", obj.GetFinalizers())
	}
	if obj.ResourceVersion != "2" {
		t.Errorf("expected caller's obj ResourceVersion synced to \"2\", got %q", obj.ResourceVersion)
	}
}

func TestAddFinalizer_NoOpWhenPresent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	obj := newFinalizerTestScaledObject([]string{testFinalizerName}, "1")

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ client.ObjectKey, fresh *kedav1alpha1.ScaledObject, _ ...client.GetOption) error {
			*fresh = *newFinalizerTestScaledObject([]string{testFinalizerName}, "1")
			return nil
		})
	// No Patch expectation: if AddFinalizer tries to patch here, the test fails
	// with "missing call" because mockClient has no matching expectation.

	if err := AddFinalizer(context.Background(), mockClient, obj, testFinalizerName); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveFinalizer_RemovesWhenPresent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	obj := newFinalizerTestScaledObject([]string{testFinalizerName}, "1")

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ client.ObjectKey, fresh *kedav1alpha1.ScaledObject, _ ...client.GetOption) error {
			*fresh = *newFinalizerTestScaledObject([]string{testFinalizerName}, "1")
			return nil
		})
	mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, patched *kedav1alpha1.ScaledObject, _ client.Patch, _ ...client.PatchOption) error {
			if Contains(patched.GetFinalizers(), testFinalizerName) {
				t.Errorf("expected patched object to not contain finalizer, got %v", patched.GetFinalizers())
			}
			patched.ResourceVersion = "2"
			return nil
		})

	if err := RemoveFinalizer(context.Background(), mockClient, obj, testFinalizerName); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if Contains(obj.GetFinalizers(), testFinalizerName) {
		t.Errorf("expected caller's obj to be synced without finalizer, got %v", obj.GetFinalizers())
	}
}

func TestRemoveFinalizer_NoOpWhenAbsent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	obj := newFinalizerTestScaledObject(nil, "1")

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ client.ObjectKey, fresh *kedav1alpha1.ScaledObject, _ ...client.GetOption) error {
			*fresh = *newFinalizerTestScaledObject(nil, "1")
			return nil
		})
	// No Patch expectation: same reasoning as TestAddFinalizer_NoOpWhenPresent.

	if err := RemoveFinalizer(context.Background(), mockClient, obj, testFinalizerName); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveFinalizer_RetriesOnConflict(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	obj := newFinalizerTestScaledObject([]string{testFinalizerName}, "1")

	conflictErr := apierrors.NewConflict(
		schema.GroupResource{Group: "keda.sh", Resource: "scaledobjects"},
		"test",
		errors.New("resourceVersion mismatch"),
	)

	gomock.InOrder(
		mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test", Namespace: "ns"}, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ client.ObjectKey, fresh *kedav1alpha1.ScaledObject, _ ...client.GetOption) error {
				*fresh = *newFinalizerTestScaledObject([]string{testFinalizerName}, "1")
				return nil
			}),
		mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Return(conflictErr),
		mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test", Namespace: "ns"}, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ client.ObjectKey, fresh *kedav1alpha1.ScaledObject, _ ...client.GetOption) error {
				// Simulates a concurrent status write bumping resourceVersion
				// between the first failed attempt and this retry.
				*fresh = *newFinalizerTestScaledObject([]string{testFinalizerName}, "2")
				return nil
			}),
		mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, patched *kedav1alpha1.ScaledObject, _ client.Patch, _ ...client.PatchOption) error {
				patched.ResourceVersion = "3"
				return nil
			}),
	)

	if err := RemoveFinalizer(context.Background(), mockClient, obj, testFinalizerName); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if Contains(obj.GetFinalizers(), testFinalizerName) {
		t.Errorf("expected caller's obj to be synced without finalizer, got %v", obj.GetFinalizers())
	}
	if obj.ResourceVersion != "3" {
		t.Errorf("expected caller's obj ResourceVersion synced to \"3\" after retry, got %q", obj.ResourceVersion)
	}
}

type fakeAuthReconciler struct {
	client.Client
	eventemitter.EventHandler
	deletedNames []string
}

func (f *fakeAuthReconciler) UpdatePromMetricsOnDelete(namespacedName string) {
	f.deletedNames = append(f.deletedNames, namespacedName)
}

func TestEnsureAuthenticationResourceFinalizer_AddsWhenAbsent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	mockEventHandler := mock_eventemitter.NewMockEventHandler(ctrl)

	triggerAuth := &kedav1alpha1.TriggerAuthentication{
		ObjectMeta: metav1.ObjectMeta{Name: "test-auth", Namespace: "ns", ResourceVersion: "1"},
	}

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test-auth", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ client.ObjectKey, fresh *kedav1alpha1.TriggerAuthentication, _ ...client.GetOption) error {
			*fresh = *triggerAuth.DeepCopy()
			return nil
		})
	mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, patched *kedav1alpha1.TriggerAuthentication, _ client.Patch, _ ...client.PatchOption) error {
			if !Contains(patched.GetFinalizers(), authenticationFinalizer) {
				t.Errorf("expected finalizer added, got %v", patched.GetFinalizers())
			}
			return nil
		})

	reconciler := &fakeAuthReconciler{Client: mockClient, EventHandler: mockEventHandler}
	logger := logr.Discard()

	if err := EnsureAuthenticationResourceFinalizer(context.Background(), logger, reconciler, triggerAuth); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !Contains(triggerAuth.GetFinalizers(), authenticationFinalizer) {
		t.Errorf("expected caller's triggerAuth synced with finalizer, got %v", triggerAuth.GetFinalizers())
	}
}

func TestFinalizeAuthenticationResource_RemovesFinalizer(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	mockEventHandler := mock_eventemitter.NewMockEventHandler(ctrl)

	triggerAuth := &kedav1alpha1.TriggerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-auth",
			Namespace:       "ns",
			Finalizers:      []string{authenticationFinalizer},
			ResourceVersion: "1",
		},
	}

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "test-auth", Namespace: "ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ client.ObjectKey, fresh *kedav1alpha1.TriggerAuthentication, _ ...client.GetOption) error {
			*fresh = *triggerAuth.DeepCopy()
			return nil
		})
	mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, patched *kedav1alpha1.TriggerAuthentication, _ client.Patch, _ ...client.PatchOption) error {
			if Contains(patched.GetFinalizers(), authenticationFinalizer) {
				t.Errorf("expected finalizer removed, got %v", patched.GetFinalizers())
			}
			return nil
		})
	mockEventHandler.EXPECT().Emit(gomock.Any(), "ns/test-auth", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	reconciler := &fakeAuthReconciler{Client: mockClient, EventHandler: mockEventHandler}
	logger := logr.Discard()

	if err := FinalizeAuthenticationResource(context.Background(), logger, reconciler, triggerAuth, "ns/test-auth"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if Contains(triggerAuth.GetFinalizers(), authenticationFinalizer) {
		t.Errorf("expected caller's triggerAuth synced without finalizer, got %v", triggerAuth.GetFinalizers())
	}
	if len(reconciler.deletedNames) != 1 || reconciler.deletedNames[0] != "ns/test-auth" {
		t.Errorf("expected UpdatePromMetricsOnDelete called with \"ns/test-auth\", got %v", reconciler.deletedNames)
	}
}
