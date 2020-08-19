package gogrpcservice

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/uber/jaeger-lib/metrics/prometheus"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"

	jaegercfg "github.com/uber/jaeger-client-go/config"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Version is incremented using bump2version
const Version = "0.0.2"

// GrpcMethodName ...
type GrpcMethodName string
type grpcMethodDescriptor string

var (
	// GrpcMethodDescriptor ...
	GrpcMethodDescriptor = grpcMethodDescriptor("methodDesc")

	// NotReady ...
	NotReady   = status.Error(codes.Unavailable, "the service is currently unavailable")
	megabyte   = 1024 * 1024
	maxMsgSize = 500 * megabyte
)

// Service ...
type Service struct {
	Name         string
	Version      string
	BuildTime    string
	HTTPServer   *http.Server
	GrpcServer   *grpc.Server
	Health       *health.Server
	TracerCloser io.Closer
	Ready        bool
	Healthy      bool

	// Hooks
	PostBootstrapHook func(bs *Service) error
	ConnectHook       func(bs *Service) error

	methods map[GrpcMethodName]pref.MethodDescriptor
}

// GracefulStop ...
func (bs *Service) GracefulStop() {
	bs.Ready = false
	bs.Healthy = false
	log.Info("graceful shutdown")
	log.Info("stopping GRPC server")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	if bs.GrpcServer != nil {
		bs.GrpcServer.GracefulStop()
	}
	if bs.HTTPServer != nil {
		_ = bs.HTTPServer.Shutdown(ctx)
		_ = bs.HTTPServer.Close()
	}
	if bs.Health != nil {
		bs.Health.Shutdown()
	}
	if bs.TracerCloser != nil {
		_ = bs.TracerCloser.Close()
	}
}

// WrappedServerStream is a thin wrapper around grpc.ServerStream that allows modifying context.
type WrappedServerStream struct {
	grpc.ServerStream
	// WrappedContext is the wrapper's own Context. You can assign it.
	WrappedContext context.Context
}

// Context returns the wrapper's WrappedContext, overwriting the nested grpc.ServerStream.Context()
func (w *WrappedServerStream) Context() context.Context {
	return w.WrappedContext
}

// WrapServerStream returns a ServerStream that has the ability to overwrite context.
func WrapServerStream(stream grpc.ServerStream) *WrappedServerStream {
	if existing, ok := stream.(*WrappedServerStream); ok {
		return existing
	}
	return &WrappedServerStream{ServerStream: stream, WrappedContext: stream.Context()}
}

func (bs *Service) bootstrap(cliCtx *cli.Context) error {
	bs.ConfigureLogging(cliCtx)
	bs.SetHealthy(false)
	if bs.PostBootstrapHook != nil {
		return bs.PostBootstrapHook(bs)
	}
	return nil
}

func (bs *Service) injectMethodDescriptors(ctx context.Context, method string) context.Context {
	methodName := GrpcMethodName(method)
	if methodDesc, ok := bs.methods[methodName]; ok {
		// Add method descriptor to context
		return context.WithValue(ctx, GrpcMethodDescriptor, methodDesc)
	}
	return ctx
}

func (bs *Service) grpcUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	return handler(bs.injectMethodDescriptors(ctx, info.FullMethod), req)
}

func (bs *Service) grpcStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	wss := WrapServerStream(ss)
	wss.WrappedContext = bs.injectMethodDescriptors(ss.Context(), info.FullMethod)
	return handler(srv, wss)
}

// BootstrapGrpcOptions ...
type BootstrapGrpcOptions struct {
	USI grpc.UnaryServerInterceptor
	SSI grpc.StreamServerInterceptor
}

// BootstrapGrpc prepares a grpc service
func (bs *Service) BootstrapGrpc(cliCtx *cli.Context, opts *BootstrapGrpcOptions) error {
	usi := grpc.UnaryInterceptor(bs.grpcUnaryInterceptor)
	if opts != nil && opts.USI != nil {
		usi = grpc.ChainUnaryInterceptor(bs.grpcUnaryInterceptor, opts.USI)
	}
	ssi := grpc.StreamInterceptor(bs.grpcStreamInterceptor)
	if opts != nil && opts.SSI != nil {
		ssi = grpc.ChainStreamInterceptor(bs.grpcStreamInterceptor, opts.SSI)
	}

	bs.methods = make(map[GrpcMethodName]pref.MethodDescriptor)
	bs.GrpcServer = grpc.NewServer(
		usi,
		ssi,
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	)
	bs.SetupGrpcHealthCheck(cliCtx)
	return bs.bootstrap(cliCtx)
}

// ServeGrpc prepares an http service
func (bs *Service) ServeGrpc(listener net.Listener) error {
	// At this point, the service is registered and we can inspect the services
	for name, info := range bs.GrpcServer.GetServiceInfo() {
		file, ok := info.Metadata.(string)
		if !ok {
			return fmt.Errorf("service %q has unexpected metadata: expecting a string; got %v", name, info.Metadata)
		}
		fileDesc, err := preg.GlobalFiles.FindFileByPath(file)
		if err != nil {
			return err
		}
		services := fileDesc.Services()
		for i := 0; i < services.Len(); i++ {
			service := services.Get(i)
			methods := service.Methods()
			for i := 0; i < methods.Len(); i++ {
				method := methods.Get(i)
				methodName := GrpcMethodName(fmt.Sprintf("/%s/%s", service.FullName(), method.Name()))
				bs.methods[methodName] = method
			}
		}
	}
	return bs.GrpcServer.Serve(listener)
}

// BootstrapHTTP prepares an http service
func (bs *Service) BootstrapHTTP(cliCtx *cli.Context, handler *gin.Engine) error {
	bs.HTTPServer = &http.Server{Handler: handler}
	bs.SetupHTTPHealthCheck(cliCtx, handler)
	return bs.bootstrap(cliCtx)
}

// SetupGrpcHealthCheck ...
func (bs *Service) SetupGrpcHealthCheck(cliCtx *cli.Context) {
	bs.Health = health.NewServer()
	healthpb.RegisterHealthServer(bs.GrpcServer, bs.Health)
}

// SetupHTTPHealthCheck ...
func (bs *Service) SetupHTTPHealthCheck(cliCtx *cli.Context, handler *gin.Engine) {
	handler.GET("healthz", func(c *gin.Context) {
		if bs.Healthy {
			c.String(http.StatusOK, "ok")
		} else {
			c.String(http.StatusServiceUnavailable, "service is not available")
		}
	})
}

// SetHealthy ...
func (bs *Service) SetHealthy(healthy bool) {
	bs.Healthy = healthy
	if bs.Health != nil {
		if healthy {
			bs.Health.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
			bs.Health.SetServingStatus(bs.Name, healthpb.HealthCheckResponse_SERVING)
		} else {
			bs.Health.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
			bs.Health.SetServingStatus(bs.Name, healthpb.HealthCheckResponse_NOT_SERVING)
		}
	}
}

// Connect connects to databases and other services
func (bs *Service) Connect(cliCtx *cli.Context) error {
	var err error
	bs.TracerCloser, err = bs.ConfigureTracing(cliCtx)
	if err != nil {
		log.Warnf("could not initialize jaeger tracer: %s", err.Error())
	}
	if bs.ConnectHook != nil {
		return bs.ConnectHook(bs)
	}
	return nil
}

// Dial connects to an external GRPC service
func (bs *Service) Dial(cliCtx *cli.Context, host string, port int) (*grpc.ClientConn, error) {
	return grpc.Dial(
		fmt.Sprintf("%s:%d", host, port),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize), grpc.MaxCallSendMsgSize(maxMsgSize)),
		grpc.WithTimeout(time.Duration(5+cliCtx.Int("connection-timeout-sec"))*time.Second),
	)
}

// ConfigureLogging ...
func (bs *Service) ConfigureLogging(cliCtx *cli.Context) {
	level, err := log.ParseLevel(cliCtx.String("log"))
	if err != nil {
		log.Warnf("log level %q does not exist", cliCtx.String("log"))
		level = log.InfoLevel
	}
	log.SetLevel(level)
}

// ConfigureTracing ...
func (bs *Service) ConfigureTracing(cliCtx *cli.Context) (io.Closer, error) {
	cfg := jaegercfg.Configuration{}
	metricsFactory := prometheus.New()
	closer, err := cfg.InitGlobalTracer(
		"upbound",
		jaegercfg.Metrics(metricsFactory),
	)
	if err != nil {
		return nil, err
	}
	return closer, nil
}
