package main

import (
	"context"
	"net"
	"testing"
	"time"

	pb "github.com/romnn/go-service/examples/reflect/gen"
	"github.com/romnn/go-service/pkg/grpc/reflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

const (
	bufSize = 1024 * 1024
)

type DialerFunc = func(string, time.Duration) (net.Conn, error)

func dailerFor(listener *bufconn.Listener) DialerFunc {
	return func(string, time.Duration) (net.Conn, error) {
		return listener.Dial()
	}
}

type test struct {
	conn    *grpc.ClientConn
	service *ReflectService
	server  *grpc.Server
	client  pb.ReflectClient
}

func (test *test) setup(t *testing.T) *test {
	var err error
	t.Parallel()

	listener := bufconn.Listen(bufSize)
	test.service = &ReflectService{}

	registry := reflect.NewRegistry()
	test.server = grpc.NewServer(
		grpc.ChainUnaryInterceptor(reflect.UnaryServerInterceptor(registry)),
		grpc.ChainStreamInterceptor(reflect.StreamServerInterceptor(registry)),
	)
	pb.RegisterReflectServer(test.server, test.service)
	registry.Load(test.server)

	go func() {
		if err := test.server.Serve(listener); err != nil {
			t.Fatalf("failed to serve service: %v", err)
		}
	}()

	test.conn, err = grpc.Dial(
		"bufnet",
		grpc.WithDialer(dailerFor(listener)),
		grpc.WithInsecure(),
		grpc.WithTimeout(20*time.Second),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("failed to dial grpc service: %v", err)
	}

	test.client = pb.NewReflectClient(test.conn)
	return test
}

func (test *test) teardown() {
	if test.conn != nil {
		_ = test.conn.Close()
	}
	if test.server != nil {
		test.server.GracefulStop()
	}
}

func TestReflectNoAnnotations(t *testing.T) {
	test := new(test).setup(t)
	defer test.teardown()

	noAnnotations, err := test.client.GetNoAnnotations(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("failed to get annotations: %v", err)
	}
	if !proto.Equal(&pb.Annotations{}, noAnnotations) {
		t.Errorf("not equal: expected %v but got %v", &pb.Annotations{}, noAnnotations)
	}
}

func TestReflectAnnotations(t *testing.T) {
	test := new(test).setup(t)
	defer test.teardown()

	annotations, err := test.client.GetAnnotations(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("failed to get annotations: %v", err)
	}
	expected := &pb.Annotations{
		BoolValue:   true,
		StringValue: "Hello World",
		IntValue:    42,
	}
	if !proto.Equal(expected, annotations) {
		t.Errorf("not equal: expected %v but got %v", expected, annotations)
	}
}
