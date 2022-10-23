package health

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Health implements a basic health check
type Health struct {
	healthy bool
	mux     sync.RWMutex
}

// SetServingStatus updates the serving status
func (health *Health) SetServingStatus(status healthpb.HealthCheckResponse_ServingStatus) {
	health.mux.Lock()
	defer health.mux.Unlock()
	switch status {
	case healthpb.HealthCheckResponse_SERVING:
		health.healthy = true
	case healthpb.HealthCheckResponse_NOT_SERVING:
		health.healthy = false
	}
}

// Use returns a new health check handler
func Use(e *echo.Echo, url string) *Health {
	health := &Health{}
	e.GET(url, func(c echo.Context) error {
		health.mux.RLock()
		defer health.mux.RUnlock()
		if health.healthy {
			c.String(http.StatusOK, "ok")
		} else {
			c.String(http.StatusServiceUnavailable, "service is not available")
		}
		return nil
	})
	return health
}
