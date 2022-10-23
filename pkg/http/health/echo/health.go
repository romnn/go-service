package health

import (
	"github.com/romnn/go-service/pkg/http/health"

	"github.com/labstack/echo/v4"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Health implements a basic health check for echo by wrapping the generic health check
type Health struct {
	health *health.Health
}

// HandlerFunc returns an echo `HandlerFunc` for this health check
// implements https://pkg.go.dev/github.com/labstack/echo/v4#HandlerFunc
func (health *Health) HandlerFunc() func(c echo.Context) error {
	return echo.WrapHandler(health.health)
}

// SetServingStatus updates the serving status
func (health *Health) SetServingStatus(status healthpb.HealthCheckResponse_ServingStatus) {
	health.health.SetServingStatus(status)
}

// Use registers a new health check handler and returns it
func Use(e *echo.Echo, url string) *Health {
	health := &Health{}
	e.GET(url, health.HandlerFunc())
	return health
}
