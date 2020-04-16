package executor

import (
	"context"
	//"fmt"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	"github.com/kedacore/keda/pkg/scalers"
	version "github.com/kedacore/keda/version"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (e *scaleExecutor) RequestJobScale(ctx context.Context, scalers []scalers.Scaler, scaledJob *kedav1alpha1.ScaledJob) {
	var queueLength int64
	var maxValue int64
	isActive := false

	for _, scaler := range scalers {
		defer scaler.Close()
		scalerLogger := e.logger.WithValues("Scaler", scaler)

		isTriggerActive, err := scaler.IsActive(ctx)
		scalerLogger.Info("Active trigger", "isTriggerActive", isTriggerActive)
		metricSpecs := scaler.GetMetricSpecForScaling()

		var metricValue int64
		for _, metric := range metricSpecs {
			metricValue, _ = metric.External.Target.AverageValue.AsInt64()
			maxValue += metricValue
		}
		scalerLogger.Info("Scaler max value", "MaxValue", maxValue)

		metrics, _ := scaler.GetMetrics(ctx, "queueLength", nil)

		for _, m := range metrics {
			if m.MetricName == "queueLength" {
				metricValue, _ = m.Value.AsInt64()
				queueLength += metricValue
			}
		}
		scalerLogger.Info("QueueLength Metric value", "queueLength", queueLength)

		if err != nil {
			scalerLogger.V(1).Info("Error getting scale decision, but continue", "Error", err)
			continue
		} else if isTriggerActive {
			isActive = true
			scalerLogger.Info("Scaler is active")
		}
	}

	e.scaleJobs(ctx, scaledJob, isActive, queueLength, maxValue)
}

func (e *scaleExecutor) scaleJobs(ctx context.Context, scaledJob *kedav1alpha1.ScaledJob, isActive bool, scaleTo int64, maxScale int64) {
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
		e.updateLastActiveTime(ctx, scaledJob)
		e.createJobs(scaledJob, scaleTo, effectiveMaxScale)

	} else {
		e.logger.V(1).Info("No change in activity")
	}
	return
}

func (e *scaleExecutor) createJobs(scaledJob *kedav1alpha1.ScaledJob, scaleTo int64, maxScale int64) {
	// scaledObject.Spec.JobTargetRef.Template.GenerateName = scaledObject.GetName() + "-"
	// if scaledObject.Spec.JobTargetRef.Template.Labels == nil {
	// 	scaledObject.Spec.JobTargetRef.Template.Labels = map[string]string{}
	// }
	// scaledObject.Spec.JobTargetRef.Template.Labels["scaledobject"] = scaledObject.GetName()

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
					"scaledobject":                 scaledJob.GetName(),
				},
			},
			//Spec: *scaledObject.Spec.JobTargetRef.DeepCopy(),
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
