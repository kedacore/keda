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
	"k8s.io/client-go/tools/record"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

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
	recorder := record.NewFakeRecorder(1)
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
	recorder := record.NewFakeRecorder(1)
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
	recorder := record.NewFakeRecorder(1)
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
	recorder := record.NewFakeRecorder(1)
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
	recorder := record.NewFakeRecorder(1)
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
	recorder := record.NewFakeRecorder(1)
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
	recorder := record.NewFakeRecorder(1)
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
	recorder := record.NewFakeRecorder(1)
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
	recorder := record.NewFakeRecorder(1)
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
	recorder := record.NewFakeRecorder(1)
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
	recorder := record.NewFakeRecorder(1)
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
		recorder: record.NewFakeRecorder(1),
	}
}

func TestGetHPAHealth_NoHPAName(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	exec := newTestExecutor(mockClient)
	logger := logf.Log.WithName("test")

	so := &v1alpha1.ScaledObject{}
	so.Status.HpaName = ""

	healthy, msg := exec.getHPAHealth(context.TODO(), logger, so)
	assert.True(t, healthy)
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

	healthy, msg := exec.getHPAHealth(context.TODO(), logger, so)
	assert.True(t, healthy, "should treat HPA read error as healthy to avoid flapping")
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

	healthy, msg := exec.getHPAHealth(context.TODO(), logger, so)
	assert.True(t, healthy)
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

	healthy, msg := exec.getHPAHealth(context.TODO(), logger, so)
	assert.False(t, healthy)
	assert.Equal(t, "FailedGetExternalMetric", msg)
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

	healthy, msg := exec.getHPAHealth(context.TODO(), logger, so)
	assert.True(t, healthy, "ScalingDisabled should be treated as healthy since KEDA manages scale-to-zero")
	assert.Empty(t, msg)
}

func TestGetHPAHealth_WithinGracePeriod(t *testing.T) {
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

	healthy, msg := exec.getHPAHealth(context.TODO(), logger, so)
	assert.True(t, healthy, "should be healthy within grace period even if ScalingActive=False")
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
	recorder := record.NewFakeRecorder(1)
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
	recorder := record.NewFakeRecorder(1)
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
	recorder := record.NewFakeRecorder(1)
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
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	exec := NewScaleExecutor(mockClient, mockScaleClient, nil, recorder)

	so := newSOWithHPA()
	so.Status.Conditions = *v1alpha1.GetInitializedConditions()

	mockDeploymentGet(mockClient)
	mockUnhealthyHPA(mockClient)

	result := exec.RequestScale(context.TODO(), &so, true, false, ScaleExecutorOptions{})

	readyCond := result.Conditions.GetReadyCondition()
	assert.True(t, readyCond.IsFalse())
	assert.Equal(t, v1alpha1.ScaledObjectConditionHPAMetricsUnavailableReason, readyCond.Reason)
	assert.Contains(t, readyCond.Message, "FailedGetExternalMetric")
}

func TestRequestScale_ScalerError_HPAUnhealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	exec := NewScaleExecutor(mockClient, mockScaleClient, nil, recorder)

	so := newSOWithHPA()
	so.Status.Conditions = *v1alpha1.GetInitializedConditions()

	mockDeploymentGet(mockClient)
	mockUnhealthyHPA(mockClient)

	// isActive=false, isError=true, no fallback → ScalingDegraded (both broken)
	result := exec.RequestScale(context.TODO(), &so, false, true, ScaleExecutorOptions{})

	readyCond := result.Conditions.GetReadyCondition()
	assert.True(t, readyCond.IsFalse())
	assert.Equal(t, v1alpha1.ScaledObjectConditionScalingDegradedReason, readyCond.Reason)
	assert.Contains(t, readyCond.Message, "FailedGetExternalMetric")
}

func TestRequestScale_ScalerErrorWithFallback_HPAHealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockClient := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
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
