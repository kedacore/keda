/*
Copyright 2022 The KEDA Authors

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

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/metrics"
)

type PrometheusMetricsReconciler struct {
	client.Client

	watchNamespace string
}

// +kubebuilder:rbac:groups=keda.sh,resources=scaledobjects;scaledobjects/finalizers;scaledobjects/status,verbs="*"
// +kubebuilder:rbac:groups=keda.sh,resources=scaledjobs;scaledjobs/finalizers;scaledjobs/status,verbs="*"
// SetupWithManager initializes the PrometheusMetricsReconciler instance and starts a new controller managed by the passed Manager instance.
func (r *PrometheusMetricsReconciler) SetupWithManager(mgr ctrl.Manager, watchNamespace string) error {
	r.watchNamespace = watchNamespace

	return ctrl.NewControllerManagedBy(mgr).
		For(&kedav1alpha1.ScaledObject{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&source.Kind{Type: &kedav1alpha1.ScaledJob{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		Complete(r)
}

func (r *PrometheusMetricsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.Log.WithValues("controller", "PrometheusMetrics")

	if err := r.updateTriggerTotals(ctx, logger); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PrometheusMetricsReconciler) updateTriggerTotals(ctx context.Context, logger logr.Logger) error {
	scaledObjects, scaledJobs, err := r.listScaledObjectsAndJobs(ctx, logger)
	if err != nil {
		return err
	}

	triggerTotals := r.countTriggerTotals(scaledObjects, scaledJobs)

	metrics.SetTriggerTotals(triggerTotals)
	return nil
}

func (r *PrometheusMetricsReconciler) listScaledObjectsAndJobs(ctx context.Context, logger logr.Logger) ([]kedav1alpha1.ScaledObject, []kedav1alpha1.ScaledJob, error) {
	scaledObjectList := kedav1alpha1.ScaledObjectList{}
	scaledJobList := kedav1alpha1.ScaledJobList{}

	listOptions := make([]client.ListOption, 0)

	if r.watchNamespace != "" {
		listOptions = append(listOptions, client.InNamespace(r.watchNamespace))
	}

	if err := r.Client.List(ctx, &scaledObjectList, listOptions...); err != nil {
		logger.Error(err, "failed to list scaled objects")
		return []kedav1alpha1.ScaledObject{}, []kedav1alpha1.ScaledJob{}, err
	}

	if err := r.Client.List(ctx, &scaledJobList, listOptions...); err != nil {
		logger.Error(err, "failed to list scaled jobs")
		return []kedav1alpha1.ScaledObject{}, []kedav1alpha1.ScaledJob{}, err
	}

	return scaledObjectList.Items, scaledJobList.Items, nil
}

func (r *PrometheusMetricsReconciler) countTriggerTotals(scaledObjects []kedav1alpha1.ScaledObject, scaledJobs []kedav1alpha1.ScaledJob) map[string]int {
	triggerTotals := make(map[string]int)

	for _, scaledObject := range scaledObjects {
		for _, trigger := range scaledObject.Spec.Triggers {
			// don't consider objects marked for deletion
			if scaledObject.GetDeletionTimestamp() == nil {
				triggerTotals[trigger.Type]++
			}
		}
	}

	for _, scaledJob := range scaledJobs {
		for _, trigger := range scaledJob.Spec.Triggers {
			// don't consider objects marked for deletion
			if scaledJob.GetDeletionTimestamp() == nil {
				triggerTotals[trigger.Type]++
			}
		}
	}

	return triggerTotals
}
