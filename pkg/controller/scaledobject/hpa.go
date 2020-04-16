package scaledobject

import (
	"context"
	"fmt"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	kedautil "github.com/kedacore/keda/pkg/util"

	"github.com/go-logr/logr"
	version "github.com/kedacore/keda/version"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultHPAMinReplicas int32 = 1
	defaultHPAMaxReplicas int32 = 100
)

// createAndDeployNewHPA creates and deploy HPA in the cluster for specifed ScaledObject
func (r *ReconcileScaledObject) createAndDeployNewHPA(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, gvkr *kedautil.GroupVersionKindResource) error {
	hpaName := getHPAName(scaledObject)
	logger.Info("Creating a new HPA", "HPA.Namespace", scaledObject.Namespace, "HPA.Name", hpaName)
	hpa, err := r.newHPAForScaledObject(logger, scaledObject, gvkr)
	if err != nil {
		logger.Error(err, "Failed to create new HPA resource", "HPA.Namespace", scaledObject.Namespace, "HPA.Name", hpaName)
		return err
	}

	// Set ScaledObject instance as the owner and controller
	if err := controllerutil.SetControllerReference(scaledObject, hpa, r.scheme); err != nil {
		return err
	}

	err = r.client.Create(context.TODO(), hpa)
	if err != nil {
		logger.Error(err, "Failed to create new HPA in cluster", "HPA.Namespace", scaledObject.Namespace, "HPA.Name", hpaName)
		return err
	}

	return nil
}

// newHPAForScaledObject returns HPA as it is specified in ScaledObject
func (r *ReconcileScaledObject) newHPAForScaledObject(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, gvkr *kedautil.GroupVersionKindResource) (*autoscalingv2beta2.HorizontalPodAutoscaler, error) {
	scaledObjectMetricSpecs, err := r.getScaledObjectMetricSpecs(logger, scaledObject)
	if err != nil {
		return nil, err
	}

	// label can have max 63 chars
	labelName := getHPAName(scaledObject)
	if len(labelName) > 63 {
		labelName = labelName[:63]
	}
	labels := map[string]string{
		"app.kubernetes.io/name":       labelName,
		"app.kubernetes.io/version":    version.Version,
		"app.kubernetes.io/part-of":    scaledObject.Name,
		"app.kubernetes.io/managed-by": "keda-operator",
	}

	return &autoscalingv2beta2.HorizontalPodAutoscaler{
		Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
			MinReplicas: getHPAMinReplicas(scaledObject),
			MaxReplicas: getHPAMaxReplicas(scaledObject),
			Metrics:     scaledObjectMetricSpecs,
			ScaleTargetRef: autoscalingv2beta2.CrossVersionObjectReference{
				Name:       scaledObject.Spec.ScaleTargetRef.Name,
				Kind:       gvkr.Kind,
				APIVersion: gvkr.GroupVersion().String(),
			}},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getHPAName(scaledObject),
			Namespace: scaledObject.Namespace,
			Labels:    labels,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2beta2",
		},
	}, nil
}

// updateHPAIfNeeded checks whether update of HPA is needed
func (r *ReconcileScaledObject) updateHPAIfNeeded(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, foundHpa *autoscalingv2beta2.HorizontalPodAutoscaler, gvkr *kedautil.GroupVersionKindResource) error {

	hpa, err := r.newHPAForScaledObject(logger, scaledObject, gvkr)
	if err != nil {
		logger.Error(err, "Failed to create new HPA resource", "HPA.Namespace", scaledObject.Namespace, "HPA.Name", getHPAName(scaledObject))
		return err
	}

	if !equality.Semantic.DeepDerivative(hpa.Spec, foundHpa.Spec) {
		if r.client.Update(context.TODO(), foundHpa) != nil {
			foundHpa.Spec = hpa.Spec
			logger.Error(err, "Failed to update HPA", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
			return err
		}
		logger.Info("Updated HPA according to ScaledObject", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
	}

	return nil
}

// getScaledObjectMetricSpecs returns MetricSpec for HPA, generater from Triggers defitinion in ScaledObject
func (r *ReconcileScaledObject) getScaledObjectMetricSpecs(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) ([]autoscalingv2beta2.MetricSpec, error) {
	var scaledObjectMetricSpecs []autoscalingv2beta2.MetricSpec
	var externalMetricNames []string

	scalers, err := r.scaleHandler.GetScalers(scaledObject)
	if err != nil {
		logger.Error(err, "Error getting scalers")
		return nil, err
	}

	for _, scaler := range scalers {
		metricSpecs := scaler.GetMetricSpecForScaling()

		// add the scaledObjectName label. This is how the MetricsAdapter will know which scaledobject a metric is for when the HPA queries it.
		for _, metricSpec := range metricSpecs {
			metricSpec.External.Metric.Selector = &metav1.LabelSelector{MatchLabels: make(map[string]string)}
			metricSpec.External.Metric.Selector.MatchLabels["scaledObjectName"] = scaledObject.Name
			externalMetricNames = append(externalMetricNames, metricSpec.External.Metric.Name)
		}
		scaledObjectMetricSpecs = append(scaledObjectMetricSpecs, metricSpecs...)
		scaler.Close()
	}

	// store External.MetricNames used by scalers defined in the ScaledObject
	scaledObject.Status.ExternalMetricNames = externalMetricNames
	err = r.client.Status().Update(context.TODO(), scaledObject)
	if err != nil {
		logger.Error(err, "Error updating scaledObject status with used externalMetricNames")
		return nil, err
	}

	return scaledObjectMetricSpecs, nil
}

// getHPAName returns generated HPA name for ScaledObject specified in the parameter
func getHPAName(scaledObject *kedav1alpha1.ScaledObject) string {
	return fmt.Sprintf("keda-hpa-%s", scaledObject.Name)
}

// getHPAMinReplicas returns MinReplicas based on definition in ScaledObject or default value if not defined
func getHPAMinReplicas(scaledObject *kedav1alpha1.ScaledObject) *int32 {
	if scaledObject.Spec.MinReplicaCount != nil && *scaledObject.Spec.MinReplicaCount > 0 {
		return scaledObject.Spec.MinReplicaCount
	}
	tmp := defaultHPAMinReplicas
	return &tmp
}

// getHPAMaxReplicas returns MaxReplicas based on definition in ScaledObject or default value if not defined
func getHPAMaxReplicas(scaledObject *kedav1alpha1.ScaledObject) int32 {
	if scaledObject.Spec.MaxReplicaCount != nil {
		return *scaledObject.Spec.MaxReplicaCount
	}
	return defaultHPAMaxReplicas
}
