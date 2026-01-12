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
	"sort"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	version "github.com/kedacore/keda/v2/version"
)

const (
	defaultSuccessfulJobsHistoryLimit = int32(100)
	defaultFailedJobsHistoryLimit     = int32(100)
)

func (e *scaleExecutor) RequestJobScale(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob, isActive, isError bool, scaleTo int64, maxScale int64) {
	logger := e.logger.WithValues("scaledJob.Name", scaledJob.Name, "scaledJob.Namespace", scaledJob.Namespace)

	runningJobCount := e.getRunningJobCount(ctx, scaledJob)
	pendingJobCount := e.getPendingJobCount(ctx, scaledJob)
	logger.Info("Scaling Jobs", "Number of running Jobs", runningJobCount)
	logger.Info("Scaling Jobs", "Number of pending Jobs", pendingJobCount)

	effectiveMaxScale, scaleTo := e.getScalingDecision(scaledJob, runningJobCount, scaleTo, maxScale, pendingJobCount, logger)

	if effectiveMaxScale < 0 {
		effectiveMaxScale = 0
	}

	if isActive {
		logger.V(1).Info("At least one scaler is active")
		now := metav1.Now()
		scaledJob.Status.LastActiveTime = &now
		err := e.updateLastActiveTime(ctx, logger, scaledJob)
		if err != nil {
			logger.Error(err, "Failed to update last active time")
		}
		e.createJobs(ctx, logger, scaledJob, scaleTo, effectiveMaxScale)
	} else {
		logger.V(1).Info("No change in activity")
	}

	readyCondition := scaledJob.Status.Conditions.GetReadyCondition()
	if isError {
		if isActive {
			// some triggers responded with error, but at least one is active
			// Set ScaledJob.Status.ReadyCondition to Unknown
			msg := "Some triggers defined in ScaledJob are not working correctly"
			logger.V(1).Info(msg)
			if !readyCondition.IsUnknown() {
				if err := e.setReadyCondition(ctx, logger, scaledJob, metav1.ConditionUnknown, "PartialTriggerError", msg); err != nil {
					logger.Error(err, "error setting ready condition")
				}
			}
		} else {
			// all triggers responded with error (no active triggers)
			// Set ScaledJob.Status.ReadyCondition to False
			msg := "Triggers defined in ScaledJob are not working correctly"
			logger.V(1).Info(msg)
			if !readyCondition.IsFalse() {
				if err := e.setReadyCondition(ctx, logger, scaledJob, metav1.ConditionFalse, "TriggerError", msg); err != nil {
					logger.Error(err, "error setting ready condition")
				}
			}
		}
	} else if !readyCondition.IsTrue() {
		// if the ScaledObject's triggers aren't in the error state,
		// but ScaledJob.Status.ReadyCondition is set not set to 'true' -> set it back to 'true'
		msg := "ScaledJob is defined correctly and is ready for scaling"
		logger.V(1).Info(msg)
		if err := e.setReadyCondition(ctx, logger, scaledJob, metav1.ConditionTrue,
			"ScaledJobReady", msg); err != nil {
			logger.Error(err, "error setting ready condition")
		}
	}

	condition := scaledJob.Status.Conditions.GetActiveCondition()
	if condition.IsUnknown() || condition.IsTrue() != isActive {
		if isActive {
			if !condition.IsTrue() {
				e.recorder.Event(scaledJob, corev1.EventTypeNormal, eventreason.ScaledJobActive, "Scaling is performed because triggers are active")
			}
			if err := e.setActiveCondition(ctx, logger, scaledJob, metav1.ConditionTrue, "ScalerActive", "Scaling is performed because triggers are active"); err != nil {
				logger.Error(err, "Error setting active condition when triggers are active")
				return
			}
		} else {
			if !condition.IsFalse() {
				e.recorder.Event(scaledJob, corev1.EventTypeNormal, eventreason.ScaledJobInactive, "Scaling is not performed because triggers are not active")
			}
			if err := e.setActiveCondition(ctx, logger, scaledJob, metav1.ConditionFalse, "ScalerNotActive", "Scaling is not performed because triggers are not active"); err != nil {
				logger.Error(err, "Error setting active condition when triggers are not active")
				return
			}
		}
	}

	err := e.cleanUp(ctx, scaledJob)
	if err != nil {
		logger.Error(err, "Failed to cleanUp jobs")
	}
}

func (e *scaleExecutor) getScalingDecision(scaledJob *kedav1alpha1.ScaledJob, runningJobCount int64, scaleTo int64, maxScale int64, pendingJobCount int64, logger logr.Logger) (int64, int64) {
	var effectiveMaxScale int64
	minReplicaCount := scaledJob.MinReplicaCount()

	if runningJobCount < minReplicaCount {
		scaleToMinReplica := minReplicaCount - runningJobCount
		scaleTo = scaleToMinReplica
		effectiveMaxScale = scaleToMinReplica
	} else {
		effectiveMaxScale, scaleTo = NewScalingStrategy(logger, scaledJob).GetEffectiveMaxScale(maxScale, runningJobCount-minReplicaCount, pendingJobCount, scaledJob.MaxReplicaCount(), scaleTo)
	}
	return effectiveMaxScale, scaleTo
}

func (e *scaleExecutor) createJobs(ctx context.Context, logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob, scaleTo int64, maxScale int64) {
	if maxScale <= 0 {
		logger.Info("No need to create jobs - all requested jobs already exist", "jobs", maxScale)
		return
	}
	logger.Info("Creating jobs", "Effective number of max jobs", maxScale)
	if scaleTo > maxScale {
		scaleTo = maxScale
	}
	logger.Info("Creating jobs", "Number of jobs", scaleTo)

	jobs := e.generateJobs(logger, scaledJob, scaleTo)
	for _, job := range jobs {
		err := e.client.Create(ctx, job)
		if err != nil {
			logger.Error(err, "Failed to create a new Job")
			e.recorder.Eventf(scaledJob, corev1.EventTypeWarning, eventreason.KEDAJobCreateFailed, "Failed to create job %s: %v", job.GenerateName, err)
		}
	}

	logger.Info("Created jobs", "Number of jobs", scaleTo)
	e.recorder.Eventf(scaledJob, corev1.EventTypeNormal, eventreason.KEDAJobsCreated, "Created %d jobs", scaleTo)
}

func (e *scaleExecutor) generateJobs(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob, scaleTo int64) []*batchv1.Job {
	scaledJob.Spec.JobTargetRef.Template.GenerateName = scaledJob.GetName() + "-"
	if scaledJob.Spec.JobTargetRef.Template.Labels == nil {
		scaledJob.Spec.JobTargetRef.Template.Labels = map[string]string{}
	}
	scaledJob.Spec.JobTargetRef.Template.Labels["scaledjob.keda.sh/name"] = scaledJob.GetName()

	labels := map[string]string{
		"app.kubernetes.io/name":       scaledJob.GetName(),
		"app.kubernetes.io/version":    version.Version,
		"app.kubernetes.io/part-of":    scaledJob.GetName(),
		"app.kubernetes.io/managed-by": "keda-operator",
		"scaledjob.keda.sh/name":       scaledJob.GetName(),
	}

	excludedLabels := map[string]struct{}{}

	if labels, ok := scaledJob.Annotations[kedav1alpha1.ScaledJobExcludedLabelsAnnotation]; ok {
		for _, excludedLabel := range strings.Split(labels, ",") {
			excludedLabels[excludedLabel] = struct{}{}
		}
	}

	for key, value := range scaledJob.Labels {
		if _, ok := excludedLabels[key]; ok {
			continue
		}

		labels[key] = value
	}

	annotations := map[string]string{
		"scaledjob.keda.sh/generation": strconv.FormatInt(scaledJob.Generation, 10),
	}
	for key, value := range scaledJob.Annotations {
		annotations[key] = value
	}

	jobs := make([]*batchv1.Job, int(scaleTo))
	for i := 0; i < int(scaleTo); i++ {
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: scaledJob.GetName() + "-",
				Namespace:    scaledJob.GetNamespace(),
				Labels:       labels,
				Annotations:  annotations,
			},
			Spec: *scaledJob.Spec.JobTargetRef.DeepCopy(),
		}

		// Job doesn't allow RestartPolicyAlways, it seems like this value is set by the client as a default one,
		// we should set this property to allowed value in that case
		if job.Spec.Template.Spec.RestartPolicy == "" {
			logger.V(1).Info("Job RestartPolicy is not set, setting it to 'OnFailure', to avoid setting it to the client's default value 'Always'")
			job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
		}

		// Set ScaledJob instance as the owner and controller
		err := controllerutil.SetControllerReference(scaledJob, job, e.reconcilerScheme)
		if err != nil {
			logger.Error(err, "Failed to set ScaledJob as the owner of the new Job")
		}

		jobs[i] = job
	}
	return jobs
}

func (e *scaleExecutor) isJobFinished(j *batchv1.Job) bool {
	for _, c := range j.Status.Conditions {
		if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (e *scaleExecutor) getRunningJobCount(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob) int64 {
	var runningJobs int64

	opts := []client.ListOption{
		client.InNamespace(scaledJob.GetNamespace()),
		client.MatchingLabels(map[string]string{"scaledjob.keda.sh/name": scaledJob.GetName()}),
	}

	jobs := &batchv1.JobList{}
	err := e.client.List(ctx, jobs, opts...)

	if err != nil {
		return 0
	}

	for _, job := range jobs.Items {
		if !e.isJobFinished(&job) {
			runningJobs++
		}
	}

	return runningJobs
}

func (e *scaleExecutor) isAnyPodRunningOrCompleted(ctx context.Context, j *batchv1.Job) bool {
	opts := []client.ListOption{
		client.InNamespace(j.GetNamespace()),
		client.MatchingLabels(map[string]string{"job-name": j.GetName()}),
	}

	pods := &corev1.PodList{}
	err := e.client.List(ctx, pods, opts...)

	if err != nil {
		return false
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodRunning {
			return true
		}
	}

	return false
}

func (e *scaleExecutor) areAllPendingPodConditionsFulfilled(ctx context.Context, j *batchv1.Job, pendingPodConditions []string) bool {
	opts := []client.ListOption{
		client.InNamespace(j.GetNamespace()),
		client.MatchingLabels(map[string]string{"job-name": j.GetName()}),
	}

	pods := &corev1.PodList{}
	err := e.client.List(ctx, pods, opts...)
	if err != nil {
		return false
	}

	// Convert pendingPodConditions to a map for faster lookup
	requiredConditions := make(map[string]struct{})
	for _, condition := range pendingPodConditions {
		requiredConditions[condition] = struct{}{}
	}

	// Check if any pod has all required conditions fulfilled
	for _, pod := range pods.Items {
		fulfilledConditions := make(map[string]struct{})

		for _, podCondition := range pod.Status.Conditions {
			if _, isRequired := requiredConditions[string(podCondition.Type)]; isRequired && podCondition.Status == corev1.ConditionTrue {
				fulfilledConditions[string(podCondition.Type)] = struct{}{}
			}
		}

		// If this pod has all required conditions fulfilled, the job is no longer pending
		if len(fulfilledConditions) == len(pendingPodConditions) {
			return true
		}
	}

	return false
}

func (e *scaleExecutor) getPendingJobCount(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob) int64 {
	var pendingJobs int64

	opts := []client.ListOption{
		client.InNamespace(scaledJob.GetNamespace()),
		client.MatchingLabels(map[string]string{"scaledjob.keda.sh/name": scaledJob.GetName()}),
	}

	jobs := &batchv1.JobList{}
	err := e.client.List(ctx, jobs, opts...)

	if err != nil {
		return 0
	}

	for _, job := range jobs.Items {
		if !e.isJobFinished(&job) {
			if len(scaledJob.Spec.ScalingStrategy.PendingPodConditions) > 0 {
				if !e.areAllPendingPodConditionsFulfilled(ctx, &job, scaledJob.Spec.ScalingStrategy.PendingPodConditions) {
					pendingJobs++
				}
			} else {
				if !e.isAnyPodRunningOrCompleted(ctx, &job) {
					pendingJobs++
				}
			}
		}
	}

	return pendingJobs
}

// Clean up will delete the jobs that is exceed historyLimit
func (e *scaleExecutor) cleanUp(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob) error {
	logger := e.logger.WithValues("scaledJob.Name", scaledJob.Name, "scaledJob.Namespace", scaledJob.Namespace)

	opts := []client.ListOption{
		client.InNamespace(scaledJob.GetNamespace()),
		client.MatchingLabels(map[string]string{"scaledjob.keda.sh/name": scaledJob.GetName()}),
	}

	jobs := &batchv1.JobList{}
	err := e.client.List(ctx, jobs, opts...)
	if err != nil {
		logger.Error(err, "Can not get list of Jobs")
		return err
	}

	var completedJobs []batchv1.Job
	var failedJobs []batchv1.Job
	for _, job := range jobs.Items {
		finishedJobConditionType := e.getFinishedJobConditionType(&job)
		switch finishedJobConditionType {
		case batchv1.JobComplete:
			completedJobs = append(completedJobs, job)
		case batchv1.JobFailed:
			failedJobs = append(failedJobs, job)
		}
	}

	sort.Sort(byCompletedTime(completedJobs))
	sort.Sort(byCompletedTime(failedJobs))

	successfulJobsHistoryLimit := defaultSuccessfulJobsHistoryLimit
	failedJobsHistoryLimit := defaultFailedJobsHistoryLimit

	if scaledJob.Spec.SuccessfulJobsHistoryLimit != nil {
		successfulJobsHistoryLimit = *scaledJob.Spec.SuccessfulJobsHistoryLimit
	}

	if scaledJob.Spec.FailedJobsHistoryLimit != nil {
		failedJobsHistoryLimit = *scaledJob.Spec.FailedJobsHistoryLimit
	}

	err = e.deleteJobsWithHistoryLimit(ctx, logger, completedJobs, successfulJobsHistoryLimit)
	if err != nil {
		return err
	}
	return e.deleteJobsWithHistoryLimit(ctx, logger, failedJobs, failedJobsHistoryLimit)
}

func (e *scaleExecutor) deleteJobsWithHistoryLimit(ctx context.Context, logger logr.Logger, jobs []batchv1.Job, historyLimit int32) error {
	if len(jobs) <= int(historyLimit) {
		return nil
	}

	deleteJobLength := len(jobs) - int(historyLimit)
	for _, j := range (jobs)[0:deleteJobLength] {
		deletePolicy := metav1.DeletePropagationBackground
		deleteOptions := &client.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}
		err := e.client.Delete(ctx, j.DeepCopy(), deleteOptions)
		if err != nil {
			return err
		}
		logger.Info("Remove a job by reaching the historyLimit", "job.Name", j.Name, "historyLimit", historyLimit)
	}
	return nil
}

type byCompletedTime []batchv1.Job

func (c byCompletedTime) Len() int { return len(c) }
func (c byCompletedTime) Less(i, j int) bool {
	return c[i].Status.CompletionTime.Before(c[j].Status.CompletionTime)
}
func (c byCompletedTime) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

func (e *scaleExecutor) getFinishedJobConditionType(j *batchv1.Job) batchv1.JobConditionType {
	for _, c := range j.Status.Conditions {
		if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == corev1.ConditionTrue {
			return c.Type
		}
	}
	return ""
}

// NewScalingStrategy returns ScalingStrategy instance
func NewScalingStrategy(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob) ScalingStrategy {
	switch scaledJob.Spec.ScalingStrategy.Strategy {
	case "custom":
		logger.V(1).Info("Selecting Scale Strategy", "specified", scaledJob.Spec.ScalingStrategy.Strategy, "selected:", "custom", "customScalingQueueLength", scaledJob.Spec.ScalingStrategy.CustomScalingQueueLengthDeduction, "customScallingRunningJobPercentage", scaledJob.Spec.ScalingStrategy.CustomScalingRunningJobPercentage)
		var err error
		if percentage, err := strconv.ParseFloat(scaledJob.Spec.ScalingStrategy.CustomScalingRunningJobPercentage, 64); err == nil {
			return customScalingStrategy{
				CustomScalingQueueLengthDeduction: scaledJob.Spec.ScalingStrategy.CustomScalingQueueLengthDeduction,
				CustomScalingRunningJobPercentage: &percentage,
			}
		}

		logger.V(1).Info("Fail to convert CustomScalingRunningJobPercentage into float", "error", err, "CustomScalingRunningJobPercentage", scaledJob.Spec.ScalingStrategy.CustomScalingRunningJobPercentage)
		logger.V(1).Info("Selecting Scale has been changed", "selected", "default")
		return defaultScalingStrategy{}

	case "accurate":
		logger.V(1).Info("Selecting Scale Strategy", "specified", scaledJob.Spec.ScalingStrategy.Strategy, "selected", "accurate")
		return accurateScalingStrategy{}
	case "eager":
		logger.V(1).Info("Selecting Scale Strategy", "specified", scaledJob.Spec.ScalingStrategy.Strategy, "selected", "eager")
		return eagerScalingStrategy{}
	default:
		logger.V(1).Info("Selecting Scale Strategy", "specified", scaledJob.Spec.ScalingStrategy.Strategy, "selected", "default")
		return defaultScalingStrategy{}
	}
}

// ScalingStrategy is an interface for switching scaling algorithm
type ScalingStrategy interface {
	GetEffectiveMaxScale(maxScale, runningJobCount, pendingJobCount, maxReplicaCount, scaleTo int64) (int64, int64)
}

type defaultScalingStrategy struct {
}

func (s defaultScalingStrategy) GetEffectiveMaxScale(maxScale, runningJobCount, _, _, scaleTo int64) (int64, int64) {
	return maxScale - runningJobCount, scaleTo
}

type customScalingStrategy struct {
	CustomScalingQueueLengthDeduction *int32
	CustomScalingRunningJobPercentage *float64
}

func (s customScalingStrategy) GetEffectiveMaxScale(maxScale, runningJobCount, _, maxReplicaCount, scaleTo int64) (int64, int64) {
	return min(maxScale-int64(*s.CustomScalingQueueLengthDeduction)-int64(float64(runningJobCount)*(*s.CustomScalingRunningJobPercentage)), maxReplicaCount), scaleTo
}

type accurateScalingStrategy struct {
}

func (s accurateScalingStrategy) GetEffectiveMaxScale(maxScale, runningJobCount, pendingJobCount, maxReplicaCount, scaleTo int64) (int64, int64) {
	if (maxScale + runningJobCount) > maxReplicaCount {
		return maxReplicaCount - runningJobCount, scaleTo
	}
	return maxScale - pendingJobCount, scaleTo
}

type eagerScalingStrategy struct {
}

func (s eagerScalingStrategy) GetEffectiveMaxScale(maxScale, runningJobCount, pendingJobCount, maxReplicaCount, _ int64) (int64, int64) {
	return min(maxReplicaCount-runningJobCount-pendingJobCount, maxScale), maxReplicaCount
}
