package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	version "github.com/kedacore/keda/v2/version"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	kedacontrollerutil "github.com/kedacore/keda/v2/controllers/util"
)

const (
	defaultHPAMinReplicas int32 = 1
	defaultHPAMaxReplicas int32 = 100
)

// createAndDeployNewHPA creates and deploy HPA in the cluster for specified ScaledObject
func (r *ScaledObjectReconciler) createAndDeployNewHPA(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, gvkr *kedav1alpha1.GroupVersionKindResource) error {
	hpaName := getHPAName(scaledObject)
	logger.Info("Creating a new HPA", "HPA.Namespace", scaledObject.Namespace, "HPA.Name", hpaName)
	hpa, err := r.newHPAForScaledObject(logger, scaledObject, gvkr)
	if err != nil {
		logger.Error(err, "Failed to create new HPA resource", "HPA.Namespace", scaledObject.Namespace, "HPA.Name", hpaName)
		return err
	}

	err = r.Client.Create(context.TODO(), hpa)
	if err != nil {
		logger.Error(err, "Failed to create new HPA in cluster", "HPA.Namespace", scaledObject.Namespace, "HPA.Name", hpaName)
		return err
	}

	return nil
}

// newHPAForScaledObject returns HPA as it is specified in ScaledObject
func (r *ScaledObjectReconciler) newHPAForScaledObject(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, gvkr *kedav1alpha1.GroupVersionKindResource) (*autoscalingv2beta2.HorizontalPodAutoscaler, error) {
	scaledObjectMetricSpecs, err := r.getScaledObjectMetricSpecs(logger, scaledObject)
	if err != nil {
		return nil, err
	}

	var behavior *autoscalingv2beta2.HorizontalPodAutoscalerBehavior
	if r.kubeVersion.MinorVersion >= 18 && scaledObject.Spec.Advanced != nil && scaledObject.Spec.Advanced.HorizontalPodAutoscalerConfig != nil {
		behavior = scaledObject.Spec.Advanced.HorizontalPodAutoscalerConfig.Behavior
	} else {
		behavior = nil
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
	for key, value := range scaledObject.ObjectMeta.Labels {
		labels[key] = value
	}

	hpa := &autoscalingv2beta2.HorizontalPodAutoscaler{
		Spec: autoscalingv2beta2.HorizontalPodAutoscalerSpec{
			MinReplicas: getHPAMinReplicas(scaledObject),
			MaxReplicas: getHPAMaxReplicas(scaledObject),
			Metrics:     scaledObjectMetricSpecs,
			Behavior:    behavior,
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
	}

	// Set ScaledObject instance as the owner and controller
	if err := controllerutil.SetControllerReference(scaledObject, hpa, r.Scheme); err != nil {
		return nil, err
	}

	return hpa, nil
}

// updateHPAIfNeeded checks whether update of HPA is needed
func (r *ScaledObjectReconciler) updateHPAIfNeeded(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, foundHpa *autoscalingv2beta2.HorizontalPodAutoscaler, gvkr *kedav1alpha1.GroupVersionKindResource) error {
	hpa, err := r.newHPAForScaledObject(logger, scaledObject, gvkr)
	if err != nil {
		logger.Error(err, "Failed to create new HPA resource", "HPA.Namespace", scaledObject.Namespace, "HPA.Name", getHPAName(scaledObject))
		return err
	}

	if !equality.Semantic.DeepDerivative(hpa.Spec, foundHpa.Spec) {
		logger.V(1).Info("Found difference in the HPA spec accordint to ScaledObject", "currentHPA", foundHpa.Spec, "newHPA", hpa.Spec)
		if r.Client.Update(context.TODO(), hpa) != nil {
			foundHpa.Spec = hpa.Spec
			logger.Error(err, "Failed to update HPA", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
			return err
		}
		// check if scaledObject.spec.behavior was defined, because it is supported only on k8s >= 1.18
		r.checkMinK8sVersionforHPABehavior(logger, scaledObject)

		logger.Info("Updated HPA according to ScaledObject", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
	}

	if !equality.Semantic.DeepDerivative(hpa.ObjectMeta.Labels, foundHpa.ObjectMeta.Labels) {
		logger.V(1).Info("Found difference in the HPA labels accordint to ScaledObject", "currentHPA", foundHpa.ObjectMeta.Labels, "newHPA", hpa.ObjectMeta.Labels)
		if r.Client.Update(context.TODO(), hpa) != nil {
			foundHpa.ObjectMeta.Labels = hpa.ObjectMeta.Labels
			logger.Error(err, "Failed to update HPA", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
			return err
		}
		logger.Info("Updated HPA according to ScaledObject", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
	}

	return nil
}

// getScaledObjectMetricSpecs returns MetricSpec for HPA, generater from Triggers defitinion in ScaledObject
func (r *ScaledObjectReconciler) getScaledObjectMetricSpecs(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) ([]autoscalingv2beta2.MetricSpec, error) {
	var scaledObjectMetricSpecs []autoscalingv2beta2.MetricSpec
	var externalMetricNames []string
	var resourceMetricNames []string

	scalers, err := r.scaleHandler.GetScalers(scaledObject)
	if err != nil {
		logger.Error(err, "Error getting scalers")
		return nil, err
	}

	for _, scaler := range scalers {
		metricSpecs := scaler.GetMetricSpecForScaling()

		for _, metricSpec := range metricSpecs {
			if metricSpec.Resource != nil {
				resourceMetricNames = append(resourceMetricNames, string(metricSpec.Resource.Name))
			}
			if metricSpec.External != nil {
				// add the scaledObjectName label. This is how the MetricsAdapter will know which scaledobject a metric is for when the HPA queries it.
				metricSpec.External.Metric.Selector = &metav1.LabelSelector{MatchLabels: make(map[string]string)}
				metricSpec.External.Metric.Selector.MatchLabels["scaledObjectName"] = scaledObject.Name
				externalMetricNames = append(externalMetricNames, metricSpec.External.Metric.Name)
			}
		}
		scaledObjectMetricSpecs = append(scaledObjectMetricSpecs, metricSpecs...)
		scaler.Close()
	}

	// store External.MetricNames,Resource.MetricsNames used by scalers defined in the ScaledObject
	status := scaledObject.Status.DeepCopy()
	status.ExternalMetricNames = externalMetricNames
	status.ResourceMetricNames = resourceMetricNames
	err = kedacontrollerutil.UpdateScaledObjectStatus(r.Client, logger, scaledObject, status)
	if err != nil {
		logger.Error(err, "Error updating scaledObject status with used externalMetricNames")
		return nil, err
	}

	return scaledObjectMetricSpecs, nil
}

// checkMinK8sVersionforHPABehavior min version (k8s v1.18) for HPA Behavior
func (r *ScaledObjectReconciler) checkMinK8sVersionforHPABehavior(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) {
	if r.kubeVersion.MinorVersion < 18 {
		if scaledObject.Spec.Advanced != nil && scaledObject.Spec.Advanced.HorizontalPodAutoscalerConfig != nil && scaledObject.Spec.Advanced.HorizontalPodAutoscalerConfig.Behavior != nil {
			logger.Info("Warning: Ignoring scaledObject.spec.behavior, it is only supported on kubernetes version >= 1.18", "kubernetes.version", r.kubeVersion.PrettyVersion)
		}
	}
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
