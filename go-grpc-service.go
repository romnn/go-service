package gogrpcservice

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"sync"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	"github.com/labstack/echo/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics/prometheus"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"

	jaegermw "github.com/labstack/echo-contrib/jaegertracing"
	prommw "github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4/middleware"
	glog "github.com/labstack/gommon/log"
	logmiddleware "github.com/neko-neko/echo-logrus/v2"
	log "github.com/neko-neko/echo-logrus/v2/log"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Version is incremented using bump2version
const Version = "0.0.10"

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

	logMux    = sync.Mutex{}
	healthMux = sync.Mutex{}
	tracerMux = sync.Mutex{}
)

// Service ...
type Service struct {
	Name                     string
	ShortName                string
	Version                  string
	BuildTime                string
	Echo                     *echo.Echo
	MetricsHTTPServer        *http.ServeMux
	HTTPServer               *http.Server
	GrpcServer               *grpc.Server
	Health                   *health.Server
	MonitoringHTTPMiddleware *prommw.Prometheus
	Tracer                   opentracing.Tracer
	TracerCloser             io.Closer
	Ready                    bool
	Healthy                  bool

	// Hooks
	PostBootstrapHook func(bs *Service) error
	ConnectHook       func(bs *Service) error

	// Configuration
	GrpcMetricsPort         uint
	GrpcMetricsURL          string
	HTTPHealthCheckURL      string
	JaegerAgentHost         string
	JaegerAgentPort         uint
	JaegerSamplingServerURL string

	tracingSetup sync.WaitGroup
	methods      map[GrpcMethodName]pref.MethodDescriptor
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
	if bs.Echo != nil {
		_ = bs.Echo.Shutdown(ctx)
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

// Bootstrap ...
func (bs *Service) Bootstrap(cliCtx *cli.Context) error {
	if bs.ShortName == "" {
		bs.ShortName = bs.Name
	}
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
	USI []grpc.UnaryServerInterceptor
	SSI []grpc.StreamServerInterceptor
}

// BootstrapGrpc prepares a grpc service
func (bs *Service) BootstrapGrpc(ctx context.Context, cliCtx *cli.Context, opts *BootstrapGrpcOptions) error {
	usi := []grpc.UnaryServerInterceptor{
		bs.grpcUnaryInterceptor,
		grpc_prometheus.UnaryServerInterceptor,
		grpc_opentracing.UnaryServerInterceptor(grpc_opentracing.WithTracer(bs.Tracer)),
	}
	ssi := []grpc.StreamServerInterceptor{
		bs.grpcStreamInterceptor,
		grpc_prometheus.StreamServerInterceptor,
		grpc_opentracing.StreamServerInterceptor(grpc_opentracing.WithTracer(bs.Tracer)),
	}
	if opts != nil && opts.USI != nil {
		usi = append(usi, opts.USI...)
	}
	if opts != nil && opts.SSI != nil {
		ssi = append(ssi, opts.SSI...)
	}

	bs.methods = make(map[GrpcMethodName]pref.MethodDescriptor)
	bs.GrpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor(usi...),
		grpc.ChainStreamInterceptor(ssi...),
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	)
	bs.SetupGrpcMonitoring(ctx)
	bs.SetupGrpcHealthCheck(ctx)
	return bs.Bootstrap(cliCtx)
}

// ServeGrpc serves an http service
func (bs *Service) ServeHTTP(listener net.Listener) error {
	bs.tracingSetup.Wait()

	// Hook up middlewares
	if bs.Tracer != nil {
		bs.SetupHTTPTracingMiddleware()
	}
	bs.SetupHTTPMonitoringMiddleware()

	if err := bs.HTTPServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// ServeGrpc serves a grpc service
func (bs *Service) ServeGrpc(listener net.Listener) error {
	bs.tracingSetup.Wait()

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
func (bs *Service) BootstrapHTTP(ctx context.Context, cliCtx *cli.Context, handler *echo.Echo, mws []echo.MiddlewareFunc) error {
	handler.HideBanner = true
	handler.Logger = log.Logger()
	for _, mw := range append(mws, []echo.MiddlewareFunc{logmiddleware.Logger(), middleware.Recover()}...) {
		handler.Use(mw)
	}
	bs.Echo = handler
	bs.HTTPServer = &http.Server{Handler: handler}
	bs.SetupHTTPHealthCheck(ctx, handler, bs.HTTPHealthCheckURL)
	return bs.Bootstrap(cliCtx)
}

func safeName(s string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return "unknown"
	}
	return reg.ReplaceAllString(s, "")
}

// SetupHTTPMonitoringMiddleware ...
func (bs *Service) SetupHTTPMonitoringMiddleware() {
	bs.MonitoringHTTPMiddleware = prommw.NewPrometheus(safeName(bs.ShortName), nil)
	bs.MonitoringHTTPMiddleware.Use(bs.Echo)
}

// SetupHTTPTracingMiddleware ...
func (bs *Service) SetupHTTPTracingMiddleware() {
	bs.Echo.Use(jaegermw.TraceWithConfig(jaegermw.TraceConfig{
		Tracer:  bs.Tracer,
		Skipper: nil,
	}))
}

// SetupGrpcMonitoring ...
func (bs *Service) SetupGrpcMonitoring(ctx context.Context) {
	grpc_prometheus.Register(bs.GrpcServer)
	bs.MetricsHTTPServer = http.NewServeMux()
	if bs.GrpcMetricsURL == "" {
		bs.GrpcMetricsURL = "/metrics"
	}
	if bs.GrpcMetricsPort == 0 {
		bs.GrpcMetricsPort = 9000
	}
	bs.MetricsHTTPServer.Handle(bs.GrpcMetricsURL, promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", bs.GrpcMetricsPort), bs.MetricsHTTPServer); err != nil {
			logMux.Lock()
			// we need to lock to make sure the logger is not configured concurrently
			log.Errorf("failed to serve metrics: %v", err)
			logMux.Unlock()
		}
	}()
}

// SetupGrpcHealthCheck ...
func (bs *Service) SetupGrpcHealthCheck(ctx context.Context) {
	bs.Health = health.NewServer()
	healthpb.RegisterHealthServer(bs.GrpcServer, bs.Health)
}

// SetupHTTPHealthCheck ...
func (bs *Service) SetupHTTPHealthCheck(ctx context.Context, handler *echo.Echo, url string) {
	if url == "" {
		url = "/healthz"
	}
	handler.GET(url, func(c echo.Context) error {
		if bs.Healthy {
			c.String(http.StatusOK, "ok")
		} else {
			c.String(http.StatusServiceUnavailable, "service is not available")
		}
		return nil
	})
}

// SetHealthy ...
func (bs *Service) SetHealthy(healthy bool) {
	healthMux.Lock()
	defer healthMux.Unlock()
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

// SetLogLevel ...
func (bs *Service) SetLogLevel(level logrus.Level) {
	logMux.Lock()
	defer logMux.Unlock()
	logrus.SetLevel(level)
	if l, ok := map[logrus.Level]glog.Lvl{
		logrus.DebugLevel: glog.DEBUG,
		logrus.InfoLevel:  glog.INFO,
		logrus.WarnLevel:  glog.WARN,
		logrus.ErrorLevel: glog.ERROR,
	}[level]; ok {
		log.Logger().SetLevel(l)
	}
}

// SetLogFormat ...
func (bs *Service) SetLogFormat(format logrus.Formatter) {
	logMux.Lock()
	defer logMux.Unlock()
	// e.g. &log.JSONFormatter{TimestampFormat: time.RFC3339}
	log.Logger().SetFormatter(format)
}

// Connect connects to databases and other services
func (bs *Service) Connect(cliCtx *cli.Context) error {
	bs.tracingSetup.Add(1)
	if err := bs.ConfigureTracing(cliCtx); err != nil {
		log.Warnf("could not initialize jaeger tracer: %s", err.Error())
	}

	if bs.ConnectHook != nil {
		return bs.ConnectHook(bs)
	}
	return nil
}

// DialOptions ...
type DialOptions struct {
	TimeoutSec int
}

// Dial connects to an external GRPC service
func (bs *Service) Dial(ctx context.Context, host string, port uint, opts *DialOptions) (*grpc.ClientConn, error) {
	if opts == nil {
		opts = &DialOptions{TimeoutSec: 5}
	}
	return grpc.Dial(
		fmt.Sprintf("%s:%d", host, port),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize), grpc.MaxCallSendMsgSize(maxMsgSize)),
		grpc.WithTimeout(time.Duration(5+opts.TimeoutSec)*time.Second),
		// prometheus
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
		// tracing
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(
			grpc_opentracing.StreamClientInterceptor(grpc_opentracing.WithTracer(bs.Tracer)),
		)),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
			grpc_opentracing.UnaryClientInterceptor(grpc_opentracing.WithTracer(bs.Tracer)),
		)),
	)
}

// ConfigureLogging ...
func (bs *Service) ConfigureLogging(cliCtx *cli.Context) {
	if cliCtx != nil {
		level, err := logrus.ParseLevel(cliCtx.String("log"))
		if err != nil {
			log.Warnf("log level %q does not exist", cliCtx.String("log"))
			level = logrus.InfoLevel
		}
		bs.SetLogLevel(level)
		return
	}
	bs.SetLogLevel(logrus.InfoLevel)
}

// ConfigureTracing ...
func (bs *Service) ConfigureTracing(cliCtx *cli.Context) error {
	defer bs.tracingSetup.Done()
	tracerMux.Lock()
	defer tracerMux.Unlock()
	metricsFactory := prometheus.New()

	name := safeName(bs.ShortName)
	cfg := jaegercfg.Configuration{
		ServiceName: name,
		Sampler: &jaegercfg.SamplerConfig{
			Type:              "const",
			Param:             1,
			SamplingServerURL: bs.JaegerSamplingServerURL,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  fmt.Sprintf("%s:%d", bs.JaegerAgentHost, bs.JaegerAgentPort),
		},
	}

	var err error
	bs.Tracer, bs.TracerCloser, err = cfg.New(name, jaegercfg.Metrics(metricsFactory))
	if err != nil {
		return err
	}
	opentracing.SetGlobalTracer(bs.Tracer)
	return nil
}
