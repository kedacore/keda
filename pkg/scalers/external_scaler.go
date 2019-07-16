package scalers

import (
	"context"
	"fmt"

	log "github.com/Sirupsen/logrus"
	keda_v1alpha1 "github.com/kedacore/keda/pkg/apis/keda/v1alpha1"
	pb "github.com/kedacore/keda/pkg/scalers/externalscaler"
	"google.golang.org/grpc"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type externalScaler struct {
	metadata        *externalScalerMetadata
	scaledObjectRef pb.ScaledObjectRef
}

type externalScalerMetadata struct {
	serviceURI string
	metadata   map[string]string
}

// NewExternalScaler creates a new external scaler - calls the GRPC interface
// to create a new scaler
func NewExternalScaler(scaledObject *keda_v1alpha1.ScaledObject, resolvedEnv, metadata map[string]string) (Scaler, error) {

	meta, err := parseExternalScalerMetadata(metadata, resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("error parsing external scaler metadata: %s", err)
	}

	scaler := &externalScaler{
		metadata: meta,
		scaledObjectRef: pb.ScaledObjectRef{
			Name:      scaledObject.Name,
			Namespace: scaledObject.Namespace,
		},
	}

	// TODO: Pass Context
	ctx := context.Background()

	// Call GRPC Interface to parse metadata
	grpcClient, err := scaler.getGRPCClient()
	if err != nil {
		return nil, err
	}

	request := &pb.NewRequest{
		ScaledObjectRef: &scaler.scaledObjectRef,
		Metadata:        scaler.metadata.metadata,
	}

	_, err = grpcClient.New(ctx, request)
	if err != nil {
		return nil, err
	}

	return scaler, nil
}

func parseExternalScalerMetadata(metadata, resolvedEnv map[string]string) (*externalScalerMetadata, error) {
	meta := externalScalerMetadata{}

	// Check if service uri is present
	if val, ok := metadata["serviceURI"]; ok && val != "" {
		meta.serviceURI = val
	} else {
		return nil, fmt.Errorf("Service URI is a required field")
	}

	meta.metadata = make(map[string]string)

	// Add elements to metadata
	for key, value := range metadata {
		// Check if key is in resolved environment and resolve
		if val, ok := resolvedEnv[value]; ok && val != "" {
			meta.metadata[key] = val
		} else {
			meta.metadata[key] = value
		}
	}

	return &meta, nil
}

// IsActive checks if there are any messages in the subscription
func (s *externalScaler) IsActive(ctx context.Context) (bool, error) {

	// Call GRPC Interface to check if active
	grpcClient, err := s.getGRPCClient()
	if err != nil {
		return false, err
	}

	response, err := grpcClient.IsActive(ctx, &s.scaledObjectRef)
	if err != nil {
		log.Errorf("error %s", err)
		return false, err
	}

	return response.Result, nil
}

func (s *externalScaler) Close() error {
	// Call GRPC Interface to close connection
	grpcClient, err := s.getGRPCClient()
	if err != nil {
		return err
	}

	// TODO: Pass Context
	ctx := context.Background()

	_, err = grpcClient.Close(ctx, &s.scaledObjectRef)
	if err != nil {
		log.Errorf("error %s", err)
		return err
	}

	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *externalScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {

	// TODO: Pass Context
	ctx := context.Background()

	// Call GRPC Interface to get metric specs
	grpcClient, err := s.getGRPCClient()
	if err != nil {
		return nil
	}

	response, err := grpcClient.GetMetricSpec(ctx, &s.scaledObjectRef)
	if err != nil {
		log.Errorf("error %s", err)
		return nil
	}

	var result []v2beta1.MetricSpec

	for _, spec := range response.MetricSpecs {
		// Construct the target subscription size as a quantity
		qty := resource.NewQuantity(int64(spec.TargetSize), resource.DecimalSI)

		externalMetric := &v2beta1.ExternalMetricSource{
			MetricName:         spec.MetricName,
			TargetAverageValue: qty,
		}

		// Create the metric spec for the HPA
		metricSpec := v2beta1.MetricSpec{
			External: externalMetric,
			Type:     externalMetricType,
		}

		result = append(result, metricSpec)
	}

	return result
}

// GetMetrics connects to Stack Driver and finds the size of the pub sub subscription
func (s *externalScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {

	var metrics []external_metrics.ExternalMetricValue
	// Call GRPC Interface to get metric specs
	grpcClient, err := s.getGRPCClient()
	if err != nil {
		return metrics, err
	}

	request := &pb.GetMetricsRequest{
		MetricName:      metricName,
		ScaledObjectRef: &s.scaledObjectRef,
	}

	response, err := grpcClient.GetMetrics(ctx, request)
	if err != nil {
		log.Errorf("error %s", err)
		return []external_metrics.ExternalMetricValue{}, err
	}

	for _, metricResult := range response.MetricValues {
		metric := external_metrics.ExternalMetricValue{
			MetricName: metricResult.MetricName,
			Value:      *resource.NewQuantity(metricResult.MetricValue, resource.DecimalSI),
			Timestamp:  metav1.Now(),
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

func (s *externalScaler) getGRPCClient() (pb.ExternalScalerClient, error) {

	var client pb.ExternalScalerClient

	conn, err := grpc.Dial(s.metadata.serviceURI, grpc.WithInsecure())
	if err != nil {
		return client, fmt.Errorf("error parsing external scaler metadata: %s", err)
	}

	client = pb.NewExternalScalerClient(conn)

	return client, nil
}
