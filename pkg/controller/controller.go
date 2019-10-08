package controller

import (
	"context"
	"sync"

	keda_v1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	clientset "github.com/kedacore/keda/pkg/client/clientset/versioned"
	kedainformer_v1alpha1 "github.com/kedacore/keda/pkg/client/informers/externalversions/keda/v1alpha1"
	"github.com/kedacore/keda/pkg/handler"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/equality"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Controller interface {
	Run(ctx context.Context)
}

type controller struct {
	scaledObjectsInformer cache.SharedInformer
	ctx                   context.Context
	kedaClient            clientset.Interface
	kubeClient            kubernetes.Interface
	scaleHandler          *handler.ScaleHandler
	scaledObjectsContexts *sync.Map
}

func NewController(kedaClient clientset.Interface, kubeClient kubernetes.Interface, scaleHandler *handler.ScaleHandler) Controller {
	c := &controller{
		kedaClient:   kedaClient,
		kubeClient:   kubeClient,
		scaleHandler: scaleHandler,
		scaledObjectsInformer: kedainformer_v1alpha1.NewScaledObjectInformer(
			kedaClient,
			meta_v1.NamespaceAll,
			0,
			cache.Indexers{},
		),
		scaledObjectsContexts: &sync.Map{},
	}

	c.scaledObjectsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.syncScaledObject(obj, false)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			new := newObj.(*keda_v1alpha1.ScaledObject)
			old := oldObj.(*keda_v1alpha1.ScaledObject)
			if new.ResourceVersion == old.ResourceVersion {
				return
			}
			if equality.Semantic.DeepEqual(old.Spec, new.Spec) {
				return
			}

			c.syncScaledObject(newObj, true)
		},
		DeleteFunc: c.syncDeletedScaledObject,
	})
	return c
}

func (c *controller) syncScaledObject(obj interface{}, isUpdate bool) {
	scaledObject := obj.(*keda_v1alpha1.ScaledObject)

	log.Printf("Detecting ScaleType from ScaledObject")
	if scaledObject.Spec.ScaleTargetRef.DeploymentName == "" {
		log.Printf("Detected ScaleType = Job")
		scaledObject.Spec.ScaleType = keda_v1alpha1.ScaleTypeJob
	} else {
		log.Printf("Detected ScaleType = Deployment")
		scaledObject.Spec.ScaleType = keda_v1alpha1.ScaleTypeDeployment
	}
	key, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		log.Errorf("Error getting key for scaledObject (%s/%s)", scaledObject.GetNamespace(), scaledObject.GetName())
		return
	}

	if !isUpdate {
		log.Infof("Watching ScaledObject: %s", key)
	}

	ctx, cancel := context.WithCancel(c.ctx)

	value, loaded := c.scaledObjectsContexts.LoadOrStore(key, cancel)
	if loaded {
		cancelValue, ok := value.(context.CancelFunc)
		if ok {
			cancelValue()
		}
		c.scaledObjectsContexts.Store(key, cancel)
	}
	// Tell the handler if this is an Update call to ScaledObject
	// to avoid status update/check loop.
	c.scaleHandler.WatchScaledObjectWithContext(ctx, scaledObject, !isUpdate)
}

func (c *controller) syncDeletedScaledObject(obj interface{}) {
	scaledObject := obj.(*keda_v1alpha1.ScaledObject)
	if scaledObject == nil {
		log.Errorf("Called syncDeletedScaledObject with an invalid scaledObject ptr")
		return
	}

	log.Debugf("Notified about deletion of ScaledObject: %s", scaledObject.GetName())
	go c.scaleHandler.HandleScaledObjectDelete(scaledObject)

	key, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		log.Errorf("Error getting key for scaledObject (%s/%s)", scaledObject.GetNamespace(), scaledObject.GetName())
		return
	}

	result, ok := c.scaledObjectsContexts.Load(key)
	if ok {
		cancel, ok := result.(context.CancelFunc)
		if ok {
			cancel()
		}
		c.scaledObjectsContexts.Delete(key)
	} else {
		log.Debugf("ScaledObject %s not found in controller cache", key)
	}
}

func (c *controller) Run(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	c.ctx = ctx
	go func() {
		<-ctx.Done()
		log.Infof("Controller is shutting down")
	}()
	log.Infof("Controller is started")
	c.scaledObjectsInformer.Run(ctx.Done())
}
