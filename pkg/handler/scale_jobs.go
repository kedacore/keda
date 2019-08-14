package handler

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	keda_v1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *ScaleHandler) scaleJobs(scaledObject *keda_v1alpha1.ScaledObject, isActive bool, scaleTo int64, maxScale int64) {
	// TODO: get current job count
	log.Println("Scaling Jobs")

	if isActive {
		log.Println("Scaler is active")
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
	var containers []apiv1.Container
	for _, c := range scaledObject.Spec.ConsumerSpec.Containers {
		containers = append(containers, apiv1.Container{
			Name:  c.Name,
			Image: c.Image,
			Env:   c.Env,
		})
	}

	if scaleTo > maxScale {
		scaleTo = maxScale
	}
	log.Printf("Creating %d jobs", scaleTo)

	for i := 0; i < int(scaleTo); i++ {

		job := &batchv1.Job{
			ObjectMeta: meta_v1.ObjectMeta{
				GenerateName: scaledObject.GetName() + "-",
				Namespace:    scaledObject.GetNamespace(),
				Labels: map[string]string{
					"scaledobject": scaledObject.GetName(),
				},
			},
			Spec: batchv1.JobSpec{
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: meta_v1.ObjectMeta{
						GenerateName: scaledObject.GetName() + "-",
						Labels: map[string]string{
							"scaledobject": scaledObject.GetName(),
						},
					},
					Spec: apiv1.PodSpec{
						RestartPolicy: apiv1.RestartPolicyOnFailure,

						Containers: containers,
					},
				},
			},
		}
		_, err := h.kubeClient.BatchV1().Jobs(scaledObject.GetNamespace()).Create(job)
		if err != nil {
			log.Fatalln(err)
		}
	}
	log.Printf("Created %d jobs", scaleTo)

}

func (h *ScaleHandler) resolveJobEnv(scaledObject *keda_v1alpha1.ScaledObject) (map[string]string, error) {
	if len(scaledObject.Spec.ConsumerSpec.Containers) < 1 {
		return nil, fmt.Errorf("Scaled Object (%s) doesn't have containers", scaledObject.GetName())
	}

	container := scaledObject.Spec.ConsumerSpec.Containers[0]

	return h.resolveEnv(&container, scaledObject.GetNamespace())
}
