package scalers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/mitchellh/hashstructure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"

	pb "github.com/kedacore/keda/v2/pkg/scalers/externalscaler"
)

type externalScaler struct {
	metricType      v2.MetricTargetType
	metadata        externalScalerMetadata
	scaledObjectRef pb.ScaledObjectRef
	logger          logr.Logger
}

type externalPushScaler struct {
	externalScaler
}

type externalScalerMetadata struct {
	scalerAddress    string
	tlsCertFile      string
	originalMetadata map[string]string
	scalerIndex      int
}

type connectionGroup struct {
	grpcConnection *grpc.ClientConn
}

// a pool of connectionGroup per metadata hash
var connectionPool sync.Map

// NewExternalScaler creates a new external scaler - calls the GRPC interface
// to create a new scaler
func NewExternalScaler(config *ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting external scaler metric type: %s", err)
	}

	meta, err := parseExternalScalerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing external scaler metadata: %s", err)
	}

	return &externalScaler{
		metricType: metricType,
		metadata:   meta,
		scaledObjectRef: pb.ScaledObjectRef{
			Name:           config.ScalableObjectName,
			Namespace:      config.ScalableObjectNamespace,
			ScalerMetadata: meta.originalMetadata,
		},
		logger: InitializeLogger(config, "external_scaler"),
	}, nil
}

// NewExternalPushScaler creates a new externalPushScaler push scaler
func NewExternalPushScaler(config *ScalerConfig) (PushScaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting external scaler metric type: %s", err)
	}

	meta, err := parseExternalScalerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing external scaler metadata: %s", err)
	}

	return &externalPushScaler{
		externalScaler{
			metricType: metricType,
			metadata:   meta,
			scaledObjectRef: pb.ScaledObjectRef{
				Name:           config.ScalableObjectName,
				Namespace:      config.ScalableObjectNamespace,
				ScalerMetadata: meta.originalMetadata,
			},
			logger: InitializeLogger(config, "external_push_scaler"),
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
	meta.scalerIndex = config.ScalerIndex
	return meta, nil
}

// IsActive checks if there are any messages in the subscription
func (s *externalScaler) IsActive(ctx context.Context) (bool, error) {
	grpcClient, err := getClientForConnectionPool(s.metadata)
	if err != nil {
		return false, err
	}

	response, err := grpcClient.IsActive(ctx, &s.scaledObjectRef)
	if err != nil {
		s.logger.Error(err, "error calling IsActive on external scaler")
		return false, err
	}

	return response.Result, nil
}

func (s *externalScaler) Close(context.Context) error {
	return nil
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *externalScaler) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	var result []v2.MetricSpec

	grpcClient, err := getClientForConnectionPool(s.metadata)
	if err != nil {
		s.logger.Error(err, "error building grpc connection")
		return result
	}

	response, err := grpcClient.GetMetricSpec(ctx, &s.scaledObjectRef)
	if err != nil {
		s.logger.Error(err, "error")
		return nil
	}

	for _, spec := range response.MetricSpecs {
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(s.metadata.scalerIndex, spec.MetricName),
			},
			Target: GetMetricTarget(s.metricType, spec.TargetSize),
		}

		// Create the metric spec for the HPA
		metricSpec := v2.MetricSpec{
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
	grpcClient, err := getClientForConnectionPool(s.metadata)
	if err != nil {
		return metrics, err
	}

	// Remove the sX- prefix as the external scaler shouldn't have to know about it
	metricNameWithoutIndex, err := RemoveIndexFromMetricName(s.metadata.scalerIndex, metricName)
	if err != nil {
		return metrics, err
	}

	request := &pb.GetMetricsRequest{
		MetricName:      metricNameWithoutIndex,
		ScaledObjectRef: &s.scaledObjectRef,
	}

	response, err := grpcClient.GetMetrics(ctx, request)
	if err != nil {
		s.logger.Error(err, "error")
		return []external_metrics.ExternalMetricValue{}, err
	}

	for _, metricResult := range response.MetricValues {
		metric := GenerateMetricInMili(metricName, float64(metricResult.MetricValue))
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// handleIsActiveStream is the only writer to the active channel and will close it on return.
func (s *externalPushScaler) Run(ctx context.Context, active chan<- bool) {
	defer close(active)
	// It's possible for the connection to get terminated anytime, we need to run this in a retry loop
	runWithLog := func() {
		grpcClient, err := getClientForConnectionPool(s.metadata)
		if err != nil {
			s.logger.Error(err, "error running internalRun")
			return
		}
		if err := handleIsActiveStream(ctx, &s.scaledObjectRef, grpcClient, active); err != nil {
			s.logger.Error(err, "error running internalRun")
			return
		}
	}

	// retry on error from runWithLog() starting by 2 sec backing off * 2 with a max of 1 minute
	retryDuration := time.Second * 2
	// the caller of this function needs to ensure that they call Stop() on the resulting
	// timer, to release background resources.
	retryBackoff := func() *time.Timer {
		tmr := time.NewTimer(retryDuration)
		retryDuration *= 2
		if retryDuration > time.Minute*1 {
			retryDuration = time.Minute * 1
		}
		return tmr
	}

	// start the first run without delay
	runWithLog()

	for {
		backoffTimer := retryBackoff()
		select {
		case <-ctx.Done():
			backoffTimer.Stop()
			return
		case <-backoffTimer.C:
			backoffTimer.Stop()
			runWithLog()
		}
	}
}

// handleIsActiveStream calls blocks on a stream call from the GRPC server. It'll only terminate on error, stream completion, or ctx cancellation.
func handleIsActiveStream(ctx context.Context, scaledObjectRef *pb.ScaledObjectRef, grpcClient pb.ExternalScalerClient, active chan<- bool) error {
	stream, err := grpcClient.StreamIsActive(ctx, scaledObjectRef)
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
func getClientForConnectionPool(metadata externalScalerMetadata) (pb.ExternalScalerClient, error) {
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

		return grpc.Dial(metadata.scalerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// create a unique key per-metadata. If scaledObjects share the same connection properties
	// in the metadata, they will share the same grpc.ClientConn
	key, err := hashstructure.Hash(metadata.scalerAddress, nil)
	if err != nil {
		return nil, err
	}

	if i, ok := connectionPool.Load(key); ok {
		if connGroup, ok := i.(*connectionGroup); ok {
			return pb.NewExternalScalerClient(connGroup.grpcConnection), nil
		}
	}

	conn, err := buildGRPCConnection(metadata)
	if err != nil {
		return nil, err
	}

	connGroup := &connectionGroup{
		grpcConnection: conn,
	}

	connectionPool.Store(key, connGroup)

	go func() {
		// clean up goroutine.
		// once gRPC client is shutdown, remove the connection from the pool and Close() grpc.ClientConn
		<-waitForState(context.TODO(), connGroup.grpcConnection, connectivity.Shutdown)
		connectionPoolMutex.Lock()
		defer connectionPoolMutex.Unlock()
		connectionPool.Delete(key)
		connGroup.grpcConnection.Close()
	}()

	return pb.NewExternalScalerClient(connGroup.grpcConnection), nil
}

func waitForState(ctx context.Context, conn *grpc.ClientConn, states ...connectivity.State) (done chan struct{}) {
	done = make(chan struct{})

	go func() {
		defer close(done)

		for {
			changeState := conn.WaitForStateChange(ctx, conn.GetState())
			if !changeState {
				// ctx is done, return
				continue
			}

			nowState := conn.GetState()
			for _, state := range states {
				if state == nowState {
					// match one of the state passed return
					return
				}
			}
		}
	}()

	return done
}
