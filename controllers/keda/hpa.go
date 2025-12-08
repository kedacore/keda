/*
Copyright 2023 The KEDA Authors

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

package keda

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-logr/logr"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedacontrollerutil "github.com/kedacore/keda/v2/controllers/keda/util"
	"github.com/kedacore/keda/v2/pkg/scaling/executor"
	kedastatus "github.com/kedacore/keda/v2/pkg/status"
	version "github.com/kedacore/keda/v2/version"
)

// storeHpaNameInStatus updates the ScaledObject status subresource with the hpaName.
func (r *ScaledObjectReconciler) storeHpaNameInStatus(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, hpaName string) error {
	status := scaledObject.Status.DeepCopy()
	status.HpaName = hpaName

	err := kedastatus.UpdateScaledObjectStatus(ctx, r.Client, logger, scaledObject, status)
	if err != nil {
		logger.Error(err, "Failed to update scaledObject status with used hpaName")
		return err
	}
	return nil
}

// createAndDeployNewHPA creates and deploy HPA in the cluster for specified ScaledObject
func (r *ScaledObjectReconciler) createAndDeployNewHPA(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, gvkr *kedav1alpha1.GroupVersionKindResource) error {
	hpaName := getHPAName(scaledObject)
	logger.Info("Creating a new HPA", "HPA.Namespace", scaledObject.Namespace, "HPA.Name", hpaName)
	hpa, err := r.newHPAForScaledObject(ctx, logger, scaledObject, gvkr)
	if err != nil {
		logger.Error(err, "Failed to create new HPA resource", "HPA.Namespace", scaledObject.Namespace, "HPA.Name", hpaName)
		return err
	}

	err = r.Client.Create(ctx, hpa)
	if err != nil {
		logger.Error(err, "Failed to create new HPA in cluster", "HPA.Namespace", scaledObject.Namespace, "HPA.Name", hpaName)
		return err
	}

	return r.storeHpaNameInStatus(ctx, logger, scaledObject, hpaName)
}

// newHPAForScaledObject returns HPA as it is specified in ScaledObject
func (r *ScaledObjectReconciler) newHPAForScaledObject(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, gvkr *kedav1alpha1.GroupVersionKindResource) (*autoscalingv2.HorizontalPodAutoscaler, error) {
	scaledObjectMetricSpecs, err := r.getScaledObjectMetricSpecs(ctx, logger, scaledObject)
	if err != nil {
		return nil, err
	}

	var behavior *autoscalingv2.HorizontalPodAutoscalerBehavior
	if scaledObject.Spec.Advanced != nil && scaledObject.Spec.Advanced.HorizontalPodAutoscalerConfig != nil {
		behavior = scaledObject.Spec.Advanced.HorizontalPodAutoscalerConfig.Behavior
	} else {
		behavior = nil
	}

	if scaledObject.NeedToPauseScaleIn() {
		// If the paused-scale-in annotation is set, set the HPA ScaleDown Select policy to Disabled
		// to prevent the HPA from scaling down the scale target
		if behavior == nil {
			behavior = &autoscalingv2.HorizontalPodAutoscalerBehavior{}
		}
		if behavior.ScaleDown == nil {
			behavior.ScaleDown = &autoscalingv2.HPAScalingRules{}
		}

		disabledPolicy := autoscalingv2.DisabledPolicySelect
		behavior.ScaleDown.SelectPolicy = &disabledPolicy

		logger.Info(
			"Scale in paused by annotation, setting HPA Scale Down Select Behavior to Disabled",
			"HPA.Namespace",
			scaledObject.Namespace,
			"HPA.Name",
			getHPAName(scaledObject),
		)
	}

	if scaledObject.NeedToPauseScaleOut() {
		// If the paused-scale-out annotation is set, set the HPA ScaleUp Select policy to Disabled
		// to prevent the HPA from scaling up the scale target
		if behavior == nil {
			behavior = &autoscalingv2.HorizontalPodAutoscalerBehavior{}
		}
		if behavior.ScaleUp == nil {
			behavior.ScaleUp = &autoscalingv2.HPAScalingRules{}
		}

		disabledPolicy := autoscalingv2.DisabledPolicySelect
		behavior.ScaleUp.SelectPolicy = &disabledPolicy

		logger.Info(
			"Scale out paused by annotation, setting HPA Scale Up Select Behavior to Disabled",
			"HPA.Namespace",
			scaledObject.Namespace,
			"HPA.Name",
			getHPAName(scaledObject),
		)
	}

	// label can have max 63 chars
	labelName := getHPAName(scaledObject)
	if len(labelName) > 63 {
		labelName = labelName[:63]
		labelName = strings.TrimRightFunc(labelName, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
	}
	labels := map[string]string{
		"app.kubernetes.io/name":       labelName,
		"app.kubernetes.io/version":    version.Version,
		"app.kubernetes.io/part-of":    scaledObject.Name,
		"app.kubernetes.io/managed-by": "keda-operator",
	}

	excludedLabels := map[string]struct{}{}

	if labels, ok := scaledObject.Annotations[kedav1alpha1.ScaledObjectExcludedLabelsAnnotation]; ok {
		for _, excludedLabel := range strings.Split(labels, ",") {
			excludedLabels[excludedLabel] = struct{}{}
		}
	}

	for key, value := range scaledObject.Labels {
		if _, ok := excludedLabels[key]; ok {
			continue
		}

		labels[key] = value
	}

	minReplicas := scaledObject.GetHPAMinReplicas()
	maxReplicas := scaledObject.GetHPAMaxReplicas()

	pausedCount, err := executor.GetPausedReplicaCount(scaledObject)
	if err != nil {
		return nil, err
	}
	if pausedCount != nil {
		// MinReplicas on HPA can't be 0
		if *pausedCount == 0 {
			*pausedCount = 1
		}
		minReplicas = pausedCount
		maxReplicas = *pausedCount
	}

	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			MinReplicas: minReplicas,
			MaxReplicas: maxReplicas,
			Metrics:     scaledObjectMetricSpecs,
			Behavior:    behavior,
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Name:       scaledObject.Spec.ScaleTargetRef.Name,
				Kind:       gvkr.Kind,
				APIVersion: gvkr.GroupVersion().String(),
			}},
		ObjectMeta: metav1.ObjectMeta{
			Name:        getHPAName(scaledObject),
			Namespace:   scaledObject.Namespace,
			Labels:      labels,
			Annotations: scaledObject.Annotations,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2",
		},
	}

	// Set ScaledObject instance as the owner and controller
	if err := controllerutil.SetControllerReference(scaledObject, hpa, r.Scheme); err != nil {
		return nil, err
	}

	return hpa, nil
}

// updateHPAIfNeeded checks whether update of HPA is needed
func (r *ScaledObjectReconciler) updateHPAIfNeeded(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, foundHpa *autoscalingv2.HorizontalPodAutoscaler, gvkr *kedav1alpha1.GroupVersionKindResource) error {
	hpa, err := r.newHPAForScaledObject(ctx, logger, scaledObject, gvkr)
	if err != nil {
		logger.Error(err, "Failed to create new HPA resource", "HPA.Namespace", scaledObject.Namespace, "HPA.Name", getHPAName(scaledObject))
		return err
	}

	// DeepDerivative ignores extra entries in arrays which makes removing the last trigger not update things, so trigger and update any time the metrics count is different.
	// DeepDerivative also treats nil as "unset" and a subset of any value, so we need to explicitly check Behavior with DeepEqual
	// to detect when paused-scale-in/out annotations are removed and Behavior should change from Disabled back to nil.
	if len(hpa.Spec.Metrics) != len(foundHpa.Spec.Metrics) ||
		!equality.Semantic.DeepEqual(hpa.Spec.Behavior, foundHpa.Spec.Behavior) ||
		!equality.Semantic.DeepDerivative(hpa.Spec, foundHpa.Spec) {
		logger.V(1).Info("Found difference in the HPA spec accordint to ScaledObject", "currentHPA", foundHpa.Spec, "newHPA", hpa.Spec)
		if err = r.Client.Update(ctx, hpa); err != nil {
			foundHpa.Spec = hpa.Spec
			logger.Error(err, "Failed to update HPA", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
			return err
		}

		logger.Info("Updated HPA according to ScaledObject", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
	}

	if !equality.Semantic.DeepDerivative(hpa.Labels, foundHpa.Labels) {
		logger.V(1).Info("Found difference in the HPA labels accordint to ScaledObject", "currentHPA", foundHpa.Labels, "newHPA", hpa.Labels)
		if err = r.Client.Update(ctx, hpa); err != nil {
			foundHpa.Labels = hpa.Labels
			logger.Error(err, "Failed to update HPA", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
			return err
		}
		logger.Info("Updated HPA according to ScaledObject", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
	}

	if (hpa.Annotations == nil && foundHpa.Annotations != nil) ||
		!equality.Semantic.DeepDerivative(hpa.Annotations, foundHpa.Annotations) {
		logger.V(1).Info("Found difference in the HPA annotations according to ScaledObject", "currentHPA", foundHpa.Annotations, "newHPA", hpa.Annotations)
		if err = r.Client.Update(ctx, hpa); err != nil {
			foundHpa.Annotations = hpa.Annotations
			logger.Error(err, "Failed to update HPA", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
			return err
		}
		logger.Info("Updated HPA according to ScaledObject", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
	}

	return nil
}

// deleteAndCreateHpa delete old HPA and create new one
func (r *ScaledObjectReconciler) renameHPA(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, foundHpa *autoscalingv2.HorizontalPodAutoscaler, gvkr *kedav1alpha1.GroupVersionKindResource) error {
	if err := r.deleteHPA(ctx, logger, scaledObject, foundHpa); err != nil {
		return err
	}
	return r.createAndDeployNewHPA(ctx, logger, scaledObject, gvkr)
}

// deleteHpa delete existing HPA
func (r *ScaledObjectReconciler) deleteHPA(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, foundHpa *autoscalingv2.HorizontalPodAutoscaler) error {
	logger.Info("Deleting existing HPA", "HPA.Namespace", scaledObject.Namespace, "HPA.Name", foundHpa.Name)
	if err := r.Client.Delete(ctx, foundHpa); err != nil {
		logger.Error(err, "Failed to delete old HPA", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
		return err
	}

	return nil
}

// getScaledObjectMetricSpecs returns MetricSpec for HPA, generater from Triggers defitinion in ScaledObject
func (r *ScaledObjectReconciler) getScaledObjectMetricSpecs(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) ([]autoscalingv2.MetricSpec, error) {
	var scaledObjectMetricSpecs []autoscalingv2.MetricSpec
	var externalMetricNames []string
	var resourceMetricNames []string

	cache, err := r.ScaleHandler.GetScalersCache(ctx, scaledObject.DeepCopy())
	if err != nil {
		logger.Error(err, "Error getting scalers")
		return nil, err
	}

	metricSpecs := cache.GetMetricSpecForScaling(ctx)

	for _, metricSpec := range metricSpecs {
		if metricSpec.Resource != nil {
			resourceMetricNames = append(resourceMetricNames, string(metricSpec.Resource.Name))
		}

		if metricSpec.External != nil {
			externalMetricName := metricSpec.External.Metric.Name
			if kedacontrollerutil.Contains(externalMetricNames, externalMetricName) {
				return nil, fmt.Errorf("metricName %s defined multiple times in ScaledObject %s", externalMetricName, scaledObject.Name)
			}

			// add the scaledobject.keda.sh/name label. This is how the MetricsAdapter will know which scaledobject a metric is for when the HPA queries it.
			metricSpec.External.Metric.Selector = &metav1.LabelSelector{MatchLabels: make(map[string]string)}
			metricSpec.External.Metric.Selector.MatchLabels[kedav1alpha1.ScaledObjectOwnerAnnotation] = scaledObject.Name
			externalMetricNames = append(externalMetricNames, externalMetricName)
		}
	}
	scaledObjectMetricSpecs = append(scaledObjectMetricSpecs, metricSpecs...)

	// sort metrics in ScaledObject, this way we always check the same resource in Reconcile loop and we can prevent unnecessary HPA updates,
	// see https://github.com/kedacore/keda/issues/1531 for details
	sort.Slice(scaledObjectMetricSpecs, func(i, j int) bool {
		return scaledObjectMetricSpecs[i].Type < scaledObjectMetricSpecs[j].Type
	})

	// store External.MetricNames,Resource.MetricsNames used by scalers defined in the ScaledObject
	status := scaledObject.Status.DeepCopy()
	status.ExternalMetricNames = externalMetricNames
	status.ResourceMetricNames = resourceMetricNames

	updateHealthStatus(scaledObject, externalMetricNames, status)

	// if ScalingModifiers struct is not empty, expect Formula and Target to be
	// non-empty (is validated beforehand - in cache). Only if target is > 0.0
	// create a compositeScaler structure
	if scaledObject.IsUsingModifiers() {
		// convert string to float (this is already validated in:
		// cache, err := r.ScaleHandler.GetScalersCache(ctx, scaledObject.DeepCopy())
		// at the beginning of this function, where the whole scalingModifiers are validated)
		validNumTarget, _ := strconv.ParseFloat(scaledObject.Spec.Advanced.ScalingModifiers.Target, 64)

		// check & get metric specs type
		metricType := autoscalingv2.AverageValueMetricType
		if scaledObject.Spec.Advanced.ScalingModifiers.MetricType != "" {
			metricType = scaledObject.Spec.Advanced.ScalingModifiers.MetricType
		}

		if metricType == autoscalingv2.UtilizationMetricType {
			err := fmt.Errorf("error metric target type is Utilization, but it needs to be AverageValue or Value for external metrics")
			return nil, err
		}

		// if target is valid, use composite scaler. Expect defined formula that returns one metric
		if validNumTarget > 0.0 {
			quan := resource.NewMilliQuantity(int64(validNumTarget*1000), resource.DecimalSI)

			correctHpaTarget := autoscalingv2.MetricTarget{
				Type: metricType,
			}
			switch metricType {
			case autoscalingv2.AverageValueMetricType:
				correctHpaTarget.AverageValue = quan
			case autoscalingv2.ValueMetricType:
				correctHpaTarget.Value = quan
			}
			compMetricName := kedav1alpha1.CompositeMetricName
			compositeSpec := autoscalingv2.MetricSpec{
				Type: autoscalingv2.MetricSourceType("External"),
				External: &autoscalingv2.ExternalMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: compMetricName,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{kedav1alpha1.ScaledObjectOwnerAnnotation: scaledObject.Name},
						},
					},
					Target: correctHpaTarget,
				},
			}
			status.CompositeScalerName = compMetricName

			// overwrite external metrics in returned array with composite metric ONLY (keep resource metrics)
			finalHpaSpecs := []autoscalingv2.MetricSpec{}
			// keep resource specs
			for _, rm := range scaledObjectMetricSpecs {
				if rm.Resource != nil {
					finalHpaSpecs = append(finalHpaSpecs, rm)
				}
			}
			finalHpaSpecs = append(finalHpaSpecs, compositeSpec)
			scaledObjectMetricSpecs = finalHpaSpecs
		}
	}
	err = kedastatus.UpdateScaledObjectStatus(ctx, r.Client, logger, scaledObject, status)

	if err != nil {
		logger.Error(err, "Error updating scaledObject status with used externalMetricNames")
		return nil, err
	}

	return scaledObjectMetricSpecs, nil
}

func updateHealthStatus(scaledObject *kedav1alpha1.ScaledObject, externalMetricNames []string, status *kedav1alpha1.ScaledObjectStatus) {
	health := scaledObject.Status.Health
	newHealth := make(map[string]kedav1alpha1.HealthStatus)
	for _, metricName := range externalMetricNames {
		entry, exists := health[metricName]
		if exists {
			newHealth[metricName] = entry
		}
	}
	status.Health = newHealth
}

// getHPAName returns generated HPA name for ScaledObject specified in the parameter
func getHPAName(scaledObject *kedav1alpha1.ScaledObject) string {
	if scaledObject.Spec.Advanced != nil && scaledObject.Spec.Advanced.HorizontalPodAutoscalerConfig != nil && scaledObject.Spec.Advanced.HorizontalPodAutoscalerConfig.Name != "" {
		return scaledObject.Spec.Advanced.HorizontalPodAutoscalerConfig.Name
	}
	return getDefaultHpaName(scaledObject)
}

func getDefaultHpaName(scaledObject *kedav1alpha1.ScaledObject) string {
	return fmt.Sprintf("keda-hpa-%s", scaledObject.Name)
}
