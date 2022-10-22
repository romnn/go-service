package main

import (
	"context"
	// "errors"
	// "fmt"
	// "io/ioutil"
	"net"
	"testing"
	"time"

	// "github.com/dgrijalva/jwt-go"
	// "github.com/romnn/go-grpc-service/auth"
	pb "github.com/romnn/go-service/examples/reflect/gen"
	// log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
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
	client  pb.ReflectClient
}

func (test *test) setup(t *testing.T) *test {
	var err error
	t.Parallel()

	listener := bufconn.Listen(bufSize)
	// test.service, err = setUpAuthServer(t, listener)
	test.service = &ReflectService{
		// Authenticator: &auth.Authenticator{
		// 	ExpireSeconds: 100,
		// 	Issuer:        "mock-issuer",
		// 	Audience:      "mock-audience",
		// },
		// UserBackend: &MockUserMgmtBackend{},
	}
	go func() {
		server := grpc.NewServer(
		// grpc.ChainUnaryInterceptor(usi...),
		// grpc.ChainStreamInterceptor(ssi...),
		// grpc.MaxRecvMsgSize(maxMsgSize),
		// grpc.MaxSendMsgSize(maxMsgSize),
		)

		pb.RegisterReflectServer(server, test.service)
		if err := server.Serve(listener); err != nil {
			t.Fatalf("failed to serve service: %v", err)
		}
	}()

	// if err != nil {
	// 	t.Fatalf("failed to setup service: %v", err)
	// }

	// context.Background(),
	test.conn, err = grpc.Dial(
		"bufnet",
		grpc.WithDialer(dailerFor(listener)),
		grpc.WithInsecure(),
		grpc.WithTimeout(20*time.Second),
		grpc.WithBlock(),
	)

	// test.conn, err = grpc.DialContext(context.Background(), "bufnet", grpc.WithDialer(dailerFor(listener)), grpc.WithInsecure(),
	// 	grpc.WithTimeout(20*time.Second),
	// 	grpc.WithBlock(),
	// )
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
	// test.service.Shutdown()
}

func TestReflect(t *testing.T) {
	test := new(test).setup(t)
	defer test.teardown()

	ctx := context.Background()
	noAnnotations, err := test.client.GetNoAnnotations(ctx, &pb.Empty{})
	if err != nil {
		t.Fatalf("failed to get annotations: %v", err)
	}

	annotations, err := test.client.GetAnnotations(ctx, &pb.Empty{})
	if err != nil {
		t.Fatalf("failed to get annotations: %v", err)
	}
}
