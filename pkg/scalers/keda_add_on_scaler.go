package scalers

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pb "github.com/kedacore/keda/v2/pkg/scalers/externalscaler"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

// AddOnResource represents the status of a KEDA add-on resource and its metadata used by the add-on scaler.
type AddOnResource struct {
	Status *struct {
		AddOnMetadata *struct {
			ServerAddress string            `json:"serverAddress"`
			Metadata      map[string]string `json:"metadata"`
			UsePushScaler bool              `json:"usePushScaler"`
		} `json:"addOnMetadata"`
	} `json:"status"`
}

type kedaAddOnScalerMetadata struct {
	triggerIndex int

	Name       string `keda:"name=name, order=triggerMetadata"`
	Kind       string `keda:"name=kind, order=triggerMetadata"`
	APIVersion string `keda:"name=apiVersion, order=triggerMetadata"`
}

// NewKedaAddOnScaler creates a new Keda Add-On Scaler
func NewKedaAddOnScaler(ctx context.Context, kubeClient client.Client, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting add-on scaler metric type: %w", err)
	}

	meta := &kedaAddOnScalerMetadata{
		triggerIndex: config.TriggerIndex,
	}
	err = config.TypedConfig(meta)
	if err != nil {
		return nil, fmt.Errorf("error parsing add-on metadata: %w", err)
	}

	gv, err := schema.ParseGroupVersion(meta.APIVersion)
	if err != nil {
		return nil, fmt.Errorf("error parsing add-on group version: %w", err)
	}

	unstruct := &unstructured.Unstructured{}
	unstruct.SetGroupVersionKind(gv.WithKind(meta.Kind))

	if err := kubeClient.Get(ctx, client.ObjectKey{Namespace: config.ScalableObjectNamespace, Name: meta.Name}, unstruct); err != nil {
		return nil, fmt.Errorf("target resource %s %s/%s not found: %w", unstruct.GroupVersionKind().String(), config.ScalableObjectNamespace, meta.Name, err)
	}
	addOnResource := &AddOnResource{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.Object, addOnResource); err != nil {
		return nil, fmt.Errorf("cannot convert Unstructured into add-on custom resource %s %s %s/%s: %w", meta.APIVersion, meta.Kind, config.ScalableObjectNamespace, meta.Name, err)
	}
	if addOnResource.Status == nil || addOnResource.Status.AddOnMetadata == nil {
		return nil, fmt.Errorf("add-on custom resource %s %s %s/%s status or add-on metadata is nil", meta.APIVersion, meta.Kind, config.ScalableObjectNamespace, meta.Name)
	}

	serverAddress := strings.TrimSpace(addOnResource.Status.AddOnMetadata.ServerAddress)
	if serverAddress == "" {
		return nil, fmt.Errorf("add-on custom resource %s %s %s/%s add-on metadata serverAddress is empty", meta.APIVersion, meta.Kind, config.ScalableObjectNamespace, meta.Name)
	}

	externalMeta := externalScalerMetadata{
		ScalerAddress: serverAddress,
		triggerIndex:  meta.triggerIndex,
	}
	if addOnResource.Status.AddOnMetadata.UsePushScaler {
		return &externalPushScaler{
			externalScaler: externalScaler{
				metricType: metricType,
				metadata:   externalMeta,
				scaledObjectRef: pb.ScaledObjectRef{
					Name:           config.ScalableObjectName,
					Namespace:      config.ScalableObjectNamespace,
					ScalerMetadata: addOnResource.Status.AddOnMetadata.Metadata,
				},
				logger: InitializeLogger(config, "add_on_external_push_scaler"),
			},
		}, nil
	}

	return &externalScaler{
		metricType: metricType,
		metadata:   externalMeta,
		scaledObjectRef: pb.ScaledObjectRef{
			Name:           config.ScalableObjectName,
			Namespace:      config.ScalableObjectNamespace,
			ScalerMetadata: addOnResource.Status.AddOnMetadata.Metadata,
		},
		logger: InitializeLogger(config, "add_on_external_scaler"),
	}, nil
}
