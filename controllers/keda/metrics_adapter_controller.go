/*
Copyright 2021 The KEDA Authors

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

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scaling"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type MetricsScaledObjectReconciler struct {
	ScaleHandler scaling.ScaleHandler
}

func (r *MetricsScaledObjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.ScaleHandler.ClearScalersCache(ctx, req.Name, req.Namespace)
	return ctrl.Result{}, nil
}

func (r *MetricsScaledObjectReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kedav1alpha1.ScaledObject{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&kedav1alpha1.ScaledObject{}).
		WithOptions(options).
		Complete(r)
}
