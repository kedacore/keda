package scalers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/mitchellh/hashstructure/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	v2 "k8s.io/api/autoscaling/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	pb "github.com/kedacore/keda/v2/pkg/scalers/externalscaler"
	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/kedacore/keda/v2/pkg/util"
)

type externalScaler struct {
	metricType      v2.MetricTargetType
	metadata        externalScalerMetadata
	scaledObjectRef pb.ScaledObjectRef
	logger          logr.Logger
	// closeOnce keeps Close releasing this scaler's share of the pooled
	// connection at most once, so a repeated Close cannot drop a reference
	// another scaler still holds.
	closeOnce sync.Once
	closeErr  error
}

type externalPushScaler struct {
	externalScaler
	metricSpecCh chan []v2.MetricSpec
}

type externalScalerMetadata struct {
	originalMetadata map[string]string
	triggerIndex     int

	ScalerAddress string `keda:"name=scalerAddress, order=triggerMetadata"`
	EnableTLS     bool   `keda:"name=enableTLS, order=triggerMetadata, optional"`
	UnsafeSsl     bool   `keda:"name=unsafeSsl, order=triggerMetadata, optional"`

	// auth
	CaCert        string `keda:"name=caCert, order=authParams, optional"`
	TLSClientCert string `keda:"name=tlsClientCert, order=authParams, optional"`
	TLSClientKey  string `keda:"name=tlsClientKey, order=authParams, optional"`
}

type connectionGroup struct {
	grpcConnection *grpc.ClientConn
	// refCount is the number of scalers currently sharing this connection. It
	// is guarded by connectionPoolMutex. The connection is closed and dropped
	// from the pool when the count reaches zero.
	refCount int
}

// a pool of connectionGroup per metadata hash
var connectionPool sync.Map

const grpcConfig = `{"loadBalancingConfig": [{"round_robin":{}}]}`

type externalScalerConnectionPoolKey struct {
	ScalerAddress string
	EnableTLS     bool
	UnsafeSsl     bool
	CaCert        string
	TLSClientCert string
	TLSClientKey  string
}

// NewExternalScaler creates a new external scaler - calls the GRPC interface
// to create a new scaler
func NewExternalScaler(config *scalersconfig.ScalerConfig) (Scaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting external scaler metric type: %w", err)
	}

	meta, err := parseExternalScalerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing external scaler metadata: %w", err)
	}

	if err := acquireConnection(meta); err != nil {
		return nil, fmt.Errorf("error acquiring connection for external scaler: %w", err)
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
func NewExternalPushScaler(config *scalersconfig.ScalerConfig) (PushScaler, error) {
	metricType, err := GetMetricTargetType(config)
	if err != nil {
		return nil, fmt.Errorf("error getting external scaler metric type: %w", err)
	}

	meta, err := parseExternalScalerMetadata(config)
	if err != nil {
		return nil, fmt.Errorf("error parsing external scaler metadata: %w", err)
	}

	if err := acquireConnection(meta); err != nil {
		return nil, fmt.Errorf("error acquiring connection for external push scaler: %w", err)
	}

	return &externalPushScaler{
		externalScaler: externalScaler{
			metricType: metricType,
			metadata:   meta,
			scaledObjectRef: pb.ScaledObjectRef{
				Name:           config.ScalableObjectName,
				Namespace:      config.ScalableObjectNamespace,
				ScalerMetadata: meta.originalMetadata,
			},
			logger: InitializeLogger(config, "external_push_scaler"),
		},
		metricSpecCh: make(chan []v2.MetricSpec, 1),
	}, nil
}

func parseExternalScalerMetadata(config *scalersconfig.ScalerConfig) (externalScalerMetadata, error) {
	meta := externalScalerMetadata{}
	meta.triggerIndex = config.TriggerIndex
	if err := config.TypedConfig(&meta); err != nil {
		return meta, fmt.Errorf("error parsing external scaler metadata: %w", err)
	}

	meta.originalMetadata = make(map[string]string)
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

// Close releases this scaler's share of the pooled gRPC connection. The
// connection is closed once no scaler is left using it. Calling Close more than
// once releases the share only once.
func (s *externalScaler) Close(context.Context) error {
	s.closeOnce.Do(func() {
		s.closeErr = releaseConnection(s.metadata)
	})
	return s.closeErr
}

// GetMetricSpecForScaling returns the metric spec for the HPA
func (s *externalScaler) GetMetricSpecForScaling(ctx context.Context) []v2.MetricSpec {
	grpcClient, err := getClientForConnectionPool(s.metadata)
	if err != nil {
		s.logger.Error(err, "error building grpc connection")
		return nil
	}

	response, err := grpcClient.GetMetricSpec(ctx, &s.scaledObjectRef)
	if err != nil {
		s.logger.Error(err, "error")
		return nil
	}

	return s.buildMetricSpecs(response)
}

// buildMetricSpecs converts a GetMetricSpecResponse into Kubernetes metric specs.
// Always returns a non-nil slice (possibly empty).
func (s *externalScaler) buildMetricSpecs(response *pb.GetMetricSpecResponse) []v2.MetricSpec {
	result := make([]v2.MetricSpec, 0, len(response.MetricSpecs))
	for _, spec := range response.MetricSpecs {
		externalMetric := &v2.ExternalMetricSource{
			Metric: v2.MetricIdentifier{
				Name: GenerateMetricNameWithIndex(s.metadata.triggerIndex, spec.MetricName),
			},
		}
		if spec.TargetSizeFloat > 0 {
			externalMetric.Target = GetMetricTargetMili(s.metricType, spec.TargetSizeFloat)
		} else {
			externalMetric.Target = GetMetricTarget(s.metricType, spec.TargetSize)
		}
		result = append(result, v2.MetricSpec{
			External: externalMetric,
			Type:     externalMetricType,
		})
	}
	return result
}

// GetMetricsAndActivity returns value for a supported metric and an error if there is a problem getting the metric
func (s *externalScaler) GetMetricsAndActivity(ctx context.Context, metricName string) ([]external_metrics.ExternalMetricValue, bool, error) {
	var metrics []external_metrics.ExternalMetricValue
	grpcClient, err := getClientForConnectionPool(s.metadata)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	// Remove the sX- prefix as the external scaler shouldn't have to know about it
	metricNameWithoutIndex, err := RemoveIndexFromMetricName(s.metadata.triggerIndex, metricName)
	if err != nil {
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	request := &pb.GetMetricsRequest{
		MetricName:      metricNameWithoutIndex,
		ScaledObjectRef: &s.scaledObjectRef,
	}

	metricsResponse, err := grpcClient.GetMetrics(ctx, request)
	if err != nil {
		s.logger.Error(err, "error")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	for _, metricResult := range metricsResponse.MetricValues {
		value := float64(metricResult.MetricValue)
		if metricResult.MetricValueFloat > 0 {
			value = metricResult.MetricValueFloat
		}
		metric := GenerateMetricInMili(metricName, value)
		metrics = append(metrics, metric)
	}

	isActiveResponse, err := grpcClient.IsActive(ctx, &s.scaledObjectRef)
	if err != nil {
		s.logger.Error(err, "error calling IsActive on external scaler")
		return []external_metrics.ExternalMetricValue{}, false, err
	}

	return metrics, isActiveResponse.Result, nil
}

// Run starts both the StreamIsActive and StreamMetricSpec stream handlers.
func (s *externalPushScaler) Run(ctx context.Context, active chan<- bool) {
	defer close(active)

	go s.runStreamMetricSpec(ctx)
	s.runStreamIsActive(ctx, active)
}

func (s *externalPushScaler) runStreamIsActive(ctx context.Context, active chan<- bool) {
	retryDuration := 2 * time.Second

	runOnce := func() {
		grpcClient, err := getClientForConnectionPool(s.metadata)
		if err != nil {
			s.logger.Error(err, "unable to get connection from the pool")
			return
		}
		err = handleIsActiveStream(ctx, &s.scaledObjectRef, grpcClient, active)
		if !errors.Is(err, io.EOF) {
			s.logger.Error(err, "error running StreamIsActive")
			return
		}
		retryDuration = 2 * time.Second
	}

	runOnce()

	for {
		tmr := time.NewTimer(retryDuration)
		s.logger.V(1).Info("StreamIsActive retry backoff", "duration", retryDuration)
		retryDuration = min(retryDuration*2, time.Minute)
		select {
		case <-ctx.Done():
			tmr.Stop()
			return
		case <-tmr.C:
			runOnce()
		}
	}
}

// runStreamMetricSpec opens a StreamMetricSpec stream and forwards updates
// to metricSpecCh. The channel is closed when this goroutine exits — either
// because the context was cancelled or because the server returned
// Unimplemented. In the latter case the channel is permanently closed; if
// the ScaledObject generation changes, startPushScalers re-creates the
// externalPushScaler (with a fresh channel) and launches a new watcher.
func (s *externalPushScaler) runStreamMetricSpec(ctx context.Context) {
	defer close(s.metricSpecCh)

	retryDuration := 2 * time.Second

	for {
		shouldStop, resetRetry := s.streamMetricSpecOnce(ctx)
		if shouldStop {
			return
		}
		if resetRetry {
			retryDuration = 2 * time.Second
		}
		tmr := time.NewTimer(retryDuration)
		s.logger.V(1).Info("StreamMetricSpec retry backoff", "duration", retryDuration)
		retryDuration = min(retryDuration*2, time.Minute)
		select {
		case <-ctx.Done():
			tmr.Stop()
			return
		case <-tmr.C:
		}
	}
}

// streamMetricSpecOnce opens a StreamMetricSpec stream and processes updates
// until the stream closes or an error occurs.
// Returns (shouldStop, resetRetry): shouldStop=true means the caller should
// exit the retry loop; resetRetry=true means the retry backoff should be
// reset because the stream terminated cleanly (io.EOF) or delivered at least
// one update before failing. A stream that opens but errors before delivering
// anything keeps the growing backoff, so a persistently broken server is not
// retried in a tight loop.
func (s *externalPushScaler) streamMetricSpecOnce(ctx context.Context) (bool, bool) {
	grpcClient, err := getClientForConnectionPool(s.metadata)
	if err != nil {
		s.logger.Error(err, "StreamMetricSpec: unable to get gRPC connection")
		return false, false
	}

	stream, err := grpcClient.StreamMetricSpec(ctx, &s.scaledObjectRef)
	if err != nil {
		if status.Code(err) == codes.Unimplemented {
			s.logger.V(1).Info("StreamMetricSpec not implemented by scaler, skipping")
			return true, false
		}
		s.logger.Error(err, "StreamMetricSpec: failed to open stream")
		return false, false
	}

	received := false
	for {
		resp, err := stream.Recv()
		if err != nil {
			if status.Code(err) == codes.Unimplemented {
				s.logger.V(1).Info("StreamMetricSpec not implemented by scaler, skipping")
				return true, false
			}
			if errors.Is(err, io.EOF) {
				return false, true
			}
			s.logger.Error(err, "StreamMetricSpec: stream error")
			return false, received
		}
		received = true

		specs := s.buildMetricSpecs(resp)
		// Single-producer drain-then-send: safe because only one goroutine
		// (runStreamMetricSpec) writes to metricSpecCh.
		select {
		case s.metricSpecCh <- specs:
		default:
			select {
			case <-s.metricSpecCh:
			default:
			}
			select {
			case s.metricSpecCh <- specs:
			case <-ctx.Done():
				return true, false
			}
		}
	}
}

// MetricSpecChan returns the channel that receives updated metric specs
// from the StreamMetricSpec stream.
func (s *externalPushScaler) MetricSpecChan() <-chan []v2.MetricSpec {
	return s.metricSpecCh
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

		select {
		case active <- resp.Result:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

var connectionPoolMutex sync.Mutex

func getConnectionPoolKey(metadata externalScalerMetadata) (uint64, error) {
	key := externalScalerConnectionPoolKey{
		ScalerAddress: metadata.ScalerAddress,
		EnableTLS:     metadata.EnableTLS,
		UnsafeSsl:     metadata.UnsafeSsl,
		CaCert:        metadata.CaCert,
		TLSClientCert: metadata.TLSClientCert,
		TLSClientKey:  metadata.TLSClientKey,
	}

	return hashstructure.Hash(key, hashstructure.FormatV1, nil)
}

func buildGRPCConnection(metadata externalScalerMetadata) (*grpc.ClientConn, error) {
	tlsConfig, err := util.NewTLSConfig(metadata.TLSClientCert, metadata.TLSClientKey, metadata.CaCert, metadata.UnsafeSsl)
	if err != nil {
		return nil, err
	}

	if metadata.EnableTLS || len(tlsConfig.Certificates) > 0 || metadata.CaCert != "" {
		// nosemgrep: go.grpc.ssrf.grpc-tainted-url-host.grpc-tainted-url-host
		return grpc.NewClient(metadata.ScalerAddress,
			grpc.WithDefaultServiceConfig(grpcConfig),
			grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}

	return grpc.NewClient(metadata.ScalerAddress,
		grpc.WithDefaultServiceConfig(grpcConfig),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
}

// connectionGroupForKey returns the pooled connection for key, creating it from
// metadata when the pool does not hold one yet. Callers must hold
// connectionPoolMutex.
func connectionGroupForKey(key uint64, metadata externalScalerMetadata) (*connectionGroup, error) {
	if i, ok := connectionPool.Load(key); ok {
		if connGroup, ok := i.(*connectionGroup); ok {
			return connGroup, nil
		}
	}

	conn, err := buildGRPCConnection(metadata)
	if err != nil {
		return nil, err
	}

	connGroup := &connectionGroup{grpcConnection: conn}
	connectionPool.Store(key, connGroup)
	return connGroup, nil
}

// acquireConnection records one more scaler sharing the connection for this
// metadata, creating the connection if the pool does not hold one yet. Every
// call must be paired with releaseConnection, which closes the connection once
// the last scaler using it has gone away.
//
// grpc.NewClient connects lazily, so acquiring here costs nothing until the
// scaler actually issues a request.
func acquireConnection(metadata externalScalerMetadata) error {
	connectionPoolMutex.Lock()
	defer connectionPoolMutex.Unlock()

	key, err := getConnectionPoolKey(metadata)
	if err != nil {
		return err
	}

	connGroup, err := connectionGroupForKey(key, metadata)
	if err != nil {
		return err
	}

	connGroup.refCount++
	return nil
}

// releaseConnection records that one scaler has stopped using the connection
// for this metadata. The connection is closed and dropped from the pool once no
// scaler is left using it.
func releaseConnection(metadata externalScalerMetadata) error {
	connectionPoolMutex.Lock()
	defer connectionPoolMutex.Unlock()

	key, err := getConnectionPoolKey(metadata)
	if err != nil {
		return err
	}

	i, ok := connectionPool.Load(key)
	if !ok {
		return nil
	}
	connGroup, ok := i.(*connectionGroup)
	if !ok {
		return nil
	}

	// Guard against releasing a share that was never taken. The entry under
	// this key may have been dropped and rebuilt by another scaler since, and
	// decrementing past zero here would close a connection that scaler is
	// still using.
	if connGroup.refCount <= 0 {
		return nil
	}

	connGroup.refCount--
	if connGroup.refCount > 0 {
		return nil
	}

	connectionPool.Delete(key)
	return connGroup.grpcConnection.Close()
}

// getClientForConnectionPool returns a client on the connection shared by every
// scaler with the same connection properties. The connection's lifetime is
// owned by acquireConnection and releaseConnection rather than by this call.
func getClientForConnectionPool(metadata externalScalerMetadata) (pb.ExternalScalerClient, error) {
	connectionPoolMutex.Lock()
	defer connectionPoolMutex.Unlock()

	// create a unique key per-metadata. If scaledObjects share the same connection properties
	// in the metadata, they will share the same grpc.ClientConn
	key, err := getConnectionPoolKey(metadata)
	if err != nil {
		return nil, err
	}

	connGroup, err := connectionGroupForKey(key, metadata)
	if err != nil {
		return nil, err
	}

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
				return
			}

			nowState := conn.GetState()
			if slices.Contains(states, nowState) {
				// match one of the state passed return
				return
			}
		}
	}()

	return done
}
