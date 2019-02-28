package controller

import (
	"context"
	"sync"

	"github.com/Azure/Kore/pkg/scalers"

	kore_v1alpha1 "github.com/Azure/Kore/pkg/apis/kore/v1alpha1"
	clientset "github.com/Azure/Kore/pkg/client/clientset/versioned"
	koreinformer_v1alpha1 "github.com/Azure/Kore/pkg/client/informers/externalversions/kore/v1alpha1"
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
	opsLock               sync.Mutex
	ctx                   context.Context
	koreClient            clientset.Interface
	kubeClient            kubernetes.Interface
	scaleHandler          *scalers.ScaleHandler
}

func NewController(koreClient clientset.Interface, kubeClient kubernetes.Interface, scaleHandler *scalers.ScaleHandler) Controller {
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
		opsLock: sync.Mutex{},
	}

	c.scaledObjectsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.syncScaledObject,
		UpdateFunc: func(oldObj, newObj interface{}) {
			new := newObj.(*kore_v1alpha1.ScaledObject)
			old := oldObj.(*kore_v1alpha1.ScaledObject)
			if new.ResourceVersion == old.ResourceVersion {
				return
			}
			c.syncScaledObject(newObj)
		},
		DeleteFunc: c.syncDeletedScaledObject,
	})

	return c
}

//TODO: might need seperate method for updates to reconcile differences when removing/changing properties
func (c *controller) syncScaledObject(obj interface{}) {
	scaledObject := obj.(*kore_v1alpha1.ScaledObject)
	c.scaleHandler.WatchScaledObject(scaledObject)
}

func (c *controller) syncDeletedScaledObject(obj interface{}) {
	scaledObject := obj.(*kore_v1alpha1.ScaledObject)
	log.Infof("Notified about deletion of ScaledObject: %s", scaledObject.GetName())
	c.scaleHandler.StopWatchingScaledObject(scaledObject)
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
	go c.scaleHandler.Run(ctx.Done())
	c.scaledObjectsInformer.Run(ctx.Done())
	cancel()
}
