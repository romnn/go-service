package reflect

import (
	"context"
	"fmt"

	grpcutils "github.com/romnn/go-service/pkg/grpc"
	"google.golang.org/grpc"
	pref "google.golang.org/protobuf/reflect/protoreflect"
	preg "google.golang.org/protobuf/reflect/protoregistry"
)

type methodInfoKey struct{}

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

// MustGetMethodInfo gets the method info from the registry by name and panics if the method does not exist
func (r *registry) MustGetMethodInfo(name string) MethodInfo {
	info, ok := r.methods[name]
	if !ok {
		err := fmt.Errorf("no method %q in registry, did you call registry.Load(&server)?", name)
		panic(err)
	}
	return info
}

// MethodInfo provides service and method descriptors
type MethodInfo interface {
	Service() pref.ServiceDescriptor
	Method() pref.MethodDescriptor
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

// UnaryServerInterceptor returns an interceptor that injects server and method info into the request context
func UnaryServerInterceptor(reg Registry) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		newCtx := WithMethodInfo(ctx, reg.MustGetMethodInfo(info.FullMethod))
		return handler(newCtx, req)
	}
}

// StreamServerInterceptor returns an interceptor that injects server and method info into the stream context
func StreamServerInterceptor(reg Registry) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		newCtx := WithMethodInfo(ctx, reg.MustGetMethodInfo(info.FullMethod))
		wrapped := grpcutils.WrapServerStream(stream)
		wrapped.WrappedContext = newCtx
		return handler(srv, wrapped)
	}
}
