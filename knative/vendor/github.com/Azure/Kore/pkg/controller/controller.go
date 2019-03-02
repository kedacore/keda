package controller

import (
	"context"
	"sync"
	"time"

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
			// https://groups.google.com/d/msg/kubernetes-sig-api-machinery/PbSCXdLDno0/dRLsMoLkDAAJ
			// Based on the discussion above, it seems that resyncPeriod can be
			// used for polling external systems like we have with autoscalers.
			// This however makes it not possible to have a custom check interval
			// per ScaledObject or deployment.
			time.Second*30,
			cache.Indexers{},
		),
		opsLock: sync.Mutex{},
	}

	c.scaledObjectsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.syncScaledObject,
		UpdateFunc: func(oldObj, newObj interface{}) {
			// always call syncScaledObject even on resync
			// this uses the informer cache for updates rather than maintaining another cache
			c.syncScaledObject(newObj)
		},
		DeleteFunc: c.syncDeletedScaledObject,
	})

	return c
}

//TODO: might need seperate method for updates to reconcile differences when removing/changing properties
func (c *controller) syncScaledObject(obj interface{}) {
	c.opsLock.Lock()
	defer c.opsLock.Unlock()

	scaledObject := obj.(*kore_v1alpha1.ScaledObject)

	go c.scaleHandler.HandleScale(scaledObject)
}

func (c *controller) syncDeletedScaledObject(obj interface{}) {
	scaledObject := obj.(*kore_v1alpha1.ScaledObject)
	log.Infof("Notified about deletion of ScaledObject: %s", scaledObject.GetName())
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
