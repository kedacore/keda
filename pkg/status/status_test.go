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

package status

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
)

func TestTransformObjectSkipsUnchangedStatus(t *testing.T) {
	objects := []client.Object{
		&kedav1alpha1.ScaledObject{},
		&kedav1alpha1.ScaledJob{},
		&kedav1alpha1.TriggerAuthentication{},
		&kedav1alpha1.ClusterTriggerAuthentication{},
		&eventingv1alpha1.CloudEventSource{},
		&eventingv1alpha1.ClusterCloudEventSource{},
	}
	for _, object := range objects {
		t.Run(fmt.Sprintf("%T", object), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockClient := mock_client.NewMockStatusClient(ctrl)
			// No Status() expectation: an unchanged object must not make a request.
			err := TransformObject(context.Background(), mockClient, logr.Discard(), object, nil, func(client.Object, any) error {
				return nil
			})
			require.NoError(t, err)
		})
	}
}

func TestUpdateScaledObjectStatusSerializedChanges(t *testing.T) {
	for _, tc := range []struct {
		name     string
		original []string
		updated  []string
		patch    string
	}{
		{name: "omitted empty list", updated: []string{}},
		{name: "unchanged list", original: []string{"metric"}, updated: []string{"metric"}},
		{name: "changed list", updated: []string{"metric"}, patch: `{"status":{"externalMetricNames":["metric"]}}`},
		{name: "cleared list", original: []string{"metric"}, patch: `{"status":{"externalMetricNames":null}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockClient := mock_client.NewMockStatusClient(ctrl)
			object := &kedav1alpha1.ScaledObject{Status: kedav1alpha1.ScaledObjectStatus{ExternalMetricNames: tc.original}}
			updated := &kedav1alpha1.ScaledObjectStatus{ExternalMetricNames: tc.updated}
			if tc.patch != "" {
				writer := mock_client.NewMockStatusWriter(ctrl)
				mockClient.EXPECT().Status().Return(writer)
				writer.EXPECT().Patch(gomock.Any(), object, gomock.Any()).DoAndReturn(func(_ context.Context, obj client.Object, patch client.Patch, _ ...client.SubResourcePatchOption) error {
					data, err := patch.Data(obj)
					require.NoError(t, err)
					assert.JSONEq(t, tc.patch, string(data))
					return nil
				})
			}
			require.NoError(t, UpdateScaledObjectStatus(context.Background(), mockClient, logr.Discard(), object, updated))
			assert.Equal(t, *updated, object.Status, "local transformation must be preserved even when no patch is sent")
		})
	}
}

func TestTransformObjectPreservesErrors(t *testing.T) {
	for _, transformFails := range []bool{true, false} {
		t.Run(fmt.Sprintf("transformFails=%t", transformFails), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockClient := mock_client.NewMockStatusClient(ctrl)
			expectedErr := errors.New("failed")
			object := &kedav1alpha1.ScaledObject{}
			if !transformFails {
				writer := mock_client.NewMockStatusWriter(ctrl)
				mockClient.EXPECT().Status().Return(writer)
				writer.EXPECT().Patch(gomock.Any(), object, gomock.Any()).Return(expectedErr)
			}
			err := TransformObject(context.Background(), mockClient, logr.Discard(), object, nil, func(obj client.Object, _ any) error {
				if transformFails {
					return expectedErr
				}
				obj.(*kedav1alpha1.ScaledObject).Status.ExternalMetricNames = []string{"metric"}
				return nil
			})
			require.ErrorIs(t, err, expectedErr)
		})
	}
}

func TestSetStatusConditionsPatchesChangedReasonAndMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockStatusClient(ctrl)
	writer := mock_client.NewMockStatusWriter(ctrl)
	object := &kedav1alpha1.ScaledObject{Status: kedav1alpha1.ScaledObjectStatus{
		Conditions: kedav1alpha1.Conditions{{Type: kedav1alpha1.ConditionReady, Status: metav1.ConditionTrue, Reason: "old", Message: "old"}},
	}}
	conditions := kedav1alpha1.Conditions{{Type: kedav1alpha1.ConditionReady, Status: metav1.ConditionTrue, Reason: "new", Message: "new"}}
	mockClient.EXPECT().Status().Return(writer)
	writer.EXPECT().Patch(gomock.Any(), object, gomock.Any()).DoAndReturn(func(_ context.Context, obj client.Object, patch client.Patch, _ ...client.SubResourcePatchOption) error {
		data, err := patch.Data(obj)
		require.NoError(t, err)
		assert.JSONEq(t, `{"status":{"conditions":[{"type":"Ready","status":"True","reason":"new","message":"new"}]}}`, string(data))
		return nil
	})
	require.NoError(t, SetStatusConditions(context.Background(), mockClient, logr.Discard(), object, &conditions))
	assert.Equal(t, conditions, object.Status.Conditions)
}
