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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

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
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

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

	result := scaleExecutor.RequestScale(context.TODO(), &scaledObject, true, false, ScaleExecutorOptions{})

	assert.Equal(t, pausedReplicaCount, scale.Spec.Replicas)
	condition := result.Conditions.GetActiveCondition()
	assert.Equal(t, false, condition.IsTrue())
	assert.NotNil(t, result.PauseReplicas)
	assert.Equal(t, pausedReplicaCount, *result.PauseReplicas)
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

func TestSkipRedundantLastActiveTimeUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	minReplicas := int32(1)
	recentTime := v1.NewTime(time.Now().Add(-5 * time.Second))

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
			LastActiveTime: &recentTime,
		},
	}

	numberOfReplicas := int32(3)

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &numberOfReplicas,
		},
	})

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(0)

	// LastActiveTime update is skipped because lastActiveTime (5s ago) is
	// within the default polling interval (30s), so result.LastActiveTime is nil.
	result := scaleExecutor.RequestScale(context.Background(), &scaledObject, true, false, ScaleExecutorOptions{})

	assert.Nil(t, result.LastActiveTime)
	condition := result.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsTrue())
}

func TestSkipRedundantLastActiveTimeUpdateWithCustomPollingInterval(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	minReplicas := int32(1)
	pollingInterval := int32(10)
	recentTime := v1.NewTime(time.Now().Add(-5 * time.Second))

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
			PollingInterval: &pollingInterval,
		},
		Status: v1alpha1.ScaledObjectStatus{
			ScaleTargetGVKR: &v1alpha1.GroupVersionKindResource{
				Group: "apps",
				Kind:  "Deployment",
			},
			LastActiveTime: &recentTime,
		},
	}

	numberOfReplicas := int32(3)

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &numberOfReplicas,
		},
	})

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(0)

	// LastActiveTime update is skipped because lastActiveTime (5s ago) is
	// within the custom polling interval (10s), so result.LastActiveTime is nil.
	result := scaleExecutor.RequestScale(context.Background(), &scaledObject, true, false, ScaleExecutorOptions{})

	assert.Nil(t, result.LastActiveTime)
	condition := result.Conditions.GetActiveCondition()
	assert.Equal(t, true, condition.IsTrue())
}

func TestUpdateLastActiveTimeWhenExpired(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_client.NewMockClient(ctrl)
	recorder := record.NewFakeRecorder(1)
	mockScaleClient := mock_scale.NewMockScalesGetter(ctrl)
	mockScaleInterface := mock_scale.NewMockScaleInterface(ctrl)

	scaleExecutor := NewScaleExecutor(client, mockScaleClient, nil, recorder)

	minReplicas := int32(1)
	oldTime := v1.NewTime(time.Now().Add(-60 * time.Second))

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
			LastActiveTime: &oldTime,
		},
	}

	numberOfReplicas := int32(3)

	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).SetArg(2, appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Replicas: &numberOfReplicas,
		},
	})

	mockScaleClient.EXPECT().Scales(gomock.Any()).Return(mockScaleInterface).Times(0)

	// lastActiveTime (60s ago) exceeds the default polling interval (30s),
	// so result.LastActiveTime should be set.
	result := scaleExecutor.RequestScale(context.Background(), &scaledObject, true, false, ScaleExecutorOptions{})

	assert.NotNil(t, result.LastActiveTime)
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
