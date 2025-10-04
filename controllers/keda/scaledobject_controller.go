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
	"strconv"
	"sync"

	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	eventingv1alpha1 "github.com/kedacore/keda/v2/apis/eventing/v1alpha1"
	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	kedacontrollerutil "github.com/kedacore/keda/v2/controllers/keda/util"
	"github.com/kedacore/keda/v2/pkg/common/message"
	"github.com/kedacore/keda/v2/pkg/eventemitter"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/fallback"
	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/scaling"
	kedastatus "github.com/kedacore/keda/v2/pkg/status"
	"github.com/kedacore/keda/v2/pkg/util"
)

// +kubebuilder:rbac:groups=keda.sh,resources=scaledobjects;scaledobjects/finalizers;scaledobjects/status,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;update;patch;create;delete
// +kubebuilder:rbac:groups="",resources=configmaps;configmaps/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=pods;services;services;secrets;external,verbs=get;list;watch
// +kubebuilder:rbac:groups="*",resources="*/scale",verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources="serviceaccounts",verbs=list;watch
// +kubebuilder:rbac:groups="*",resources="*",verbs=get
// +kubebuilder:rbac:groups="apps",resources=deployments;statefulsets,verbs=list;watch
// +kubebuilder:rbac:groups="coordination.k8s.io",namespace=keda,resources=leases,verbs=get;list;watch;update;patch;create;delete
// +kubebuilder:rbac:groups="",resources="limitranges",verbs=list;watch

// ScaledObjectReconciler reconciles a ScaledObject object
type ScaledObjectReconciler struct {
	Client       client.Client
	Scheme       *runtime.Scheme
	ScaleClient  scale.ScalesGetter
	ScaleHandler scaling.ScaleHandler
	EventEmitter eventemitter.EventHandler

	restMapper               meta.RESTMapper
	scaledObjectsGenerations *sync.Map
}

type scaledObjectMetricsData struct {
	namespace    string
	triggerTypes []string
}

var (
	// A cache mapping "resource.group" to true or false if we know if this resource is scalable.
	isScalableCache *sync.Map

	scaledObjectPromMetricsMap  map[string]scaledObjectMetricsData
	scaledObjectPromMetricsLock *sync.Mutex
)

func init() {
	// Prefill the cache with some known values for core resources in case of future parallelism to avoid stampeding herd on startup.
	isScalableCache = &sync.Map{}
	isScalableCache.Store("deployments.apps", true)
	isScalableCache.Store("statefulsets.apps", true)

	scaledObjectPromMetricsMap = make(map[string]scaledObjectMetricsData)
	scaledObjectPromMetricsLock = &sync.Mutex{}
}

// SetupWithManager initializes the ScaledObjectReconciler instance and starts a new controller managed by the passed Manager instance.
func (r *ScaledObjectReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	r.restMapper = mgr.GetRESTMapper()
	r.scaledObjectsGenerations = &sync.Map{}

	if r.ScaleHandler == nil {
		return fmt.Errorf("ScaledObjectReconciler.ScaleHandler is not initialized")
	}
	if r.Client == nil {
		return fmt.Errorf("ScaledObjectReconciler.Client is not initialized")
	}
	if r.ScaleClient == nil {
		return fmt.Errorf("ScaledObjectReconciler.ScaleClient is not initialized")
	}
	if r.Scheme == nil {
		return fmt.Errorf("ScaledObjectReconciler.Scheme is not initialized")
	}
	if r.EventEmitter == nil {
		return fmt.Errorf("ScaledObjectReconciler.EventEmitter is not initialized")
	}
	// Start controller
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		// predicate.GenerationChangedPredicate{} ignore updates to ScaledObject Status
		// (in this case metadata.Generation does not change)
		// so reconcile loop is not started on Status updates
		For(&kedav1alpha1.ScaledObject{}, builder.WithPredicates(
			predicate.Or(
				kedacontrollerutil.PausedPredicate{},
				kedacontrollerutil.PausedReplicasPredicate{},
				kedacontrollerutil.PausedScaleInPredicate{},
				kedacontrollerutil.PausedScaleOutPredicate{},
				kedacontrollerutil.ScaleObjectReadyConditionPredicate{},
				kedacontrollerutil.ForceActivationPredicate{},
				predicate.GenerationChangedPredicate{},
			),
		)).
		WithEventFilter(util.IgnoreOtherNamespaces()).
		// Trigger a reconcile only when the HPA spec,label or annotation changes.
		// Ignore updates to HPA status
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}, builder.WithPredicates(
			predicate.Or(
				predicate.LabelChangedPredicate{},
				predicate.AnnotationChangedPredicate{},
				kedacontrollerutil.HPASpecChangedPredicate{},
			))).
		Complete(r)
}

// Reconcile performs reconciliation on the identified ScaledObject resource based on the request information passed, returns the result and an error (if any).
func (r *ScaledObjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	// Fetch the ScaledObject instance
	scaledObject := &kedav1alpha1.ScaledObject{}
	err := r.Client.Get(ctx, req.NamespacedName, scaledObject)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "failed to get ScaledObject")
		return ctrl.Result{}, err
	}

	reqLogger.Info("Reconciling ScaledObject")

	// Check if the ScaledObject instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if scaledObject.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.finalizeScaledObject(ctx, reqLogger, scaledObject, req.String())
	}
	r.updatePromMetrics(scaledObject, req.String())

	// ensure finalizer is set on this CR
	if err := r.ensureFinalizer(ctx, reqLogger, scaledObject); err != nil {
		return ctrl.Result{}, err
	}

	// ensure Status Conditions are initialized
	if !scaledObject.Status.Conditions.AreInitialized() {
		conditions := kedav1alpha1.GetInitializedConditions()
		if err := kedastatus.SetStatusConditions(ctx, r.Client, reqLogger, scaledObject, conditions); err != nil {
			r.EventEmitter.Emit(scaledObject, req.Namespace, corev1.EventTypeWarning, eventingv1alpha1.ScaledObjectFailedType, eventreason.ScaledObjectUpdateFailed, err.Error())
			return ctrl.Result{}, err
		}
	}

	conditions := scaledObject.Status.Conditions.DeepCopy()
	// reconcile ScaledObject and set status appropriately
	msg, err := r.reconcileScaledObject(ctx, reqLogger, scaledObject, &conditions)
	if err != nil {
		reqLogger.Error(err, msg)
		fullErrMsg := fmt.Sprintf("%s: %s", msg, err.Error())
		conditions.SetReadyCondition(metav1.ConditionFalse, "ScaledObjectCheckFailed", fullErrMsg)
		conditions.SetActiveCondition(metav1.ConditionUnknown, "UnknownState", "ScaledObject check failed")
		r.EventEmitter.Emit(scaledObject, req.Namespace, corev1.EventTypeWarning, eventingv1alpha1.ScaledObjectFailedType, eventreason.ScaledObjectCheckFailed, fullErrMsg)
	} else {
		wasReady := conditions.GetReadyCondition()
		if wasReady.IsFalse() || wasReady.IsUnknown() {
			r.EventEmitter.Emit(scaledObject, req.Namespace, corev1.EventTypeNormal, eventingv1alpha1.ScaledObjectReadyType, eventreason.ScaledObjectReady, message.ScalerReadyMsg)
		}
		reqLogger.V(1).Info(msg)
		conditions.SetReadyCondition(metav1.ConditionTrue, kedav1alpha1.ScaledObjectConditionReadySuccessReason, msg)
	}

	if scaledObject.Spec.Fallback == nil || !fallback.HasValidFallback(scaledObject) {
		conditions.SetFallbackCondition(metav1.ConditionFalse, "NoFallbackFound", "No fallbacks are active on this scaled object")
	}

	metricscollector.RecordScaledObjectPaused(scaledObject.Namespace, scaledObject.Name, conditions.GetPausedCondition().Status == metav1.ConditionTrue)

	if err := kedastatus.SetStatusConditions(ctx, r.Client, reqLogger, scaledObject, &conditions); err != nil {
		r.EventEmitter.Emit(scaledObject, req.Namespace, corev1.EventTypeWarning, eventingv1alpha1.ScaledObjectFailedType, eventreason.ScaledObjectUpdateFailed, err.Error())
		return ctrl.Result{}, err
	}

	if _, err := r.updateTriggerAuthenticationStatus(ctx, reqLogger, scaledObject); err != nil {
		reqLogger.Error(err, "Failed to update TriggerAuthentication Status after removing a finalizer")
	}

	return ctrl.Result{}, err
}

// reconcileScaledObject implements reconciler logic for ScaledObject
func (r *ScaledObjectReconciler) reconcileScaledObject(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, conditions *kedav1alpha1.Conditions) (string, error) {
	// Check the presence of  the following annotations on the scaledObject:
	// - "autoscaling.keda.sh/paused"
	// - "autoscaling.keda.sh/paused-replicas"
	// and if so, stop the scale loop and delete the HPA on the scaledObject.
	// Additionally, if the following annotations are present:
	// - "autoscaling.keda.sh/paused-scale-in"
	// - "autoscaling.keda.sh/paused-scale-out"
	// we also set the status to paused but we allow the scale loop to continue and do not delete the HPA because these are unidirectional pauses.
	needsToPause := scaledObject.NeedToBePausedByAnnotation()
	switch {
	case needsToPause:
		scaledToPausedCount := true
		if conditions.GetPausedCondition().Status == metav1.ConditionTrue {
			// If scaledobject is in paused condition but replica count is not equal to paused replica count, the following scaling logic needs to be trigger again.
			scaledToPausedCount = r.checkIfTargetResourceReachPausedCount(ctx, logger, scaledObject)
			if scaledToPausedCount {
				return kedav1alpha1.ScaledObjectConditionReadySuccessMessage, nil
			}
		}
		if scaledToPausedCount {
			msg := kedav1alpha1.ScaledObjectConditionPausedMessage
			if err := r.stopScaleLoop(ctx, logger, scaledObject); err != nil {
				msg = "failed to stop the scale loop for paused ScaledObject"
				return msg, err
			}
			if deleted, err := r.ensureHPAForScaledObjectIsDeleted(ctx, logger, scaledObject); !deleted {
				msg = "failed to delete HPA for paused ScaledObject"
				return msg, err
			}
			conditions.SetPausedCondition(metav1.ConditionTrue, kedav1alpha1.ScaledObjectConditionPausedReason, msg)
			return msg, nil
		}
	case scaledObject.NeedToPauseScaleIn() || scaledObject.NeedToPauseScaleOut():
		conditions.SetPausedCondition(metav1.ConditionTrue, kedav1alpha1.ScaledObjectConditionPausedReason, kedav1alpha1.ScaledObjectConditionPausedMessage)
	case conditions.GetPausedCondition().Status == metav1.ConditionTrue:
		conditions.SetPausedCondition(metav1.ConditionFalse, "ScaledObjectUnpaused", "pause annotation removed for ScaledObject")
	}

	// Check scale target Name is specified
	if scaledObject.Spec.ScaleTargetRef.Name == "" {
		err := fmt.Errorf("ScaledObject.spec.scaleTargetRef.name is missing")
		return message.ScaleTargetErrMsg, err
	}

	// Check the label needed for Metrics servers is present on ScaledObject
	err := r.ensureScaledObjectLabel(ctx, logger, scaledObject)
	if err != nil {
		return "failed to update ScaledObject with scaledObjectName label", err
	}

	// Check if resource targeted for scaling exists and exposes /scale subresource
	gvkr, err := r.checkTargetResourceIsScalable(ctx, logger, scaledObject)
	if err != nil {
		return message.ScaleTargetErrMsg, err
	}

	err = kedav1alpha1.CheckReplicaCountBoundsAreValid(scaledObject)
	if err != nil {
		return "ScaledObject doesn't have correct Idle/Min/Max Replica Counts specification", err
	}

	err = kedav1alpha1.ValidateTriggers(scaledObject.Spec.Triggers)
	if err != nil {
		return "ScaledObject doesn't have correct triggers specification", err
	}

	err = r.updateStatusWithTriggersAndAuthsTypes(ctx, logger, scaledObject)
	if err != nil {
		return "Cannot update ScaledObject status with triggers'types and authentications'types", err
	}

	// Create a new HPA or update existing one according to ScaledObject
	newHPACreated, err := r.ensureHPAForScaledObjectExists(ctx, logger, scaledObject, &gvkr)
	if err != nil {
		return "failed to ensure HPA is correctly created for ScaledObject", err
	}
	scaleObjectSpecChanged := false
	if !newHPACreated {
		// Let's Check whether ScaledObject generation was changed, i.e. there were changes in ScaledObject.Spec
		// if it was changed we should start a new ScaleLoop
		// (we can omit this check if a new HPA was created, which fires new ScaleLoop anyway)
		scaleObjectSpecChanged, err = r.scaledObjectGenerationChanged(logger, scaledObject)
		if err != nil {
			return "failed to check whether ScaledObject's Generation was changed", err
		}
	}

	// Notify ScaleHandler if a new HPA was created or if ScaledObject was updated
	if newHPACreated || scaleObjectSpecChanged {
		if r.requestScaleLoop(ctx, logger, scaledObject) != nil {
			return "failed to start a new scale loop with scaling logic", err
		}
		logger.Info("Initializing Scaling logic according to ScaledObject Specification")
	}
	if scaledObject.NeedToBePausedByAnnotation() && conditions.GetPausedCondition().Status != metav1.ConditionTrue {
		return "ScaledObject paused replicas are being scaled", fmt.Errorf("ScaledObject paused replicas are being scaled")
	}
	return kedav1alpha1.ScaledObjectConditionReadySuccessMessage, nil
}

// ensureScaledObjectLabel ensures that scaledobject.keda.sh/name=<scaledObject.Name> label exist in the ScaledObject
// This is how the MetricsAdapter will know which ScaledObject a metric is for when the HPA queries it.
func (r *ScaledObjectReconciler) ensureScaledObjectLabel(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
	if scaledObject.Labels == nil {
		scaledObject.Labels = map[string]string{kedav1alpha1.ScaledObjectOwnerAnnotation: scaledObject.Name}
	} else {
		value, found := scaledObject.Labels[kedav1alpha1.ScaledObjectOwnerAnnotation]
		if found && value == scaledObject.Name {
			return nil
		}
		scaledObject.Labels[kedav1alpha1.ScaledObjectOwnerAnnotation] = scaledObject.Name
	}

	logger.V(1).Info("Adding \"scaledobject.keda.sh/name\" label on ScaledObject", "value", scaledObject.Name)
	return r.Client.Update(ctx, scaledObject)
}

func (r *ScaledObjectReconciler) checkIfTargetResourceReachPausedCount(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) bool {
	pausedReplicaCount, pausedReplicasAnnotationFound := scaledObject.GetAnnotations()[kedav1alpha1.PausedReplicasAnnotation]
	if !pausedReplicasAnnotationFound {
		return true
	}
	pausedReplicaCountNum, err := strconv.ParseInt(pausedReplicaCount, 10, 32)
	if err != nil {
		return true
	}

	gvkr, err := kedav1alpha1.ParseGVKR(r.restMapper, scaledObject.Spec.ScaleTargetRef.APIVersion, scaledObject.Spec.ScaleTargetRef.Kind)
	if err != nil {
		logger.Error(err, "failed to parse Group, Version, Kind, Resource", "apiVersion", scaledObject.Spec.ScaleTargetRef.APIVersion, "kind", scaledObject.Spec.ScaleTargetRef.Kind)
		return true
	}
	gvkString := gvkr.GVKString()
	logger.V(1).Info("Parsed Group, Version, Kind, Resource", "GVK", gvkString, "Resource", gvkr.Resource)

	// check if we already know.
	var scale *autoscalingv1.Scale
	gr := gvkr.GroupResource()
	scale, errScale := (r.ScaleClient).Scales(scaledObject.Namespace).Get(ctx, gr, scaledObject.Spec.ScaleTargetRef.Name, metav1.GetOptions{})
	if errScale != nil {
		return true
	}
	return scale.Spec.Replicas == int32(pausedReplicaCountNum)
}

// checkTargetResourceIsScalable checks if resource targeted for scaling exists and exposes /scale subresource
func (r *ScaledObjectReconciler) checkTargetResourceIsScalable(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) (kedav1alpha1.GroupVersionKindResource, error) {
	gvkr, err := kedav1alpha1.ParseGVKR(r.restMapper, scaledObject.Spec.ScaleTargetRef.APIVersion, scaledObject.Spec.ScaleTargetRef.Kind)
	if err != nil {
		msg := "Failed to parse Group, Version, Kind, Resource"
		logger.Error(err, msg, "apiVersion", scaledObject.Spec.ScaleTargetRef.APIVersion, "kind", scaledObject.Spec.ScaleTargetRef.Kind)
		r.EventEmitter.Emit(scaledObject, scaledObject.Namespace, corev1.EventTypeWarning, eventingv1alpha1.ScaledObjectFailedType, eventreason.ScaledObjectUpdateFailed, err.Error())
		return gvkr, err
	}
	gvkString := gvkr.GVKString()
	logger.V(1).Info("Parsed Group, Version, Kind, Resource", "GVK", gvkString, "Resource", gvkr.Resource)

	statusGvkString := ""
	if scaledObject.Status.ScaleTargetGVKR != nil {
		statusGvkr, _ := kedav1alpha1.ParseGVKR(r.restMapper, scaledObject.Status.ScaleTargetGVKR.Version, scaledObject.Status.ScaleTargetGVKR.Kind)
		statusGvkString = statusGvkr.GVKString()
		logger.V(1).Info("Status Group, Version, Kind, Resource", "GVK", statusGvkString, "Resource", statusGvkr.Resource)
	}

	// do we need the scale to update the status later?
	present := scaledObject.HasPausedAnnotation()
	removePausedStatus := scaledObject.Status.PausedReplicaCount != nil && !present
	wantStatusUpdate := scaledObject.Status.ScaleTargetKind != gvkString ||
		statusGvkString != gvkString ||
		scaledObject.Status.OriginalReplicaCount == nil ||
		removePausedStatus

	// check if we already know.
	var scale *autoscalingv1.Scale
	gr := gvkr.GroupResource()
	_, isScalable := isScalableCache.Load(gr.String())
	if !isScalable || wantStatusUpdate {
		// not cached, let's try to detect /scale subresource
		// also rechecks when we need to update the status.
		var errScale error
		scale, errScale = (r.ScaleClient).Scales(scaledObject.Namespace).Get(ctx, gr, scaledObject.Spec.ScaleTargetRef.Name, metav1.GetOptions{})
		if errScale != nil {
			// not able to get /scale subresource -> let's check if the resource even exist in the cluster
			unstruct := &unstructured.Unstructured{}
			unstruct.SetGroupVersionKind(gvkr.GroupVersionKind())
			if err := r.Client.Get(ctx, client.ObjectKey{Namespace: scaledObject.Namespace, Name: scaledObject.Spec.ScaleTargetRef.Name}, unstruct); err != nil {
				// resource doesn't exist
				logger.Error(err, message.ScaleTargetNotFoundMsg, "resource", gvkString, "name", scaledObject.Spec.ScaleTargetRef.Name)
				r.EventEmitter.Emit(scaledObject, scaledObject.Namespace, corev1.EventTypeWarning, eventingv1alpha1.ScaledObjectFailedType, eventreason.ScaledObjectCheckFailed, message.ScaleTargetNotFoundMsg)
				return gvkr, err
			}
			// resource exist but doesn't expose /scale subresource
			logger.Error(errScale, message.ScaleTargetNoSubresourceMsg, "resource", gvkString, "name", scaledObject.Spec.ScaleTargetRef.Name)
			r.EventEmitter.Emit(scaledObject, scaledObject.Namespace, corev1.EventTypeWarning, eventingv1alpha1.ScaledObjectFailedType, eventreason.ScaledObjectCheckFailed, message.ScaleTargetNoSubresourceMsg)
			return gvkr, errScale
		}
		isScalableCache.Store(gr.String(), true)
	}

	// if it is not already present in ScaledObject Status:
	// - store discovered GVK and GVKR
	// - store original scaleTarget's replica count (before scaling with KEDA)
	if wantStatusUpdate {
		status := scaledObject.Status.DeepCopy()
		if scaledObject.Status.ScaleTargetKind != gvkString || gvkString != statusGvkString {
			status.ScaleTargetKind = gvkString
			status.ScaleTargetGVKR = &gvkr
		}
		if scaledObject.Status.OriginalReplicaCount == nil {
			status.OriginalReplicaCount = &scale.Spec.Replicas
		}

		if removePausedStatus {
			status.PausedReplicaCount = nil
		}

		if err := kedastatus.UpdateScaledObjectStatus(ctx, r.Client, logger, scaledObject, status); err != nil {
			return gvkr, err
		}
		logger.Info("Detected resource targeted for scaling", "resource", gvkString, "name", scaledObject.Spec.ScaleTargetRef.Name)
	}

	return gvkr, nil
}

// ensureHPAForScaledObjectExists ensures that in cluster exist up-to-date HPA for specified ScaledObject, returns true if a new HPA was created
func (r *ScaledObjectReconciler) ensureHPAForScaledObjectExists(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, gvkr *kedav1alpha1.GroupVersionKindResource) (bool, error) {
	hpaName := getHPANameOnEnsure(scaledObject)
	foundHpa := &autoscalingv2.HorizontalPodAutoscaler{}
	// Check if HPA for this ScaledObject already exists
	err := r.Client.Get(ctx, types.NamespacedName{Name: hpaName, Namespace: scaledObject.Namespace}, foundHpa)
	if err != nil && errors.IsNotFound(err) {
		// HPA wasn't found -> let's create a new one
		err = r.createAndDeployNewHPA(ctx, logger, scaledObject, gvkr)
		if err != nil {
			return false, err
		}

		// new HPA created successfully -> notify Reconcile function so it could fire a new ScaleLoop
		return true, nil
	} else if err != nil {
		logger.Error(err, "failed to get HPA from cluster")
		return false, err
	}

	// check if hpa name is changed, and if so we need to delete the old hpa before creating new one
	if isHpaRenamed(scaledObject, foundHpa) {
		err = r.renameHPA(ctx, logger, scaledObject, foundHpa, gvkr)
		if err != nil {
			return false, err
		}
		// new HPA created successfully -> notify Reconcile function so it could fire a new ScaleLoop
		return true, nil
	}

	// HPA was found -> let's check if we need to update it
	err = r.updateHPAIfNeeded(ctx, logger, scaledObject, foundHpa, gvkr)
	if err != nil {
		logger.Error(err, "failed to check HPA for possible update")
		return false, err
	}

	// If the HPA name does not match the one in ScaledObject status, we need to update the status
	if scaledObject.Status.HpaName != hpaName {
		err = r.storeHpaNameInStatus(ctx, logger, scaledObject, hpaName)
		if err != nil {
			return false, err
		}
	}

	return false, nil
}

// ensureHPAForScaledObjectIsDeleted ensures that in cluster any HPA for specified ScaledObject is deleted, returns true if no HPA exists
func (r *ScaledObjectReconciler) ensureHPAForScaledObjectIsDeleted(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) (bool, error) {
	hpaName := getHPANameOnEnsure(scaledObject)
	foundHpa := &autoscalingv2.HorizontalPodAutoscaler{}
	// Check if HPA for this ScaledObject already exists
	err := r.Client.Get(ctx, types.NamespacedName{Name: hpaName, Namespace: scaledObject.Namespace}, foundHpa)
	if err != nil && errors.IsNotFound(err) {
		return true, nil
	} else if err != nil {
		logger.Error(err, "failed to get HPA from cluster")
		return false, err
	}

	if err := r.deleteHPA(ctx, logger, scaledObject, foundHpa); err != nil {
		logger.Error(err, "failed to delete HPA from cluster")
		return false, err
	}
	return true, nil
}

func getHPANameOnEnsure(scaledObject *kedav1alpha1.ScaledObject) string {
	if scaledObject.Status.HpaName != "" {
		return scaledObject.Status.HpaName
	}
	return getHPAName(scaledObject)
}

func isHpaRenamed(scaledObject *kedav1alpha1.ScaledObject, foundHpa *autoscalingv2.HorizontalPodAutoscaler) bool {
	// if HPA name defined in SO -> check if equals to the found HPA
	if scaledObject.Spec.Advanced != nil && scaledObject.Spec.Advanced.HorizontalPodAutoscalerConfig != nil && scaledObject.Spec.Advanced.HorizontalPodAutoscalerConfig.Name != "" {
		return scaledObject.Spec.Advanced.HorizontalPodAutoscalerConfig.Name != foundHpa.Name
	}
	// if HPA name not defined in SO -> check if the found HPA is equals to the default
	return foundHpa.Name != getDefaultHpaName(scaledObject)
}

// requestScaleLoop tries to start ScaleLoop handler for the respective ScaledObject
func (r *ScaledObjectReconciler) requestScaleLoop(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
	logger.V(1).Info("Notify scaleHandler of an update in scaledObject")

	key, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		logger.Error(err, "error getting key for scaledObject")
		return err
	}

	if err = r.ScaleHandler.HandleScalableObject(ctx, scaledObject); err != nil {
		return err
	}

	// store ScaledObject's current Generation
	r.scaledObjectsGenerations.Store(key, scaledObject.Generation)

	return nil
}

// stopScaleLoop stops ScaleLoop handler for the respective ScaledObject
func (r *ScaledObjectReconciler) stopScaleLoop(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
	key, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		logger.Error(err, "error getting key for scaledObject")
		return err
	}

	if err := r.ScaleHandler.DeleteScalableObject(ctx, scaledObject); err != nil {
		return err
	}
	// delete ScaledObject's current Generation
	r.scaledObjectsGenerations.Delete(key)
	return nil
}

// scaledObjectGenerationChanged returns true if ScaledObject's Generation was changed, ie. ScaledObject.Spec was changed
func (r *ScaledObjectReconciler) scaledObjectGenerationChanged(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) (bool, error) {
	key, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		logger.Error(err, "error getting key for scaledObject")
		return true, err
	}

	value, loaded := r.scaledObjectsGenerations.Load(key)
	if loaded {
		generation := value.(int64)
		if generation == scaledObject.Generation {
			return false, nil
		}
	}
	return true, nil
}

func (r *ScaledObjectReconciler) updatePromMetrics(scaledObject *kedav1alpha1.ScaledObject, namespacedName string) {
	scaledObjectPromMetricsLock.Lock()
	defer scaledObjectPromMetricsLock.Unlock()

	metricsData, ok := scaledObjectPromMetricsMap[namespacedName]

	if ok {
		metricscollector.DecrementCRDTotal(metricscollector.ScaledObjectResource, metricsData.namespace)
		for _, triggerType := range metricsData.triggerTypes {
			metricscollector.DecrementTriggerTotal(triggerType)
		}
	}

	metricscollector.IncrementCRDTotal(metricscollector.ScaledObjectResource, scaledObject.Namespace)
	metricsData.namespace = scaledObject.Namespace

	metricscollector.DeleteScalerMetrics(scaledObject.Namespace, scaledObject.Name, true)
	triggerTypes := make([]string, 0, len(scaledObject.Spec.Triggers))
	for _, trigger := range scaledObject.Spec.Triggers {
		metricscollector.IncrementTriggerTotal(trigger.Type)
		triggerTypes = append(triggerTypes, trigger.Type)
	}
	metricsData.triggerTypes = triggerTypes

	scaledObjectPromMetricsMap[namespacedName] = metricsData
}

func (r *ScaledObjectReconciler) updatePromMetricsOnDelete(namespacedName string) {
	scaledObjectPromMetricsLock.Lock()
	defer scaledObjectPromMetricsLock.Unlock()

	if metricsData, ok := scaledObjectPromMetricsMap[namespacedName]; ok {
		metricscollector.DecrementCRDTotal(metricscollector.ScaledObjectResource, metricsData.namespace)
		for _, triggerType := range metricsData.triggerTypes {
			metricscollector.DecrementTriggerTotal(triggerType)
		}
	}

	delete(scaledObjectPromMetricsMap, namespacedName)
}

func (r *ScaledObjectReconciler) updateTriggerAuthenticationStatus(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) (string, error) {
	return kedastatus.UpdateTriggerAuthenticationStatusFromTriggers(ctx, logger, r.Client, scaledObject.GetNamespace(), scaledObject.Spec.Triggers,
		func(triggerAuthenticationStatus *kedav1alpha1.TriggerAuthenticationStatus) *kedav1alpha1.TriggerAuthenticationStatus {
			triggerAuthenticationStatus.ScaledObjectNamesStr = kedacontrollerutil.AppendIntoString(triggerAuthenticationStatus.ScaledObjectNamesStr, scaledObject.GetName(), ",")
			return triggerAuthenticationStatus
		})
}

func (r *ScaledObjectReconciler) updateTriggerAuthenticationStatusOnDelete(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) (string, error) {
	return kedastatus.UpdateTriggerAuthenticationStatusFromTriggers(ctx, logger, r.Client, scaledObject.GetNamespace(), scaledObject.Spec.Triggers,
		func(triggerAuthenticationStatus *kedav1alpha1.TriggerAuthenticationStatus) *kedav1alpha1.TriggerAuthenticationStatus {
			triggerAuthenticationStatus.ScaledObjectNamesStr = kedacontrollerutil.RemoveFromString(triggerAuthenticationStatus.ScaledObjectNamesStr, scaledObject.GetName(), ",")
			return triggerAuthenticationStatus
		})
}

func (r *ScaledObjectReconciler) updateStatusWithTriggersAndAuthsTypes(ctx context.Context, logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
	triggersTypes, authsTypes := kedav1alpha1.CombinedTriggersAndAuthenticationsTypes(scaledObject.Spec.Triggers)
	status := scaledObject.Status.DeepCopy()
	status.TriggersTypes = &triggersTypes
	status.AuthenticationsTypes = &authsTypes

	logger.V(1).Info("Updating ScaledObject status with triggers and authentications types", "triggersTypes", triggersTypes, "authenticationsTypes", authsTypes)

	return kedastatus.UpdateScaledObjectStatus(ctx, r.Client, logger, scaledObject, status)
}
