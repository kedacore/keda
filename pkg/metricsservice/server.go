/*
Copyright 2022 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metricsservice

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/metrics/pkg/apis/external_metrics/v1beta1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kedacore/keda/v2/pkg/metricscollector"
	"github.com/kedacore/keda/v2/pkg/metricsservice/api"
	"github.com/kedacore/keda/v2/pkg/metricsservice/utils"
	"github.com/kedacore/keda/v2/pkg/scaling"
)

var log = logf.Log.WithName("grpc_server")

type GrpcServer struct {
	server        *grpc.Server
	address       string
	certDir       string
	certsReady    chan struct{}
	scalerHandler *scaling.ScaleHandler
	api.UnimplementedMetricsServiceServer
	api.UnimplementedRawMetricsServiceServer
}

// GetMetrics returns metrics values in form of ExternalMetricValueList for specified ScaledObject reference
func (s *GrpcServer) GetMetrics(ctx context.Context, in *api.ScaledObjectRef) (*v1beta1.ExternalMetricValueList, error) {
	v1beta1ExtMetrics := &v1beta1.ExternalMetricValueList{}
	extMetrics, err := (*s.scalerHandler).GetScaledObjectMetrics(ctx, in.Name, in.Namespace, in.MetricName)
	if err != nil {
		return v1beta1ExtMetrics, fmt.Errorf("error when getting metric values %w", err)
	}

	err = v1beta1.Convert_external_metrics_ExternalMetricValueList_To_v1beta1_ExternalMetricValueList(extMetrics, v1beta1ExtMetrics, nil)
	if err != nil {
		return v1beta1ExtMetrics, fmt.Errorf("error when converting metric values %w", err)
	}

	log.V(1).WithValues("scaledObjectName", in.Name, "scaledObjectNamespace", in.Namespace, "metrics", v1beta1ExtMetrics).Info("Providing metrics")

	return v1beta1ExtMetrics, nil
}

// GetRawMetricsStream opens the gRPC stream for sending metrics
func (s *GrpcServer) GetRawMetricsStream(request *api.RawMetricsRequest, stream grpc.ServerStreamingServer[api.RawMetricsResponse]) error {
	logger := log.WithName("GetRawMetricsStream").WithValues("subscriber", request.GetSubscriber())
	if request.GetSubscriber() == "" {
		return fmt.Errorf("subscriber must be specified, request: %+v", request)
	}
	// create a channel that will be receiving the metrics and for each metric on that channel, send it to the client
	rawMetricsCh, doneCh := (*s.scalerHandler).GetRawMetricsChan(request.GetSubscriber())
	for {
		select {
		case <-stream.Context().Done():
			// client went away or canceled
			return status.FromContextError(stream.Context().Err()).Err()
		case rm, ok := <-rawMetricsCh:
			if !ok {
				return nil
			}
			resp := &api.RawMetricsResponse{}
			var metrics []*api.RawMetric
			for _, v := range rm.Values {
				logger.V(10).Info("Sending raw metric", "MetricName", v.MetricName, "Time", v.Timestamp.String(), "Value", v.Value.AsApproximateFloat64())
				metrics = append(metrics, &api.RawMetric{
					Value:     v.Value.AsApproximateFloat64(),
					Timestamp: timestamppb.New(v.Timestamp.Time),
					IsActive:  rm.IsActive,
					Metadata: &api.ScaledObjectRef{
						Name:       rm.Meta.ScaledObjectName,
						Namespace:  rm.Meta.Namespace,
						MetricName: rm.Meta.TriggerName,
					},
				})
			}
			resp.Metrics = metrics
			if err := stream.Send(resp); err != nil {
				st, _ := status.FromError(err)
				switch st.Code() {
				case codes.Canceled, codes.Unavailable:
					return nil
				default:
					// genuine server-side send error
					return err
				}
			}
		case val, open := <-doneCh:
			if open && !val {
				logger.V(10).Info(fmt.Sprintf("Channel closed for subscriber %s", request.GetSubscriber()))
				return nil
			}
			if !open {
				logger.V(10).Info(fmt.Sprintf("Done channel closed for subscriber %s", request.GetSubscriber()))
				return nil
			}
		}
	}
}

// SubscribeMetric returns true in SubscriptionAck.WasSubscribed is the metric has been already subscribed
// false for new a subscription
func (s *GrpcServer) SubscribeMetric(ctx context.Context, request *api.SubscriptionRequest) (*api.SubscriptionAck, error) {
	alreadySubscribed := (*s.scalerHandler).SubscribeMetric(ctx, request.GetSubscriber(), request.GetMetricMetadata())
	return &api.SubscriptionAck{
		WasSubscribed: alreadySubscribed,
	}, nil
}

// UnsubscribeMetric returns true in SubscriptionAck.WasSubscribed is the metric has been successfully unsubscribed
// false if it wasn't subscribed
func (s *GrpcServer) UnsubscribeMetric(ctx context.Context, request *api.SubscriptionRequest) (*api.SubscriptionAck, error) {
	wasSubscribed := (*s.scalerHandler).UnsubscribeMetric(ctx, request.GetSubscriber(), request.GetMetricMetadata())
	return &api.SubscriptionAck{
		WasSubscribed: wasSubscribed,
	}, nil
}

// NewGrpcServer creates a new instance of GrpcServer
func NewGrpcServer(scaleHandler *scaling.ScaleHandler, address, certDir string, certsReady chan struct{}) GrpcServer {
	return GrpcServer{
		address:       address,
		scalerHandler: scaleHandler,
		certDir:       certDir,
		certsReady:    certsReady,
	}
}

func (s *GrpcServer) startServer() error {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Start starts a new gRPC Metrics Service, this implements Runnable interface
// of controller-runtime Manager, so we can use mgr.Add() to start this component.
func (s *GrpcServer) Start(ctx context.Context) error {
	<-s.certsReady
	if s.server == nil {
		creds, err := utils.LoadGrpcTLSCredentials(ctx, s.certDir, true)
		if err != nil {
			return err
		}

		grpcServerOpts := []grpc.ServerOption{
			grpc.Creds(creds),
		}

		if metricscollector.GetServerMetrics() != nil {
			grpcServerOpts = append(
				grpcServerOpts,
				grpc.ChainStreamInterceptor(metricscollector.GetServerMetrics().StreamServerInterceptor()),
				grpc.ChainUnaryInterceptor(metricscollector.GetServerMetrics().UnaryServerInterceptor()),
			)
		}

		s.server = grpc.NewServer(grpcServerOpts...)
		api.RegisterMetricsServiceServer(s.server, s)
		api.RegisterRawMetricsServiceServer(s.server, s)
	}

	errChan := make(chan error)

	go func() {
		log.Info("Starting Metrics Service gRPC Server", "address", s.address)
		if err := s.startServer(); err != nil {
			err := fmt.Errorf("unable to start Metrics Service gRPC server on address %s, error: %w", s.address, err)
			log.Error(err, "error starting Metrics Service gRPC server")
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return nil
	}
}

// NeedLeaderElection is needed to implement LeaderElectionRunnable interface
// of controller-runtime. This assures that the component is started/stoped
// when this particular instance is selected/deselected as a leader.
func (s *GrpcServer) NeedLeaderElection() bool {
	return true
}
