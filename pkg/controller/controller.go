package controller

import (
	"context"
	"sync"

	kore_v1alpha1 "github.com/Azure/Kore/pkg/apis/kesc/v1alpha1"
	clientset "github.com/Azure/Kore/pkg/client/clientset/versioned"
	koreinformer_v1alpha1 "github.com/Azure/Kore/pkg/client/informers/externalversions/kesc/v1alpha1"
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
}

func NewController(koreClient clientset.Interface, kubeClient kubernetes.Interface) Controller {
	c := &controller{
		koreClient: koreClient,
		kubeClient: kubeClient,
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
	c.opsLock.Lock()
	defer c.opsLock.Unlock()

	scaledObject := obj.(*kore_v1alpha1.ScaledObject)
	deploymentName := scaledObject.Spec.DeploymentName
	if deploymentName == "" {
		log.Infof("Notified about ScaledObject with missing deployment name: %s", scaledObject.GetName())
		return
	}

	deployment, err := c.kubeClient.AppsV1().Deployments(scaledObject.GetNamespace()).Get(deploymentName, meta_v1.GetOptions{})
	if err != nil {
		log.Errorf("Error getting deployment: %s", err)
		return
	}

	log.Infof("Starting autoscalers for: %s. target deployment: %s", scaledObject.GetName(), deployment.GetName())
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
