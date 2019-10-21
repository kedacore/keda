package scaledobject

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/go-logr/logr"
	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	scalehandler "github.com/kedacore/keda/pkg/handler"

	autoscalingv2beta1 "k8s.io/api/autoscaling/v2beta1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	defaultHPAMinReplicas int32 = 1
	defaultHPAMaxReplicas int32 = 100
)

var log = logf.Log.WithName("controller_scaledobject")

// Add creates a new ScaledObject Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileScaledObject{client: mgr.GetClient(), scheme: mgr.GetScheme(), scaleLoopContexts: &sync.Map{}}
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
	return nil
}

// blank assignment to verify that ReconcileScaledObject implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileScaledObject{}

// ReconcileScaledObject reconciles a ScaledObject object
type ReconcileScaledObject struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client            client.Client
	scheme            *runtime.Scheme
	scaleLoopContexts *sync.Map
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
	isScaledObjectMarkedToBeDeleted := scaledObject.GetDeletionTimestamp() != nil
	if isScaledObjectMarkedToBeDeleted {
		if contains(scaledObject.GetFinalizers(), scaledObjectFinalizer) {
			// Run finalization logic for scaledObjectFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeScaledObject(reqLogger, scaledObject); err != nil {
				return reconcile.Result{}, err
			}

			// Remove scaledObjectFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			scaledObject.SetFinalizers(remove(scaledObject.GetFinalizers(), scaledObjectFinalizer))
			err := r.client.Update(context.TODO(), scaledObject)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(scaledObject.GetFinalizers(), scaledObjectFinalizer) {
		if err := r.addFinalizer(reqLogger, scaledObject); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	reqLogger.Info("Detecting ScaleType from ScaledObject")
	if scaledObject.Spec.ScaleTargetRef.DeploymentName == "" {
		reqLogger.Info("Detected ScaleType = Job")
		return r.reconcileJobType(reqLogger, scaledObject)
	} else {
		reqLogger.Info("Detected ScaleType = Deployment")
		return r.reconcileDeploymentType(reqLogger, scaledObject)
	}
}

// reconcileJobType implemets reconciler logic for K8s Jobs based ScaleObject
func (r *ReconcileScaledObject) reconcileJobType(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) (reconcile.Result, error) {
	scaledObject.Spec.ScaleType = kedav1alpha1.ScaleTypeJob

	// Delete Jobs owned by the previous version of the ScaledObject
	opts := []client.ListOption{
		client.InNamespace(scaledObject.GetNamespace()),
		client.MatchingLabels(map[string]string{"scaledobject": scaledObject.GetName()}),
	}
	jobs := &batchv1.JobList{}
	err := r.client.List(context.TODO(), jobs, opts...)
	if err != nil {
		logger.Error(err, "Cannot get list of Jobs owned by this ScaledObject")
		return reconcile.Result{}, err
	}

	if jobs.Size() > 0 {
		logger.Info("Deleting jobs owned by the previous version of the ScaledObject", "Number of jobs to delete", jobs.Size())
	}
	for _, job := range jobs.Items {
		err = r.client.Delete(context.TODO(), &job, client.PropagationPolicy(metav1.DeletePropagationBackground))
		if err != nil {
			logger.Error(err, "Not able to delete job", "Job", job.Name)
			return reconcile.Result{}, err
		}
	}

	// ScaledObject was created or modified - let's start a new ScaleLoop
	r.startScaleLoop(logger, scaledObject)

	return reconcile.Result{}, nil
}

// reconcileDeploymentType implements reconciler logic for Deployment based ScaleObject
func (r *ReconcileScaledObject) reconcileDeploymentType(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) (reconcile.Result, error) {
	scaledObject.Spec.ScaleType = kedav1alpha1.ScaleTypeDeployment

	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	if deploymentName == "" {
		err := fmt.Errorf("Notified about ScaledObject with missing deployment name")
		logger.Error(err, "Notified about ScaledObject with missing deployment")
		return reconcile.Result{}, err
	}

	hpaName := getHpaName(deploymentName)
	hpaNamespace := scaledObject.Namespace

	// Check if this HPA already exists
	foundHpa := &autoscalingv2beta1.HorizontalPodAutoscaler{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: hpaName, Namespace: hpaNamespace}, foundHpa)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new HPA", "HPA.Namespace", hpaNamespace, "HPA.Name", hpaName)
		hpa, err := r.newHPAForScaledObject(logger, scaledObject)
		if err != nil {
			logger.Error(err, "Failed to create new HPA resource", "HPA.Namespace", hpaNamespace, "HPA.Name", hpaName)
			return reconcile.Result{}, err
		}

		// Set ScaledObject instance as the owner and controller
		if err := controllerutil.SetControllerReference(scaledObject, hpa, r.scheme); err != nil {
			return reconcile.Result{}, err
		}

		err = r.client.Create(context.TODO(), hpa)
		if err != nil {
			logger.Error(err, "Failed to create new HPA in cluster", "HPA.Namespace", hpaNamespace, "HPA.Name", hpaName)
			return reconcile.Result{}, err
		}

		// ScaledObject was created - let's start a new ScaleLoop
		r.startScaleLoop(logger, scaledObject)

		// HPA created successfully & ScaleLoop started - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get HPA")
		return reconcile.Result{}, err
	}

	// Check whether update of HPA is needed
	updateHPA := false
	scaledObjectMinReplicaCount := getHpaMinReplicas(scaledObject)
	if foundHpa.Spec.MinReplicas != scaledObjectMinReplicaCount {
		updateHPA = true
		foundHpa.Spec.MinReplicas = scaledObjectMinReplicaCount
	}

	scaledObjectMaxReplicaCount := getHpaMaxReplicas(scaledObject)
	if foundHpa.Spec.MaxReplicas != scaledObjectMaxReplicaCount {
		updateHPA = true
		foundHpa.Spec.MaxReplicas = scaledObjectMaxReplicaCount
	}

	newMetricSpec, err := r.getScaledObjectMetricSpecs(logger, scaledObject, deploymentName)
	if err != nil {
		logger.Error(err, "Failed to create MetricSpec")
		return reconcile.Result{}, err
	}
	if !reflect.DeepEqual(foundHpa.Spec.Metrics, newMetricSpec) {
		updateHPA = true
		foundHpa.Spec.Metrics = newMetricSpec
	}

	if updateHPA {
		err = r.client.Update(context.TODO(), foundHpa)
		if err != nil {
			logger.Error(err, "Failed to update HPA", "HPA.Namespace", foundHpa.Namespace, "HPA.Name", foundHpa.Name)
			return reconcile.Result{}, err
		}
		logger.Info("Updated HPA according to ScaledObject", "HPA.Namespace", hpaNamespace, "HPA.Name", hpaName)
	}

	// ScaledObject was modified - let's start a new ScaleLoop
	r.startScaleLoop(logger, scaledObject)

	return reconcile.Result{}, nil
}

// startScaleLoop starts ScaleLoop handler for the respective ScaledObject
func (r *ReconcileScaledObject) startScaleLoop(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) {

	scaleHandler := scalehandler.NewScaleHandler(r.client, r.scheme)

	key, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		logger.Error(err, "Error getting key for scaledObject")
		return
	}

	ctx, cancel := context.WithCancel(context.TODO())

	// cancel the outdated ScaleLoop for the same ScaledObject (if exists)
	value, loaded := r.scaleLoopContexts.LoadOrStore(key, cancel)
	if loaded {
		cancelValue, ok := value.(context.CancelFunc)
		if ok {
			cancelValue()
		}
		r.scaleLoopContexts.Store(key, cancel)
	}
	go scaleHandler.HandleScaleLoop(ctx, scaledObject)
}

// newHPAForScaledObject returns HPA as it is specified in ScaledObject
func (r *ReconcileScaledObject) newHPAForScaledObject(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject) (*autoscalingv2beta1.HorizontalPodAutoscaler, error) {
	deploymentName := scaledObject.Spec.ScaleTargetRef.DeploymentName
	scaledObjectMetricSpecs, err := r.getScaledObjectMetricSpecs(logger, scaledObject, deploymentName)

	if err != nil {
		return nil, err
	}

	return &autoscalingv2beta1.HorizontalPodAutoscaler{
		Spec: autoscalingv2beta1.HorizontalPodAutoscalerSpec{
			MinReplicas: getHpaMinReplicas(scaledObject),
			MaxReplicas: getHpaMaxReplicas(scaledObject),
			Metrics:     scaledObjectMetricSpecs,
			ScaleTargetRef: autoscalingv2beta1.CrossVersionObjectReference{
				Name:       deploymentName,
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			}},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getHpaName(deploymentName),
			Namespace: scaledObject.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v2beta1",
		},
	}, nil
}

// getScaledObjectMetricSpecs returns MetricSpec for HPA, generater from Triggers defitinion in ScaledObject
func (r *ReconcileScaledObject) getScaledObjectMetricSpecs(logger logr.Logger, scaledObject *kedav1alpha1.ScaledObject, deploymentName string) ([]autoscalingv2beta1.MetricSpec, error) {
	var scaledObjectMetricSpecs []autoscalingv2beta1.MetricSpec
	var externalMetricNames []string

	scalers, _, err := scalehandler.NewScaleHandler(r.client, r.scheme).GetDeploymentScalers(scaledObject)
	if err != nil {
		logger.Error(err, "Error getting scalers")
		return nil, err
	}

	for _, scaler := range scalers {
		metricSpecs := scaler.GetMetricSpecForScaling()

		// add the deploymentName label. This is how the MetricsAdapter will know which scaledobject a metric is for when the HPA queries it.
		for _, metricSpec := range metricSpecs {
			metricSpec.External.MetricSelector = &metav1.LabelSelector{MatchLabels: make(map[string]string)}
			metricSpec.External.MetricSelector.MatchLabels["deploymentName"] = deploymentName
			externalMetricNames = append(externalMetricNames, metricSpec.External.MetricName)
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

// getHpaName returns generated HPA name for DeploymentName specified in the parameter
func getHpaName(deploymentName string) string {
	return fmt.Sprintf("keda-hpa-%s", deploymentName)
}

// getHpaMinReplicas returns MinReplicas based on definition in ScaledObject or default value if not defined
func getHpaMinReplicas(scaledObject *kedav1alpha1.ScaledObject) *int32 {
	if scaledObject.Spec.MinReplicaCount != nil && *scaledObject.Spec.MinReplicaCount > 0 {
		return scaledObject.Spec.MinReplicaCount
	}
	tmp := defaultHPAMinReplicas
	return &tmp
}

// getHpaMaxReplicas returns MaxReplicas based on definition in ScaledObject or default value if not defined
func getHpaMaxReplicas(scaledObject *kedav1alpha1.ScaledObject) int32 {
	if scaledObject.Spec.MaxReplicaCount != nil {
		return *scaledObject.Spec.MaxReplicaCount
	}
	return defaultHPAMaxReplicas
}
