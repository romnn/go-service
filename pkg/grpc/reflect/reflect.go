package reflect

import (
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"
)

// GrpcMethodName ...
type GrpcMethodName string
type grpcMethodDescriptor string

// GrpcMethodDescriptor ...
const (
	GrpcMethodDescriptor = grpcMethodDescriptor("methodDesc")
)

// methods      map[GrpcMethodName]pref.MethodDescriptor

func (bs *Service) injectMethodDescriptors(ctx context.Context, method string) context.Context {
	methodName := GrpcMethodName(method)
	if methodDesc, ok := bs.methods[methodName]; ok {
		// Add method descriptor to context
		return context.WithValue(ctx, GrpcMethodDescriptor, methodDesc)
	}
	return ctx
}

// InspectService injects metadata about the GPRC service to be used for tracing
func (bs *Service) InspectService() error {
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
	return nil
}

// UnaryServerInterceptor returns a new unary server interceptors that lazily injects server and method information
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// var newCtx context.Context
		// var err error
		// if overrideSrv, ok := info.Server.(ServiceAuthFuncOverride); ok {
		// 	newCtx, err = overrideSrv.AuthFuncOverride(ctx, info.FullMethod)
		// } else {
		// 	newCtx, err = authFunc(ctx)
		// }
		// if err != nil {
		// 	return nil, err
		// }
		// return handler(newCtx, req)
	}
}

// StreamServerInterceptor returns a new unary server interceptors that lazily injects server and method information
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// var newCtx context.Context
		// var err error
		// if overrideSrv, ok := srv.(ServiceAuthFuncOverride); ok {
		// 	newCtx, err = overrideSrv.AuthFuncOverride(stream.Context(), info.FullMethod)
		// } else {
		// 	newCtx, err = authFunc(stream.Context())
		// }
		// if err != nil {
		// 	return err
		// }
		// wrapped := grpc_middleware.WrapServerStream(stream)
		// wrapped.WrappedContext = newCtx
		// return handler(srv, wrapped)
	}
}
