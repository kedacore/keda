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

package scaling

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/eventreason"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scaling/cache"
	"github.com/kedacore/keda/v2/pkg/scaling/executor"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
)

// ScaleHandler encapsulates the logic of calling the right scalers for
// each ScaledObject and making the final scale decision and operation
type ScaleHandler interface {
	HandleScalableObject(ctx context.Context, scalableObject interface{}) error
	DeleteScalableObject(ctx context.Context, scalableObject interface{}) error
	GetScalersCache(ctx context.Context, scalableObject interface{}) (*cache.ScalersCache, error)
	ClearScalersCache(ctx context.Context, name, namespace string)
}

type scaleHandler struct {
	client            client.Client
	logger            logr.Logger
	scaleLoopContexts *sync.Map
	scaleExecutor     executor.ScaleExecutor
	globalHTTPTimeout time.Duration
	recorder          record.EventRecorder
	scalerCaches      map[string]*cache.ScalersCache
	lock              *sync.RWMutex
}

// NewScaleHandler creates a ScaleHandler object
func NewScaleHandler(client client.Client, scaleClient scale.ScalesGetter, reconcilerScheme *runtime.Scheme, globalHTTPTimeout time.Duration, recorder record.EventRecorder) ScaleHandler {
	return &scaleHandler{
		client:            client,
		logger:            logf.Log.WithName("scalehandler"),
		scaleLoopContexts: &sync.Map{},
		scaleExecutor:     executor.NewScaleExecutor(client, scaleClient, reconcilerScheme, recorder),
		globalHTTPTimeout: globalHTTPTimeout,
		recorder:          recorder,
		scalerCaches:      map[string]*cache.ScalersCache{},
		lock:              &sync.RWMutex{},
	}
}

func (h *scaleHandler) HandleScalableObject(ctx context.Context, scalableObject interface{}) error {
	withTriggers, err := asDuckWithTriggers(scalableObject)
	if err != nil {
		h.logger.Error(err, "error duck typing object into withTrigger")
		return err
	}

	key := withTriggers.GenerateIdenitifier()
	ctx, cancel := context.WithCancel(ctx)

	// cancel the outdated ScaleLoop for the same ScaledObject (if exists)
	value, loaded := h.scaleLoopContexts.LoadOrStore(key, cancel)
	if loaded {
		cancelValue, ok := value.(context.CancelFunc)
		if ok {
			cancelValue()
		}
		h.scaleLoopContexts.Store(key, cancel)
	} else {
		h.recorder.Event(withTriggers, corev1.EventTypeNormal, eventreason.KEDAScalersStarted, "Started scalers watch")
	}

	// a mutex is used to synchronize scale requests per scalableObject
	scalingMutex := &sync.Mutex{}

	// passing deep copy of ScaledObject/ScaledJob to the scaleLoop go routines, it's a precaution to not have global objects shared between threads
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		go h.startPushScalers(ctx, withTriggers, obj.DeepCopy(), scalingMutex)
		go h.startScaleLoop(ctx, withTriggers, obj.DeepCopy(), scalingMutex)
	case *kedav1alpha1.ScaledJob:
		go h.startPushScalers(ctx, withTriggers, obj.DeepCopy(), scalingMutex)
		go h.startScaleLoop(ctx, withTriggers, obj.DeepCopy(), scalingMutex)
	}
	return nil
}

func (h *scaleHandler) DeleteScalableObject(ctx context.Context, scalableObject interface{}) error {
	withTriggers, err := asDuckWithTriggers(scalableObject)
	if err != nil {
		h.logger.Error(err, "error duck typing object into withTrigger")
		return err
	}

	key := withTriggers.GenerateIdenitifier()
	result, ok := h.scaleLoopContexts.Load(key)
	if ok {
		cancel, ok := result.(context.CancelFunc)
		if ok {
			cancel()
		}
		h.scaleLoopContexts.Delete(key)
		h.recorder.Event(withTriggers, corev1.EventTypeNormal, eventreason.KEDAScalersStopped, "Stopped scalers watch")
	} else {
		h.logger.V(1).Info("ScaleObject was not found in controller cache", "key", key)
	}

	return nil
}

// startScaleLoop blocks forever and checks the scaledObject based on its pollingInterval
func (h *scaleHandler) startScaleLoop(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, scalableObject interface{}, scalingMutex sync.Locker) {
	logger := h.logger.WithValues("type", withTriggers.Kind, "namespace", withTriggers.Namespace, "name", withTriggers.Name)

	pollingInterval := withTriggers.GetPollingInterval()
	logger.V(1).Info("Watching with pollingInterval", "PollingInterval", pollingInterval)

	for {
		tmr := time.NewTimer(pollingInterval)
		h.checkScalers(ctx, scalableObject, scalingMutex)

		select {
		case <-tmr.C:
			tmr.Stop()
		case <-ctx.Done():
			logger.V(1).Info("Context canceled")
			h.ClearScalersCache(ctx, withTriggers.Name, withTriggers.Namespace)
			tmr.Stop()
			return
		}
	}
}

func (h *scaleHandler) GetScalersCache(ctx context.Context, scalableObject interface{}) (*cache.ScalersCache, error) {
	withTriggers, err := asDuckWithTriggers(scalableObject)
	if err != nil {
		return nil, err
	}

	key := strings.ToLower(fmt.Sprintf("%s.%s.%s", withTriggers.Kind, withTriggers.Name, withTriggers.Namespace))

	h.lock.RLock()
	if cache, ok := h.scalerCaches[key]; ok && cache.Generation == withTriggers.Generation {
		h.lock.RUnlock()
		return cache, nil
	}
	h.lock.RUnlock()

	h.lock.Lock()
	defer h.lock.Unlock()
	if cache, ok := h.scalerCaches[key]; ok && cache.Generation == withTriggers.Generation {
		return cache, nil
	} else if ok {
		cache.Close(ctx)
	}

	podTemplateSpec, containerName, err := resolver.ResolveScaleTargetPodSpec(ctx, h.client, h.logger, scalableObject)
	if err != nil {
		return nil, err
	}

	scalers := h.buildScalers(ctx, withTriggers, podTemplateSpec, containerName)

	h.scalerCaches[key] = &cache.ScalersCache{
		Generation: withTriggers.Generation,
		Scalers:    scalers,
		Logger:     h.logger,
		Recorder:   h.recorder,
	}

	return h.scalerCaches[key], nil
}

func (h *scaleHandler) ClearScalersCache(ctx context.Context, name, namespace string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	key := strings.ToLower(fmt.Sprintf("%s.%s", name, namespace))
	if cache, ok := h.scalerCaches[key]; ok {
		cache.Close(ctx)
		delete(h.scalerCaches, key)
	}
}

func (h *scaleHandler) startPushScalers(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, scalableObject interface{}, scalingMutex sync.Locker) {
	logger := h.logger.WithValues("type", withTriggers.Kind, "namespace", withTriggers.Namespace, "name", withTriggers.Name)
	cache, err := h.GetScalersCache(ctx, scalableObject)
	if err != nil {
		logger.Error(err, "Error getting scalers", "object", scalableObject)
		return
	}

	for _, ps := range cache.GetPushScalers() {
		go func(s scalers.PushScaler) {
			activeCh := make(chan bool)
			go s.Run(ctx, activeCh)
			defer s.Close(ctx)
			for {
				select {
				case <-ctx.Done():
					return
				case active := <-activeCh:
					scalingMutex.Lock()
					switch obj := scalableObject.(type) {
					case *kedav1alpha1.ScaledObject:
						h.scaleExecutor.RequestScale(ctx, obj, active, false)
					case *kedav1alpha1.ScaledJob:
						h.logger.Info("Warning: External Push Scaler does not support ScaledJob", "object", scalableObject)
					}
					scalingMutex.Unlock()
				}
			}
		}(ps)
	}
}

// checkScalers contains the main logic for the ScaleHandler scaling logic.
// It'll check each trigger active status then call RequestScale
func (h *scaleHandler) checkScalers(ctx context.Context, scalableObject interface{}, scalingMutex sync.Locker) {
	cache, err := h.GetScalersCache(ctx, scalableObject)
	if err != nil {
		h.logger.Error(err, "Error getting scalers", "object", scalableObject)
		return
	}

	scalingMutex.Lock()
	defer scalingMutex.Unlock()
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		err = h.client.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj)
		if err != nil {
			h.logger.Error(err, "Error getting scaledObject", "object", scalableObject)
			return
		}
		isActive, isError, _ := cache.IsScaledObjectActive(ctx, obj)
		h.scaleExecutor.RequestScale(ctx, obj, isActive, isError)
	case *kedav1alpha1.ScaledJob:
		err = h.client.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj)
		if err != nil {
			h.logger.Error(err, "Error getting scaledJob", "object", scalableObject)
			return
		}
		isActive, scaleTo, maxScale := cache.IsScaledJobActive(ctx, obj)
		h.scaleExecutor.RequestJobScale(ctx, obj, isActive, scaleTo, maxScale)
	}
}

// buildScalers returns list of Scalers for the specified triggers
func (h *scaleHandler) buildScalers(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, podTemplateSpec *corev1.PodTemplateSpec, containerName string) []cache.ScalerBuilder {
	logger := h.logger.WithValues("type", withTriggers.Kind, "namespace", withTriggers.Namespace, "name", withTriggers.Name)
	var err error
	resolvedEnv := make(map[string]string)
	result := make([]cache.ScalerBuilder, 0, len(withTriggers.Spec.Triggers))

	for scalerIndex, t := range withTriggers.Spec.Triggers {
		triggerName, trigger := scalerIndex, t
		factory := func() (scalers.Scaler, error) {
			if podTemplateSpec != nil {
				resolvedEnv, err = resolver.ResolveContainerEnv(ctx, h.client, logger, &podTemplateSpec.Spec, containerName, withTriggers.Namespace)
				if err != nil {
					return nil, fmt.Errorf("error resolving secrets for ScaleTarget: %s", err)
				}
			}
			config := &scalers.ScalerConfig{
				Name:              withTriggers.Name,
				Namespace:         withTriggers.Namespace,
				TriggerMetadata:   trigger.Metadata,
				ResolvedEnv:       resolvedEnv,
				AuthParams:        make(map[string]string),
				GlobalHTTPTimeout: h.globalHTTPTimeout,
				ScalerIndex:       scalerIndex,
			}

			config.AuthParams, config.PodIdentity, err = resolver.ResolveAuthRefAndPodIdentity(ctx, h.client, logger, trigger.AuthenticationRef, podTemplateSpec, withTriggers.Namespace)
			if err != nil {
				return nil, err
			}

			return buildScaler(ctx, h.client, trigger.Type, config)
		}

		scaler, err := factory()
		if err != nil {
			h.recorder.Event(withTriggers, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
			h.logger.Error(err, "error resolving auth params", "scalerIndex", scalerIndex, "object", withTriggers, "trigger", triggerName)
			if scaler != nil {
				scaler.Close(ctx)
			}
			continue
		}

		result = append(result, cache.ScalerBuilder{
			Scaler:  scaler,
			Factory: factory,
		})
	}

	return result
}

func buildScaler(ctx context.Context, client client.Client, triggerType string, config *scalers.ScalerConfig) (scalers.Scaler, error) {
	// TRIGGERS-START
	switch triggerType {
	case "artemis-queue":
		return scalers.NewArtemisQueueScaler(config)
	case "aws-cloudwatch":
		return scalers.NewAwsCloudwatchScaler(config)
	case "aws-kinesis-stream":
		return scalers.NewAwsKinesisStreamScaler(config)
	case "aws-sqs-queue":
		return scalers.NewAwsSqsQueueScaler(config)
	case "azure-blob":
		return scalers.NewAzureBlobScaler(config)
	case "azure-eventhub":
		return scalers.NewAzureEventHubScaler(config)
	case "azure-log-analytics":
		return scalers.NewAzureLogAnalyticsScaler(config)
	case "azure-monitor":
		return scalers.NewAzureMonitorScaler(config)
	case "azure-pipelines":
		return scalers.NewAzurePipelinesScaler(config)
	case "azure-queue":
		return scalers.NewAzureQueueScaler(config)
	case "azure-servicebus":
		return scalers.NewAzureServiceBusScaler(ctx, config)
	case "cassandra":
		return scalers.NewCassandraScaler(config)
	case "cpu":
		return scalers.NewCPUMemoryScaler(corev1.ResourceCPU, config)
	case "cron":
		return scalers.NewCronScaler(config)
	case "elasticsearch":
		return scalers.NewElasticsearchScaler(config)
	case "external":
		return scalers.NewExternalScaler(config)
	case "external-push":
		return scalers.NewExternalPushScaler(config)
	case "gcp-pubsub":
		return scalers.NewPubSubScaler(config)
	case "graphite":
		return scalers.NewGraphiteScaler(config)
	case "huawei-cloudeye":
		return scalers.NewHuaweiCloudeyeScaler(config)
	case "ibmmq":
		return scalers.NewIBMMQScaler(config)
	case "influxdb":
		return scalers.NewInfluxDBScaler(config)
	case "kafka":
		return scalers.NewKafkaScaler(config)
	case "kubernetes-workload":
		return scalers.NewKubernetesWorkloadScaler(client, config)
	case "liiklus":
		return scalers.NewLiiklusScaler(config)
	case "memory":
		return scalers.NewCPUMemoryScaler(corev1.ResourceMemory, config)
	case "metrics-api":
		return scalers.NewMetricsAPIScaler(config)
	case "mongodb":
		return scalers.NewMongoDBScaler(ctx, config)
	case "mssql":
		return scalers.NewMSSQLScaler(config)
	case "mysql":
		return scalers.NewMySQLScaler(config)
	case "openstack-metric":
		return scalers.NewOpenstackMetricScaler(ctx, config)
	case "openstack-swift":
		return scalers.NewOpenstackSwiftScaler(ctx, config)
	case "postgresql":
		return scalers.NewPostgreSQLScaler(config)
	case "predictkube":
		return scalers.NewPredictKubeScaler(ctx, config)
	case "prometheus":
		return scalers.NewPrometheusScaler(config)
	case "rabbitmq":
		return scalers.NewRabbitMQScaler(config)
	case "redis":
		return scalers.NewRedisScaler(ctx, false, false, config)
	case "redis-cluster":
		return scalers.NewRedisScaler(ctx, true, false, config)
	case "redis-cluster-streams":
		return scalers.NewRedisStreamsScaler(ctx, true, false, config)
	case "redis-sentinel":
		return scalers.NewRedisScaler(ctx, false, true, config)
	case "redis-sentinel-streams":
		return scalers.NewRedisStreamsScaler(ctx, false, true, config)
	case "redis-streams":
		return scalers.NewRedisStreamsScaler(ctx, false, false, config)
	case "selenium-grid":
		return scalers.NewSeleniumGridScaler(config)
	case "solace-event-queue":
		return scalers.NewSolaceScaler(config)
	case "stan":
		return scalers.NewStanScaler(config)
	default:
		return nil, fmt.Errorf("no scaler found for type: %s", triggerType)
	}
	// TRIGGERS-END
}

func asDuckWithTriggers(scalableObject interface{}) (*kedav1alpha1.WithTriggers, error) {
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		return &kedav1alpha1.WithTriggers{
			TypeMeta:   obj.TypeMeta,
			ObjectMeta: obj.ObjectMeta,
			Spec: kedav1alpha1.WithTriggersSpec{
				PollingInterval: obj.Spec.PollingInterval,
				Triggers:        obj.Spec.Triggers,
			},
		}, nil
	case *kedav1alpha1.ScaledJob:
		return &kedav1alpha1.WithTriggers{
			TypeMeta:   obj.TypeMeta,
			ObjectMeta: obj.ObjectMeta,
			Spec: kedav1alpha1.WithTriggersSpec{
				PollingInterval: obj.Spec.PollingInterval,
				Triggers:        obj.Spec.Triggers,
			},
		}, nil
	default:
		// here could be the conversion from unknown Duck type potentially in the future
		return nil, fmt.Errorf("unknown scalable object type %v", scalableObject)
	}
}
