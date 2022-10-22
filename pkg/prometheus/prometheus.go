package prometheus

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

func NewMetricsServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}

func NewMetricsServer(addr string) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: NewMetricsServeMux(),
	}
}
