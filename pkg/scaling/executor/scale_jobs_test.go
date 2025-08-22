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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/mock/mock_client"
)

func TestCleanUpNormalCase(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup ScaledJob
	// successfulJobHistoryLimit = 2
	// failedJobHistoryLimit = 2
	scaledJob := getMockScaledJob(2, 2)

	var actualDeletedJobName = make(map[string]string)

	// Setup current running jobs
	client := getMockClient(t, ctrl, &[]mockJobParameter{
		{Name: "name1", CompletionTime: "2020-07-29T15:37:00Z", JobConditionType: batchv1.JobComplete},
		{Name: "name2", CompletionTime: "2020-07-29T15:36:00Z", JobConditionType: batchv1.JobComplete},
		{Name: "name3", CompletionTime: "2020-07-29T15:38:00Z", JobConditionType: batchv1.JobComplete},
	}, &actualDeletedJobName)

	scaleExecutor := getMockScaleExecutor(client)

	err := scaleExecutor.cleanUp(ctx, scaledJob)
	if err != nil {
		t.Errorf("Unable to cleanup as: %v", err)
		return
	}

	_, ok := actualDeletedJobName["name2"]
	assert.True(t, ok)
}

func TestNewNewScalingStrategy(t *testing.T) {
	logger := logf.Log.WithName("ScaledJobTest")
	strategy := NewScalingStrategy(logger, getMockScaledJobWithStrategy("custom", "custom", int32(10), "0"))
	assert.Equal(t, "executor.customScalingStrategy", fmt.Sprintf("%T", strategy))
	strategy = NewScalingStrategy(logger, getMockScaledJobWithStrategy("accurate", "accurate", int32(0), "0"))
	assert.Equal(t, "executor.accurateScalingStrategy", fmt.Sprintf("%T", strategy))
	strategy = NewScalingStrategy(logger, getMockScaledJobWithDefaultStrategy("default"))
	assert.Equal(t, "executor.defaultScalingStrategy", fmt.Sprintf("%T", strategy))
	strategy = NewScalingStrategy(logger, getMockScaledJobWithStrategy("default", "default", int32(0), "0"))
	assert.Equal(t, "executor.defaultScalingStrategy", fmt.Sprintf("%T", strategy))
}

func maxScaleValue(maxValue, _ int64) int64 {
	return maxValue
}

func TestDefaultScalingStrategy(t *testing.T) {
	logger := logf.Log.WithName("ScaledJobTest")
	strategy := NewScalingStrategy(logger, getMockScaledJobWithDefaultStrategy("default"))
	// maxScale doesn't exceed MaxReplicaCount. You can ignore on this sceanrio
	// pendingJobCount isn't relevant on this scenario
	assert.Equal(t, int64(1), maxScaleValue(strategy.GetEffectiveMaxScale(3, 2, 0, 5, 1)))
	assert.Equal(t, int64(2), maxScaleValue(strategy.GetEffectiveMaxScale(2, 0, 0, 5, 1)))
}

func TestCustomScalingStrategy(t *testing.T) {
	logger := logf.Log.WithName("ScaledJobTest")
	customScalingQueueLengthDeduction := int32(1)
	customScalingRunningJobPercentage := "0.5"
	strategy := NewScalingStrategy(logger, getMockScaledJobWithStrategy("custom", "custom", customScalingQueueLengthDeduction, customScalingRunningJobPercentage))
	// maxScale doesn't exceed MaxReplicaCount. You can ignore on this sceanrio
	// pendingJobCount isn't relevant on this scenario
	assert.Equal(t, int64(1), maxScaleValue(strategy.GetEffectiveMaxScale(3, 2, 0, 5, 1)))
	assert.Equal(t, int64(9), maxScaleValue(strategy.GetEffectiveMaxScale(10, 0, 0, 10, 1)))
	strategy = NewScalingStrategy(logger, getMockScaledJobWithCustomStrategyWithNilParameter("custom", "custom"))

	// If you don't set the two parameters is the same behavior as DefaultStrategy
	assert.Equal(t, int64(1), maxScaleValue(strategy.GetEffectiveMaxScale(3, 2, 0, 5, 1)))
	assert.Equal(t, int64(2), maxScaleValue(strategy.GetEffectiveMaxScale(2, 0, 0, 5, 1)))

	// Empty String will be DefaultStrategy
	customScalingQueueLengthDeduction = int32(1)
	customScalingRunningJobPercentage = ""
	strategy = NewScalingStrategy(logger, getMockScaledJobWithStrategy("custom", "custom", customScalingQueueLengthDeduction, customScalingRunningJobPercentage))
	assert.Equal(t, "executor.defaultScalingStrategy", fmt.Sprintf("%T", strategy))

	// Set 0 as customScalingRunningJobPercentage
	customScalingQueueLengthDeduction = int32(2)
	customScalingRunningJobPercentage = "0"
	strategy = NewScalingStrategy(logger, getMockScaledJobWithStrategy("custom", "custom", customScalingQueueLengthDeduction, customScalingRunningJobPercentage))
	assert.Equal(t, int64(1), maxScaleValue(strategy.GetEffectiveMaxScale(3, 2, 0, 5, 1)))

	// Exceed the MaxReplicaCount
	customScalingQueueLengthDeduction = int32(-2)
	customScalingRunningJobPercentage = "0"
	strategy = NewScalingStrategy(logger, getMockScaledJobWithStrategy("custom", "custom", customScalingQueueLengthDeduction, customScalingRunningJobPercentage))
	assert.Equal(t, int64(4), maxScaleValue(strategy.GetEffectiveMaxScale(3, 2, 0, 4, 1)))
}

func TestAccurateScalingStrategy(t *testing.T) {
	logger := logf.Log.WithName("ScaledJobTest")
	strategy := NewScalingStrategy(logger, getMockScaledJobWithStrategy("accurate", "accurate", 0, "0"))
	// maxScale doesn't exceed MaxReplicaCount. You can ignore on this sceanrio
	assert.Equal(t, int64(3), maxScaleValue(strategy.GetEffectiveMaxScale(3, 2, 0, 5, 1)))
	assert.Equal(t, int64(3), maxScaleValue(strategy.GetEffectiveMaxScale(5, 2, 0, 5, 1)))

	// Test with 2 pending jobs
	assert.Equal(t, int64(1), maxScaleValue(strategy.GetEffectiveMaxScale(3, 4, 2, 10, 1)))
	assert.Equal(t, int64(1), maxScaleValue(strategy.GetEffectiveMaxScale(5, 4, 2, 5, 1)))
}

func TestEagerScalingStrategy(t *testing.T) {
	logger := logf.Log.WithName("ScaledJobTest")
	strategy := NewScalingStrategy(logger, getMockScaledJobWithStrategy("eager", "eager", 0, "0"))

	maxScale, scaleTo := strategy.GetEffectiveMaxScale(4, 3, 0, 10, 1)
	assert.Equal(t, int64(4), maxScale)
	assert.Equal(t, int64(10), scaleTo)
	maxScale, scaleTo = strategy.GetEffectiveMaxScale(4, 0, 3, 10, 1)
	assert.Equal(t, int64(4), maxScale)
	assert.Equal(t, int64(10), scaleTo)

	maxScale, scaleTo = strategy.GetEffectiveMaxScale(4, 7, 0, 10, 1)
	assert.Equal(t, int64(3), maxScale)
	assert.Equal(t, int64(10), scaleTo)
	maxScale, scaleTo = strategy.GetEffectiveMaxScale(4, 1, 6, 10, 1)
	assert.Equal(t, int64(3), maxScale)
	assert.Equal(t, int64(10), scaleTo)

	maxScale, scaleTo = strategy.GetEffectiveMaxScale(15, 0, 0, 10, 1)
	assert.Equal(t, int64(10), maxScale)
	assert.Equal(t, int64(10), scaleTo)
}

func TestCleanUpMixedCaseWithSortByTime(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup ScaledJob
	// successfulJobHistoryLimit = 1
	// failedJobHistoryLimit = 2
	scaledJob := getMockScaledJob(1, 2)

	var actualDeletedJobName = make(map[string]string)

	// Setup current running jobs
	client := getMockClient(t, ctrl, &[]mockJobParameter{
		{Name: "success1", CompletionTime: "2020-07-29T15:37:00Z", JobConditionType: batchv1.JobComplete},
		{Name: "success2", CompletionTime: "2020-07-29T15:36:00Z", JobConditionType: batchv1.JobComplete},
		{Name: "success3", CompletionTime: "2020-07-29T15:35:00Z", JobConditionType: batchv1.JobComplete},
		{Name: "fail1", CompletionTime: "2020-07-29T15:37:00Z", JobConditionType: batchv1.JobFailed},
		{Name: "fail2", CompletionTime: "2020-07-29T15:36:00Z", JobConditionType: batchv1.JobFailed},
		{Name: "fail3", CompletionTime: "2020-07-29T15:38:00Z", JobConditionType: batchv1.JobFailed},
	}, &actualDeletedJobName)

	scaleExecutor := getMockScaleExecutor(client)

	err := scaleExecutor.cleanUp(ctx, scaledJob)
	if err != nil {
		t.Errorf("Unable to cleanup as: %v", err)
		return
	}
	assert.Equal(t, 3, len(actualDeletedJobName))
	_, ok := actualDeletedJobName["success2"]
	assert.True(t, ok)
	_, ok = actualDeletedJobName["success3"]
	assert.True(t, ok)
	_, ok = actualDeletedJobName["fail2"]
	assert.True(t, ok)
}

func TestRunningJobCountSmallerMinReplicaCount(t *testing.T) {
	scaleExecutor := getMockScaleExecutor(nil)
	scaledJob := getMockScaledJobWithMinReplicaCountAndDefaultStrategy(2)

	var runningJobCount int64
	var scaleTo int64
	var maxScale int64
	var pendingJobCount int64

	effectiveMaxScale, scaleTo := scaleExecutor.getScalingDecision(scaledJob, runningJobCount, scaleTo, maxScale, pendingJobCount, scaleExecutor.logger)
	assert.Equal(t, int64(2), effectiveMaxScale)
	assert.Equal(t, int64(2), scaleTo)
}

func TestRunningJobCountIsDeductedFromMinReplicaCount(t *testing.T) {
	scaleExecutor := getMockScaleExecutor(nil)
	scaledJob := getMockScaledJobWithMinReplicaCountAndDefaultStrategy(2)

	var runningJobCount int64 = 1
	var scaleTo int64
	var maxScale int64
	var pendingJobCount int64

	effectiveMaxScale, scaleTo := scaleExecutor.getScalingDecision(scaledJob, runningJobCount, scaleTo, maxScale, pendingJobCount, scaleExecutor.logger)
	assert.Equal(t, int64(1), effectiveMaxScale)
	assert.Equal(t, int64(1), scaleTo)
}

func TestRunningJobCountGreaterOrEqualMinReplicaCountExecutesScalingStrategy(t *testing.T) {
	scaleExecutor := getMockScaleExecutor(nil)
	scaledJob := getMockScaledJobWithMinReplicaCountAndDefaultStrategy(1)

	var runningJobCount int64 = 1
	var scaleTo int64 = 2
	var maxScale int64 = 2
	var pendingJobCount int64

	effectiveMaxScale, scaleTo := scaleExecutor.getScalingDecision(scaledJob, runningJobCount, scaleTo, maxScale, pendingJobCount, scaleExecutor.logger)
	assert.Equal(t, int64(2), effectiveMaxScale)
	assert.Equal(t, int64(2), scaleTo)
}

func TestCleanUpDefaultValue(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup ScaledJob
	// The optional value is not configured 100 by default
	// successfulJobHistoryLimit = nil
	// failedJobHistoryLimit = nil
	scaledJob := getMockScaledJobWithDefault()

	var actualDeletedJobName = make(map[string]string)
	mockJobParameters := make([]mockJobParameter, 202)
	// Setup the first one will be removed.
	mockJobParameters[0] = mockJobParameter{Name: "success0", CompletionTime: "2020-07-29T15:35:00Z", JobConditionType: batchv1.JobComplete}
	for i := 1; i < 101; i++ {
		mockJobParameters[i] = mockJobParameter{Name: fmt.Sprintf("success%d", i), CompletionTime: "2020-07-29T15:40:00Z", JobConditionType: batchv1.JobComplete}
	}
	mockJobParameters[101] = mockJobParameter{Name: "fail101", CompletionTime: "2020-07-29T15:35:00Z", JobConditionType: batchv1.JobFailed}
	for i := 102; i < 202; i++ {
		mockJobParameters[i] = mockJobParameter{Name: fmt.Sprintf("fail%d", i), CompletionTime: "2020-07-29T15:40:00Z", JobConditionType: batchv1.JobFailed}
	}

	// Setup current running jobs
	client := getMockClient(t, ctrl, &mockJobParameters, &actualDeletedJobName)

	scaleExecutor := getMockScaleExecutor(client)

	err := scaleExecutor.cleanUp(ctx, scaledJob)
	if err != nil {
		t.Errorf("Unable to cleanup as: %v", err)
		return
	}

	assert.Equal(t, 2, len(actualDeletedJobName))
	_, ok := actualDeletedJobName["success0"]
	assert.True(t, ok)
	_, ok = actualDeletedJobName["fail101"]
	assert.True(t, ok)
}

func TestGetPendingJobCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pendingPodConditions := []string{"Ready", "PodScheduled"}
	readyCondition := getPodCondition(v1.PodReady)
	podScheduledCondition := getPodCondition(v1.PodScheduled)

	testPendingJobTestData := []pendingJobTestData{
		{PodStatus: v1.PodStatus{Phase: v1.PodSucceeded}, PendingJobCount: 0},
		{PodStatus: v1.PodStatus{Phase: v1.PodRunning}, PendingJobCount: 0},
		{PodStatus: v1.PodStatus{Phase: v1.PodFailed}, PendingJobCount: 1},
		{PodStatus: v1.PodStatus{Phase: v1.PodPending}, PendingJobCount: 1},
		{PodStatus: v1.PodStatus{Phase: v1.PodUnknown}, PendingJobCount: 1},
		{PendingPodConditions: pendingPodConditions, PodStatus: v1.PodStatus{Conditions: []v1.PodCondition{}}, PendingJobCount: 1},
		{PendingPodConditions: pendingPodConditions, PodStatus: v1.PodStatus{Conditions: []v1.PodCondition{readyCondition}}, PendingJobCount: 1},
		{PendingPodConditions: pendingPodConditions, PodStatus: v1.PodStatus{Conditions: []v1.PodCondition{podScheduledCondition}}, PendingJobCount: 1},
		{PendingPodConditions: pendingPodConditions, PodStatus: v1.PodStatus{Conditions: []v1.PodCondition{readyCondition, podScheduledCondition}}, PendingJobCount: 0},

		// Test case: Job with 2 pods, both have all conditions - should NOT be pending
		{
			PendingPodConditions: pendingPodConditions,
			PodStatuses: []v1.PodStatus{
				{Conditions: []v1.PodCondition{readyCondition, podScheduledCondition}},
				{Conditions: []v1.PodCondition{readyCondition, podScheduledCondition}},
			},
			PendingJobCount: 0,
		},

		// Test case: Job with 2 pods, 1 has all conditions, 1 doesn't - should NOT be pending
		{
			PendingPodConditions: pendingPodConditions,
			PodStatuses: []v1.PodStatus{
				{Conditions: []v1.PodCondition{readyCondition, podScheduledCondition}},
				{Conditions: []v1.PodCondition{readyCondition}}, // missing PodScheduled
			},
			PendingJobCount: 0,
		},

		// Test case: Job with 2 pods, neither has all conditions - should be pending
		{
			PendingPodConditions: pendingPodConditions,
			PodStatuses: []v1.PodStatus{
				{Conditions: []v1.PodCondition{readyCondition}},        // missing PodScheduled
				{Conditions: []v1.PodCondition{podScheduledCondition}}, // missing Ready
			},
			PendingJobCount: 1,
		},

		// Test case: Job with 3 pods, 1 has all conditions - should NOT be pending
		{
			PendingPodConditions: pendingPodConditions,
			PodStatuses: []v1.PodStatus{
				{Conditions: []v1.PodCondition{}},                                      // no conditions
				{Conditions: []v1.PodCondition{readyCondition}},                        // partial conditions
				{Conditions: []v1.PodCondition{readyCondition, podScheduledCondition}}, // all conditions
			},
			PendingJobCount: 0,
		},

		// Test case: Job with 3 pods, none have all conditions - should be pending
		{
			PendingPodConditions: pendingPodConditions,
			PodStatuses: []v1.PodStatus{
				{Conditions: []v1.PodCondition{}},                      // no conditions
				{Conditions: []v1.PodCondition{readyCondition}},        // partial conditions
				{Conditions: []v1.PodCondition{podScheduledCondition}}, // partial conditions
			},
			PendingJobCount: 1,
		},
		// Edge case: Empty conditions list
		{
			PendingPodConditions: []string{},
			PodStatuses: []v1.PodStatus{
				{Phase: v1.PodRunning, Conditions: []v1.PodCondition{readyCondition}},
			},
			PendingJobCount: 0, // No conditions to check, pod is running, so not pending
		},
	}

	for i, testData := range testPendingJobTestData {
		ctx := context.Background()

		// Support both single pod and multiple pods test cases
		var client *mock_client.MockClient
		if len(testData.PodStatuses) > 0 {
			client = getMockClientForTestingPendingPodsMultiple(t, ctrl, testData.PodStatuses)
		} else {
			client = getMockClientForTestingPendingPods(t, ctrl, testData.PodStatus)
		}

		scaleExecutor := getMockScaleExecutor(client)
		scaledJob := getMockScaledJobWithPendingPodConditions(testData.PendingPodConditions)
		result := scaleExecutor.getPendingJobCount(ctx, scaledJob)

		assert.Equal(t, testData.PendingJobCount, result, "Test case %d failed", i)
	}
}

// Test to check issue #6727
func TestAreAllPendingPodConditionsFulfilledBugScenario(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pendingPodConditions := []string{"Ready", "PodScheduled"}
	readyCondition := getPodCondition(v1.PodReady)
	podScheduledCondition := getPodCondition(v1.PodScheduled)

	podStatuses := []v1.PodStatus{
		{Conditions: []v1.PodCondition{readyCondition, podScheduledCondition}},
		{Conditions: []v1.PodCondition{readyCondition, podScheduledCondition}},
	}

	ctx := context.Background()
	client := getMockClientForTestingPendingPodsMultiple(t, ctrl, podStatuses)
	scaleExecutor := getMockScaleExecutor(client)
	scaledJob := getMockScaledJobWithPendingPodConditions(pendingPodConditions)

	result := scaleExecutor.getPendingJobCount(ctx, scaledJob)

	// Should be 0, because both pods have all required conditions
	assert.Equal(t, int64(0), result, "Job with pods having all conditions should not be pending")
}

func TestCreateJobs(t *testing.T) {
	ctx := context.Background()
	logger := logf.Log.WithName("CreateJobsTest")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock_client.NewMockClient(ctrl)
	scaleExecutor := getMockScaleExecutor(client)

	client.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, obj runtime.Object, _ ...runtimeclient.CreateOption) {
		j, ok := obj.(*batchv1.Job)
		if !ok {
			t.Error("Cast failed on batchv1.Job at mocking client.Create()")
		}
		if ok {
			assert.Equal(t, "test-", j.ObjectMeta.GenerateName)
			assert.Equal(t, "test", j.ObjectMeta.Namespace)
		}
	}).Times(2).
		Return(nil)

	scaledJob := getMockScaledJobWithDefaultStrategyAndMeta("test")
	scaleExecutor.createJobs(ctx, logger, scaledJob, 2, 2)
}

func TestGenerateJobs(t *testing.T) {
	var (
		expectedAnnotations = map[string]string{
			"test":                         "test",
			"scaledjob.keda.sh/generation": "0",
		}
		expectedLabels = map[string]string{
			"app.kubernetes.io/managed-by": "keda-operator",
			"app.kubernetes.io/name":       "test",
			"app.kubernetes.io/part-of":    "test",
			"app.kubernetes.io/version":    "main",
			"scaledjob.keda.sh/name":       "test",
			"test":                         "test",
		}
	)

	logger := logf.Log.WithName("GenerateJobsTest")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock_client.NewMockClient(ctrl)
	scaleExecutor := getMockScaleExecutor(client)
	scaledJob := getMockScaledJobWithDefaultStrategyAndMeta("test")

	jobs := scaleExecutor.generateJobs(logger, scaledJob, 2)

	assert.Equal(t, 2, len(jobs))
	for _, j := range jobs {
		assert.Equal(t, expectedAnnotations, j.ObjectMeta.Annotations)
		assert.Equal(t, expectedLabels, j.ObjectMeta.Labels)
		assert.Equal(t, v1.RestartPolicyOnFailure, j.Spec.Template.Spec.RestartPolicy)
	}
}

type mockJobParameter struct {
	Name             string
	CompletionTime   string
	JobConditionType batchv1.JobConditionType
}

type pendingJobTestData struct {
	PendingPodConditions []string
	PodStatus            v1.PodStatus   // For single pod tests
	PodStatuses          []v1.PodStatus // For multiple pods tests
	PendingJobCount      int64
}

func getMockScaleExecutor(client *mock_client.MockClient) *scaleExecutor {
	scheme := runtime.NewScheme()
	utilruntime.Must(kedav1alpha1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	return &scaleExecutor{
		client:           client,
		scaleClient:      nil,
		reconcilerScheme: scheme,
		logger:           logf.Log.WithName("scaleexecutor"),
		recorder:         record.NewFakeRecorder(1),
	}
}

func getMockScaledJob(successfulJobHistoryLimit, failedJobHistoryLimit int) *kedav1alpha1.ScaledJob {
	successfulJobHistoryLimit32 := int32(successfulJobHistoryLimit)
	failedJobHistoryLimit32 := int32(failedJobHistoryLimit)
	scaledJob := &kedav1alpha1.ScaledJob{
		Spec: kedav1alpha1.ScaledJobSpec{
			SuccessfulJobsHistoryLimit: &successfulJobHistoryLimit32,
			FailedJobsHistoryLimit:     &failedJobHistoryLimit32,
		},
	}
	scaledJob.ObjectMeta.Name = "azure-storage-queue-consumer"
	return scaledJob
}

func getMockScaledJobWithDefault() *kedav1alpha1.ScaledJob {
	scaledJob := &kedav1alpha1.ScaledJob{
		Spec: kedav1alpha1.ScaledJobSpec{},
	}
	scaledJob.ObjectMeta.Name = "azure-storage-queue-consumer"
	return scaledJob
}

func getMockScaledJobWithMinReplicaCountAndDefaultStrategy(minReplicaCount int32) *kedav1alpha1.ScaledJob {
	maxReplicaCount := int32(100)
	scaledJob := &kedav1alpha1.ScaledJob{
		Spec: kedav1alpha1.ScaledJobSpec{
			MinReplicaCount: &minReplicaCount,
			MaxReplicaCount: &maxReplicaCount,
		},
	}
	return scaledJob
}

func getMockScaledJobWithStrategy(name, scalingStrategy string, customScalingQueueLengthDeduction int32, customScalingRunningJobPercentage string) *kedav1alpha1.ScaledJob {
	scaledJob := &kedav1alpha1.ScaledJob{
		Spec: kedav1alpha1.ScaledJobSpec{
			ScalingStrategy: kedav1alpha1.ScalingStrategy{
				Strategy:                          scalingStrategy,
				CustomScalingQueueLengthDeduction: &customScalingQueueLengthDeduction,
				CustomScalingRunningJobPercentage: customScalingRunningJobPercentage,
			},
		},
	}
	scaledJob.ObjectMeta.Name = name
	return scaledJob
}

func getMockScaledJobWithCustomStrategyWithNilParameter(name, scalingStrategy string) *kedav1alpha1.ScaledJob {
	scaledJob := &kedav1alpha1.ScaledJob{
		Spec: kedav1alpha1.ScaledJobSpec{
			ScalingStrategy: kedav1alpha1.ScalingStrategy{
				Strategy: scalingStrategy,
			},
		},
	}
	scaledJob.ObjectMeta.Name = name
	return scaledJob
}

func getMockScaledJobWithDefaultStrategy(name string) *kedav1alpha1.ScaledJob {
	scaledJob := &kedav1alpha1.ScaledJob{
		Spec: kedav1alpha1.ScaledJobSpec{
			JobTargetRef: &batchv1.JobSpec{},
		},
	}
	scaledJob.ObjectMeta.Name = name
	return scaledJob
}

func getMockScaledJobWithDefaultStrategyAndMeta(name string) *kedav1alpha1.ScaledJob {
	sc := getMockScaledJobWithDefaultStrategy(name)
	sc.ObjectMeta.Namespace = "test"
	sc.ObjectMeta.Labels = map[string]string{"test": "test"}
	sc.ObjectMeta.Annotations = map[string]string{"test": "test"}
	return sc
}

func getMockScaledJobWithPendingPodConditions(pendingPodConditions []string) *kedav1alpha1.ScaledJob {
	scaledJob := &kedav1alpha1.ScaledJob{
		Spec: kedav1alpha1.ScaledJobSpec{
			ScalingStrategy: kedav1alpha1.ScalingStrategy{
				PendingPodConditions: pendingPodConditions,
			},
		},
	}

	return scaledJob
}

func getMockClient(t *testing.T, ctrl *gomock.Controller, jobs *[]mockJobParameter, deletedJobName *map[string]string) *mock_client.MockClient {
	client := mock_client.NewMockClient(ctrl)

	client.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, list runtime.Object, _ ...runtimeclient.ListOption) {
		j, ok := list.(*batchv1.JobList)
		if ok {
			for _, job := range *jobs {
				j.Items = append(j.Items, *getJob(t, job.Name, job.CompletionTime, job.JobConditionType))
			}
		}
	}).
		Return(nil)

	client.EXPECT().
		Delete(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, obj runtime.Object, opt *runtimeclient.DeleteOptions) {
		j, ok := obj.(*batchv1.Job)
		if !ok {
			t.Error("Cast failed on batchv1.Job at mocking client.Delete()")
		}
		if *opt.PropagationPolicy != metav1.DeletePropagationBackground {
			t.Error("Job Delete PropagationPolicy is not DeletePropagationForeground")
		}
		(*deletedJobName)[j.GetName()] = j.GetName()
	}).
		Return(nil).AnyTimes()
	return client
}

func getMockClientForTestingPendingPods(t *testing.T, ctrl *gomock.Controller, podStatus v1.PodStatus) *mock_client.MockClient {
	client := mock_client.NewMockClient(ctrl)
	gomock.InOrder(
		// listing jobs
		client.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, list runtime.Object, _ ...runtimeclient.ListOption) {
			j, ok := list.(*batchv1.JobList)

			if !ok {
				t.Error("Cast failed on batchv1.JobList at mocking client.List()")
			}

			if ok {
				j.Items = append(j.Items, batchv1.Job{})
			}
		}).
			Return(nil),
		// listing pods
		client.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, list runtime.Object, _ ...runtimeclient.ListOption) {
			p, ok := list.(*v1.PodList)

			if !ok {
				t.Error("Cast failed on v1.PodList at mocking client.List()")
			}

			if ok {
				p.Items = append(p.Items, v1.Pod{Status: podStatus})
			}
		}).
			Return(nil),
	)

	return client
}

// getMockClientForTestingPendingPodsMultiple returns mock client function for testing pending pod conditions with multiple pods in a job
func getMockClientForTestingPendingPodsMultiple(t *testing.T, ctrl *gomock.Controller, podStatuses []v1.PodStatus) *mock_client.MockClient {
	client := mock_client.NewMockClient(ctrl)
	gomock.InOrder(
		// listing jobs
		client.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, list runtime.Object, _ ...runtimeclient.ListOption) {
			j, ok := list.(*batchv1.JobList)

			if !ok {
				t.Error("Cast failed on batchv1.JobList at mocking client.List()")
			}

			if ok {
				j.Items = append(j.Items, batchv1.Job{})
			}
		}).
			Return(nil),
		// listing pods - multiple pods this time
		client.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_ context.Context, list runtime.Object, _ ...runtimeclient.ListOption) {
			p, ok := list.(*v1.PodList)

			if !ok {
				t.Error("Cast failed on v1.PodList at mocking client.List()")
			}

			if ok {
				for _, podStatus := range podStatuses {
					p.Items = append(p.Items, v1.Pod{Status: podStatus})
				}
			}
		}).
			Return(nil),
	)

	return client
}

func getJob(t *testing.T, name string, completionTime string, jobConditionType batchv1.JobConditionType) *batchv1.Job {
	parsedCompletionTime, err := time.Parse(time.RFC3339, completionTime)
	completionTimeT := metav1.NewTime(parsedCompletionTime)
	if err != nil {
		t.Errorf("Can not parse %s as RFC3339: %v", completionTime, err)
	}
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: batchv1.JobSpec{},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{
					Type:   jobConditionType,
					Status: v1.ConditionTrue,
				},
			},
			CompletionTime: &completionTimeT,
		},
	}
}

func getPodCondition(podConditionType v1.PodConditionType) v1.PodCondition {
	return v1.PodCondition{
		Type:   podConditionType,
		Status: v1.ConditionTrue,
	}
}
