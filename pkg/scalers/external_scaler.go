package scalers

import (
	"context"
	"fmt"

	pb "github.com/kedacore/keda/pkg/scalers/externalscaler"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	v2beta1 "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type externalScaler struct {
	metadata        *externalScalerMetadata
	scaledObjectRef pb.ScaledObjectRef
	grpcClient      pb.ExternalScalerClient
	grpcConnection  *grpc.ClientConn
}

type externalScalerMetadata struct {
	scalerAddress string
	tlsCertFile   string
	metadata      map[string]string
}

// NewExternalScaler creates a new external scaler - calls the GRPC interface
// to create a new scaler
func NewExternalScaler(name, namespace string, resolvedEnv, metadata map[string]string) (Scaler, error) {

	meta, err := parseExternalScalerMetadata(metadata, resolvedEnv)
	if err != nil {
		return nil, fmt.Errorf("error parsing external scaler metadata: %s", err)
	}

	scaler := &externalScaler{
		metadata: meta,
		scaledObjectRef: pb.ScaledObjectRef{
			Name:      name,
			Namespace: namespace,
		},
	}

	// TODO: Pass Context
	ctx := context.Background()

	// Call GRPC Interface to parse metadata
	err = scaler.getGRPCClient()
	if err != nil {
		return nil, err
	}

	request := &pb.NewRequest{
		ScaledObjectRef: &scaler.scaledObjectRef,
		Metadata:        scaler.metadata.metadata,
	}

	_, err = scaler.grpcClient.New(ctx, request)
	if err != nil {
		return nil, err
	}

	return scaler, nil
}

func parseExternalScalerMetadata(metadata, resolvedEnv map[string]string) (*externalScalerMetadata, error) {
	meta := externalScalerMetadata{}

	// Check if scalerAddress is present
	if val, ok := metadata["scalerAddress"]; ok && val != "" {
		meta.scalerAddress = val
	} else {
		return nil, fmt.Errorf("Scaler Address is a required field")
	}

	if val, ok := metadata["tlsCertFile"]; ok && val != "" {
		meta.tlsCertFile = val
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
	response, err := s.grpcClient.IsActive(ctx, &s.scaledObjectRef)
	if err != nil {
		log.Errorf("error %s", err)
		return false, err
	}

	return response.Result, nil
}

func (s *externalScaler) Close() error {
	// Call GRPC Interface to close connection

	// TODO: Pass Context
	ctx := context.Background()

	_, err := s.grpcClient.Close(ctx, &s.scaledObjectRef)
	if err != nil {
		log.Errorf("error %s", err)
		return err
	}
	defer s.grpcConnection.Close()

	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *externalScaler) GetMetricSpecForScaling() []v2beta1.MetricSpec {

	// TODO: Pass Context
	ctx := context.Background()

	// Call GRPC Interface to get metric specs
	response, err := s.grpcClient.GetMetricSpec(ctx, &s.scaledObjectRef)
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

// GetMetrics connects calls the gRPC interface to get the metrics with a specific name
func (s *externalScaler) GetMetrics(ctx context.Context, metricName string, metricSelector labels.Selector) ([]external_metrics.ExternalMetricValue, error) {

	var metrics []external_metrics.ExternalMetricValue
	// Call GRPC Interface to get metric specs

	request := &pb.GetMetricsRequest{
		MetricName:      metricName,
		ScaledObjectRef: &s.scaledObjectRef,
	}

	response, err := s.grpcClient.GetMetrics(ctx, request)
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

// getGRPCClient creates a new gRPC client
func (s *externalScaler) getGRPCClient() error {

	var err error

	if s.metadata.tlsCertFile != "" {
		certFile := fmt.Sprintf("/grpccerts/%s", s.metadata.tlsCertFile)
		creds, err := credentials.NewClientTLSFromFile(certFile, "")
		if err != nil {
			return err
		}
		s.grpcConnection, err = grpc.Dial(s.metadata.scalerAddress, grpc.WithTransportCredentials(creds))
	} else {
		s.grpcConnection, err = grpc.Dial(s.metadata.scalerAddress, grpc.WithInsecure())
	}

	if err != nil {
		return fmt.Errorf("cannot connect to external scaler over grpc interface: %s", err)
	}

	s.grpcClient = pb.NewExternalScalerClient(s.grpcConnection)

	return nil
}
