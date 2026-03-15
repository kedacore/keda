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
		return nil, fmt.Errorf("target resource doesn't exist: %w", err)
	}
	addOnResource := &AddOnResource{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.Object, addOnResource); err != nil {
		return nil, fmt.Errorf("cannot convert Unstructured into AddOnCRD: %w", err)
	}
	if addOnResource.Status == nil || addOnResource.Status.AddOnMetadata == nil {
		return nil, fmt.Errorf("add-on CRD status or add-on metadata is nil")
	}

	serverAddress := strings.TrimSpace(addOnResource.Status.AddOnMetadata.ServerAddress)
	if serverAddress == "" {
		return nil, fmt.Errorf("add-on metadata serverAddress is empty")
	}

	externalScalerMetadata := externalScalerMetadata{
		ScalerAddress: serverAddress,
		triggerIndex:  meta.triggerIndex,
	}
	if addOnResource.Status.AddOnMetadata.UsePushScaler {
		return &externalPushScaler{
			externalScaler: externalScaler{
				metricType: metricType,
				metadata:   externalScalerMetadata,
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
		metadata:   externalScalerMetadata,
		scaledObjectRef: pb.ScaledObjectRef{
			Name:           config.ScalableObjectName,
			Namespace:      config.ScalableObjectNamespace,
			ScalerMetadata: addOnResource.Status.AddOnMetadata.Metadata,
		},
		logger: InitializeLogger(config, "add_on_external_scaler"),
	}, nil
}
