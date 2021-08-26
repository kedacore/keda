package scaling

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kedacore/keda/v2/pkg/eventreason"
	"k8s.io/client-go/tools/record"

	"github.com/go-logr/logr"
	"k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/scale"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scaling/executor"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
)

// ScaleHandler encapsulates the logic of calling the right scalers for
// each ScaledObject and making the final scale decision and operation
type ScaleHandler interface {
	HandleScalableObject(scalableObject interface{}) error
	DeleteScalableObject(scalableObject interface{}) error
	GetScalers(scalableObject interface{}) ([]scalers.Scaler, error)
}

type scaleHandler struct {
	client            client.Client
	logger            logr.Logger
	scaleLoopContexts *sync.Map
	scaleExecutor     executor.ScaleExecutor
	globalHTTPTimeout time.Duration
	recorder          record.EventRecorder
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
	}
}

func (h *scaleHandler) GetScalers(scalableObject interface{}) ([]scalers.Scaler, error) {
	withTriggers, err := asDuckWithTriggers(scalableObject)
	if err != nil {
		return nil, err
	}

	podTemplateSpec, containerName, err := resolver.ResolveScaleTargetPodSpec(h.client, h.logger, scalableObject)
	if err != nil {
		return nil, err
	}

	return h.buildScalers(withTriggers, podTemplateSpec, containerName)
}

func (h *scaleHandler) HandleScalableObject(scalableObject interface{}) error {
	withTriggers, err := asDuckWithTriggers(scalableObject)
	if err != nil {
		h.logger.Error(err, "error duck typing object into withTrigger")
		return err
	}

	key := withTriggers.GenerateIdenitifier()
	ctx, cancel := context.WithCancel(context.TODO())

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

func (h *scaleHandler) DeleteScalableObject(scalableObject interface{}) error {
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

	// kick off one check to the scalers now
	h.checkScalers(ctx, scalableObject, scalingMutex)

	pollingInterval := withTriggers.GetPollingInterval()
	logger.V(1).Info("Watching with pollingInterval", "PollingInterval", pollingInterval)

	for {
		tmr := time.NewTimer(pollingInterval)

		select {
		case <-tmr.C:
			h.checkScalers(ctx, scalableObject, scalingMutex)
			tmr.Stop()
		case <-ctx.Done():
			logger.V(1).Info("Context canceled")
			tmr.Stop()
			return
		}
	}
}

func (h *scaleHandler) startPushScalers(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, scalableObject interface{}, scalingMutex sync.Locker) {
	logger := h.logger.WithValues("type", withTriggers.Kind, "namespace", withTriggers.Namespace, "name", withTriggers.Name)
	ss, err := h.GetScalers(scalableObject)
	if err != nil {
		logger.Error(err, "Error getting scalers", "object", scalableObject)
		return
	}

	for _, s := range ss {
		scaler, ok := s.(scalers.PushScaler)
		if !ok {
			s.Close()
			continue
		}

		go func() {
			activeCh := make(chan bool)
			go scaler.Run(ctx, activeCh)
			defer scaler.Close()
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
		}()
	}
}

// checkScalers contains the main logic for the ScaleHandler scaling logic.
// It'll check each trigger active status then call RequestScale
func (h *scaleHandler) checkScalers(ctx context.Context, scalableObject interface{}, scalingMutex sync.Locker) {
	scalers, err := h.GetScalers(scalableObject)
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
		isActive, isError := h.isScaledObjectActive(ctx, scalers, obj)
		h.scaleExecutor.RequestScale(ctx, obj, isActive, isError)
	case *kedav1alpha1.ScaledJob:
		err = h.client.Get(ctx, types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, obj)
		if err != nil {
			h.logger.Error(err, "Error getting scaledJob", "object", scalableObject)
			return
		}
		isActive, scaleTo, maxScale := h.isScaledJobActive(ctx, scalers, obj)
		h.scaleExecutor.RequestJobScale(ctx, obj, isActive, scaleTo, maxScale)
	}
}

func (h *scaleHandler) isScaledObjectActive(ctx context.Context, scalers []scalers.Scaler, scaledObject *kedav1alpha1.ScaledObject) (bool, bool) {
	isActive := false
	isError := false
	for i, scaler := range scalers {
		isTriggerActive, err := scaler.IsActive(ctx)
		scaler.Close()

		if err != nil {
			h.logger.V(1).Info("Error getting scale decision", "Error", err)
			isError = true
			h.recorder.Event(scaledObject, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
			continue
		} else if isTriggerActive {
			isActive = true
			if externalMetricsSpec := scaler.GetMetricSpecForScaling()[0].External; externalMetricsSpec != nil {
				h.logger.V(1).Info("Scaler for scaledObject is active", "Metrics Name", externalMetricsSpec.Metric.Name)
			}
			if resourceMetricsSpec := scaler.GetMetricSpecForScaling()[0].Resource; resourceMetricsSpec != nil {
				h.logger.V(1).Info("Scaler for scaledObject is active", "Metrics Name", resourceMetricsSpec.Name)
			}
			closeScalers(scalers[i+1:])
			break
		}
	}
	return isActive, isError
}

func (h *scaleHandler) isScaledJobActive(ctx context.Context, scalers []scalers.Scaler, scaledJob *kedav1alpha1.ScaledJob) (bool, int64, int64) {
	var queueLength int64
	var targetAverageValue int64
	var maxValue int64
	isActive := false

	// TODO refactor this, do chores, reduce the verbosity ie: V(1) and frequency of logs
	// move relevant funcs getTargetAverageValue(), min() and divideWithCeil() out of scaler_handler.go
	for _, scaler := range scalers {
		scalerLogger := h.logger.WithValues("Scaler", scaler)

		metricSpecs := scaler.GetMetricSpecForScaling()

		// skip scaler that doesn't return any metric specs (usually External scaler with incorrect metadata)
		// or skip cpu/memory resource scaler
		if len(metricSpecs) < 1 || metricSpecs[0].External == nil {
			continue
		}

		isTriggerActive, err := scaler.IsActive(ctx)

		scalerLogger.Info("Active trigger", "isTriggerActive", isTriggerActive)

		targetAverageValue = getTargetAverageValue(metricSpecs)

		scalerLogger.Info("Scaler targetAverageValue", "targetAverageValue", targetAverageValue)

		metrics, _ := scaler.GetMetrics(ctx, "queueLength", nil)

		var metricValue int64

		for _, m := range metrics {
			if m.MetricName == "queueLength" {
				metricValue, _ = m.Value.AsInt64()
				queueLength += metricValue
			}
		}
		scalerLogger.Info("QueueLength Metric value", "queueLength", queueLength)

		scaler.Close()
		if err != nil {
			scalerLogger.V(1).Info("Error getting scale decision, but continue", "Error", err)
			h.recorder.Event(scaledJob, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
			continue
		} else if isTriggerActive {
			isActive = true
			scalerLogger.Info("Scaler is active")
		}
	}
	if targetAverageValue != 0 {
		maxValue = min(scaledJob.MaxReplicaCount(), divideWithCeil(queueLength, targetAverageValue))
	}
	h.logger.Info("Scaler maxValue", "maxValue", maxValue)
	return isActive, queueLength, maxValue
}

func getTargetAverageValue(metricSpecs []v2beta2.MetricSpec) int64 {
	var targetAverageValue int64
	var metricValue int64
	var flag bool
	for _, metric := range metricSpecs {
		if metric.External.Target.AverageValue == nil {
			metricValue = 0
		} else {
			metricValue, flag = metric.External.Target.AverageValue.AsInt64()
			if !flag {
				metricValue = 0
			}
		}

		targetAverageValue += metricValue
	}
	count := int64(len(metricSpecs))
	if count != 0 {
		return targetAverageValue / count
	}
	return 0
}

func divideWithCeil(x, y int64) int64 {
	ans := x / y
	reminder := x % y
	if reminder != 0 {
		return ans + 1
	}
	return ans
}

// Min function for int64
func min(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}

// buildScalers returns list of Scalers for the specified triggers
func (h *scaleHandler) buildScalers(withTriggers *kedav1alpha1.WithTriggers, podTemplateSpec *corev1.PodTemplateSpec, containerName string) ([]scalers.Scaler, error) {
	logger := h.logger.WithValues("type", withTriggers.Kind, "namespace", withTriggers.Namespace, "name", withTriggers.Name)
	var scalersRes []scalers.Scaler
	var err error
	resolvedEnv := make(map[string]string)
	if podTemplateSpec != nil {
		resolvedEnv, err = resolver.ResolveContainerEnv(h.client, logger, &podTemplateSpec.Spec, containerName, withTriggers.Namespace)
		if err != nil {
			return scalersRes, fmt.Errorf("error resolving secrets for ScaleTarget: %s", err)
		}
	}

	for i, trigger := range withTriggers.Spec.Triggers {
		config := &scalers.ScalerConfig{
			Name:              withTriggers.Name,
			Namespace:         withTriggers.Namespace,
			TriggerMetadata:   trigger.Metadata,
			ResolvedEnv:       resolvedEnv,
			AuthParams:        make(map[string]string),
			GlobalHTTPTimeout: h.globalHTTPTimeout,
		}

		config.AuthParams, config.PodIdentity, err = resolver.ResolveAuthRefAndPodIdentity(h.client, logger, trigger.AuthenticationRef, podTemplateSpec, withTriggers.Namespace)
		if err != nil {
			closeScalers(scalersRes)
			return []scalers.Scaler{}, err
		}

		scaler, err := buildScaler(h.client, trigger.Type, config)
		if err != nil {
			closeScalers(scalersRes)
			h.recorder.Event(withTriggers, corev1.EventTypeWarning, eventreason.KEDAScalerFailed, err.Error())
			return []scalers.Scaler{}, fmt.Errorf("error getting scaler for trigger #%d: %s", i, err)
		}

		scalersRes = append(scalersRes, scaler)
	}

	return scalersRes, nil
}

func buildScaler(client client.Client, triggerType string, config *scalers.ScalerConfig) (scalers.Scaler, error) {
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
		return scalers.NewAzureServiceBusScaler(config)
	case "cpu":
		return scalers.NewCPUMemoryScaler(corev1.ResourceCPU, config)
	case "cron":
		return scalers.NewCronScaler(config)
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
		return scalers.NewMongoDBScaler(config)
	case "mssql":
		return scalers.NewMSSQLScaler(config)
	case "mysql":
		return scalers.NewMySQLScaler(config)
	case "openstack-metric":
		return scalers.NewOpenstackMetricScaler(config)
	case "openstack-swift":
		return scalers.NewOpenstackSwiftScaler(config)
	case "postgresql":
		return scalers.NewPostgreSQLScaler(config)
	case "prometheus":
		return scalers.NewPrometheusScaler(config)
	case "rabbitmq":
		return scalers.NewRabbitMQScaler(config)
	case "redis":
		return scalers.NewRedisScaler(false, config)
	case "redis-cluster":
		return scalers.NewRedisScaler(true, config)
	case "redis-cluster-streams":
		return scalers.NewRedisStreamsScaler(true, config)
	case "redis-streams":
		return scalers.NewRedisStreamsScaler(false, config)
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

func closeScalers(scalers []scalers.Scaler) {
	for _, scaler := range scalers {
		defer scaler.Close()
	}
}
