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
	"reflect"
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

const (
	defaultHPAMinReplicas int32 = 1
	defaultHPAMaxReplicas int32 = 100
)

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

	// store hpaName in the ScaledObject
	status := scaledObject.Status.DeepCopy()
	status.HpaName = hpaName

	err = kedastatus.UpdateScaledObjectStatus(ctx, r.Client, logger, scaledObject, status)
	if err != nil {
		logger.Error(err, "Error updating scaledObject status with used hpaName")
		return err
	}

	return nil
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
	for key, value := range scaledObject.ObjectMeta.Labels {
		labels[key] = value
	}

	minReplicas := getHPAMinReplicas(scaledObject)
	maxReplicas := getHPAMaxReplicas(scaledObject)

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
	if len(hpa.Spec.Metrics) != len(foundHpa.Spec.Metrics) || !equality.Semantic.DeepDerivative(hpa.Spec, foundHpa.Spec) {
		logger.V(1).Info("Found difference in the HPA spec accordint to ScaledObject", "currentHPA", foundHpa.Spec, "newHPA", hpa.Spec)
		if err = r.Client.Update(ctx, hpa); err != nil {
			foundHpa.Spec = hpa.Spec
			logger.Error(err, "Failed to update HPA", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
			return err
		}

		logger.Info("Updated HPA according to ScaledObject", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
	}

	if !equality.Semantic.DeepDerivative(hpa.ObjectMeta.Labels, foundHpa.ObjectMeta.Labels) {
		logger.V(1).Info("Found difference in the HPA labels accordint to ScaledObject", "currentHPA", foundHpa.ObjectMeta.Labels, "newHPA", hpa.ObjectMeta.Labels)
		if err = r.Client.Update(ctx, hpa); err != nil {
			foundHpa.ObjectMeta.Labels = hpa.ObjectMeta.Labels
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

	// TODO: it returns indexed names here
	metricSpecs := cache.GetMetricSpecForScaling(ctx)

	for _, metricSpec := range metricSpecs {
		if metricSpec.Resource != nil {
			resourceMetricNames = append(resourceMetricNames, string(metricSpec.Resource.Name))
		}

		if metricSpec.External != nil {
			externalMetricName := metricSpec.External.Metric.Name
			if kedacontrollerutil.Contains(externalMetricNames, externalMetricName) {
				return nil, fmt.Errorf("metricName %s defined multiple times in ScaledObject %s, please refer the documentation how to define metricName manually", externalMetricName, scaledObject.Name)
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

	// if ComplexScalingLogic struct is not empty, expect Formula or ExternalCalculation
	// to be non-empty. If target is > 0.0 create a compositeScaler structure
	if scaledObject.Spec.Advanced != nil &&
		!reflect.DeepEqual(scaledObject.Spec.Advanced.ComplexScalingLogic, kedav1alpha1.ComplexScalingLogic{}) {
		validNumTarget, validMetricType, err := validateCompositeScalingLogic(scaledObject, scaledObjectMetricSpecs)
		if err != nil {
			logger.Error(err, "error validating compositeScalingLogic")
			return nil, err
		}

		// if target is valid, use composite scaler.
		// Expect Formula or ExternalCalculation that returns one metric
		if validNumTarget > 0.0 {
			qual := resource.NewMilliQuantity(int64(validNumTarget*1000), resource.DecimalSI)

			if err != nil {
				logger.Error(err, "Error parsing Quantity elements for composite scaler")
				return nil, err
			}
			compositeSpec := autoscalingv2.MetricSpec{
				Type: autoscalingv2.MetricSourceType("External"),
				External: &autoscalingv2.ExternalMetricSource{
					Metric: autoscalingv2.MetricIdentifier{
						Name: "composite-metric-name",
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"scaledobject.keda.sh/name": scaledObject.Name},
						},
					},
					Target: autoscalingv2.MetricTarget{
						Type:         validMetricType,
						AverageValue: qual,
					},
				},
			}
			status.CompositeScalerName = "composite-metric-name"

			// overwrite returned array with composite metric ONLY
			scaledObjectMetricSpecs = []autoscalingv2.MetricSpec{compositeSpec}
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

// validateCompositeScalingLogic validates all combinations of given arguments
// and their values
func validateCompositeScalingLogic(so *kedav1alpha1.ScaledObject, specs []autoscalingv2.MetricSpec) (float64, autoscalingv2.MetricTargetType, error) {
	csl := so.Spec.Advanced.ComplexScalingLogic

	// if Formula AND ExternalCalculation is empty, return an error
	if csl.Formula == "" && len(csl.ExternalCalculations) == 0 {
		return -1, autoscalingv2.MetricTargetType(""), fmt.Errorf("error atleast one complex scaling logic needs to be specified (formula or externalCalculation)")
	}

	var num float64
	var metricType autoscalingv2.MetricTargetType
	var err error
	// If ComplexScalingLogic.Formula is non-empty, target needs to be specified
	if csl.Formula != "" {
		if csl.Target == "" {
			return -1, autoscalingv2.MetricTargetType(""), fmt.Errorf("error complexScalingLogic formula is given, but target is empty")
		}
		// TODO: possibly validate formula here otherwise combine the two ifs above
	}

	// if ExternalCalculation is given, target doesnt need to be specified but can depending
	// on if the user wants to use custom composite scaler
	// if len(csl.ExternalCalculations) > 0 {
	// TODO: check if connection to the endpoints are valid?
	// }

	if csl.Target != "" {
		// convert string to float
		num, err = strconv.ParseFloat(csl.Target, 64)
		if err != nil || num <= 0.0 {
			return -1, autoscalingv2.MetricTargetType(""), fmt.Errorf("error converting target for complex logic (string->float): %w", err)
		}
	}

	// if both are empty OR both are given its an error
	// if (csl.Formula == "" && len(csl.ComplexScalingLogic) == 0) ||
	// (csl.Formula != "" && len(csl.ComplexScalingLogic) > 0) {
	// err := fmt.Errorf("error exactly one of Formula or ExternalCalculator can be given")
	// return -1, autoscalingv2.MetricTargetType(""), err
	// }

	// if target is given, complex custom scaler for metric collection will be
	// passed to HPA config -> all types need to be the same
	if csl.Target != "" {
		// make sure all scalers have the same metricTargetType
		for i, metric := range specs {
			if i == 0 {
				metricType = metric.External.Target.Type
			} else if metric.External.Target.Type != metricType {
				err := fmt.Errorf("error metric target type not the same for composite scaler: %s & %s", metricType, metric.External.Target.Type)
				return -1, metricType, err
			}
		}
	}
	return num, metricType, nil
}
