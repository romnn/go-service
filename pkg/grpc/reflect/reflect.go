package reflect

import (
	"context"
	"fmt"

	grpcutils "github.com/romnn/go-service/pkg/grpc"
	"google.golang.org/grpc"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"
)

// GrpcMethodName is the type alias for
// type GrpcMethodName string

// type grpcMethodInfo string
type methodInfoKey struct{}

// // GrpcMethodInfo is the index for the method info
// const (
// 	GrpcMethodInfo = grpcMethodInfo("methodInfo")
// )

// type ServerInfo interface {
// 	GetFullMethod() string
// }

// type StreamServerInfo grpc.StreamServerInfo

// func (info StreamServerInfo) GetFullMethod() string {
// 	return info.FullMethod
// }

// type UnaryServerInfo grpc.UnaryServerInfo

// func (info UnaryServerInfo) GetFullMethod() string {
// 	return info.FullMethod
// }

// Registry provides efficient access to method and service descriptors for grpc servers
type Registry interface {
	Load(srv *grpc.Server) error
	LoadFile(file string) error
	GetMethodInfo(name string) (MethodInfo, bool)
	MustGetMethodInfo(name string) MethodInfo
}

// NewRegistry creates a new method info registry
func NewRegistry() Registry {
	return &registry{
		methods: make(map[string]*methodInfo),
	}
}

// GetMethodInfo gets the method info from the registry by name
func (r *registry) GetMethodInfo(name string) (MethodInfo, bool) {
	info, ok := r.methods[name]
	return info, ok
}

func (r *registry) MustGetMethodInfo(name string) MethodInfo {
	info, ok := r.methods[name]
	if !ok {
		err := fmt.Errorf("no method %q found in registry, did you forget to call registry.Load(&server)?", name)
		panic(err)
	}
	return info
	// if info, ok := r.methods[info.GetFullMethod()]; ok {
	// 	return info
	// }
	// grpcServer, ok := server.(grpc.Server)
	// if !ok {
	// 	panic("is not a gprc server!")
	// }
	// r.Load(&grpcServer)
	// return r.methods[info.GetFullMethod()]
	// return r.methods[name]
}

type methodInfo struct {
	service pref.ServiceDescriptor
	method  pref.MethodDescriptor
}

// Method provides the service descriptor
func (info *methodInfo) Service() pref.ServiceDescriptor {
	return info.service
}

// Method provides the method descriptor
func (info *methodInfo) Method() pref.MethodDescriptor {
	return info.method
}

// MethodInfo provides service and method descriptors
type MethodInfo interface {
	Service() pref.ServiceDescriptor
	Method() pref.MethodDescriptor
}

type registry struct {
	methods map[string]*methodInfo
}

// Load loads service and method definitions for a grpc server
func (r *registry) Load(server *grpc.Server) error {
	for name, info := range server.GetServiceInfo() {
		file, ok := info.Metadata.(string)
		if !ok {
			return fmt.Errorf("service %q has unexpected metadata (expected string, got %v)", name, info.Metadata)
		}
		if err := r.LoadFile(file); err != nil {
			return err
		}
	}
	return nil
}

// LoadFile loads service and method definitions from proto file
func (r *registry) LoadFile(file string) error {
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
			methodName := fmt.Sprintf("/%s/%s", service.FullName(), method.Name())
			r.methods[methodName] = &methodInfo{
				method:  method,
				service: service,
			}
		}
	}
	return nil
}

// WithMethodInfo injects method info into context
func WithMethodInfo(ctx context.Context, info MethodInfo) context.Context {
	return context.WithValue(ctx, methodInfoKey{}, info)
}

// GetMethodInfo extracts method info from context
func GetMethodInfo(ctx context.Context) (MethodInfo, bool) {
	info, ok := ctx.Value(methodInfoKey{}).(MethodInfo)
	return info, ok
}

// UnaryServerInterceptor returns a new unary server interceptors that lazily injects server and method information
func UnaryServerInterceptor(reg Registry) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

		// for name, info := range info.Server.GetServiceInfo() {
		//   file, ok := info.Metadata.(string)
		//   if !ok {
		//     return fmt.Errorf("service %q has unexpected metadata: expecting a string; got %v", name, info.Metadata)
		//   }
		//   fileDesc, err := preg.GlobalFiles.FindFileByPath(file)
		//   if err != nil {
		//     return err
		//   }
		//   services := fileDesc.Services()
		//   for i := 0; i < services.Len(); i++ {
		//     service := services.Get(i)
		//     methods := service.Methods()
		//     for i := 0; i < methods.Len(); i++ {
		//       method := methods.Get(i)
		//       methodName := GrpcMethodName(fmt.Sprintf("/%s/%s", service.FullName(), method.Name()))
		//       bs.methods[methodName] = method
		//     }
		//   }
		// }

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

		// newCtx := WithMethodInfo(ctx, reg.GetMethodInfo(info.Server, UnaryServerInfo(*info)))
		newCtx := WithMethodInfo(ctx, reg.MustGetMethodInfo(info.FullMethod))
		return handler(newCtx, req)
		// mi :=
		// if minfo, ok := reg.GetMethodInfo(info); ok {
		// 	newCtx := WithMethodInfo(ctx, minfo)
		// 	return handler(newCtx, req)
		// }

		// return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a new unary server interceptors that lazily injects server and method information
func StreamServerInterceptor(reg Registry) grpc.StreamServerInterceptor {
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
		ctx := stream.Context()
		// newCtx := WithMethodInfo(ctx, reg.MustGetMethodInfo(srv, StreamServerInfo(*info)))
		newCtx := WithMethodInfo(ctx, reg.MustGetMethodInfo(info.FullMethod))
		wrapped := grpcutils.WrapServerStream(stream)
		wrapped.WrappedContext = newCtx
		return handler(srv, wrapped)
	}
}

// if info, ok := ctx.Value(GrpcMethodInfo).(MethodInfo); ok {
// 	return info
// }
// return nil

// methods      map[GrpcMethodName]pref.MethodDescriptor

// func (bs *Service) injectMethodDescriptors(ctx context.Context, method string) context.Context {
// 	methodName := GrpcMethodName(method)
// 	if methodDesc, ok := bs.methods[methodName]; ok {
// 		// Add method descriptor to context
// 		return context.WithValue(ctx, GrpcMethodDescriptor, methodDesc)
// 	}
// 	return ctx
// }

// InspectService injects metadata about the GPRC service to be used for tracing
// func (bs *Service) InspectService() error {
// 	// At this point, the service is registered and we can inspect the services
// 	for name, info := range bs.GrpcServer.GetServiceInfo() {
// 		file, ok := info.Metadata.(string)
// 		if !ok {
// 			return fmt.Errorf("service %q has unexpected metadata: expecting a string; got %v", name, info.Metadata)
// 		}
// 		fileDesc, err := preg.GlobalFiles.FindFileByPath(file)
// 		if err != nil {
// 			return err
// 		}
// 		services := fileDesc.Services()
// 		for i := 0; i < services.Len(); i++ {
// 			service := services.Get(i)
// 			methods := service.Methods()
// 			for i := 0; i < methods.Len(); i++ {
// 				method := methods.Get(i)
// 				methodName := GrpcMethodName(fmt.Sprintf("/%s/%s", service.FullName(), method.Name()))
// 				bs.methods[methodName] = method
// 			}
// 		}
// 	}
// 	return nil
// }
