package scalers

import (
	"context"
	"fmt"

	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	keda "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	smi "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/metrics/v1alpha1"
)

type smiScaler struct {
	metadata      *smiMetadata
	namespace     string
	scaledObjName string
}

type smiMetadata struct {
	metricName  string
	metricValue resource.Quantity
}

const (
	smiMetricName   = "metricName"
	smiMetricValue  = "metricValue"
	scaledObjectRes = "scaledobjects"
	scaledTargetRes = "deployments"
)

var smiLog = logf.Log.WithName("smi_scaler")

func NewSmiScaler(name, namespace string, resolvedEnv, metadata map[string]string) (Scaler, error) {
	meta, err := parseSmiMetadata(metadata)
	if err != nil {
		return nil, fmt.Errorf("error parsing service mesh interface metadata: %s", err)
	}

	return &smiScaler{
		metadata:      meta,
		namespace:     namespace,
		scaledObjName: name,
	}, nil
}

func parseSmiMetadata(metadata map[string]string) (*smiMetadata, error) {
	meta := smiMetadata{}

	if val, ok := metadata[smiMetricName]; ok && val != "" {
		meta.metricName = val
	} else {
		return nil, fmt.Errorf("no %s given", smiMetricName)
	}

	if val, ok := metadata[smiMetricValue]; ok && val != "" {
		mValue, err := resource.ParseQuantity(val)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %s", smiMetricValue, err)

		}
		meta.metricValue = mValue
	} else {
		return nil, fmt.Errorf("no %s given", smiMetricValue)
	}
	return &meta, nil
}

func (s *smiScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {
	count, err := s.getSmiMetricValue(ctx)

	if err != nil {
		return []external_metrics.ExternalMetricValue{}, err
	}

	metric := external_metrics.ExternalMetricValue{
		MetricName: metricName,
		Value:      count,
		Timestamp:  metav1.Now(),
	}

	return append([]external_metrics.ExternalMetricValue{}, metric), nil
}

func (s *smiScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {
	targetQty := s.metadata.metricValue

	metricsSpec := []v2beta1.MetricSpec{
		{
			External: &v2beta1.ExternalMetricSource{
				TargetAverageValue: &targetQty,
				MetricName:         s.metadata.metricName,
			},
			Type: externalMetricType,
		},
	}
	return metricsSpec
}

func (s *smiScaler) IsActive(ctx context.Context) (bool, error) {
	count, err := s.getSmiMetricValue(ctx)
	if err != nil {
		smiLog.Error(err, "error retrieving smi metrics value")
		return false, err
	}
	ret := count.CmpInt64(int64(0)) > 0
	return ret, nil
}

func (s *smiScaler) Close() error {
	return nil
}

func (s *smiScaler) getSmiMetricValue(ctx context.Context) (resource.Quantity, error) {
	var ret resource.Quantity

	cfg, _ := rest.InClusterConfig()
	dynCfg := dynamic.ConfigFor(cfg)
	dynClient, _ := dynamic.NewForConfig(dynCfg)

	scaledObjRes := schema.GroupVersionResource{
		Group:    keda.SchemeGroupVersion.Group,
		Version:  keda.SchemeGroupVersion.Version,
		Resource: scaledObjectRes,
	}

	scaledObjectClient := dynClient.Resource(scaledObjRes)
	so, err := scaledObjectClient.Namespace(s.namespace).Get(s.scaledObjName, metav1.GetOptions{})
	if err != nil {
		smiLog.Error(err, "error retrieving ScaledObject", "namespace", s.namespace, "scaledObjectName", s.scaledObjName)
		return ret, err
	}

	var sot keda.ScaledObject
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(so.UnstructuredContent(), &sot)

	// TODO: jobs not supported
	if sot.Spec.ScaleType == "jobs" {
		return resource.Quantity{}, nil
	}

	scaleTargetName := sot.Spec.ScaleTargetRef.DeploymentName

	scaleTypeRes := schema.GroupVersionResource{
		Group:    smi.SchemeGroupVersion.Group,
		Version:  smi.SchemeGroupVersion.Version,
		Resource: scaledTargetRes,
	}

	smiClient := dynClient.Resource(scaleTypeRes)
	crd, err := smiClient.Namespace(s.namespace).Get(scaleTargetName, metav1.GetOptions{})
	if err != nil {
		smiLog.Error(err, "error retrieving TrafficMetrics", "namespace", s.namespace, "scaleType", scaledTargetRes, "scaleTargetName", scaleTargetName)
		return ret, err
	}

	var tm smi.TrafficMetrics
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(crd.UnstructuredContent(), &tm)
	if err != nil {
		smiLog.Error(err, "error deserialzing TrafficMetrics", "scaleTargetName", scaleTargetName)
		return ret, err
	}

	for _, m := range tm.Metrics {
		if s.metadata.metricName == m.Name {
			m.Value.DeepCopyInto(&ret)
		}
	}
	return ret, nil
}
