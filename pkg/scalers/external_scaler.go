package scalers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/hashstructure"

	pb "github.com/kedacore/keda/v2/pkg/scalers/externalscaler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	v2beta2 "k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type externalScaler struct {
	metadata        externalScalerMetadata
	scaledObjectRef pb.ScaledObjectRef
}

type externalPushScaler struct {
	externalScaler
}

type externalScalerMetadata struct {
	scalerAddress    string
	tlsCertFile      string
	originalMetadata map[string]string
}

type connectionGroup struct {
	grpcConnection *grpc.ClientConn
	waitGroup      *sync.WaitGroup
}

// a pool of connectionGroup per metadata hash
var connectionPool sync.Map

var externalLog = logf.Log.WithName("external_scaler")

// NewExternalScaler creates a new external scaler - calls the GRPC interface
// to create a new scaler
func NewExternalScaler(config *ScalerConfig) (Scaler, error) {
	meta, err := parseExternalScalerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing external scaler metadata: %s", err)
	}

	return &externalScaler{
		metadata: meta,
		scaledObjectRef: pb.ScaledObjectRef{
			Name:           config.Name,
			Namespace:      config.Namespace,
			ScalerMetadata: meta.originalMetadata,
		},
	}, nil
}

// NewExternalPushScaler creates a new externalPushScaler push scaler
func NewExternalPushScaler(config *ScalerConfig) (PushScaler, error) {
	meta, err := parseExternalScalerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing external scaler metadata: %s", err)
	}

	return &externalPushScaler{
		externalScaler{
			metadata: meta,
			scaledObjectRef: pb.ScaledObjectRef{
				Name:           config.Name,
				Namespace:      config.Namespace,
				ScalerMetadata: meta.originalMetadata,
			},
		},
	}, nil
}

func parseExternalScalerMetadata(config *ScalerConfig) (externalScalerMetadata, error) {
	meta := externalScalerMetadata{
		originalMetadata: config.TriggerMetadata,
	}

	// Check if scalerAddress is present
	if val, ok := config.TriggerMetadata["scalerAddress"]; ok && val != "" {
		meta.scalerAddress = val
	} else {
		return meta, fmt.Errorf("scaler Address is a required field")
	}

	if val, ok := config.TriggerMetadata["tlsCertFile"]; ok && val != "" {
		meta.tlsCertFile = val
	}

	meta.originalMetadata = make(map[string]string)

	// Add elements to metadata
	for key, value := range config.TriggerMetadata {
		// Check if key is in resolved environment and resolve
		if strings.HasSuffix(key, "FromEnv") {
			if val, ok := config.ResolvedEnv[value]; ok && val != "" {
				meta.originalMetadata[key] = val
			}
		} else {
			meta.originalMetadata[key] = value
		}
	}

	return meta, nil
}

// IsActive checks if there are any messages in the subscription
func (s *externalScaler) IsActive(ctx context.Context) (bool, error) {
	grpcClient, done, err := getClientForConnectionPool(s.metadata)
	if err != nil {
		return false, err
	}
	defer done()

	response, err := grpcClient.IsActive(ctx, &s.scaledObjectRef)
	if err != nil {
		externalLog.Error(err, "error calling IsActive on external scaler")
		return false, err
	}

	return response.Result, nil
}

func (s *externalScaler) Close() error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *externalScaler) GetMetricSpecForScaling() []v2beta2.MetricSpec {
	var result []v2beta2.MetricSpec

	grpcClient, done, err := getClientForConnectionPool(s.metadata)
	if err != nil {
		externalLog.Error(err, "error building grpc connection")
		return result
	}
	defer done()

	response, err := grpcClient.GetMetricSpec(context.TODO(), &s.scaledObjectRef)
	if err != nil {
		externalLog.Error(err, "error")
		return nil
	}

	for _, spec := range response.MetricSpecs {
		// Construct the target subscription size as a quantity
		qty := resource.NewQuantity(spec.TargetSize, resource.DecimalSI)

		externalMetric := &v2beta2.ExternalMetricSource{
			Metric: v2beta2.MetricIdentifier{
				Name: spec.MetricName,
			},
			Target: v2beta2.MetricTarget{
				Type:         v2beta2.AverageValueMetricType,
				AverageValue: qty,
			},
		}

		// Create the metric spec for the HPA
		metricSpec := v2beta2.MetricSpec{
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
	grpcClient, done, err := getClientForConnectionPool(s.metadata)
	if err != nil {
		return metrics, err
	}
	defer done()

	request := &pb.GetMetricsRequest{
		MetricName:      metricName,
		ScaledObjectRef: &s.scaledObjectRef,
	}

	response, err := grpcClient.GetMetrics(ctx, request)
	if err != nil {
		externalLog.Error(err, "error")
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

// handleIsActiveStream is the only writer to the active channel and will close it on return.
func (s *externalPushScaler) Run(ctx context.Context, active chan<- bool) {
	defer close(active)
	// It's possible for the connection to get terminated anytime, we need to run this in a retry loop
	runWithLog := func() {
		grpcClient, done, err := getClientForConnectionPool(s.metadata)
		if err != nil {
			externalLog.Error(err, "error running internalRun")
			return
		}
		if err := handleIsActiveStream(ctx, s.scaledObjectRef, grpcClient, active); err != nil {
			externalLog.Error(err, "error running internalRun")
			return
		}
		done()
	}

	// retry on error from runWithLog() starting by 2 sec backing off * 2 with a max of 1 minute
	retryDuration := time.Second * 2
	retryBackoff := func() <-chan time.Time {
		ch := time.After(retryDuration)
		retryDuration *= time.Second * 2
		if retryDuration > time.Minute*1 {
			retryDuration = time.Minute * 1
		}
		return ch
	}

	// start the first run without delay
	runWithLog()

	for {
		select {
		case <-ctx.Done():
			return
		case <-retryBackoff():
			runWithLog()
		}
	}
}

// handleIsActiveStream calls blocks on a stream call from the GRPC server. It'll only terminate on error, stream completion, or ctx cancellation.
func handleIsActiveStream(ctx context.Context, scaledObjectRef pb.ScaledObjectRef, grpcClient pb.ExternalScalerClient, active chan<- bool) error {
	stream, err := grpcClient.StreamIsActive(ctx, &scaledObjectRef)
	if err != nil {
		return err
	}

	for {
		resp, err := stream.Recv()
		if err != nil {
			return err
		}

		active <- resp.Result
	}
}

var connectionPoolMutex sync.Mutex

// getClientForConnectionPool returns a grpcClient and a done() Func. The done() function must be called once the client is no longer
// in use to clean up the shared grpc.ClientConn
func getClientForConnectionPool(metadata externalScalerMetadata) (pb.ExternalScalerClient, func(), error) {
	connectionPoolMutex.Lock()
	defer connectionPoolMutex.Unlock()

	buildGRPCConnection := func(metadata externalScalerMetadata) (*grpc.ClientConn, error) {
		if metadata.tlsCertFile != "" {
			creds, err := credentials.NewClientTLSFromFile(metadata.tlsCertFile, "")
			if err != nil {
				return nil, err
			}
			return grpc.Dial(metadata.scalerAddress, grpc.WithTransportCredentials(creds))
		}

		return grpc.Dial(metadata.scalerAddress, grpc.WithInsecure())
	}

	// create a unique key per-metadata. If scaledObjects share the same connection properties
	// in the metadata, they will share the same grpc.ClientConn
	key, err := hashstructure.Hash(metadata, nil)
	if err != nil {
		return nil, nil, err
	}

	if i, ok := connectionPool.Load(key); ok {
		if connGroup, ok := i.(*connectionGroup); ok {
			connGroup.waitGroup.Add(1)
			return pb.NewExternalScalerClient(connGroup.grpcConnection), func() {
				connGroup.waitGroup.Done()
			}, nil
		}
	}

	conn, err := buildGRPCConnection(metadata)
	if err != nil {
		return nil, nil, err
	}

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)
	connGroup := connectionGroup{
		grpcConnection: conn,
		waitGroup:      waitGroup,
	}

	connectionPool.Store(key, connGroup)

	go func() {
		// clean up goroutine.
		// once all waitGroup is done, remove the connection from the pool and Close() grpc.ClientConn
		connGroup.waitGroup.Wait()
		connectionPoolMutex.Lock()
		defer connectionPoolMutex.Unlock()
		connectionPool.Delete(key)
		connGroup.grpcConnection.Close()
	}()

	return pb.NewExternalScalerClient(connGroup.grpcConnection), func() {
		connGroup.waitGroup.Done()
	}, nil
}
