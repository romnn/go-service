package main

import (
	"context"

	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	jaegermw "github.com/labstack/echo-contrib/jaegertracing"
	prometheusmw "github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	logrusmw "github.com/neko-neko/echo-logrus/v2"
	"github.com/romnn/go-service/pkg/http/health/echo"
	"github.com/romnn/go-service/pkg/jaeger"
	log "github.com/sirupsen/logrus"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// HTTPService implements a HTTP service
type HTTPService struct {
	Server *http.Server
	Health *health.Health
}

// GreetingRoute sends a JSON encoded greetign message
func (s *HTTPService) GreetingRoute(c echo.Context) error {
	s.Health.SetServingStatus(healthpb.HealthCheckResponse_SERVING)
	json := map[string]string{"message": "welcome"}
	return c.JSONPretty(http.StatusOK, json, "  ")
}

func main() {
	serviceName := "service_name"
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	tracer, _, err := jaeger.DefaultJaegerTracer(serviceName, "jaeger-agent:3000")
	if err != nil {
		log.Fatalf("failed to setup jaeger tracer: %v", err)
	}

	service := HTTPService{}

	e := echo.New()
	e.GET("/", service.GreetingRoute)
	e.HideBanner = true
	middlewares := []echo.MiddlewareFunc{
		jaegermw.TraceWithConfig(jaegermw.TraceConfig{
			Tracer:  tracer,
			Skipper: nil,
		}),
		logrusmw.Logger(),
		middleware.Recover(),
	}
	for _, mw := range middlewares {
		e.Use(mw)
	}

	metrics := prometheusmw.NewPrometheus(serviceName, nil)
	metrics.Use(e)

	// service.Health = health.Health{}
	service.Health = health.Use(e, "/healthz")

	// e.GET("/healthz", service.Health.HandlerFunc())
	// Use(e, "/healthz")
	service.Server = &http.Server{
		Handler: e,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdown
		log.Println("shutdown ...")
		service.Server.Shutdown(context.Background())
		listener.Close()
	}()

	log.Printf("listening on: %v", listener.Addr())
	err = service.Server.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to serve: %v", err)
	}
}
