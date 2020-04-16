package scaledobject

import (
	"context"
	"fmt"
	"sync"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	"github.com/kedacore/keda/pkg/scaling"
	kedautil "github.com/kedacore/keda/pkg/util"

	"github.com/go-logr/logr"
	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_scaledobject")

// Add creates a new ScaledObject Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	// Create Scale Client
	scaleClient, err := initScaleClient(mgr)
	if err != nil {
		return err
	}
	return add(mgr, newReconciler(mgr, &scaleClient))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, scaleClient *scale.ScalesGetter) reconcile.Reconciler {
	return &ReconcileScaledObject{
		client:                   mgr.GetClient(),
		scaleClient:              scaleClient,
		restMapper:               mgr.GetRESTMapper(),
		scheme:                   mgr.GetScheme(),
		scaledObjectsGenerations: &sync.Map{},
		scaleHandler:             scaling.NewScaleHandler(mgr.GetClient(), scaleClient, mgr.GetScheme()),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("scaledobject-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ScaledObject
	err = c.Watch(&source.Kind{Type: &kedav1alpha1.ScaledObject{}},
		&handler.EnqueueRequestForObject{},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// Ignore updates to ScaledObject Status (in this case metadata.Generation does not change)
				// so reconcile loop is not started on Status updates
				return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
			},
		})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource HPA and requeue the owner ScaledObject
	err = c.Watch(&source.Kind{Type: &autoscalingv2beta2.HorizontalPodAutoscaler{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kedav1alpha1.ScaledObject{},
	})
	if err != nil {
		return err
	}
	return nil
}

func initScaleClient(mgr manager.Manager) (scale.ScalesGetter, error) {
	clientset, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	scaleKindResolver := scale.NewDiscoveryScaleKindResolver(clientset)
	return scale.New(
		clientset.RESTClient(), mgr.GetRESTMapper(),
		dynamic.LegacyAPIPathResolverFunc,
		scaleKindResolver,
	), nil
}

// blank assignment to verify that ReconcileScaledObject implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileScaledObject{}

// ReconcileScaledObject reconciles a ScaledObject object
type ReconcileScaledObject struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client                   client.Client
	scaleClient              *scale.ScalesGetter
	restMapper               meta.RESTMapper
	scheme                   *runtime.Scheme
	scaledObjectsGenerations *sync.Map
	scaleHandler             scaling.ScaleHandler
}

// Reconcile reads that state of the cluster for a ScaledObject object and makes changes based on the state read
// and what is in the ScaledObject.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileScaledObject) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling ScaledObject")

	// Fetch the ScaledObject instance
	scaledObject := &kedav1alpha1.ScaledObject{}
	err := r.client.Get(context.TODO(), request.NamespacedName, scaledObject)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get ScaledObject")
		return reconcile.Result{}, err
	}

	// Check if the ScaledObject instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if scaledObject.GetDeletionTimestamp() != nil {
		return reconcile.Result{}, r.finalizeScaledObject(reqLogger, scaledObject)
	}

	// ensure finalizer is set on this CR
	if err := r.ensureFinalizer(reqLogger, scaledObject); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, r.reconcileScaledObject(reqLogger, scaledObject)
}

// reconcileScaledObject implements reconciler logic for ScaleObject
func (r *ReconcileScaledObject) reconcileScaledObject(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {

	// Check scale target Name is specified
	if scaledObject.Spec.ScaleTargetRef.Name == "" {
		err := fmt.Errorf("ScaledObject.spec.scaleTargetRef.name is missing")
		logger.Error(err, "Notified about ScaledObject with incorrect scaleTargetRef specification")
		return err
	}

	// Check the label needed for Metrics servers is present on ScaledObject
	err := r.ensureScaledObjectLabel(logger, scaledObject)
	if err != nil {
		logger.Error(err, "Failed to update ScaledObject with scaledObjectName label")
		return err
	}

	// Check if resource targeted for scaling exists and exposes /scale subresource
	gvkr, err := r.checkTargetResourceIsScalable(logger, scaledObject)
	if err != nil {
		logger.Error(err, "Notified about ScaledObject with incorrect scaleTargetRef specification")
		return err
	}

	// Create a new HPA or update existing one according to ScaledObject
	newHPACreated, err := r.ensureHPAForScaledObjectExists(logger, scaledObject, &gvkr)
	if err != nil {
		logger.Error(err, "Failed to reconcile HPA for ScaledObject")
		return err
	}
	scaleObjectSpecChanged := false
	if !newHPACreated {
		// Lets Check whether ScaledObject generation was changed, ie. there were changes in ScaledObject.Spec
		// if it was changed we should start a new ScaleLoop
		// (we can omit this check if a new HPA was created, which fires new ScaleLoop anyway)
		scaleObjectSpecChanged, err = r.scaledObjectGenerationChanged(logger, scaledObject)
		if err != nil {
			logger.Error(err, "Failed to check whether ScaledObject's Generation was changed")
			return err
		}
	}

	// Notify ScaleHandler if a new HPA was created or if ScaledObject was updated
	if newHPACreated || scaleObjectSpecChanged {
		if r.requestScaleLoop(logger, scaledObject) != nil {
			logger.Error(err, "Failed to start a new ScaleLoop")
			return err
		}
	}

	return nil
}

// ensureScaledObjectLabel ensures that scaledObjectName=<scaledObject.Name> label exist in the ScaledObject
// This is how the MetricsAdapter will know which ScaledObject a metric is for when the HPA queries it.
func (r *ReconcileScaledObject) ensureScaledObjectLabel(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
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
	return r.client.Update(context.TODO(), scaledObject)
}

// checkTargetResourceIsScalable checks if resource targeted for scaling exists and exposes /scale subresource
func (r *ReconcileScaledObject) checkTargetResourceIsScalable(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) (kedautil.GroupVersionKindResource, error) {
	gvkr, err := kedautil.ParseGVKR(r.restMapper, scaledObject.Spec.ScaleTargetRef.ApiVersion, scaledObject.Spec.ScaleTargetRef.Kind)
	if err != nil {
		logger.Error(err, "Failed to parse Group, Version, Kind, Resource", "apiVersion", scaledObject.Spec.ScaleTargetRef.ApiVersion, "kind", scaledObject.Spec.ScaleTargetRef.Kind)
		return gvkr, err
	}
	logger.V(1).Info("Parsed Group, Version, Kind, Resource", "GVK", gvkr.GVKString(), "Resource", gvkr.Resource)

	// let's try to detect /scale subresource
	_, errScale := (*r.scaleClient).Scales(scaledObject.Namespace).Get(gvkr.GroupResource(), scaledObject.Spec.ScaleTargetRef.Name)
	if errScale != nil {
		// not able to get /scale subresource -> let's check if the resource even exist in the cluster
		unstruct := &unstructured.Unstructured{}
		unstruct.SetGroupVersionKind(gvkr.GroupVersionKind())
		if err := r.client.Get(context.TODO(), client.ObjectKey{Namespace: scaledObject.Namespace, Name: scaledObject.Spec.ScaleTargetRef.Name}, unstruct); err != nil {
			// resource doesn't exist
			logger.Error(err, "Target resource doesn't exist", "resource", gvkr.GVKString(), "name", scaledObject.Spec.ScaleTargetRef.Name)
			return gvkr, err
		} else {
			// resource exist but doesn't expose /scale subresource
			logger.Error(errScale, "Target resource doesn't expose /scale subresource", "resource", gvkr.GVKString(), "name", scaledObject.Spec.ScaleTargetRef.Name)
			return gvkr, errScale
		}
	}

	// store discovered GVK and GVKR into the Status
	scaledObject.Status.ScaleTargetKind = gvkr.GVKString()
	scaledObject.Status.ScaleTargetGVKR = &gvkr
	if err = r.client.Status().Update(context.TODO(), scaledObject); err != nil {
		logger.Error(err, "Failed to update ScaledObject.Status", "scaledObject.Status.ScaleTargetKind", gvkr.GVKString(), "scaledObject.Status.ScaleTargetGVKR", gvkr)
	}

	logger.Info("Detected resource targeted for scaling", "resource", gvkr.GVKString(), "name", scaledObject.Spec.ScaleTargetRef.Name)
	return gvkr, nil
}

// ensureHPAForScaledObjectExists ensures that in cluster exist up-to-date HPA for specified ScaledObject, returns true if a new HPA was created
func (r *ReconcileScaledObject) ensureHPAForScaledObjectExists(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, gvkr *kedautil.GroupVersionKindResource) (bool, error) {
	hpaName := getHPAName(scaledObject)
	foundHpa := &autoscalingv2beta2.HorizontalPodAutoscaler{}
	// Check if HPA for this ScaledObject already exists
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: hpaName, Namespace: scaledObject.Namespace}, foundHpa)
	if err != nil && errors.IsNotFound(err) {
		// HPA wasn't found -> let's create a new one
		err = r.createAndDeployNewHPA(logger, scaledObject, gvkr)
		if err != nil {
			return false, err
		}
		// new HPA created successfully -> notify Reconcile function so it could fire a new ScaleLoop
		return true, nil
	} else if err != nil {
		logger.Error(err, "Failed to get HPA")
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
func (r *ReconcileScaledObject) requestScaleLoop(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
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
func (r *ReconcileScaledObject) stopScaleLoop(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) error {
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
func (r *ReconcileScaledObject) scaledObjectGenerationChanged(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) (bool, error) {
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
