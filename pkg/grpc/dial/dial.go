package dial

// import (
// 	"context"
// 	"fmt"
// 	"time"

// 	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
// 	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
// 	"google.golang.org/grpc"
// )

// // DialOptions ...
// type DialOptions struct {
// 	TimeoutSec int
// }

// // Dial connects to an external GRPC service
// func (bs *Service) Dial(ctx context.Context, host string, port uint, path string, opts *DialOptions) (*grpc.ClientConn, error) {
// 	if opts == nil {
// 		opts = &DialOptions{TimeoutSec: 5}
// 	}
// 	return grpc.Dial(
// 		fmt.Sprintf("%s:%d%s", host, port, path),
// 		grpc.WithInsecure(),
// 		grpc.WithBlock(),
// 		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize), grpc.MaxCallSendMsgSize(maxMsgSize)),
// 		grpc.WithTimeout(time.Duration(5+opts.TimeoutSec)*time.Second),
// 		// prometheus
// 		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
// 		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
// 		// tracing
// 		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(
// 			grpc_opentracing.StreamClientInterceptor(grpc_opentracing.WithTracer(bs.Tracer)),
// 		)),
// 		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(
// 			grpc_opentracing.UnaryClientInterceptor(grpc_opentracing.WithTracer(bs.Tracer)),
// 		)),
// 	)
// }
