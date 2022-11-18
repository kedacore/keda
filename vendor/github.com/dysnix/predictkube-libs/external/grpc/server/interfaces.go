package server

import (
	pb "google.golang.org/grpc/health/grpc_health_v1"
)

type Server interface {
	Start() <-chan error
	Stop() error
}

type Health interface {
	Server
	pb.HealthServer
}
