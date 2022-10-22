package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/romnn/go-service/examples/grpc/gen"
	"github.com/romnn/go-service/pkg/grpc/reflect"
	"github.com/romnn/go-service/pkg/jaeger"
	"github.com/romnn/go-service/pkg/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	opentracing "github.com/opentracing/opentracing-go"
	tracelog "github.com/opentracing/opentracing-go/log"
	log "github.com/sirupsen/logrus"
)

// GrpcService implements the gRPC service
type GrpcService struct {
	pb.UnimplementedGrpcServer
	Health  *health.Server
	Metrics *http.Server
}

// Get returns a sample response
func (s *GrpcService) Get(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	info, ok := reflect.GetMethodInfo(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to get grpc method info")
	}
	methodName := string(info.Method().Name())
	serviceName := string(info.Service().Name())
	log.Infof("called method %q of service %q", methodName, serviceName)

	span, ctx := opentracing.StartSpanFromContext(ctx, methodName)
	defer span.Finish()

	s.Health.SetServingStatus(serviceName, healthpb.HealthCheckResponse_SERVING)

	span.SetTag("sample-tag", "test")
	span.LogFields(tracelog.Object("request", req))

	return &pb.Response{Value: "Hello World"}, nil
}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	megabyte := 1024 * 1024
	maxMsgSize := 500 * megabyte

	tracer, _, err := jaeger.DefaultJaegerTracer("service-name", "jaeger-agent:3000")
	if err != nil {
		log.Fatalf("failed to setup jaeger tracer: %v", err)
	}

	service := GrpcService{
		Health:  health.NewServer(),
		Metrics: prometheus.NewMetricsServer(":9000"),
	}
	registry := reflect.NewRegistry()

	unaryInterceptors := []grpc.UnaryServerInterceptor{
		reflect.UnaryServerInterceptor(registry),
		grpc_prometheus.UnaryServerInterceptor,
		grpc_opentracing.UnaryServerInterceptor(grpc_opentracing.WithTracer(tracer)),
	}
	streamInterceptors := []grpc.StreamServerInterceptor{
		reflect.StreamServerInterceptor(registry),
		grpc_prometheus.StreamServerInterceptor,
		grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(tracer)),
	}
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	)
	pb.RegisterGrpcServer(server, &service)
	registry.Load(server)

	healthpb.RegisterHealthServer(server, service.Health)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdown
		log.Println("shutdown ...")
		server.GracefulStop()
		listener.Close()
		service.Metrics.Shutdown(context.Background())
	}()

	go func() {
		_ = service.Metrics.ListenAndServe()
	}()

	log.Printf("listening on: %v", listener.Addr())
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
