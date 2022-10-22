package health

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// RegisterHealthServer ...
func RegisterHealthServer(server *grpc.Server, healthServer *health.Server) {
	healthpb.RegisterHealthServer(server, healthServer)
}
