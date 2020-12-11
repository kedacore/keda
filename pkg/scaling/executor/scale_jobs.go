package executor

import (
	"context"
	"sort"
	"strconv"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	version "github.com/kedacore/keda/v2/version"
)

const (
	defaultSuccessfulJobsHistoryLimit = int32(100)
	defaultFailedJobsHistoryLimit     = int32(100)
)

func (e *scaleExecutor) RequestJobScale(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob, isActive bool, scaleTo int64, maxScale int64) {
	logger := e.logger.WithValues("scaledJob.Name", scaledJob.Name, "scaledJob.Namespace", scaledJob.Namespace)

	runningJobCount := e.getRunningJobCount(scaledJob)
	pendingJobCount := e.getPendingJobCount(scaledJob)
	logger.Info("Scaling Jobs", "Number of running Jobs", runningJobCount)
	logger.Info("Scaling Jobs", "Number of pending Jobs ", pendingJobCount)

	effectiveMaxScale := NewScalingStrategy(logger, scaledJob).GetEffectiveMaxScale(maxScale, runningJobCount, pendingJobCount, scaledJob.MaxReplicaCount())

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
		e.createJobs(logger, scaledJob, scaleTo, effectiveMaxScale)
	} else {
		logger.V(1).Info("No change in activity")
	}

	err := e.cleanUp(scaledJob)
	if err != nil {
		logger.Error(err, "Failed to cleanUp jobs")
	}
}

func (e *scaleExecutor) createJobs(logger logr.Logger, scaledJob *kedav1alpha1.ScaledJob, scaleTo int64, maxScale int64) {
	scaledJob.Spec.JobTargetRef.Template.GenerateName = scaledJob.GetName() + "-"
	if scaledJob.Spec.JobTargetRef.Template.Labels == nil {
		scaledJob.Spec.JobTargetRef.Template.Labels = map[string]string{}
	}
	scaledJob.Spec.JobTargetRef.Template.Labels["scaledjob"] = scaledJob.GetName()

	logger.Info("Creating jobs", "Effective number of max jobs", maxScale)

	if scaleTo > maxScale {
		scaleTo = maxScale
	}
	logger.Info("Creating jobs", "Number of jobs", scaleTo)

	labels := map[string]string{
		"app.kubernetes.io/name":       scaledJob.GetName(),
		"app.kubernetes.io/version":    version.Version,
		"app.kubernetes.io/part-of":    scaledJob.GetName(),
		"app.kubernetes.io/managed-by": "keda-operator",
		"scaledjob":                    scaledJob.GetName(),
	}
	for key, value := range scaledJob.ObjectMeta.Labels {
		labels[key] = value
	}

	for i := 0; i < int(scaleTo); i++ {
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: scaledJob.GetName() + "-",
				Namespace:    scaledJob.GetNamespace(),
				Labels:       labels,
			},
			Spec: *scaledJob.Spec.JobTargetRef.DeepCopy(),
		}

		// Job doesn't allow RestartPolicyAlways, it seems like this value is set by the client as a default one,
		// we should set this property to allowed value in that case
		if job.Spec.Template.Spec.RestartPolicy == "" {
			logger.V(1).Info("Job RestartPolicy is not set, setting it to 'OnFailure', to avoid setting it to the client's default value 'Always'")
			job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
		}

		// Set ScaledObject instance as the owner and controller
		err := controllerutil.SetControllerReference(scaledJob, job, e.reconcilerScheme)
		if err != nil {
			logger.Error(err, "Failed to set ScaledObject as the owner of the new Job")
		}

		err = e.client.Create(context.TODO(), job)
		if err != nil {
			logger.Error(err, "Failed to create a new Job")
		}
	}
	logger.Info("Created jobs", "Number of jobs", scaleTo)
}

func (e *scaleExecutor) isJobFinished(j *batchv1.Job) bool {
	for _, c := range j.Status.Conditions {
		if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (e *scaleExecutor) getRunningJobCount(scaledJob *kedav1alpha1.ScaledJob) int64 {
	var runningJobs int64

	opts := []client.ListOption{
		client.InNamespace(scaledJob.GetNamespace()),
		client.MatchingLabels(map[string]string{"scaledjob": scaledJob.GetName()}),
	}

	jobs := &batchv1.JobList{}
	err := e.client.List(context.TODO(), jobs, opts...)

	if err != nil {
		return 0
	}

	for _, job := range jobs.Items {
		job := job
		if !e.isJobFinished(&job) {
			runningJobs++
		}
	}

	return runningJobs
}

func (e *scaleExecutor) isAnyPodRunningOrCompleted(j *batchv1.Job) bool {
	opts := []client.ListOption{
		client.InNamespace(j.GetNamespace()),
		client.MatchingLabels(map[string]string{"job-name": j.GetName()}),
	}

	pods := &corev1.PodList{}
	err := e.client.List(context.TODO(), pods, opts...)

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

func (e *scaleExecutor) getPendingJobCount(scaledJob *kedav1alpha1.ScaledJob) int64 {
	var pendingJobs int64

	opts := []client.ListOption{
		client.InNamespace(scaledJob.GetNamespace()),
		client.MatchingLabels(map[string]string{"scaledjob": scaledJob.GetName()}),
	}

	jobs := &batchv1.JobList{}
	err := e.client.List(context.TODO(), jobs, opts...)

	if err != nil {
		return 0
	}

	for _, job := range jobs.Items {
		job := job
		if !e.isJobFinished(&job) && !e.isAnyPodRunningOrCompleted(&job) {
			pendingJobs++
		}
	}

	return pendingJobs
}

// Clean up will delete the jobs that is exceed historyLimit
func (e *scaleExecutor) cleanUp(scaledJob *kedav1alpha1.ScaledJob) error {
	logger := e.logger.WithValues("scaledJob.Name", scaledJob.Name, "scaledJob.Namespace", scaledJob.Namespace)

	opts := []client.ListOption{
		client.InNamespace(scaledJob.GetNamespace()),
		client.MatchingLabels(map[string]string{"scaledjob": scaledJob.GetName()}),
	}

	jobs := &batchv1.JobList{}
	err := e.client.List(context.TODO(), jobs, opts...)
	if err != nil {
		logger.Error(err, "Can not get list of Jobs")
		return err
	}

	completedJobs := []batchv1.Job{}
	failedJobs := []batchv1.Job{}
	for _, job := range jobs.Items {
		job := job
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

	err = e.deleteJobsWithHistoryLimit(logger, completedJobs, successfulJobsHistoryLimit)
	if err != nil {
		return err
	}
	err = e.deleteJobsWithHistoryLimit(logger, failedJobs, failedJobsHistoryLimit)
	if err != nil {
		return err
	}
	return nil
}

func (e *scaleExecutor) deleteJobsWithHistoryLimit(logger logr.Logger, jobs []batchv1.Job, historyLimit int32) error {
	if len(jobs) <= int(historyLimit) {
		return nil
	}

	deleteJobLength := len(jobs) - int(historyLimit)
	for _, j := range (jobs)[0:deleteJobLength] {
		deletePolicy := metav1.DeletePropagationBackground
		deleteOptions := &client.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}
		err := e.client.Delete(context.TODO(), j.DeepCopyObject(), deleteOptions)
		if err != nil {
			return err
		}
		logger.Info("Remove a job by reaching the historyLimit", "job.Name", j.ObjectMeta.Name, "historyLimit", historyLimit)
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
	default:
		logger.V(1).Info("Selecting Scale Strategy", "specified", scaledJob.Spec.ScalingStrategy.Strategy, "selected", "default")
		return defaultScalingStrategy{}
	}
}

// ScalingStrategy is an interface for switching scaling algorithm
type ScalingStrategy interface {
	GetEffectiveMaxScale(maxScale, runningJobCount, pendingJobCount, maxReplicaCount int64) int64
}

type defaultScalingStrategy struct {
}

func (s defaultScalingStrategy) GetEffectiveMaxScale(maxScale, runningJobCount, pendingJobCount, maxReplicaCount int64) int64 {
	return maxScale - runningJobCount
}

type customScalingStrategy struct {
	CustomScalingQueueLengthDeduction *int32
	CustomScalingRunningJobPercentage *float64
}

func (s customScalingStrategy) GetEffectiveMaxScale(maxScale, runningJobCount, pendingJobCount, maxReplicaCount int64) int64 {
	return min(maxScale-int64(*s.CustomScalingQueueLengthDeduction)-int64(float64(runningJobCount)*(*s.CustomScalingRunningJobPercentage)), maxReplicaCount)
}

type accurateScalingStrategy struct {
}

func (s accurateScalingStrategy) GetEffectiveMaxScale(maxScale, runningJobCount, pendingJobCount, maxReplicaCount int64) int64 {
	if (maxScale + runningJobCount) > maxReplicaCount {
		return maxReplicaCount - runningJobCount
	}
	return maxScale - pendingJobCount
}

func min(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}
