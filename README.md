## go-service

[![Build Status](https://github.com/romnn/go-service/workflows/test/badge.svg)](https://github.com/romnn/go-service/actions)
[![GitHub](https://img.shields.io/github/license/romnn/go-service)](https://github.com/romnn/go-service)
[![GoDoc](https://godoc.org/github.com/romnn/go-service?status.svg)](https://godoc.org/github.com/romnn/go-service)
[![Test Coverage](https://codecov.io/gh/romnn/go-service/branch/master/graph/badge.svg)](https://codecov.io/gh/romnn/go-service)

Service utilities for building gRPC and HTTP services in Go.

Some features:

- composable authentication using JWT
- gRPC interceptors for method reflection

##### Example: Reflection

```go
// examples/reflect/service.proto

syntax = "proto3";
package reflect;

import "google/protobuf/descriptor.proto";

// we define custom options
extend google.protobuf.MethodOptions {
  bool bool_value = 51234;
  string string_value = 51235;
  int32 int_value = 51236;
}

message Empty {}

message Annotations {
  bool bool_value = 1;
  string string_value = 2;
  int32 int_value = 3;
}

service Reflect {
  // we will read the options of this method using reflection
  rpc GetNoAnnotations(Empty) returns (Annotations) {}

  // we will read the options of this method using reflection
  rpc GetAnnotations(Empty) returns (Annotations) {
    option (bool_value) = true;
    option (string_value) = "Hello World";
    option (int_value) = 42;
  }
}

```

```go
// examples/reflect/server.go

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
	var annotations pb.Annotations
	info, ok := reflect.GetMethodInfo(ctx)
	if !ok {
		return &annotations, status.Error(codes.Internal, "failed to get grpc method info")
	}
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

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdown
		log.Println("shutdown ...")
		server.GracefulStop()
	}()

	log.Printf("listening on: %v", listener.Addr())
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

```

For more examples, see `examples/`.

#### Development

##### Tooling

Before you get started, make sure you have installed the following tools:

    $ python3 -m pip install pre-commit bump2version invoke
    $ go install golang.org/x/tools/cmd/goimports
    $ go install golang.org/x/lint/golint
    $ go install github.com/fzipp/gocyclo

**Remember**: To be able to excecute the tools downloaded with `go get`,
make sure to include `$GOPATH/bin` in your `$PATH`.
If `echo $GOPATH` does not give you a path make sure to run
(`export GOPATH="$HOME/go"` to set it). In order for your changes to persist,
do not forget to add these to your shells `.bashrc`.

With the tools in place, it is strongly advised to install the git commit hooks to make sure checks are passing in CI:

```bash
inv install-hooks
```

You can check if all checks pass at any time:

```bash
inv pre-commit
```

##### Compiling proto files

If you want to (re-)compile the sample grpc `.proto` services, you will need `protoc`, `protoc-gen-go` and `protoc-gen-go-grpc`.

```bash
apt install -y protobuf-compiler
brew install protobuf

go get -u google.golang.org/protobuf/cmd/protoc-gen-go
go install google.golang.org/protobuf/cmd/protoc-gen-go

go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

To compile, you can use the provided utility

```bash
inv compile-proto
```
