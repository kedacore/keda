package handler

import (
	"context"
	"fmt"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (h *ScaleHandler) scaleJobs(scaledObject *kedav1alpha1.ScaledObject, isActive bool, scaleTo int64, maxScale int64) {
	// TODO: get current job count
	h.logger.V(1).Info("Scaling Jobs")

	if isActive {
		h.logger.V(1).Info("At least one scaler is active")
		now := metav1.Now()
		scaledObject.Status.LastActiveTime = &now
		h.updateScaledObjectStatus(scaledObject)
		h.createJobs(scaledObject, scaleTo, maxScale)
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
					"scaledobject": scaledObject.GetName(),
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

func (h *ScaleHandler) parseJobAuthRef(triggerAuthRef kedav1alpha1.ScaledObjectAuthRef, scaledObject *kedav1alpha1.ScaledObject) (map[string]string, string) {
	return h.parseAuthRef(triggerAuthRef, scaledObject, func(name, containerName string) string {
		env, err := h.resolveJobEnv(scaledObject)
		if err != nil {
			return ""
		}
		return env[name]
	})
}
