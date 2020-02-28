package handler

import (
	"context"
	"fmt"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	version "github.com/kedacore/keda/version"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (h *ScaleHandler) scaleJobs(scaledObject *kedav1alpha1.ScaledObject, isActive bool, scaleTo int64, maxScale int64) {
	runningJobCount := h.getRunningJobCount(scaledObject, maxScale)
	pendingJobCount := h.getPendingJobCount(scaledObject, maxScale)
	h.logger.Info("Scaling Jobs", "Number of running Jobs ", runningJobCount)
	h.logger.Info("Scaling Jobs", "Number of pending Jobs ", pendingJobCount)

	var effectiveMaxScale int64
	effectiveMaxScale = maxScale - runningJobCount
	if effectiveMaxScale < 0 {
		effectiveMaxScale = 0
	}

	var effectiveScaleTo int64
	effectiveScaleTo = scaleTo - pendingJobCount
	if effectiveScaleTo < 0 {
		effectiveScaleTo = 0
	}

	h.logger.Info("Scaling Jobs")

	if isActive {
		h.logger.V(1).Info("At least one scaler is active")
		now := metav1.Now()
		scaledObject.Status.LastActiveTime = &now
		h.updateScaledObjectStatus(scaledObject)
		h.createJobs(scaledObject, effectiveScaleTo, effectiveMaxScale)

	} else {
		h.logger.V(1).Info("No change in activity")
	}
	return
}

func (h *ScaleHandler) createJobs(scaledObject *kedav1alpha1.ScaledObject, scaleTo int64, maxScale int64) {
	scaledObject.Spec.JobTargetRef.Template.GenerateName = scaledObject.GetName() + "-"
	if scaledObject.Spec.JobTargetRef.Template.Labels == nil {
		scaledObject.Spec.JobTargetRef.Template.Labels = map[string]string{}
	}
	scaledObject.Spec.JobTargetRef.Template.Labels["scaledobject"] = scaledObject.GetName()

	h.logger.Info("Creating jobs", "Effective number of max jobs", maxScale)

	if scaleTo > maxScale {
		scaleTo = maxScale
	}
	h.logger.Info("Creating jobs", "Number of jobs", scaleTo)

	for i := 0; i < int(scaleTo); i++ {

		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: scaledObject.GetName() + "-",
				Namespace:    scaledObject.GetNamespace(),
				Labels: map[string]string{
					"app.kubernetes.io/name":       scaledObject.GetName(),
					"app.kubernetes.io/version":    version.Version,
					"app.kubernetes.io/part-of":    scaledObject.GetName(),
					"app.kubernetes.io/managed-by": "keda-operator",
					"scaledobject":                 scaledObject.GetName(),
				},
			},
			Spec: *scaledObject.Spec.JobTargetRef.DeepCopy(),
		}

		// Job doesn't allow RestartPolicyAlways, it seems like this value is set by the client as a default one,
		// we should set this property to allowed value in that case
		if job.Spec.Template.Spec.RestartPolicy == "" {
			h.logger.V(1).Info("Job RestartPolicy is not set, setting it to 'OnFailure', to avoid setting it to the client's default value 'Always'")
			job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
		}

		// Set ScaledObject instance as the owner and controller
		err := controllerutil.SetControllerReference(scaledObject, job, h.reconcilerScheme)
		if err != nil {
			h.logger.Error(err, "Failed to set ScaledObject as the owner of the new Job")
		}

		err = h.client.Create(context.TODO(), job)
		if err != nil {
			h.logger.Error(err, "Failed to create a new Job")

		}
	}
	h.logger.Info("Created jobs", "Number of jobs", scaleTo)

}

func (h *ScaleHandler) resolveJobEnv(scaledObject *kedav1alpha1.ScaledObject) (map[string]string, error) {

	if len(scaledObject.Spec.JobTargetRef.Template.Spec.Containers) < 1 {
		return nil, fmt.Errorf("Scaled Object (%s) doesn't have containers", scaledObject.GetName())
	}

	container := scaledObject.Spec.JobTargetRef.Template.Spec.Containers[0]

	return h.resolveEnv(&container, scaledObject.GetNamespace())
}

func (h *ScaleHandler) parseJobAuthRef(triggerAuthRef *kedav1alpha1.ScaledObjectAuthRef, scaledObject *kedav1alpha1.ScaledObject) (map[string]string, string) {
	return h.parseAuthRef(triggerAuthRef, scaledObject, func(name, containerName string) string {
		env, err := h.resolveJobEnv(scaledObject)
		if err != nil {
			return ""
		}
		return env[name]
	})
}

func (h *ScaleHandler) isJobFinished(j *batchv1.Job) bool {
	for _, c := range j.Status.Conditions {
		if (c.Type == batchv1.JobComplete || c.Type == batchv1.JobFailed) && c.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}

func (h *ScaleHandler) getRunningJobCount(scaledObject *kedav1alpha1.ScaledObject, maxScale int64) int64 {
	var runningJobs int64

	opts := []client.ListOption{
		client.InNamespace(scaledObject.GetNamespace()),
		client.MatchingLabels(map[string]string{"scaledobject": scaledObject.GetName()}),
	}

	jobs := &batchv1.JobList{}
	err := h.client.List(context.TODO(), jobs, opts...)

	if err != nil {
		return 0
	}

	for _, job := range jobs.Items {
		if !h.isJobFinished(&job) {
			runningJobs++
		}
	}

	return runningJobs
}

func (h *ScaleHandler) isAnyPodRunningOrCompleted(job batchv1.Job) bool {
	opts := []client.ListOption{
		client.InNamespace(job.GetNamespace()),
		client.MatchingLabels(map[string]string{"job-name": job.GetName()}),
	}

	pods := &corev1.PodList{}
	err := h.client.List(context.TODO(), pods, opts...)

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

func (h *ScaleHandler) getPendingJobCount(scaledObject *kedav1alpha1.ScaledObject, maxScale int64) int64 {
	var pendingJobs int64

	opts := []client.ListOption{
		client.InNamespace(scaledObject.GetNamespace()),
		client.MatchingLabels(map[string]string{"scaledobject": scaledObject.GetName()}),
	}

	jobs := &batchv1.JobList{}
	err := h.client.List(context.TODO(), jobs, opts...)

	if err != nil {
		return 0
	}

	for _, job := range jobs.Items {
		if !h.isAnyPodRunningOrCompleted(job) {
			pendingJobs++
		}
	}

	return pendingJobs
}
