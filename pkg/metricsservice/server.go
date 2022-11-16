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
	"k8s.io/metrics/pkg/apis/external_metrics/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kedacore/keda/v2/pkg/metricsservice/api"
	"github.com/kedacore/keda/v2/pkg/scaling"
)

var log = ctrl.Log.WithName("grpc_server")

type GrpcServer struct {
	server        *grpc.Server
	address       string
	scalerHandler *scaling.ScaleHandler
	api.UnimplementedMetricsServiceServer
}

// GetMetrics returns metrics values in form of ExternalMetricValueList for specified ScaledObject reference
func (s *GrpcServer) GetMetrics(ctx context.Context, in *api.ScaledObjectRef) (*v1beta1.ExternalMetricValueList, error) {
	// TODO hit the metrics cache here first

	cache, err := (*s.scalerHandler).GetScalersCacheForScaledObject(ctx, in.Name, in.Namespace)
	// TODO fix Prom metrics recorder
	// metricsServer.RecordScalerObjectError(scaledObject.Namespace, scaledObject.Name, err)
	if err != nil {
		return nil, fmt.Errorf("error when getting scalers %s", err)
	}

	v1beta1ExtMetrics := &v1beta1.ExternalMetricValueList{}
	extMetrics, err := (*s.scalerHandler).GetExternalMetricsValuesList(ctx, cache, &cache.ScaledObject, in.MetricName)
	if err != nil {
		return nil, fmt.Errorf("error when getting metric values %s", err)
	}

	err = v1beta1.Convert_external_metrics_ExternalMetricValueList_To_v1beta1_ExternalMetricValueList(extMetrics, v1beta1ExtMetrics, nil)
	if err != nil {
		return nil, fmt.Errorf("error when converting metric values %s", err)
	}

	log.V(1).WithValues("scaledObjectName", in.Name, "scaledObjectNamespace", in.Namespace, "metrics", v1beta1ExtMetrics).Info("Providing metrics")

	return v1beta1ExtMetrics, nil
}

// NewGrpcServer creates a new instance of GrpcServer
func NewGrpcServer(scaleHandler *scaling.ScaleHandler, address string) GrpcServer {
	gsrv := grpc.NewServer()
	srv := GrpcServer{
		server:        gsrv,
		address:       address,
		scalerHandler: scaleHandler,
	}

	api.RegisterMetricsServiceServer(gsrv, &srv)
	return srv
}

func (s *GrpcServer) startServer() error {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}

// Start starts a new gRPC Metrics Service, this implements Runnable interface
// of controller-runtime Manager, so we can use mgr.Add() to start this component.
func (s *GrpcServer) Start(ctx context.Context) error {
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
