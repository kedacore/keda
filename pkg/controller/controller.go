package controller

import (
	"context"
	"sync"

	kore_v1alpha1 "github.com/Azure/Kore/pkg/apis/kore/v1alpha1"
	clientset "github.com/Azure/Kore/pkg/client/clientset/versioned"
	koreinformer_v1alpha1 "github.com/Azure/Kore/pkg/client/informers/externalversions/kore/v1alpha1"
	"github.com/Azure/Kore/pkg/handler"
	log "github.com/Sirupsen/logrus"
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
	koreClient            clientset.Interface
	kubeClient            kubernetes.Interface
	scaleHandler          *handler.ScaleHandler
	scaledObjectsContexts *sync.Map
}

func NewController(koreClient clientset.Interface, kubeClient kubernetes.Interface, scaleHandler *handler.ScaleHandler) Controller {
	c := &controller{
		koreClient:   koreClient,
		kubeClient:   kubeClient,
		scaleHandler: scaleHandler,
		scaledObjectsInformer: koreinformer_v1alpha1.NewScaledObjectInformer(
			koreClient,
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
			new := newObj.(*kore_v1alpha1.ScaledObject)
			old := oldObj.(*kore_v1alpha1.ScaledObject)
			if new.ResourceVersion == old.ResourceVersion {
				return
			}
			c.syncScaledObject(newObj, true)
		},
		DeleteFunc: c.syncDeletedScaledObject,
	})
	return c
}

func (c *controller) syncScaledObject(obj interface{}, isUpdate bool) {
	scaledObject := obj.(*kore_v1alpha1.ScaledObject)
	key, err := cache.MetaNamespaceKeyFunc(scaledObject)
	if err != nil {
		log.Errorf("Error getting key for scaledObject (%s/%s)", scaledObject.GetNamespace(), scaledObject.GetName())
		return
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
	scaledObject := obj.(*kore_v1alpha1.ScaledObject)
	log.Debugf("Notified about deletion of ScaledObject: %s", scaledObject.GetName())

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
	cancel()
}
