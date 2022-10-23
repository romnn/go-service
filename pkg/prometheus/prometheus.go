package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewMetricsServeMux returns a new http.ServeMux that serves prometheus metrics at /metrics
func NewMetricsServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}

// NewMetricsServer returns a http.Server that serves prometheus metrics under /metrics
func NewMetricsServer(addr string) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: NewMetricsServeMux(),
	}
}
