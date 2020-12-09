package controllers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	kedacontrollerutil "github.com/kedacore/keda/v2/controllers/util"
	"github.com/kedacore/keda/v2/pkg/scaling"
	kedautil "github.com/kedacore/keda/v2/pkg/util"
)

// +kubebuilder:rbac:groups=keda.sh,resources=scaledobjects;scaledobjects/finalizers;scaledobjects/status,verbs="*"
// +kubebuilder:rbac:groups=keda.sh,resources=triggerauthentications;triggerauthentications/status,verbs="*"
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs="*"
// +kubebuilder:rbac:groups="",resources=configmaps;configmaps/status;events,verbs="*"
// +kubebuilder:rbac:groups="",resources=pods;services;services;secrets;external,verbs=get;list;watch
// +kubebuilder:rbac:groups="*",resources="*/scale",verbs="*"
// +kubebuilder:rbac:groups="*",resources="*",verbs=get

// ScaledObjectReconciler reconciles a ScaledObject object
type ScaledObjectReconciler struct {
	Log    logr.Logger
	Client client.Client
	Scheme *runtime.Scheme

	scaleClient              *scale.ScalesGetter
	restMapper               meta.RESTMapper
	scaledObjectsGenerations *sync.Map
	scaleHandler             scaling.ScaleHandler
	kubeVersion              kedautil.K8sVersion

	globalHTTPTimeout time.Duration
}

// SetupWithManager initializes the ScaledObjectReconciler instance and starts a new controller managed by the passed Manager instance.
func (r *ScaledObjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// create Discovery clientset
	clientset, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		r.Log.Error(err, "Not able to create Discovery clientset")
		return err
	}

	// Find out Kubernetes version
	version, err := clientset.ServerVersion()
	if err == nil {
		r.kubeVersion = kedautil.NewK8sVersion(version)
		r.Log.Info("Running on Kubernetes "+r.kubeVersion.PrettyVersion, "version", version)
	} else {
		r.Log.Error(err, "Not able to get Kubernetes version")
	}

	// Create Scale Client
	scaleClient := initScaleClient(mgr, clientset)
	r.scaleClient = &scaleClient

	// Init the rest of ScaledObjectReconciler
	r.restMapper = mgr.GetRESTMapper()
	r.scaledObjectsGenerations = &sync.Map{}
	r.scaleHandler = scaling.NewScaleHandler(mgr.GetClient(), r.scaleClient, mgr.GetScheme(), r.globalHTTPTimeout)

	// Start controller
	return ctrl.NewControllerManagedBy(mgr).
		// predicate.GenerationChangedPredicate{} ignore updates to ScaledObject Status
		// (in this case metadata.Generation does not change)
		// so reconcile loop is not started on Status updates
		For(&kedav1alpha1.ScaledObject{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&autoscalingv2beta2.HorizontalPodAutoscaler{}).
		Complete(r)
}

func initScaleClient(mgr manager.Manager, clientset *discovery.DiscoveryClient) scale.ScalesGetter {
	scaleKindResolver := scale.NewDiscoveryScaleKindResolver(clientset)
	return scale.New(
		clientset.RESTClient(), mgr.GetRESTMapper(),
		dynamic.LegacyAPIPathResolverFunc,
		scaleKindResolver,
	)
}

// Reconcile performs reconciliation on the identified ScaledObject resource based on the request information passed, returns the result and an error (if any).
func (r *ScaledObjectReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("ScaledObject.Namespace", req.Namespace, "ScaledObject.Name", req.Name)

	// Fetch the ScaledObject instance
	scaledObject := &kedav1alpha1.ScaledObject{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, scaledObject)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get ScaledObject")
		return ctrl.Result{}, err
	}

	reqLogger.Info("Reconciling ScaledObject")

	// Check if the ScaledObject instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if scaledObject.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, r.finalizeScaledObject(reqLogger, scaledObject)
	}

	// ensure finalizer is set on this CR
	if err := r.ensureFinalizer(reqLogger, scaledObject); err != nil {
		return ctrl.Result{}, err
	}

	// ensure Status Conditions are initialized
	if !scaledObject.Status.Conditions.AreInitialized() {
		conditions := kedav1alpha1.GetInitializedConditions()
		if err := kedacontrollerutil.SetStatusConditions(r.Client, reqLogger, scaledObject, conditions); err != nil {
			return ctrl.Result{}, err
		}
	}

	// reconcile ScaledObject and set status appropriately
	msg, err := r.reconcileScaledObject(reqLogger, scaledObject)
	conditions := scaledObject.Status.Conditions.DeepCopy()
	if err != nil {
		reqLogger.Error(err, msg)
		conditions.SetReadyCondition(metav1.ConditionFalse, "ScaledObjectCheckFailed", msg)
		conditions.SetActiveCondition(metav1.ConditionUnknown, "UnkownState", "ScaledObject check failed")
	} else {
		reqLogger.V(1).Info(msg)
		conditions.SetReadyCondition(metav1.ConditionTrue, "ScaledObjectReady", msg)
	}
	if err := kedacontrollerutil.SetStatusConditions(r.Client, reqLogger, scaledObject, &conditions); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, err
}

// reconcileScaledObject implements reconciler logic for ScaleObject
func (r *ScaledObjectReconciler) reconcileScaledObject(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) (string, error) {
	// Check scale target Name is specified
	if scaledObject.Spec.ScaleTargetRef.Name == "" {
		err := fmt.Errorf("ScaledObject.spec.scaleTargetRef.name is missing")
		return "ScaledObject doesn't have correct scaleTargetRef specification", err
	}

	// Check the label needed for Metrics servers is present on ScaledObject
	err := r.ensureScaledObjectLabel(logger, scaledObject)
	if err != nil {
		return "Failed to update ScaledObject with scaledObjectName label", err
	}

	// Check if resource targeted for scaling exists and exposes /scale subresource
	gvkr, err := r.checkTargetResourceIsScalable(logger, scaledObject)
	if err != nil {
		return "ScaledObject doesn't have correct scaleTargetRef specification", err
	}

	// Create a new HPA or update existing one according to ScaledObject
	newHPACreated, err := r.ensureHPAForScaledObjectExists(logger, scaledObject, &gvkr)
	if err != nil {
		return "Failed to ensure HPA is correctly created for ScaledObject", err
	}
	scaleObjectSpecChanged := false
	if !newHPACreated {
		// Lets Check whether ScaledObject generation was changed, ie. there were changes in ScaledObject.Spec
		// if it was changed we should start a new ScaleLoop
		// (we can omit this check if a new HPA was created, which fires new ScaleLoop anyway)
		scaleObjectSpecChanged, err = r.scaledObjectGenerationChanged(logger, scaledObject)
		if err != nil {
			return "Failed to check whether ScaledObject's Generation was changed", err
		}
	}

	// Notify ScaleHandler if a new HPA was created or if ScaledObject was updated
	if newHPACreated || scaleObjectSpecChanged {
		if r.requestScaleLoop(logger, scaledObject) != nil {
			return "Failed to start a new scale loop with scaling logic", err
		}
		logger.Info("Initializing Scaling logic according to ScaledObject Specification")
	}
	return "ScaledObject is defined correctly and is ready for scaling", nil
}

// ensureScaledObjectLabel ensures that scaledObjectName=<scaledObject.Name> label exist in the ScaledObject
// This is how the MetricsAdapter will know which ScaledObject a metric is for when the HPA queries it.
func (r *ScaledObjectReconciler) ensureScaledObjectLabel(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
	const labelScaledObjectName = "scaledObjectName"

	if scaledObject.Labels == nil {
		scaledObject.Labels = map[string]string{labelScaledObjectName: scaledObject.Name}
	} else {
		value, found := scaledObject.Labels[labelScaledObjectName]
		if found && value == scaledObject.Name {
			return nil
		}
		scaledObject.Labels[labelScaledObjectName] = scaledObject.Name
	}

	logger.V(1).Info("Adding scaledObjectName label on ScaledObject", "value", scaledObject.Name)
	return r.Client.Update(context.TODO(), scaledObject)
}

// checkTargetResourceIsScalable checks if resource targeted for scaling exists and exposes /scale subresource
func (r *ScaledObjectReconciler) checkTargetResourceIsScalable(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) (kedav1alpha1.GroupVersionKindResource, error) {
	gvkr, err := kedautil.ParseGVKR(r.restMapper, scaledObject.Spec.ScaleTargetRef.APIVersion, scaledObject.Spec.ScaleTargetRef.Kind)
	if err != nil {
		logger.Error(err, "Failed to parse Group, Version, Kind, Resource", "apiVersion", scaledObject.Spec.ScaleTargetRef.APIVersion, "kind", scaledObject.Spec.ScaleTargetRef.Kind)
		return gvkr, err
	}
	gvkString := gvkr.GVKString()
	logger.V(1).Info("Parsed Group, Version, Kind, Resource", "GVK", gvkString, "Resource", gvkr.Resource)

	// let's try to detect /scale subresource
	scale, errScale := (*r.scaleClient).Scales(scaledObject.Namespace).Get(context.TODO(), gvkr.GroupResource(), scaledObject.Spec.ScaleTargetRef.Name, metav1.GetOptions{})
	if errScale != nil {
		// not able to get /scale subresource -> let's check if the resource even exist in the cluster
		unstruct := &unstructured.Unstructured{}
		unstruct.SetGroupVersionKind(gvkr.GroupVersionKind())
		if err := r.Client.Get(context.TODO(), client.ObjectKey{Namespace: scaledObject.Namespace, Name: scaledObject.Spec.ScaleTargetRef.Name}, unstruct); err != nil {
			// resource doesn't exist
			logger.Error(err, "Target resource doesn't exist", "resource", gvkString, "name", scaledObject.Spec.ScaleTargetRef.Name)
			return gvkr, err
		}
		// resource exist but doesn't expose /scale subresource
		logger.Error(errScale, "Target resource doesn't expose /scale subresource", "resource", gvkString, "name", scaledObject.Spec.ScaleTargetRef.Name)
		return gvkr, errScale
	}

	// if it is not already present in ScaledObject Status:
	// - store discovered GVK and GVKR
	// - store original scaleTarget's replica count (before scaling with KEDA)
	if scaledObject.Status.ScaleTargetKind != gvkString || scaledObject.Status.OriginalReplicaCount == nil {
		status := scaledObject.Status.DeepCopy()
		if scaledObject.Status.ScaleTargetKind != gvkString {
			status.ScaleTargetKind = gvkString
			status.ScaleTargetGVKR = &gvkr
		}
		if scaledObject.Status.OriginalReplicaCount == nil {
			status.OriginalReplicaCount = &scale.Spec.Replicas
		}

		if err := kedacontrollerutil.UpdateScaledObjectStatus(r.Client, logger, scaledObject, status); err != nil {
			return gvkr, err
		}
		logger.Info("Detected resource targeted for scaling", "resource", gvkString, "name", scaledObject.Spec.ScaleTargetRef.Name)
	}

	return gvkr, nil
}

// ensureHPAForScaledObjectExists ensures that in cluster exist up-to-date HPA for specified ScaledObject, returns true if a new HPA was created
func (r *ScaledObjectReconciler) ensureHPAForScaledObjectExists(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, gvkr *kedav1alpha1.GroupVersionKindResource) (bool, error) {
	hpaName := getHPAName(scaledObject)
	foundHpa := &autoscalingv2beta2.HorizontalPodAutoscaler{}
	// Check if HPA for this ScaledObject already exists
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: hpaName, Namespace: scaledObject.Namespace}, foundHpa)
	if err != nil && errors.IsNotFound(err) {
		// HPA wasn't found -> let's create a new one
		err = r.createAndDeployNewHPA(logger, scaledObject, gvkr)
		if err != nil {
			return false, err
		}

		// check if scaledObject.spec.behavior was defined, because it is supported only on k8s >= 1.18
		r.checkMinK8sVersionforHPABehavior(logger, scaledObject)

		// new HPA created successfully -> notify Reconcile function so it could fire a new ScaleLoop
		return true, nil
	} else if err != nil {
		logger.Error(err, "Failed to get HPA from cluster")
		return false, err
	}

	// HPA was found -> let's check if we need to update it
	err = r.updateHPAIfNeeded(logger, scaledObject, foundHpa, gvkr)
	if err != nil {
		logger.Error(err, "Failed to check HPA for possible update")
		return false, err
	}

	return false, nil
}

// startScaleLoop starts ScaleLoop handler for the respective ScaledObject
func (r *ScaledObjectReconciler) requestScaleLoop(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
	logger.V(1).Info("Notify scaleHandler of an update in scaledObject")

	key, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		logger.Error(err, "Error getting key for scaledObject")
		return err
	}

	if err = r.scaleHandler.HandleScalableObject(scaledObject); err != nil {
		return err
	}

	// store ScaledObject's current Generation
	r.scaledObjectsGenerations.Store(key, scaledObject.Generation)

	return nil
}

// stopScaleLoop stops ScaleLoop handler for the respective ScaleObject
func (r *ScaledObjectReconciler) stopScaleLoop(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
	key, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		logger.Error(err, "Error getting key for scaledObject")
		return err
	}

	if err := r.scaleHandler.DeleteScalableObject(scaledObject); err != nil {
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
		logger.Error(err, "Error getting key for scaledObject")
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
