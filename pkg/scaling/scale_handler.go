package scaling

import (
	"context"
	"fmt"
	"github.com/kedacore/keda/pkg/scaling/resolver"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"knative.dev/pkg/apis/duck"
	"sync"
	"time"

	kedav1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	"github.com/kedacore/keda/pkg/scalers"
	"github.com/kedacore/keda/pkg/scaling/executor"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/scale"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
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
}

// NewScaleHandler creates a ScaleHandler object
func NewScaleHandler(client client.Client, scaleClient *scale.ScalesGetter, reconcilerScheme *runtime.Scheme) ScaleHandler {
	return &scaleHandler{
		client:            client,
		logger:            logf.Log.WithName("scalehandler"),
		scaleLoopContexts: &sync.Map{},
		scaleExecutor:     executor.NewScaleExecutor(client, scaleClient, reconcilerScheme),
	}
}

func (h *scaleHandler) GetScalers(scalableObject interface{}) ([]scalers.Scaler, error) {
	withTriggers, err := asDuckWithTriggers(scalableObject)
	if err != nil {
		return nil, err
	}

	withPods, containerName, err := h.getPods(scalableObject)
	if err != nil {
		return nil, err
	}

	return h.buildScalers(withTriggers, withPods, containerName)
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

	go h.startScaleLoop(ctx, withTriggers, scalableObject)
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
func (h *scaleHandler) startScaleLoop(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, scalableObject interface{}) {
	logger := h.logger.WithValues("namespace", withTriggers.GetNamespace(), "name", withTriggers.GetName())

	// kick off one check to the scalers now
	h.checkScalers(ctx, withTriggers, scalableObject)

	pollingInterval := getPollingInterval(withTriggers)
	logger.V(1).Info("Watching with pollingInterval", "PollingInterval", pollingInterval)

	for {
		select {
		case <-time.After(pollingInterval):
			h.checkScalers(ctx, withTriggers, scalableObject)
		case <-ctx.Done():
			logger.V(1).Info("Context canceled")
			return
		}
	}
}

// checkScalers contains the main logic for the ScaleHandler scaling logic.
// It'll check each trigger active status then call RequestScale
func (h *scaleHandler) checkScalers(ctx context.Context, withTriggers *kedav1alpha1.WithTriggers, scalableObject interface{}) {
	scalers, err := h.GetScalers(scalableObject)
	if err != nil {
		h.logger.Error(err, "Error getting scalers", "object", scalableObject)
		return
	}

	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		h.scaleExecutor.RequestScale(ctx, scalers, obj)
	case *kedav1alpha1.ScaledJob:
		h.scaleExecutor.RequestJobScale(ctx, scalers, obj)
	}
}

// GetScaledObjectScalers returns list of Scalers for the specified ScaledObject
func (h *scaleHandler) buildScalers(withTriggers *kedav1alpha1.WithTriggers, withPods *duckv1.WithPod, containerName string) ([]scalers.Scaler, error) {
	var scalersRes []scalers.Scaler
	logger := h.logger.WithValues("name", withTriggers.Name, "namespace", withTriggers.Namespace)

	resolvedEnv, err := resolver.ResolveContainerEnv(h.client, logger, &withPods.Spec.Template.Spec, containerName, withTriggers.Namespace)
	if err != nil {
		return scalersRes, fmt.Errorf("error resolving secrets for ScaleTarget: %s", err)
	}

	for i, trigger := range withTriggers.Spec.Triggers {
		authParams, podIdentity := resolver.ResolveAuthRef(h.client, logger, trigger.AuthenticationRef, &withPods.Spec.Template.Spec, withTriggers.Namespace)

		if podIdentity == kedav1alpha1.PodIdentityProviderAwsEKS {
			serviceAccountName := withPods.Spec.Template.Spec.ServiceAccountName
			serviceAccount := &corev1.ServiceAccount{}
			err = h.client.Get(context.TODO(), types.NamespacedName{Name: serviceAccountName, Namespace: withTriggers.Namespace}, serviceAccount)
			if err != nil {
				closeScalers(scalersRes)
				return []scalers.Scaler{}, fmt.Errorf("error getting service account: %s", err)
			}
			authParams["awsRoleArn"] = serviceAccount.Annotations[kedav1alpha1.PodIdentityAnnotationEKS]
		} else if podIdentity == kedav1alpha1.PodIdentityProviderAwsKiam {
			authParams["awsRoleArn"] = withPods.Spec.Template.ObjectMeta.Annotations[kedav1alpha1.PodIdentityAnnotationKiam]
		}

		scaler, err := buildScaler(withTriggers.Name, withTriggers.Namespace, trigger.Type, resolvedEnv, trigger.Metadata, authParams, podIdentity)
		if err != nil {
			closeScalers(scalersRes)
			return []scalers.Scaler{}, fmt.Errorf("error getting scaler for trigger #%d: %s", i, err)
		}

		scalersRes = append(scalersRes, scaler)
	}

	return scalersRes, nil
}

func (h *scaleHandler) getPods(scalableObject interface{}) (*duckv1.WithPod, string, error) {
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
			h.logger.Info("There aren't any containers in the ScaleTarget", "resource", obj.Status.ScaleTargetGVKR.GVKString(), "name", obj.Spec.ScaleTargetRef.Name)
			return nil, "", fmt.Errorf("no containers found")
		}

		return withPods, obj.Spec.ScaleTargetRef.ContainerName, nil
	}

	// TODO: implement this for ScaledJobs!!
	return nil, "", fmt.Errorf("resolvePods is only implemented for ScaledObjects so far")
}

func buildScaler(name, namespace, triggerType string, resolvedEnv, triggerMetadata, authParams map[string]string, podIdentity string) (scalers.Scaler, error) {
	switch triggerType {
	case "azure-queue":
		return scalers.NewAzureQueueScaler(resolvedEnv, triggerMetadata, authParams, podIdentity)
	case "azure-servicebus":
		return scalers.NewAzureServiceBusScaler(resolvedEnv, triggerMetadata, authParams, podIdentity)
	case "aws-sqs-queue":
		return scalers.NewAwsSqsQueueScaler(resolvedEnv, triggerMetadata, authParams)
	case "aws-cloudwatch":
		return scalers.NewAwsCloudwatchScaler(resolvedEnv, triggerMetadata, authParams)
	case "aws-kinesis-stream":
		return scalers.NewAwsKinesisStreamScaler(resolvedEnv, triggerMetadata, authParams)
	case "kafka":
		return scalers.NewKafkaScaler(resolvedEnv, triggerMetadata, authParams)
	case "rabbitmq":
		return scalers.NewRabbitMQScaler(resolvedEnv, triggerMetadata, authParams)
	case "azure-eventhub":
		return scalers.NewAzureEventHubScaler(resolvedEnv, triggerMetadata)
	case "prometheus":
		return scalers.NewPrometheusScaler(resolvedEnv, triggerMetadata)
	case "redis":
		return scalers.NewRedisScaler(resolvedEnv, triggerMetadata, authParams)
	case "gcp-pubsub":
		return scalers.NewPubSubScaler(resolvedEnv, triggerMetadata)
	case "external":
		return scalers.NewExternalScaler(name, namespace, resolvedEnv, triggerMetadata)
	case "liiklus":
		return scalers.NewLiiklusScaler(resolvedEnv, triggerMetadata)
	case "stan":
		return scalers.NewStanScaler(resolvedEnv, triggerMetadata)
	case "huawei-cloudeye":
		return scalers.NewHuaweiCloudeyeScaler(triggerMetadata, authParams)
	case "azure-blob":
		return scalers.NewAzureBlobScaler(resolvedEnv, triggerMetadata, authParams, podIdentity)
	case "postgresql":
		return scalers.NewPostgreSQLScaler(resolvedEnv, triggerMetadata, authParams)
	case "mysql":
		return scalers.NewMySQLScaler(resolvedEnv, triggerMetadata, authParams)
	case "azure-monitor":
		return scalers.NewAzureMonitorScaler(resolvedEnv, triggerMetadata, authParams)
	default:
		return nil, fmt.Errorf("no scaler found for type: %s", triggerType)
	}
}

func asDuckWithTriggers(scalableObject interface{}) (*kedav1alpha1.WithTriggers, error) {
	withTriggers := &kedav1alpha1.WithTriggers{}
	switch obj := scalableObject.(type) {
	case *kedav1alpha1.ScaledObject:
		withTriggers = &kedav1alpha1.WithTriggers{
			Spec: kedav1alpha1.WithTriggersSpec{
				PollingInterval: obj.Spec.PollingInterval,
				Triggers:        obj.Spec.Triggers,
			},
		}
	case *kedav1alpha1.ScaledJob:
		withTriggers = &kedav1alpha1.WithTriggers{
			Spec: kedav1alpha1.WithTriggersSpec{
				PollingInterval: obj.Spec.PollingInterval,
				Triggers:        obj.Spec.Triggers,
			},
		}
	default:
		// here could be the conversion from unknown Duck type potentially in the future
		return nil, fmt.Errorf("unknown scalable object type %v", scalableObject)
	}
	return withTriggers, nil
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
