package scaling

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/scale"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	kedav1alpha1 "github.com/kedacore/keda/v2/api/v1alpha1"
	"github.com/kedacore/keda/v2/pkg/scalers"
	"github.com/kedacore/keda/v2/pkg/scaling/executor"
	"github.com/kedacore/keda/v2/pkg/scaling/resolver"
)

const (
	// Default polling interval for a ScaledObject triggers if no pollingInterval is defined.
	defaultPollingInterval = 30
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
}

// NewScaleHandler creates a ScaleHandler object
func NewScaleHandler(client client.Client, scaleClient *scale.ScalesGetter, reconcilerScheme *runtime.Scheme, globalHTTPTimeout time.Duration) ScaleHandler {
	return &scaleHandler{
		client:            client,
		logger:            logf.Log.WithName("scalehandler"),
		scaleLoopContexts: &sync.Map{},
		scaleExecutor:     executor.NewScaleExecutor(client, scaleClient, reconcilerScheme),
		globalHTTPTimeout: globalHTTPTimeout,
	}
}

func (h *scaleHandler) GetScalers(scalableObject interface{}) ([]scalers.Scaler, error) {
	withTriggers, err := asDuckWithTriggers(scalableObject)
	if err != nil {
		return nil, err
	}

	podTemplateSpec, containerName, err := h.getPods(scalableObject)
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

	key := generateKey(withTriggers)

	ctx, cancel := context.WithCancel(context.TODO())

	// cancel the outdated ScaleLoop for the same ScaledObject (if exists)
	value, loaded := h.scaleLoopContexts.LoadOrStore(key, cancel)
	if loaded {
		cancelValue, ok := value.(context.CancelFunc)
		if ok {
			cancelValue()
		}
		h.scaleLoopContexts.Store(key, cancel)
	}

	// a mutex is used to synchronize scale requests per scalableObject
	scalingMutex := &sync.Mutex{}
	go h.startPushScalers(ctx, withTriggers, scalableObject, scalingMutex)
	go h.startScaleLoop(ctx, withTriggers, scalableObject, scalingMutex)
	return nil
}

func (h *scaleHandler) DeleteScalableObject(scalableObject interface{}) error {
	withTriggers, err := asDuckWithTriggers(scalableObject)
	if err != nil {
		h.logger.Error(err, "error duck typing object into withTrigger")
		return err
	}

	key := generateKey(withTriggers)

	result, ok := h.scaleLoopContexts.Load(key)
	if ok {
		cancel, ok := result.(context.CancelFunc)
		if ok {
			cancel()
		}
		h.scaleLoopContexts.Delete(key)
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

	pollingInterval := getPollingInterval(withTriggers)
	logger.V(1).Info("Watching with pollingInterval", "PollingInterval", pollingInterval)

	for {
		select {
		case <-time.After(pollingInterval):
			h.checkScalers(ctx, scalableObject, scalingMutex)
		case <-ctx.Done():
			logger.V(1).Info("Context canceled")
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
			continue
		}

		go func() {
			activeCh := make(chan bool)
			go scaler.Run(ctx, activeCh)
			for {
				select {
				case <-ctx.Done():
					return
				case active := <-activeCh:
					scalingMutex.Lock()
					switch obj := scalableObject.(type) {
					case *kedav1alpha1.ScaledObject:
						h.scaleExecutor.RequestScale(ctx, obj, active)
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
		h.scaleExecutor.RequestScale(ctx, obj, h.checkScaledObjectScalers(ctx, scalers))
	case *kedav1alpha1.ScaledJob:
		scaledJob := scalableObject.(*kedav1alpha1.ScaledJob)
		isActive, scaleTo, maxScale := h.checkScaledJobScalers(ctx, scalers, scaledJob)
		h.scaleExecutor.RequestJobScale(ctx, obj, isActive, scaleTo, maxScale)
	}
}

func (h *scaleHandler) checkScaledObjectScalers(ctx context.Context, scalers []scalers.Scaler) bool {
	isActive := false
	for i, scaler := range scalers {
		isTriggerActive, err := scaler.IsActive(ctx)
		scaler.Close()

		if err != nil {
			h.logger.V(1).Info("Error getting scale decision", "Error", err)
			continue
		} else if isTriggerActive {
			isActive = true
			if scaler.GetMetricSpecForScaling()[0].External != nil {
				h.logger.V(1).Info("Scaler for scaledObject is active", "Metrics Name", scaler.GetMetricSpecForScaling()[0].External.Metric.Name)
			}
			if scaler.GetMetricSpecForScaling()[0].Resource != nil {
				h.logger.V(1).Info("Scaler for scaledObject is active", "Metrics Name", scaler.GetMetricSpecForScaling()[0].Resource.Name)
			}
			closeScalers(scalers[i+1:])
			break
		}
	}
	return isActive
}

func (h *scaleHandler) checkScaledJobScalers(ctx context.Context, scalers []scalers.Scaler, scaledJob *kedav1alpha1.ScaledJob) (bool, int64, int64) {
	var queueLength int64
	var targetAverageValue int64
	var maxValue int64
	isActive := false

	for _, scaler := range scalers {
		scalerLogger := h.logger.WithValues("Scaler", scaler)

		metricSpecs := scaler.GetMetricSpecForScaling()
		// skip cpu/memory resource scaler
		if metricSpecs[0].External == nil {
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
			continue
		} else if isTriggerActive {
			isActive = true
			scalerLogger.Info("Scaler is active")
		}
	}
	if targetAverageValue != 0 {
		maxValue = min(scaledJob.MaxReplicaCount(), devideWithCeil(queueLength, targetAverageValue))
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

func devideWithCeil(x, y int64) int64 {
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
		if podTemplateSpec != nil {
			authParams, podIdentity := resolver.ResolveAuthRef(h.client, logger, trigger.AuthenticationRef, &podTemplateSpec.Spec, withTriggers.Namespace)

			if podIdentity == kedav1alpha1.PodIdentityProviderAwsEKS {
				serviceAccountName := podTemplateSpec.Spec.ServiceAccountName
				serviceAccount := &corev1.ServiceAccount{}
				err = h.client.Get(context.TODO(), types.NamespacedName{Name: serviceAccountName, Namespace: withTriggers.Namespace}, serviceAccount)
				if err != nil {
					closeScalers(scalersRes)
					return []scalers.Scaler{}, fmt.Errorf("error getting service account: %s", err)
				}
				authParams["awsRoleArn"] = serviceAccount.Annotations[kedav1alpha1.PodIdentityAnnotationEKS]
			} else if podIdentity == kedav1alpha1.PodIdentityProviderAwsKiam {
				authParams["awsRoleArn"] = podTemplateSpec.ObjectMeta.Annotations[kedav1alpha1.PodIdentityAnnotationKiam]
			}
			config.AuthParams = authParams
			config.PodIdentity = podIdentity
		} else {
			authParams, _ := resolver.ResolveAuthRef(h.client, logger, trigger.AuthenticationRef, nil, withTriggers.Namespace)
			config.AuthParams = authParams
		}

		scaler, err := buildScaler(trigger.Type, config)
		if err != nil {
			closeScalers(scalersRes)
			return []scalers.Scaler{}, fmt.Errorf("error getting scaler for trigger #%d: %s", i, err)
		}

		scalersRes = append(scalersRes, scaler)
	}

	return scalersRes, nil
}

func (h *scaleHandler) getPods(scalableObject interface{}) (*corev1.PodTemplateSpec, string, error) {
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		unstruct := &unstructured.Unstructured{}
		unstruct.SetGroupVersionKind(obj.Status.ScaleTargetGVKR.GroupVersionKind())
		if err := h.client.Get(context.TODO(), client.ObjectKey{Namespace: obj.Namespace, Name: obj.Spec.ScaleTargetRef.Name}, unstruct); err != nil {
			// resource doesn't exist
			h.logger.Error(err, "Target resource doesn't exist", "resource", obj.Status.ScaleTargetGVKR.GVKString(), "name", obj.Spec.ScaleTargetRef.Name)
			return nil, "", err
		}

		withPods := &duckv1.WithPod{}
		if err := duck.FromUnstructured(unstruct, withPods); err != nil {
			h.logger.Error(err, "Cannot convert unstructured into PodSpecable Duck-type", "object", unstruct)
		}

		if withPods.Spec.Template.Spec.Containers == nil {
			h.logger.V(1).Info("There aren't any containers found in the ScaleTarget, therefore it is no possible to inject environment properties", "resource", obj.Status.ScaleTargetGVKR.GVKString(), "name", obj.Spec.ScaleTargetRef.Name)
			return nil, "", nil
		}

		podTemplateSpec := corev1.PodTemplateSpec{
			ObjectMeta: withPods.ObjectMeta,
			Spec:       withPods.Spec.Template.Spec,
		}
		return &podTemplateSpec, obj.Spec.ScaleTargetRef.EnvSourceContainerName, nil
	case *kedav1alpha1.ScaledJob:
		return &obj.Spec.JobTargetRef.Template, obj.Spec.EnvSourceContainerName, nil
	default:
		return nil, "", fmt.Errorf("unknown scalable object type %v", scalableObject)
	}
}

func buildScaler(triggerType string, config *scalers.ScalerConfig) (scalers.Scaler, error) {
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
	case "huawei-cloudeye":
		return scalers.NewHuaweiCloudeyeScaler(config)
	case "ibmmq":
		return scalers.NewIBMMQScaler(config)
	case "influxdb":
		return scalers.NewInfluxDBScaler(config)
	case "kafka":
		return scalers.NewKafkaScaler(config)
	case "liiklus":
		return scalers.NewLiiklusScaler(config)
	case "memory":
		return scalers.NewCPUMemoryScaler(corev1.ResourceMemory, config)
	case "metrics-api":
		return scalers.NewMetricsAPIScaler(config)
	case "mysql":
		return scalers.NewMySQLScaler(config)
	case "postgresql":
		return scalers.NewPostgreSQLScaler(config)
	case "prometheus":
		return scalers.NewPrometheusScaler(config)
	case "rabbitmq":
		return scalers.NewRabbitMQScaler(config)
	case "redis":
		return scalers.NewRedisScaler(config)
	case "redis-streams":
		return scalers.NewRedisStreamsScaler(config)
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

func getPollingInterval(withTriggers *kedav1alpha1.WithTriggers) time.Duration {
	if withTriggers.Spec.PollingInterval != nil {
		return time.Second * time.Duration(*withTriggers.Spec.PollingInterval)
	}

	return time.Second * time.Duration(defaultPollingInterval)
}

func generateKey(scalableObject *kedav1alpha1.WithTriggers) string {
	return fmt.Sprintf("%s.%s.%s", scalableObject.Kind, scalableObject.Namespace, scalableObject.Name)
}
