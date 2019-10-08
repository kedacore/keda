package handler

import (
	"fmt"

	keda_v1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *ScaleHandler) scaleJobs(scaledObject *keda_v1alpha1.ScaledObject, isActive bool, scaleTo int64, maxScale int64) {
	// TODO: get current job count
	log.Infoln("Scaling Jobs")

	if isActive {
		log.Infoln("Scaler is active")
		now := meta_v1.Now()
		scaledObject.Status.LastActiveTime = &now
		h.updateScaledObject(scaledObject)
		h.createJobs(scaledObject, scaleTo, maxScale)
	} else {
		log.Debugf("scaledObject (%s) no change", scaledObject.GetName())
	}
	return
}

func (h *ScaleHandler) createJobs(scaledObject *keda_v1alpha1.ScaledObject, scaleTo int64, maxScale int64) {
	scaledObject.Spec.JobTargetRef.Template.GenerateName = scaledObject.GetName() + "-"
	if scaledObject.Spec.JobTargetRef.Template.Labels == nil {
		scaledObject.Spec.JobTargetRef.Template.Labels = map[string]string{}
	}
	scaledObject.Spec.JobTargetRef.Template.Labels["scaledobject"] = scaledObject.GetName()

	if scaleTo > maxScale {
		scaleTo = maxScale
	}
	log.Infof("Creating %d jobs", scaleTo)

	for i := 0; i < int(scaleTo); i++ {

		job := &batchv1.Job{
			ObjectMeta: meta_v1.ObjectMeta{
				GenerateName: scaledObject.GetName() + "-",
				Namespace:    scaledObject.GetNamespace(),
				Labels: map[string]string{
					"scaledobject": scaledObject.GetName(),
				},
			},
			Spec: scaledObject.Spec.JobTargetRef,
		}
		_, err := h.kubeClient.BatchV1().Jobs(scaledObject.GetNamespace()).Create(job)
		if err != nil {
			log.Fatalln(err)
		}
	}
	log.Infof("Created %d jobs", scaleTo)

}

func (h *ScaleHandler) resolveJobEnv(scaledObject *keda_v1alpha1.ScaledObject) (map[string]string, error) {

	if len(scaledObject.Spec.JobTargetRef.Template.Spec.Containers) < 1 {
		return nil, fmt.Errorf("Scaled Object (%s) doesn't have containers", scaledObject.GetName())
	}

	container := scaledObject.Spec.JobTargetRef.Template.Spec.Containers[0]

	return h.resolveEnv(&container, scaledObject.GetNamespace())
}
