package executor

import (
	"context"
	"fmt"
	"sort"

	//"fmt"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	version "github.com/kedacore/keda/version"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (e *scaleExecutor) RequestJobScale(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob, isActive bool, scaleTo int64, maxScale int64) {
	runningJobCount := e.getRunningJobCount(scaledJob, maxScale)
	e.logger.Info("Scaling Jobs", "Number of running Jobs ", runningJobCount)

	var effectiveMaxScale int64
	effectiveMaxScale = maxScale - runningJobCount
	if effectiveMaxScale < 0 {
		effectiveMaxScale = 0
	}

	e.logger.Info("Scaling Jobs")

	if isActive {
		e.logger.V(1).Info("At least one scaler is active")
		now := metav1.Now()
		scaledJob.Status.LastActiveTime = &now
		e.updateLastActiveTime(ctx, e.logger, scaledJob)
		e.createJobs(scaledJob, scaleTo, effectiveMaxScale)
	} else {
		e.logger.V(1).Info("No change in activity")
	}

	err := e.cleanUp(scaledJob)
	if err != nil {
		e.logger.Error(err, "Failed to cleanUp jobs")
	}

	return
}

func (e *scaleExecutor) createJobs(scaledJob *kedav1alpha1.ScaledJob, scaleTo int64, maxScale int64) {
	scaledJob.Spec.JobTargetRef.Template.GenerateName = scaledJob.GetName() + "-"
	if scaledJob.Spec.JobTargetRef.Template.Labels == nil {
		scaledJob.Spec.JobTargetRef.Template.Labels = map[string]string{}
	}
	scaledJob.Spec.JobTargetRef.Template.Labels["scaledjob"] = scaledJob.GetName()

	e.logger.Info("Creating jobs", "Effective number of max jobs", maxScale)

	if scaleTo > maxScale {
		scaleTo = maxScale
	}
	e.logger.Info("Creating jobs", "Number of jobs", scaleTo)

	for i := 0; i < int(scaleTo); i++ {

		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: scaledJob.GetName() + "-",
				Namespace:    scaledJob.GetNamespace(),
				Labels: map[string]string{
					"app.kubernetes.io/name":       scaledJob.GetName(),
					"app.kubernetes.io/version":    version.Version,
					"app.kubernetes.io/part-of":    scaledJob.GetName(),
					"app.kubernetes.io/managed-by": "keda-operator",
					"scaledjob":                    scaledJob.GetName(),
				},
			},
			Spec: *scaledJob.Spec.JobTargetRef.DeepCopy(),
		}

		// Job doesn't allow RestartPolicyAlways, it seems like this value is set by the client as a default one,
		// we should set this property to allowed value in that case
		if job.Spec.Template.Spec.RestartPolicy == "" {
			e.logger.V(1).Info("Job RestartPolicy is not set, setting it to 'OnFailure', to avoid setting it to the client's default value 'Always'")
			job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
		}

		// Set ScaledObject instance as the owner and controller
		err := controllerutil.SetControllerReference(scaledJob, job, e.reconcilerScheme)
		if err != nil {
			e.logger.Error(err, "Failed to set ScaledObject as the owner of the new Job")
		}

		err = e.client.Create(context.TODO(), job)
		if err != nil {
			e.logger.Error(err, "Failed to create a new Job")

		}
	}
	e.logger.Info("Created jobs", "Number of jobs", scaleTo)

}

func (e *scaleExecutor) isJobFinished(j *batchv1.Job) bool {
	for _, c := range j.Status.Conditions {
		if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func (e *scaleExecutor) getRunningJobCount(scaledJob *kedav1alpha1.ScaledJob, maxScale int64) int64 {
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
		if !e.isJobFinished(&job) {
			runningJobs++
		}
	}

	return runningJobs
}

// Clean up will delete the jobs that is exceed historyLimit
func (e *scaleExecutor) cleanUp(scaledJob *kedav1alpha1.ScaledJob) error {
	opts := []client.ListOption{
		client.InNamespace(scaledJob.GetNamespace()),
		client.MatchingLabels(map[string]string{"scaledjob": scaledJob.GetName()}),
	}

	jobs := &batchv1.JobList{}
	err := e.client.List(context.TODO(), jobs, opts...)
	if err != nil {
		e.logger.Info("Can not get job list: ", scaledJob.GetName())
		return err
	}

	completedJobs := []batchv1.Job{}
	failedJobs := []batchv1.Job{}
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

	successfulJobsHistoryLimit := int32(100) // TODO Default value should be somewhere constant.
	failedJobsHistoryLimit := int32(100)     // TODO Default value should be somewhere constant.

	if scaledJob.Spec.SuccessfulJobsHistoryLimit != nil {
		successfulJobsHistoryLimit = *scaledJob.Spec.SuccessfulJobsHistoryLimit
	}

	if scaledJob.Spec.FailedJobsHistoryLimit != nil {
		failedJobsHistoryLimit = *scaledJob.Spec.FailedJobsHistoryLimit
	}

	err = e.deleteJobsWithHistoryLimit(completedJobs, successfulJobsHistoryLimit)
	if err != nil {
		return err
	}
	err = e.deleteJobsWithHistoryLimit(failedJobs, failedJobsHistoryLimit)
	if err != nil {
		return err
	}
	return nil
}

func (e *scaleExecutor) deleteJobsWithHistoryLimit(jobs []batchv1.Job, historyLimit int32) error {
	if len(jobs) <= int(historyLimit) {
		return nil
	}

	deleteJobLength := len(jobs) - int(historyLimit)
	for _, j := range (jobs)[0:deleteJobLength] {
		err := e.client.Delete(context.TODO(), j.DeepCopyObject())
		if err != nil {
			return err
		}
		e.logger.Info(fmt.Sprintf("Remove a job (%s) by reaching the historyLimit: %d", j.ObjectMeta.Name, historyLimit))
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
		if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == v1.ConditionTrue {
			return c.Type
		}
	}
	return ""
}
