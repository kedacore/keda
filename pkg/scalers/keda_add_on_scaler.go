package scalers

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	pb "github.com/kedacore/keda/v2/pkg/scalers/externalscaler"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
)

type AddOnCRD struct {
	Status *struct {
		AddOnMetadata *struct {
			ServerAddress string            `json:"serverAddress"`
			Metadata      map[string]string `json:"metadata"`
			UsePushScaler bool              `json:"usePushScaler"`
		} `json:"addOnMetadata"`
	} `json:"status"`
}

type kedaAddOnScalerMetadata struct {
	triggerIndex int //nolint:unused // This is needed as marker for schema generation

	Name       string `keda:"name=name, order=triggerMetadata"`
	Kind       string `keda:"name=kind, order=triggerMetadata"`
	APIVersion string `keda:"name=apiVersion, order=triggerMetadata"`
}

// NewKedaAddOnScaler creates a new Keda Add-On Scaler
func NewKedaAddOnScaler(ctx context.Context, kubeClient client.Client, config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting external scaler metric type: %w", err)
	}

	meta := &kedaAddOnScalerMetadata{}
	err = config.TypedConfig(meta)
	if err != nil {
		return nil, fmt.Errorf("error parsing add-on metadata: %w", err)
	}

	gvk, err := v1alpha1.ParseGVKR(kubeClient.RESTMapper(), meta.APIVersion, meta.Kind)
	if err != nil {
		return nil, fmt.Errorf("error parsing add-on gvkr: %w", err)
	}
	unstruct := &unstructured.Unstructured{}
	unstruct.SetGroupVersionKind(gvk.GroupVersionKind())

	if err := kubeClient.Get(ctx, client.ObjectKey{Namespace: config.ScalableObjectNamespace, Name: meta.Name}, unstruct); err != nil {
		return nil, fmt.Errorf("target resource doesn't exist: %w", err)
	}
	addOnCRD := &AddOnCRD{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct.Object, addOnCRD); err != nil {
		return nil, fmt.Errorf("cannot convert Unstructured into AddOnCRD: %w", err)
	}
	if addOnCRD.Status == nil || addOnCRD.Status.AddOnMetadata == nil {
		return nil, fmt.Errorf("add-on CRD status or add-on metadata is nil")
	}

	serverAddress := strings.TrimSpace(addOnCRD.Status.AddOnMetadata.ServerAddress)
	if serverAddress == "" {
		return nil, fmt.Errorf("add-on metadata serverAddress is empty")
	}

	externalScalerMetadata := externalScalerMetadata{
		ScalerAddress: serverAddress,
	}
	if addOnCRD.Status.AddOnMetadata.UsePushScaler {
		return &externalPushScaler{
			externalScaler: externalScaler{
				metricType: metricType,
				metadata:   externalScalerMetadata,
				scaledObjectRef: pb.ScaledObjectRef{
					Name:           config.ScalableObjectName,
					Namespace:      config.ScalableObjectNamespace,
					ScalerMetadata: addOnCRD.Status.AddOnMetadata.Metadata,
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
			ScalerMetadata: addOnCRD.Status.AddOnMetadata.Metadata,
		},
		logger: InitializeLogger(config, "add_on_external_scaler"),
	}, nil
}
