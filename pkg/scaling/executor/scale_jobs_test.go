package executor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/api/v1alpha1"
	"github.com/kedacore/keda/pkg/mock/mock_client"
)

func TestCleanUpNormalCase(t *testing.T) {
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

	scaleExecutor.cleanUp(scaledJob)

	_, ok := actualDeletedJobName["name2"]
	assert.True(t, ok)
}

func TestCleanUpMixedCaseWithSortByTime(t *testing.T) {
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

	scaleExecutor.cleanUp(scaledJob)

	assert.Equal(t, 3, len(actualDeletedJobName))
	_, ok := actualDeletedJobName["success2"]
	assert.True(t, ok)
	_, ok = actualDeletedJobName["success3"]
	assert.True(t, ok)
	_, ok = actualDeletedJobName["fail2"]
	assert.True(t, ok)
}

func TestCleanUpDefaultValue(t *testing.T) {
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

	scaleExecutor.cleanUp(scaledJob)

	assert.Equal(t, 2, len(actualDeletedJobName))
	_, ok := actualDeletedJobName["success0"]
	assert.True(t, ok)
	_, ok = actualDeletedJobName["fail101"]
	assert.True(t, ok)
}

type mockJobParameter struct {
	Name             string
	CompletionTime   string
	JobConditionType batchv1.JobConditionType
}

func getMockScaleExecutor(client *mock_client.MockClient) *scaleExecutor {
	return &scaleExecutor{
		client:           client,
		scaleClient:      nil,
		reconcilerScheme: nil,
		logger:           logf.Log.WithName("scaleexecutor"),
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
		Delete(gomock.Any(), gomock.Any()).Do(func(_ context.Context, obj runtime.Object) {
		j, ok := obj.(*batchv1.Job)
		if !ok {
			t.Error("Cast failed on batchv1.Job at mocking client.Delete()")
		}
		(*deletedJobName)[j.GetName()] = j.GetName()
	}).
		Return(nil).AnyTimes()
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
