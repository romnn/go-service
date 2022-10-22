package jaeger

import (
	"io"
	"regexp"
	"time"

	"github.com/opentracing/opentracing-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
)

// SafeServiceName returns a sanitized ASCII service name
func SafeServiceName(name string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9]+")
	return re.ReplaceAllString(name, "")
}

// DefaultJaegerTracer sets a default jaeger tracer
func DefaultJaegerTracer(serviceName, agentHost string) (opentracing.Tracer, io.Closer, error) {
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  agentHost,
		},
	}

	tracer, closer, err := cfg.New(serviceName)
	if err != nil {
		return nil, nil, err
	}
	opentracing.SetGlobalTracer(tracer)
	return tracer, closer, nil
}
