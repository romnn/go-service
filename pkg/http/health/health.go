package health

import (
	"net/http"
	"sync"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Health implements a basic health check
type Health struct {
	healthy bool
	mux     sync.RWMutex
}

// Healthy returns `true` if the service is healthy, or `false` otherwise
func (health *Health) Healthy() bool {
	health.mux.RLock()
	defer health.mux.RUnlock()
	return health.healthy
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

// ServeHTTP implements a `http.Handler`
// implements https://pkg.go.dev/net/http#Handler
func (health *Health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if health.Healthy() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("service is not available"))
	}
}
