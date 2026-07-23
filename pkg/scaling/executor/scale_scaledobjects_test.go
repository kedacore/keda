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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
	"github.com/kedacore/keda/v2/pkg/mock/mock_scale"
)

func TestScaleToMinReplicasWhenNotActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

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

	result := scaleExecutor.RequestScale(context.TODO(), &scaledObject, false, false, ScaleExecutorOptions{})

	assert.Equal(t, minReplicas, scale.Spec.Replicas)
	condition := result.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsFalse())
}

func TestScaleToMinReplicasFromLowerInitialReplicaCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

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

	result := scaleExecutor.RequestScale(context.TODO(), &scaledObject, false, false, ScaleExecutorOptions{})

	assert.Equal(t, minReplicas, scale.Spec.Replicas)
	condition := result.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsFalse())
}

func TestScaleFromMinReplicasWhenActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

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

	result := scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, ScaleExecutorOptions{})

	assert.Equal(t, int32(1), scale.Spec.Replicas)
	condition := result.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsTrue())
	assert.Equal(t, "ScalerActive", condition.Reason)
	assert.Equal(t, "Scaling is performed because triggers are active", condition.Message)
	assert.NotNil(t, result.LastActiveTime)
}

func TestScaleToIdleReplicasWhenNotActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

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

	result := scaleExecutor.RequestScale(context.TODO(), &scaledObject, false, false, ScaleExecutorOptions{})

	assert.Equal(t, idleReplicas, scale.Spec.Replicas)
	condition := result.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsFalse())
}

func TestScaleFromIdleToMinReplicasWhenActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

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

	result := scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, ScaleExecutorOptions{})

	assert.Equal(t, minReplicas, scale.Spec.Replicas)
	condition := result.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsTrue())
	assert.Equal(t, "ScalerActive", condition.Reason)
	assert.Equal(t, "Scaling is performed because triggers are active", condition.Message)
	assert.NotNil(t, result.LastActiveTime)
}

func TestScaleToPausedReplicasCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	scaledObject := v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
			Annotations: map[string]string{
				"autoscaling.keda.sh/paused-replicas": "0",
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

	// GetCurrentReplicas is called before handlePaused, so we need a mock for it.
	replicaCount := int32(2)
	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicaCount,
		},
	})

	result := scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, ScaleExecutorOptions{})

	// PauseReplicas should not be set
	assert.Nil(t, result.PauseReplicas)
	// Executor should set paused condition
	condition := result.Conditions.GetPausedCondition()
	assert.Equal(t, true, condition.IsTrue())
	// Active condition should not be set (we returned early before checking triggers)
	activeCondition := result.Conditions.GetActiveCondition()
	assert.Equal(t, false, activeCondition.IsTrue())
}

func TestEventWitTriggerInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

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

	scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, ScaleExecutorOptions{ActiveTriggers: []string{"testTrigger"}})

	eventstring := <-recorder.Events
	assert.Equal(t, "Normal KEDAScaleTargetActivated Scaled  namespace/name from 2 to 5, triggered by testTrigger", eventstring)
}

func TestNoScaleToMinReplicasWhenNotActiveAndPauseScaleInAnnotationSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

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

	result := scaleExecutor.RequestScale(context.TODO(), &scaledObject, false, false, ScaleExecutorOptions{})

	assert.Equal(t, numberOfReplicas, scale.Spec.Replicas)
	condition := result.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsFalse())
}

func TestNoScaleToIdleReplicasWhenNotActiveAndPauseScaleInAnnotationSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

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

	result := scaleExecutor.RequestScale(context.TODO(), &scaledObject, false, false, ScaleExecutorOptions{})

	assert.Equal(t, numberOfReplicas, scale.Spec.Replicas)
	condition := result.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsFalse())
}

func TestScaleFromMinReplicasWhenActivationForced(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

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

	result := scaleExecutor.RequestScale(context.TODO(), &scaledObject, false, false, ScaleExecutorOptions{})

	assert.Equal(t, int32(1), scale.Spec.Replicas)
	condition := result.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsTrue())
	assert.Equal(t, "ScalerActive", condition.Reason)
	assert.Equal(t, "Scaling is performed because activation is being forced by annotation", condition.Message)
	assert.NotNil(t, result.LastActiveTime)
}

func TestNoScaleFromMinReplicasWhenActiveAndPausedScaleOutAnnotationSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

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

	result := scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, ScaleExecutorOptions{})

	assert.Equal(t, int32(0), scale.Spec.Replicas)
	condition := result.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsTrue())
}

func TestNoScaleFromIdleReplicasToMinReplicasWhenActiveAndPausedScaleOutAnnotationSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

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

	result := scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, ScaleExecutorOptions{})

	assert.Equal(t, int32(0), scale.Spec.Replicas)
	condition := result.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsTrue())
}

// --------------------------------------------------------------------------- //
// ----------         getHPAHealth tests                             --------- //
// --------------------------------------------------------------------------- //

func newTestExecutor(client *mock_client.MockClient) *scaleExecutor {
	return &scaleExecutor{
		client:   client,
		logger:   logf.Log.WithName("test"),
		recorder: events.NewFakeRecorder(1),
	}
}

func TestGetHPAHealth_NoHPAName(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	exec := newTestExecutor(mockClient)
	logger := logf.Log.WithName("test")

	so := &v1alpha1.ScaledObject{}
	so.Status.HpaName = ""

	status, reason, msg := exec.getHPAHealth(context.TODO(), logger, so)
	assert.Empty(t, status, "no HPA name yet means the HPA cannot be observed, not that it is healthy")
	assert.Empty(t, reason)
	assert.Empty(t, msg)
}

func TestGetHPAHealth_HPAReadError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	exec := newTestExecutor(mockClient)
	logger := logf.Log.WithName("test")

	so := &v1alpha1.ScaledObject{}
	so.Name = "test-so"
	so.Namespace = "test-ns"
	so.Status.HpaName = "my-hpa"

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "my-hpa", Namespace: "test-ns"}, gomock.Any()).
		Return(errors.New("api unavailable"))

	status, reason, msg := exec.getHPAHealth(context.TODO(), logger, so)
	assert.Empty(t, status, "a read error means the HPA cannot be observed, not that it is healthy")
	assert.Empty(t, reason)
	assert.Empty(t, msg)
}

func TestGetHPAHealth_ScalingActiveTrue(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	exec := newTestExecutor(mockClient)
	logger := logf.Log.WithName("test")

	so := &v1alpha1.ScaledObject{}
	so.Name = "test-so"
	so.Namespace = "test-ns"
	so.Status.HpaName = "my-hpa"

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "my-hpa", Namespace: "test-ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *autoscalingv2.HorizontalPodAutoscaler, _ ...interface{}) error {
			obj.Status.Conditions = []autoscalingv2.HorizontalPodAutoscalerCondition{
				{Type: autoscalingv2.ScalingActive, Status: corev1.ConditionTrue},
			}
			return nil
		})

	status, reason, msg := exec.getHPAHealth(context.TODO(), logger, so)
	assert.Equal(t, v1.ConditionTrue, status)
	assert.Equal(t, v1alpha1.ScaledObjectConditionHPAActiveReason, reason)
	assert.Empty(t, msg)
}

func TestGetHPAHealth_ScalingActiveFalse(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	exec := newTestExecutor(mockClient)
	logger := logf.Log.WithName("test")

	so := &v1alpha1.ScaledObject{}
	so.Name = "test-so"
	so.Namespace = "test-ns"
	so.Status.HpaName = "my-hpa"

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "my-hpa", Namespace: "test-ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *autoscalingv2.HorizontalPodAutoscaler, _ ...interface{}) error {
			obj.Status.Conditions = []autoscalingv2.HorizontalPodAutoscalerCondition{
				{Type: autoscalingv2.ScalingActive, Status: corev1.ConditionFalse, Reason: "FailedGetExternalMetric", Message: "unable to get metrics"},
			}
			return nil
		})

	status, reason, msg := exec.getHPAHealth(context.TODO(), logger, so)
	assert.Equal(t, v1.ConditionFalse, status)
	assert.Equal(t, v1alpha1.ScaledObjectConditionHPAMetricsUnavailableReason, reason)
	assert.Equal(t, "FailedGetExternalMetric: unable to get metrics", msg, "both the HPA's reason and its own message must be preserved")
}

func TestGetHPAHealth_ScalingDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	exec := newTestExecutor(mockClient)
	logger := logf.Log.WithName("test")

	so := &v1alpha1.ScaledObject{}
	so.Name = "test-so"
	so.Namespace = "test-ns"
	so.Status.HpaName = "my-hpa"

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "my-hpa", Namespace: "test-ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *autoscalingv2.HorizontalPodAutoscaler, _ ...interface{}) error {
			obj.Status.Conditions = []autoscalingv2.HorizontalPodAutoscalerCondition{
				// ScalingDisabled is set by the HPA controller when the target has 0 replicas
				// (scale-to-zero managed by KEDA). This should NOT be treated as unhealthy.
				{Type: autoscalingv2.ScalingActive, Status: corev1.ConditionFalse, Reason: "ScalingDisabled", Message: "scaling is disabled since the replica count of the target is zero"},
			}
			return nil
		})

	status, reason, msg := exec.getHPAHealth(context.TODO(), logger, so)
	assert.Equal(t, v1.ConditionTrue, status, "ScalingDisabled should be treated as healthy since KEDA manages scale-to-zero")
	assert.Equal(t, v1alpha1.ScaledObjectConditionHPAScalingDisabledReason, reason, "ScalingDisabled must be distinguishable from a normally-scaling HPA")
	assert.Equal(t, "scaling is disabled since the replica count of the target is zero", msg)
}

func TestGetHPAHealth_NoGracePeriod(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	exec := newTestExecutor(mockClient)
	logger := logf.Log.WithName("test")

	so := &v1alpha1.ScaledObject{}
	so.Name = "test-so"
	so.Namespace = "test-ns"
	so.Status.HpaName = "my-hpa"

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "my-hpa", Namespace: "test-ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *autoscalingv2.HorizontalPodAutoscaler, _ ...interface{}) error {
			obj.CreationTimestamp = v1.Now() // just created
			obj.Status.Conditions = []autoscalingv2.HorizontalPodAutoscalerCondition{
				{Type: autoscalingv2.ScalingActive, Status: corev1.ConditionFalse, Reason: "FailedGetExternalMetric"},
			}
			return nil
		})

	// #7914: the anti-flap grace period was removed. HPAActive now absorbs any flapping instead,
	// so a freshly created but unhealthy HPA must be reported unhealthy immediately.
	status, reason, msg := exec.getHPAHealth(context.TODO(), logger, so)
	assert.Equal(t, v1.ConditionFalse, status, "grace period was removed; a freshly created but unhealthy HPA must be reported unhealthy immediately")
	assert.Equal(t, v1alpha1.ScaledObjectConditionHPAMetricsUnavailableReason, reason)
	assert.Equal(t, "FailedGetExternalMetric", msg, "cond.Message was empty, so the message falls back to the HPA's own reason alone")
}

// TestGetHPAHealth_NoScalingActiveConditionYet covers the third "cannot observe" path: the HPA
// exists and was read successfully, but hasn't reported a ScalingActive condition yet (e.g. right
// after creation, before the HPA controller's first sync). This must not be treated as healthy.
func TestGetHPAHealth_NoScalingActiveConditionYet(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	exec := newTestExecutor(mockClient)
	logger := logf.Log.WithName("test")

	so := &v1alpha1.ScaledObject{}
	so.Name = "test-so"
	so.Namespace = "test-ns"
	so.Status.HpaName = "my-hpa"

	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "my-hpa", Namespace: "test-ns"}, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *autoscalingv2.HorizontalPodAutoscaler, _ ...interface{}) error {
			obj.Status.Conditions = []autoscalingv2.HorizontalPodAutoscalerCondition{
				{Type: autoscalingv2.AbleToScale, Status: corev1.ConditionTrue},
			}
			return nil
		})

	status, reason, msg := exec.getHPAHealth(context.TODO(), logger, so)
	assert.Empty(t, status, "an HPA that hasn't reported ScalingActive yet cannot be observed, not treated as healthy")
	assert.Empty(t, reason)
	assert.Empty(t, msg)
}

// --------------------------------------------------------------------------- //
// ----------    HPA health aggregation in RequestScale tests        --------- //
// --------------------------------------------------------------------------- //

// newSOWithHPA creates a minimal ScaledObject with an HPA name for testing.
func newSOWithHPA() v1alpha1.ScaledObject {
	minReplicas := int32(1)
	return v1alpha1.ScaledObject{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-so",
			Namespace: "test-ns",
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef:  &v1alpha1.ScaleTarget{Name: "test-deploy"},
			MinReplicaCount: &minReplicas,
		},
		Status: v1alpha1.ScaledObjectStatus{
			HpaName: "my-hpa",
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
		},
	}
}

// mockHealthyHPA sets up the mock to return an HPA with ScalingActive=True.
func mockHealthyHPA(mockClient *mock_client.MockClient) {
	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "my-hpa", Namespace: "test-ns"}, gomock.AssignableToTypeOf(&autoscalingv2.HorizontalPodAutoscaler{})).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *autoscalingv2.HorizontalPodAutoscaler, _ ...interface{}) error {
			obj.Status.Conditions = []autoscalingv2.HorizontalPodAutoscalerCondition{
				{Type: autoscalingv2.ScalingActive, Status: corev1.ConditionTrue},
			}
			obj.Status.CurrentMetrics = []autoscalingv2.MetricStatus{{}}
			return nil
		})
}

// mockUnhealthyHPA sets up the mock to return an HPA with ScalingActive=False.
func mockUnhealthyHPA(mockClient *mock_client.MockClient) {
	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "my-hpa", Namespace: "test-ns"}, gomock.AssignableToTypeOf(&autoscalingv2.HorizontalPodAutoscaler{})).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *autoscalingv2.HorizontalPodAutoscaler, _ ...interface{}) error {
			obj.Status.Conditions = []autoscalingv2.HorizontalPodAutoscalerCondition{
				{Type: autoscalingv2.ScalingActive, Status: corev1.ConditionFalse, Reason: "FailedGetExternalMetric", Message: "metrics not available"},
			}
			return nil
		})
}

// mockDeploymentGet sets up the mock for the deployment Get that resolver.GetCurrentReplicas calls.
func mockDeploymentGet(mockClient *mock_client.MockClient) {
	replicas := int32(1)
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *appsv1.Deployment, _ ...interface{}) error {
			obj.Spec.Replicas = &replicas
			return nil
		})
}

func TestRequestScale_AllHealthy_HPAHealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	exec := NewScaleExecutor(mockClient, mockScaleClient, nil, recorder)

	so := newSOWithHPA()
	so.Status.Conditions = *v1alpha1.GetInitializedConditions()

	// GetCurrentReplicas: return 1 replica (at minReplicas, so no scaling needed)
	mockDeploymentGet(mockClient)
	// HPA health check
	mockHealthyHPA(mockClient)

	result := exec.RequestScale(context.TODO(), &so, true, false, ScaleExecutorOptions{})

	readyCond := result.Conditions.GetReadyCondition()
	assert.True(t, readyCond.IsTrue())
	assert.Equal(t, v1alpha1.ScaledObjectConditionReadySuccessReason, readyCond.Reason)
}

func TestRequestScale_ScalerError_HPAHealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	exec := NewScaleExecutor(mockClient, mockScaleClient, nil, recorder)

	so := newSOWithHPA()
	so.Status.Conditions = *v1alpha1.GetInitializedConditions()

	mockDeploymentGet(mockClient)
	mockHealthyHPA(mockClient)

	// isActive=false, isError=true, no fallback → TriggerError
	result := exec.RequestScale(context.TODO(), &so, false, true, ScaleExecutorOptions{})

	readyCond := result.Conditions.GetReadyCondition()
	assert.True(t, readyCond.IsFalse())
	assert.Equal(t, "TriggerError", readyCond.Reason)
}

func TestRequestScale_PartialError_HPAHealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	exec := NewScaleExecutor(mockClient, mockScaleClient, nil, recorder)

	so := newSOWithHPA()
	so.Status.Conditions = *v1alpha1.GetInitializedConditions()

	mockDeploymentGet(mockClient)
	mockHealthyHPA(mockClient)

	// isActive=true, isError=true → PartialTriggerError
	result := exec.RequestScale(context.TODO(), &so, true, true, ScaleExecutorOptions{})

	readyCond := result.Conditions.GetReadyCondition()
	assert.True(t, readyCond.IsUnknown())
	assert.Equal(t, "PartialTriggerError", readyCond.Reason)
}

func TestRequestScale_NoError_HPAUnhealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	exec := NewScaleExecutor(mockClient, mockScaleClient, nil, recorder)

	so := newSOWithHPA()
	so.Status.Conditions = *v1alpha1.GetInitializedConditions()

	mockDeploymentGet(mockClient)
	mockUnhealthyHPA(mockClient)

	result := exec.RequestScale(context.TODO(), &so, true, false, ScaleExecutorOptions{})

	// #7914: HPA health no longer affects Ready. The ScaledObject itself is valid, so Ready stays True.
	readyCond := result.Conditions.GetReadyCondition()
	assert.True(t, readyCond.IsTrue())
	assert.Equal(t, v1alpha1.ScaledObjectConditionReadySuccessReason, readyCond.Reason)

	// HPA unhealthiness is now surfaced exclusively via the HPAActive condition, with both the
	// HPA's reason (normalized into HPAMetricsUnavailable) and its own message preserved.
	hpaActiveCond := result.Conditions.GetHPAActiveCondition()
	assert.True(t, hpaActiveCond.IsFalse())
	assert.Equal(t, v1alpha1.ScaledObjectConditionHPAMetricsUnavailableReason, hpaActiveCond.Reason)
	assert.Equal(t, "FailedGetExternalMetric: metrics not available", hpaActiveCond.Message)
}

func TestRequestScale_ScalerError_HPAUnhealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	exec := NewScaleExecutor(mockClient, mockScaleClient, nil, recorder)

	so := newSOWithHPA()
	so.Status.Conditions = *v1alpha1.GetInitializedConditions()

	mockDeploymentGet(mockClient)
	mockUnhealthyHPA(mockClient)

	// isActive=false, isError=true, no fallback → TriggerError (SO-level reason only, no more combined ScalingDegraded)
	result := exec.RequestScale(context.TODO(), &so, false, true, ScaleExecutorOptions{})

	readyCond := result.Conditions.GetReadyCondition()
	assert.True(t, readyCond.IsFalse())
	assert.Equal(t, "TriggerError", readyCond.Reason)
	assert.NotEqual(t, v1alpha1.ScaledObjectConditionScalingDegradedReason, readyCond.Reason)

	// HPA unhealthiness is now surfaced exclusively via the HPAActive condition, independent of Ready.
	hpaActiveCond := result.Conditions.GetHPAActiveCondition()
	assert.True(t, hpaActiveCond.IsFalse())
	assert.Equal(t, v1alpha1.ScaledObjectConditionHPAMetricsUnavailableReason, hpaActiveCond.Reason)
	assert.Equal(t, "FailedGetExternalMetric: metrics not available", hpaActiveCond.Message)
}

// TestRequestScale_TransientHPAGap_ReadyStaysTrue proves the #7914 fix: a transient HPA metric gap
// (e.g. during a rolling restart of the metrics adapter) must no longer flip the ScaledObject's Ready
// condition to False. Only the dedicated HPAActive condition should reflect the HPA's unhealthy state,
// with the underlying HPA condition reason preserved in the message.
func TestRequestScale_TransientHPAGap_ReadyStaysTrue(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	exec := NewScaleExecutor(mockClient, mockScaleClient, nil, recorder)

	so := newSOWithHPA()
	so.Status.Conditions = *v1alpha1.GetInitializedConditions()
	// Ready was already True before this reconcile, as it would be in steady state.
	so.Status.Conditions.SetReadyCondition(v1.ConditionTrue, v1alpha1.ScaledObjectConditionReadySuccessReason, v1alpha1.ScaledObjectConditionReadySuccessMessage)

	mockDeploymentGet(mockClient)
	mockUnhealthyHPA(mockClient) // simulates a transient HPAMetricsUnavailable-style gap

	result := exec.RequestScale(context.TODO(), &so, true, false, ScaleExecutorOptions{})

	readyCond := result.Conditions.GetReadyCondition()
	assert.True(t, readyCond.IsTrue(), "Ready must stay True during a transient HPA metric gap")

	hpaActiveCond := result.Conditions.GetHPAActiveCondition()
	assert.True(t, hpaActiveCond.IsFalse(), "HPAActive must reflect the transient HPA unhealthiness")
	assert.Equal(t, v1alpha1.ScaledObjectConditionHPAMetricsUnavailableReason, hpaActiveCond.Reason)
	assert.Equal(t, "FailedGetExternalMetric: metrics not available", hpaActiveCond.Message, "both the underlying HPA reason and its own message must be preserved")
}

// TestRequestScale_HPAHealthy_HPAActiveTrue proves a healthy, actively-scaling HPA produces
// HPAActive=True with the active reason, passing through the HPA's own (here empty) message.
func TestRequestScale_HPAHealthy_HPAActiveTrue(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	exec := NewScaleExecutor(mockClient, mockScaleClient, nil, recorder)

	so := newSOWithHPA()
	so.Status.Conditions = *v1alpha1.GetInitializedConditions()

	mockDeploymentGet(mockClient)
	mockHealthyHPA(mockClient)

	result := exec.RequestScale(context.TODO(), &so, true, false, ScaleExecutorOptions{})

	hpaActiveCond := result.Conditions.GetHPAActiveCondition()
	assert.True(t, hpaActiveCond.IsTrue(), "HPAActive must be True when the HPA is actively scaling")
	assert.Equal(t, v1alpha1.ScaledObjectConditionHPAActiveReason, hpaActiveCond.Reason)
	assert.Empty(t, hpaActiveCond.Message, "the mocked HPA's ScalingActive condition has no message, so HPAActive.Message passes that through")
}

// TestRequestScale_HPAScalingDisabled_HPAActiveTrueWithDistinctReason proves the ScalingDisabled
// case (KEDA-managed scale-to-zero) is surfaced as HPAActive=True but with a reason distinct from a
// normally-scaling HPA, so consumers can tell "actively scaling" apart from "intentionally idle".
func TestRequestScale_HPAScalingDisabled_HPAActiveTrueWithDistinctReason(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	exec := NewScaleExecutor(mockClient, mockScaleClient, nil, recorder)

	so := newSOWithHPA()
	so.Status.Conditions = *v1alpha1.GetInitializedConditions()

	mockDeploymentGet(mockClient)
	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "my-hpa", Namespace: "test-ns"}, gomock.AssignableToTypeOf(&autoscalingv2.HorizontalPodAutoscaler{})).
		DoAndReturn(func(_ context.Context, _ types.NamespacedName, obj *autoscalingv2.HorizontalPodAutoscaler, _ ...interface{}) error {
			obj.Status.Conditions = []autoscalingv2.HorizontalPodAutoscalerCondition{
				{Type: autoscalingv2.ScalingActive, Status: corev1.ConditionFalse, Reason: "ScalingDisabled", Message: "scaling is disabled since the replica count of the target is zero"},
			}
			return nil
		})

	result := exec.RequestScale(context.TODO(), &so, true, false, ScaleExecutorOptions{})

	hpaActiveCond := result.Conditions.GetHPAActiveCondition()
	assert.True(t, hpaActiveCond.IsTrue(), "ScalingDisabled must still be reported as HPAActive=True, not unhealthy")
	assert.Equal(t, v1alpha1.ScaledObjectConditionHPAScalingDisabledReason, hpaActiveCond.Reason, "the reason must distinguish this from a normally-scaling HPA")
	assert.Equal(t, "scaling is disabled since the replica count of the target is zero", hpaActiveCond.Message)
}

// TestCheckHPAHealth_CannotObserve_HPAActiveUnchanged proves the tri-state contract of getHPAHealth:
// when the HPA cannot currently be observed (e.g. a transient read error), checkHPAHealth must leave
// a previously-observed HPAActive condition exactly as it was, instead of optimistically flipping it
// to True or otherwise touching it.
func TestCheckHPAHealth_CannotObserve_HPAActiveUnchanged(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	exec := newTestExecutor(mockClient)
	logger := logf.Log.WithName("test")

	so := &v1alpha1.ScaledObject{}
	so.Name = "test-so"
	so.Namespace = "test-ns"
	so.Status.HpaName = "my-hpa"

	// HPA read fails (e.g. a transient API server hiccup): getHPAHealth cannot observe the HPA.
	mockClient.EXPECT().Get(gomock.Any(), types.NamespacedName{Name: "my-hpa", Namespace: "test-ns"}, gomock.Any()).
		Return(errors.New("api unavailable"))

	result := &ScaleResult{Conditions: v1alpha1.Conditions{}}
	// Simulate a previously-observed HPAActive=False from an earlier, successful reconcile.
	result.Conditions.SetHPAActiveCondition(v1.ConditionFalse, v1alpha1.ScaledObjectConditionHPAMetricsUnavailableReason, "FailedGetExternalMetric: metrics not available")

	exec.checkHPAHealth(context.TODO(), logger, so, result)

	hpaActiveCond := result.Conditions.GetHPAActiveCondition()
	assert.True(t, hpaActiveCond.IsFalse(), "a transient read error must not flip HPAActive away from its last observed state")
	assert.Equal(t, v1alpha1.ScaledObjectConditionHPAMetricsUnavailableReason, hpaActiveCond.Reason)
	assert.Equal(t, "FailedGetExternalMetric: metrics not available", hpaActiveCond.Message, "the last genuinely observed HPAActive state must persist untouched")
}

// TestRequestScale_HPAActive_NotInSharedDefaults proves HPAActive is lazy-set only: it is not part of
// GetInitializedConditions/AreInitialized, which are shared with ScaledJob (a kind that has no HPA).
// This is what keeps ScaledJob unaffected by this change - no HPAActive condition is ever initialized
// on ScaledJob, since ScaledJob never calls checkHPAHealth (only RequestScale for ScaledObject does).
func TestRequestScale_HPAActive_NotInSharedDefaults(t *testing.T) {
	initialized := v1alpha1.GetInitializedConditions()
	assert.Len(t, *initialized, 4, "GetInitializedConditions must remain unchanged so ScaledJob is unaffected")
	for _, cond := range *initialized {
		assert.NotEqual(t, v1alpha1.ConditionHPAActive, cond.Type, "HPAActive must not be part of the shared default conditions")
	}

	// Before checkHPAHealth ever runs (e.g. fresh ScaledJob-style conditions), HPAActive is unset.
	conditions := *v1alpha1.GetInitializedConditions()
	hpaActiveCond := conditions.GetHPAActiveCondition()
	assert.Empty(t, hpaActiveCond.Type, "HPAActive should not appear until explicitly set")
}

func TestRequestScale_ScalerErrorWithFallback_HPAHealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	recorder := events.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	exec := NewScaleExecutor(mockClient, mockScaleClient, nil, recorder)

	so := newSOWithHPA()
	so.Status.Conditions = *v1alpha1.GetInitializedConditions()
	fallbackReplicas := int32(5)
	so.Spec.Fallback = &v1alpha1.Fallback{Replicas: fallbackReplicas}

	mockDeploymentGet(mockClient)
	mockHealthyHPA(mockClient)

	// isActive=false, isError=true, fallback configured → Ready stays True
	result := exec.RequestScale(context.TODO(), &so, false, true, ScaleExecutorOptions{})

	readyCond := result.Conditions.GetReadyCondition()
	assert.Truef(t, readyCond.IsTrue(), "with fallback configured and HPA healthy, Ready should be True, got %s/%s", readyCond.Status, readyCond.Reason)
}
