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

extend google.protobuf.MethodOptions {
  bool bool_value = 51234;
  string string_value = 51235;
  int32 int_value = 51236;
}

message Empty {
}

message Annotations {
  bool bool_value = 1;
  string string_value = 2;
  int32 int_value = 3;
}

service Reflect {
  rpc GetNoAnnotations(Empty) returns (Annotations) {}

  rpc GetAnnotations(Empty) returns (Annotations) {
    option (bool_value) = true;
    option (string_value) = "Hello World";
    option (int_value) = 42;
  }
}

message LoginRequest {
  string username = 1;
  string password = 2;
}

message Token {
  string token = 1;
}

message GetSecretRequest {
  Token token = 1;
}

message Secret {
  string secret = 1;
}

/* service Reflect { */
/*   rpc Login(LoginRequest) returns (Token) {} */

/*   rpc GetSecret(GetSecretRequest) returns (Secret) { */
/*     option (require_admin) = true; */
/*   } */
/* } */

/* message LoginRequest { */
/*   string username = 1; */
/*   string password = 2; */
/* } */

/* message Token { */
/*   string token = 1; */
/* } */

/* message GetSecretRequest { */
/*   Token token = 1; */
/* } */

/* message Secret { */
/*   string secret = 1; */
/* } */

```

```go
// examples/reflect/server.go

package main

import (
	"context"
	"fmt"

	"net"
	// "os"
	// "os/signal"
	// "syscall"

	// gogrpcservice "github.com/romnn/go-grpc-service"
	pb "github.com/romnn/go-service/examples/reflect/gen"
	"github.com/romnn/go-service/pkg/grpc/reflect"
	// "github.com/romnn/flags4urfavecli/flags"
	// log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/proto"
	// pref "google.golang.org/protobuf/reflect/protoreflect"
	// "google.golang.org/grpc/codes"
	// "google.golang.org/grpc/status"
)

// ReflectService ...
type ReflectService struct {
	// gogrpcservice.Service
	pb.UnimplementedReflectServer

	// connected bool
}

func (s *ReflectService) getAnnotations(ctx context.Context) (*pb.Annotations, error) {
	var annotations pb.Annotations
	info, ok := reflect.GetMethodInfo(ctx)
	if !ok {
		return &annotations, fmt.Errorf("failed to get method descriptor")
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

// GetNoAnnotations ...
func (s *ReflectService) GetNoAnnotations(ctx context.Context, req *pb.Empty) (*pb.Annotations, error) {
	return s.getAnnotations(ctx)
}

// GetAnnotations ...
func (s *ReflectService) GetAnnotations(ctx context.Context, req *pb.Empty) (*pb.Annotations, error) {
	return s.getAnnotations(ctx)
}

// Serve starts to serve the service
func (s *ReflectService) Serve(ctx *cli.Context, listener net.Listener) error {
	// go func() {
	// 	log.Info("connecting...")
	// 	if err := server.Service.Connect(ctx); err != nil {
	// 		log.Error(err)
	// 		s.Shutdown()
	// 	}
	// 	s.Service.Ready = true
	// 	s.Service.SetHealthy(true)
	// 	log.Infof("%s ready at %s", s.Service.Name, listener.Addr())
	// }()

	// pb.RegisterSampleServer(s.Service.GrpcServer, s)
	// if err := server.Service.ServeGrpc(listener); err != nil {
	// 	return err
	// }
	// log.Info("closing socket")
	// listener.Close()
	return nil
}

func main() {
	// shutdown := make(chan os.Signal)
	// signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	// go func() {
	// 	<-shutdown
	// 	server.Shutdown()
	// }()

	// cliFlags := []cli.Flag{
	// 	&flags.LogLevelFlag,
	// 	&cli.IntFlag{
	// 		Name:    "port",
	// 		Value:   80,
	// 		Aliases: []string{"p"},
	// 		EnvVars: []string{"PORT"},
	// 		Usage:   "service port",
	// 	},
	// }

	// name := "sample service"

	// app := &cli.App{
	// 	Name:  name,
	// 	Usage: "serves as an example",
	// 	Flags: cliFlags,
	// 	Action: func(cliCtx *cli.Context) error {
	// 		server = SampleServer{
	// 			Service: gogrpcservice.Service{
	// 				Name:      name,
	// 				Version:   Version,
	// 				BuildTime: BuildTime,
	// 				PostBootstrapHook: func(bs *gogrpcservice.Service) error {
	// 					log.Info("<your app name> (c) <your name>")
	// 					return nil
	// 				},
	// 				ConnectHook: func(bs *gogrpcservice.Service) error {
	// 					server.connected = true
	// 					return nil
	// 				},
	// 			},
	// 		}
	// 		port := fmt.Sprintf(":%d", cliCtx.Int("port"))
	// 		listener, err := net.Listen("tcp", port)
	// 		if err != nil {
	// 			return fmt.Errorf("failed to listen: %v", err)
	// 		}

	// 		if err := server.Service.BootstrapGrpc(context.Background(), cliCtx, nil); err != nil {
	// 			return err
	// 		}
	// 		return server.Serve(cliCtx, listener)
	// 	},
	// }
	// err := app.Run(os.Args)
	// if err != nil {
	// 	log.Fatal(err)
	// }
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
