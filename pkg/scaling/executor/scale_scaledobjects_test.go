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

package executor

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scale"
)

func TestScaleToMinReplicasWhenNotActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	minReplicas := int32(0)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "name",
			},
			MinReplicaCount: &minReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	numberOfReplicas := int32(10)

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &numberOfReplicas,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: numberOfReplicas,
		},
	}

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(2)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(scale, nil)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(scale), gomock.Any())

	client.EXPECT().Status().Return(statusWriter).Times(2)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

	scaleExecutor.RequestScale(context.TODO(), &scaledObject, false, false, &ScaleExecutorOptions{})

	assert.Equal(t, minReplicas, scale.Spec.Replicas)
	condition := scaledObject.Status.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsFalse())
}

func TestScaleToMinReplicasFromLowerInitialReplicaCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	minReplicas := int32(5)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "name",
			},
			MinReplicaCount: &minReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	numberOfReplicas := int32(1)

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &numberOfReplicas,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: numberOfReplicas,
		},
	}

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(2)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(scale, nil)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(scale), gomock.Any())

	client.EXPECT().Status().Return(statusWriter).Times(2)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

	scaleExecutor.RequestScale(context.TODO(), &scaledObject, false, false, &ScaleExecutorOptions{})

	assert.Equal(t, minReplicas, scale.Spec.Replicas)
	condition := scaledObject.Status.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsFalse())
}

func TestScaleFromMinReplicasWhenActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	minReplicas := int32(0)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "name",
			},
			MinReplicaCount: &minReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &minReplicas,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: minReplicas,
		},
	}

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(2)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(scale, nil)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(scale), gomock.Any())

	client.EXPECT().Status().Times(2).Return(statusWriter).Times(3)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(3)

	scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, &ScaleExecutorOptions{})

	assert.Equal(t, int32(1), scale.Spec.Replicas)
	condition := scaledObject.Status.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsTrue())
	assert.Equal(t, "ScalerActive", condition.Reason)
	assert.Equal(t, "Scaling is performed because triggers are active", condition.Message)
}

func TestScaleToIdleReplicasWhenNotActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	idleReplicas := int32(0)
	minReplicas := int32(5)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "name",
			},
			IdleReplicaCount: &idleReplicas,
			MinReplicaCount:  &minReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	numberOfReplicas := int32(10)

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &numberOfReplicas,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: numberOfReplicas,
		},
	}

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(2)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(scale, nil)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(scale), gomock.Any())

	client.EXPECT().Status().Return(statusWriter).Times(2)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

	scaleExecutor.RequestScale(context.TODO(), &scaledObject, false, false, &ScaleExecutorOptions{})

	assert.Equal(t, idleReplicas, scale.Spec.Replicas)
	condition := scaledObject.Status.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsFalse())
}

func TestScaleFromIdleToMinReplicasWhenActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	idleReplicas := int32(0)
	minReplicas := int32(5)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "name",
			},
			IdleReplicaCount: &idleReplicas,
			MinReplicaCount:  &minReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &idleReplicas,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: idleReplicas,
		},
	}

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(2)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(scale, nil)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(scale), gomock.Any())

	client.EXPECT().Status().Times(2).Return(statusWriter).Times(3)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(3)

	scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, &ScaleExecutorOptions{})

	assert.Equal(t, minReplicas, scale.Spec.Replicas)
	condition := scaledObject.Status.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsTrue())
	assert.Equal(t, "ScalerActive", condition.Reason)
	assert.Equal(t, "Scaling is performed because triggers are active", condition.Message)
}

func TestScaleToPausedReplicasCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	pausedReplicaCount := int32(0)
	replicaCount := int32(2)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
			Annotations: map[string]string{
				"autoscaling.keda.sh/paused-replicas": strconv.Itoa(int(pausedReplicaCount)),
			},
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "name",
			},
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicaCount,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: replicaCount,
		},
	}

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(2)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(scale, nil)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(scale), gomock.Any())

	client.EXPECT().Status().Return(statusWriter).Times(2)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

	scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, &ScaleExecutorOptions{})

	assert.Equal(t, pausedReplicaCount, scale.Spec.Replicas)
	condition := scaledObject.Status.Conditions.GetActiveCondition()
	assert.Equal(t, false, condition.IsTrue())
}

func TestEventWitTriggerInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	replicaCount := int32(2)
	idleReplicas := int32(0)
	minReplicas := int32(5)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "name",
			},
			IdleReplicaCount: &idleReplicas,
			MinReplicaCount:  &minReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	// scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicaCount,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: replicaCount,
		},
	}

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(2)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(scale, nil)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(scale), gomock.Any())

	client.EXPECT().Status().Return(statusWriter).AnyTimes()
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, &ScaleExecutorOptions{ActiveTriggers: []string{"testTrigger"}})

	eventstring := <-recorder.Events
	assert.Equal(t, "Normal KEDAScaleTargetActivated Scaled  namespace/name from 2 to 5, triggered by testTrigger", eventstring)
}

func TestNoScaleToMinReplicasWhenNotActiveAndPauseScaleInAnnotationSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	minReplicas := int32(0)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
			Annotations: map[string]string{
				v1alpha1.PausedScaleInAnnotation: "true",
			},
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "name",
			},
			MinReplicaCount: &minReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	numberOfReplicas := int32(10)

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &numberOfReplicas,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: numberOfReplicas,
		},
	}

	// Expect no calls to Scale
	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(0)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	client.EXPECT().Status().Return(statusWriter).Times(2)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

	scaleExecutor.RequestScale(context.TODO(), &scaledObject, false, false, &ScaleExecutorOptions{})

	assert.Equal(t, numberOfReplicas, scale.Spec.Replicas)
	condition := scaledObject.Status.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsFalse())
}

func TestNoScaleToIdleReplicasWhenNotActiveAndPauseScaleInAnnotationSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	idleReplicas := int32(0)
	minReplicas := int32(5)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
			Annotations: map[string]string{
				v1alpha1.PausedScaleInAnnotation: "true",
			},
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "name",
			},
			IdleReplicaCount: &idleReplicas,
			MinReplicaCount:  &minReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	numberOfReplicas := int32(10)

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &numberOfReplicas,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: numberOfReplicas,
		},
	}

	// Expect no calls to Scale
	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(0)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	client.EXPECT().Status().Return(statusWriter).Times(2)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

	scaleExecutor.RequestScale(context.TODO(), &scaledObject, false, false, &ScaleExecutorOptions{})

	assert.Equal(t, numberOfReplicas, scale.Spec.Replicas)
	condition := scaledObject.Status.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsFalse())
}

func TestScaleFromMinReplicasWhenActivationForced(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	minReplicas := int32(0)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
			Annotations: map[string]string{
				v1alpha1.ForceActivationAnnotation: "true",
			},
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "name",
			},
			MinReplicaCount: &minReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &minReplicas,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: minReplicas,
		},
	}

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(2)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(scale, nil)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(scale), gomock.Any())

	client.EXPECT().Status().Return(statusWriter).Times(3)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(3)

	scaleExecutor.RequestScale(context.TODO(), &scaledObject, false, false, &ScaleExecutorOptions{})

	assert.Equal(t, int32(1), scale.Spec.Replicas)
	condition := scaledObject.Status.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsTrue())
	assert.Equal(t, "ScalerActive", condition.Reason)
	assert.Equal(t, "Scaling is performed because activation is being forced by annotation", condition.Message)
}

func TestNoScaleFromMinReplicasWhenActiveAndPausedScaleOutAnnotationSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	minReplicas := int32(0)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
			Annotations: map[string]string{
				v1alpha1.PausedScaleOutAnnotation: "true",
			},
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "name",
			},
			MinReplicaCount: &minReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &minReplicas,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: minReplicas,
		},
	}

	// Expect no calls to Scale
	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(0)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	client.EXPECT().Status().Times(2).Return(statusWriter)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

	scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, &ScaleExecutorOptions{})

	assert.Equal(t, int32(0), scale.Spec.Replicas)
	condition := scaledObject.Status.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsTrue())
}

func TestNoScaleFromIdleReplicasToMinReplicasWhenActiveAndPausedScaleOutAnnotationSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	idleReplicaCount := int32(0)
	minReplicas := int32(5)
	maxReplicas := int32(10)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
			Annotations: map[string]string{
				v1alpha1.PausedScaleOutAnnotation: "true",
			},
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{
				Name: "name",
			},
			IdleReplicaCount: &idleReplicaCount,
			MinReplicaCount:  &minReplicas,
			MaxReplicaCount:  &maxReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}

	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &idleReplicaCount,
		},
	})

	scale := &autoscalingv1.Scale{
		Spec: autoscalingv1.ScaleSpec{
			Replicas: idleReplicaCount,
		},
	}

	// Expect no calls to Scale
	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(0)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	client.EXPECT().Status().Return(statusWriter).Times(2)
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

	scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, &ScaleExecutorOptions{})

	assert.Equal(t, int32(0), scale.Spec.Replicas)
	condition := scaledObject.Status.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsTrue())
}

func TestIsHPAScalingActive(t *testing.T) {
	tests := []struct {
		name     string
		hpaName  string
		soName   string
		getCb    func(ctx context.Context, key runtimeclient.ObjectKey, obj runtimeclient.Object, opts ...runtimeclient.GetOption) error
		expected bool
	}{
		{
			name:    "returns true when ScalingActive is True",
			hpaName: "keda-hpa-test",
			soName:  "test",
			getCb: func(_ context.Context, _ runtimeclient.ObjectKey, obj runtimeclient.Object, _ ...runtimeclient.GetOption) error {
				hpa := obj.(*autoscalingv2.HorizontalPodAutoscaler)
				hpa.Status.Conditions = []autoscalingv2.HorizontalPodAutoscalerCondition{
					{Type: autoscalingv2.ScalingActive, Status: corev1.ConditionTrue},
				}
				return nil
			},
			expected: true,
		},
		{
			name:    "returns false when ScalingActive is False",
			hpaName: "keda-hpa-test",
			soName:  "test",
			getCb: func(_ context.Context, _ runtimeclient.ObjectKey, obj runtimeclient.Object, _ ...runtimeclient.GetOption) error {
				hpa := obj.(*autoscalingv2.HorizontalPodAutoscaler)
				hpa.Status.Conditions = []autoscalingv2.HorizontalPodAutoscalerCondition{
					{Type: autoscalingv2.ScalingActive, Status: corev1.ConditionFalse},
				}
				return nil
			},
			expected: false,
		},
		{
			name:    "returns false when no conditions present",
			hpaName: "keda-hpa-test",
			soName:  "test",
			getCb: func(_ context.Context, _ runtimeclient.ObjectKey, _ runtimeclient.Object, _ ...runtimeclient.GetOption) error {
				return nil
			},
			expected: false,
		},
		{
			name:    "returns false on Get error (fail-open)",
			hpaName: "keda-hpa-test",
			soName:  "test",
			getCb: func(_ context.Context, _ runtimeclient.ObjectKey, _ runtimeclient.Object, _ ...runtimeclient.GetOption) error {
				return fmt.Errorf("not found")
			},
			expected: false,
		},
		{
			name:    "falls back to keda-hpa-<name> when HpaName is empty",
			hpaName: "",
			soName:  "my-so",
			getCb: func(_ context.Context, key runtimeclient.ObjectKey, obj runtimeclient.Object, _ ...runtimeclient.GetOption) error {
				if key.Name != "keda-hpa-my-so" {
					return fmt.Errorf("unexpected HPA name: %s", key.Name)
				}
				hpa := obj.(*autoscalingv2.HorizontalPodAutoscaler)
				hpa.Status.Conditions = []autoscalingv2.HorizontalPodAutoscalerCondition{
					{Type: autoscalingv2.ScalingActive, Status: corev1.ConditionTrue},
				}
				return nil
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			client := mock_client.NewMockClient(ctrl)
			client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(tt.getCb)

			executor := &scaleExecutor{client: client}
			so := &v1alpha1.ScaledObject{}
			so.Name = tt.soName
			so.Namespace = "default"
			so.Status.HpaName = tt.hpaName

			result := executor.isHPAScalingActive(context.TODO(), so)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// clientGetRouter dispatches mock client.Get calls by object type:
// *appsv1.Deployment is served with the given replicas,
// *autoscalingv2.HorizontalPodAutoscaler is served with the given ScalingActive status.
func clientGetRouter(replicas int32, hpaActive corev1.ConditionStatus) func(ctx context.Context, key runtimeclient.ObjectKey, obj runtimeclient.Object, opts ...runtimeclient.GetOption) error {
	return func(_ context.Context, _ runtimeclient.ObjectKey, obj runtimeclient.Object, _ ...runtimeclient.GetOption) error {
		switch o := obj.(type) {
		case *appsv1.Deployment:
			o.Spec.Replicas = &replicas
		case *autoscalingv2.HorizontalPodAutoscaler:
			o.Status.Conditions = []autoscalingv2.HorizontalPodAutoscalerCondition{
				{Type: autoscalingv2.ScalingActive, Status: hpaActive},
			}
		}
		return nil
	}
}

func TestCatchUpScaling_HPAInactive_ScalesDirectly(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	executor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	currentReplicas := int32(1)
	desiredReplicas := int32(3)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{Name: "test-so", Namespace: "ns"},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{Name: "deploy"},
		},
		Status: v1alpha1.ScaledObjectStatus{
			HpaName:         "keda-hpa-test-so",
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{Group: "apps", Kind: "Deployment"},
		},
	}
	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	// client.Get: first for GetCurrentReplicas (Deployment), second for isHPAScalingActive (HPA)
	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(clientGetRouter(currentReplicas, corev1.ConditionFalse)).Times(2)

	scale := &autoscalingv1.Scale{Spec: autoscalingv1.ScaleSpec{Replicas: currentReplicas}}
	mockScaleClient.EXPECT().Scales("ns").Return(mockScaleInterface).Times(2)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(scale, nil)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(scale), gomock.Any()).Return(scale, nil)

	client.EXPECT().Status().Return(statusWriter).AnyTimes()
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	executor.RequestScale(context.TODO(), &scaledObject, true, false, &ScaleExecutorOptions{
		DesiredReplicas: &desiredReplicas,
	})

	assert.Equal(t, desiredReplicas, scale.Spec.Replicas)

	eventMsg := <-recorder.Events
	assert.Contains(t, eventMsg, "KEDAScaleTargetCatchUpScaled")
	assert.Contains(t, eventMsg, "Catch-up scaled")
}

func TestCatchUpScaling_HPAActive_NoDirectScale(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	executor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	currentReplicas := int32(1)
	desiredReplicas := int32(3)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{Name: "test-so", Namespace: "ns"},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{Name: "deploy"},
		},
		Status: v1alpha1.ScaledObjectStatus{
			HpaName:         "keda-hpa-test-so",
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{Group: "apps", Kind: "Deployment"},
		},
	}
	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	// HPA is active -> no catch-up scaling
	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(clientGetRouter(currentReplicas, corev1.ConditionTrue)).Times(2)

	// No scale calls expected
	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(0)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	client.EXPECT().Status().Return(statusWriter).AnyTimes()
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	executor.RequestScale(context.TODO(), &scaledObject, true, false, &ScaleExecutorOptions{
		DesiredReplicas: &desiredReplicas,
	})

	// Verify no events were emitted (no catch-up happened)
	assert.Equal(t, 0, len(recorder.Events))
}

func TestCatchUpScaling_CappedByMaxReplicas(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	executor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	currentReplicas := int32(1)
	desiredReplicas := int32(10)
	maxReplicas := int32(3)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{Name: "test-so", Namespace: "ns"},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef:  &v1alpha1.ScaleTarget{Name: "deploy"},
			MaxReplicaCount: &maxReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			HpaName:         "keda-hpa-test-so",
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{Group: "apps", Kind: "Deployment"},
		},
	}
	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(clientGetRouter(currentReplicas, corev1.ConditionFalse)).Times(2)

	scale := &autoscalingv1.Scale{Spec: autoscalingv1.ScaleSpec{Replicas: currentReplicas}}
	mockScaleClient.EXPECT().Scales("ns").Return(mockScaleInterface).Times(2)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(scale, nil)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(scale), gomock.Any()).Return(scale, nil)

	client.EXPECT().Status().Return(statusWriter).AnyTimes()
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	executor.RequestScale(context.TODO(), &scaledObject, true, false, &ScaleExecutorOptions{
		DesiredReplicas: &desiredReplicas,
	})

	// desired=10 capped to maxReplicas=3
	assert.Equal(t, maxReplicas, scale.Spec.Replicas)
}

func TestCatchUpScaling_SkippedWhenPauseScaleOut(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	executor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	currentReplicas := int32(1)
	desiredReplicas := int32(3)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-so",
			Namespace: "ns",
			Annotations: map[string]string{
				v1alpha1.PausedScaleOutAnnotation: "true",
			},
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{Name: "deploy"},
		},
		Status: v1alpha1.ScaledObjectStatus{
			HpaName:         "keda-hpa-test-so",
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{Group: "apps", Kind: "Deployment"},
		},
	}
	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(clientGetRouter(currentReplicas, corev1.ConditionFalse)).AnyTimes()

	// No scale calls expected
	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(0)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	client.EXPECT().Status().Return(statusWriter).AnyTimes()
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	executor.RequestScale(context.TODO(), &scaledObject, true, false, &ScaleExecutorOptions{
		DesiredReplicas: &desiredReplicas,
	})

	assert.Equal(t, 0, len(recorder.Events))
}

func TestCatchUpScaling_NoOpWhenDesiredEqualsCurrent(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	executor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	currentReplicas := int32(2)
	maxReplicas := int32(2)
	desiredReplicas := int32(5) // will be capped to 2, same as current

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{Name: "test-so", Namespace: "ns"},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef:  &v1alpha1.ScaleTarget{Name: "deploy"},
			MaxReplicaCount: &maxReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			HpaName:         "keda-hpa-test-so",
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{Group: "apps", Kind: "Deployment"},
		},
	}
	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(clientGetRouter(currentReplicas, corev1.ConditionFalse)).Times(2)

	// No scale calls -- desired (5) capped to max (2) == current (2)
	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(0)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	client.EXPECT().Status().Return(statusWriter).AnyTimes()
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	executor.RequestScale(context.TODO(), &scaledObject, true, false, &ScaleExecutorOptions{
		DesiredReplicas: &desiredReplicas,
	})

	assert.Equal(t, 0, len(recorder.Events))
}

func TestCatchUpScaling_DesiredReplicasNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)
	statusWriter := mock_client.NewMockStatusWriter(ctrl)

	executor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	currentReplicas := int32(1)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{Name: "test-so", Namespace: "ns"},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef: &v1alpha1.ScaleTarget{Name: "deploy"},
		},
		Status: v1alpha1.ScaledObjectStatus{
			HpaName:         "keda-hpa-test-so",
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{Group: "apps", Kind: "Deployment"},
		},
	}
	scaledObject.Status.Conditions = *v1alpha1.GetInitializedConditions()

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(clientGetRouter(currentReplicas, corev1.ConditionFalse)).AnyTimes()

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(0)
	mockScaleInterface.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockScaleInterface.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	client.EXPECT().Status().Return(statusWriter).AnyTimes()
	statusWriter.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	executor.RequestScale(context.TODO(), &scaledObject, true, false, &ScaleExecutorOptions{
		DesiredReplicas: nil,
	})

	assert.Equal(t, 0, len(recorder.Events))
}
