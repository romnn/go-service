package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/romnn/go-service/examples/reflect/gen"
	"github.com/romnn/go-service/pkg/grpc/reflect"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ReflectService implements the reflect service
type ReflectService struct {
	pb.UnimplementedReflectServer
}

func (s *ReflectService) getAnnotations(ctx context.Context) (*pb.Annotations, error) {
	info, ok := reflect.GetMethodInfo(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to get grpc method info")
	}
	var annotations pb.Annotations
	methodOptions := info.Method().Options()
	if boolValue, ok := proto.GetExtension(methodOptions, pb.E_BoolValue).(bool); ok {
		annotations.BoolValue = boolValue
	}
	if stringValue, ok := proto.GetExtension(methodOptions, pb.E_StringValue).(string); ok {
		annotations.StringValue = stringValue
	}
	if intValue, ok := proto.GetExtension(methodOptions, pb.E_IntValue).(int32); ok {
		annotations.IntValue = intValue
	}
	return &annotations, nil
}

// GetNoAnnotations returns the options of this GRPC method
func (s *ReflectService) GetNoAnnotations(ctx context.Context, req *pb.Empty) (*pb.Annotations, error) {
	return s.getAnnotations(ctx)
}

// GetAnnotations returns the options of this GRPC method
func (s *ReflectService) GetAnnotations(ctx context.Context, req *pb.Empty) (*pb.Annotations, error) {
	return s.getAnnotations(ctx)
}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	service := ReflectService{}
	registry := reflect.NewRegistry()
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(reflect.UnaryServerInterceptor(registry)),
		grpc.ChainStreamInterceptor(reflect.StreamServerInterceptor(registry)),
	)
	pb.RegisterReflectServer(server, &service)
	registry.Load(server)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdown
		log.Println("shutdown ...")
		server.GracefulStop()
		listener.Close()
	}()

	log.Printf("listening on: %v", listener.Addr())
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
