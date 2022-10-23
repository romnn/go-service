## go-service

[![Build Status](https://github.com/romnn/go-service/workflows/test/badge.svg)](https://github.com/romnn/go-service/actions)
[![GitHub](https://img.shields.io/github/license/romnn/go-service)](https://github.com/romnn/go-service)
[![GoDoc](https://godoc.org/github.com/romnn/go-service?status.svg)](https://godoc.org/github.com/romnn/go-service)
[![Test Coverage](https://codecov.io/gh/romnn/go-service/branch/master/graph/badge.svg)](https://codecov.io/gh/romnn/go-service)

Service utilities for building gRPC and HTTP services in Go.

Some features:

- composable authentication using JWT
- gRPC interceptors for method reflection

### Example: Authentication

```proto
// examples/auth/auth.proto

syntax = "proto3";
package auth;

import "google/protobuf/timestamp.proto";

service Auth {
  rpc Login(LoginRequest) returns (AuthToken) {}
  rpc Validate(ValidationRequest) returns (ValidationResult) {}
}

message LoginRequest {
  string email = 1;
  string password = 2;
}

message ValidationRequest { string token = 1; }

message ValidationResult { bool valid = 1; }

message AuthToken {
  string token = 1;
  string email = 2;
  google.protobuf.Timestamp expires = 10;
}

```

```go
// examples/auth/server.go

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang-jwt/jwt/v4"
	pb "github.com/romnn/go-service/examples/auth/gen"
	"github.com/romnn/go-service/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// User represents a user
type User struct {
	Email          string
	HashedPassword string
}

// UserDatabase is a mock user database
type UserDatabase interface {
	GetUserByEmail(email string) (*User, error)
	AddUser(user *User)
	RemoveUserByEmail(email string) (*User, error)
}

type userDatabase struct {
	users map[string]*User
}

// AddUser adds a user to the database
func (db *userDatabase) AddUser(user *User) {
	db.users[user.Email] = user
}

// RemoveUserByEmail removes a user
func (db *userDatabase) RemoveUserByEmail(email string) (*User, error) {
	if user, ok := db.users[email]; ok {
		delete(db.users, email)
		return user, nil
	}
	return nil, fmt.Errorf("no user with email %q", email)
}

// GetUserByEmail gets a user
func (db *userDatabase) GetUserByEmail(email string) (*User, error) {
	if user, ok := db.users[email]; ok {
		return user, nil
	}
	return nil, fmt.Errorf("no user with email %q", email)
}

// AuthService ...
type AuthService struct {
	pb.UnimplementedAuthServer
	Authenticator *auth.Authenticator
	Database      UserDatabase
}

// Claims encode the JWT token claims
type Claims struct {
	UserEmail string `json:"user-email"`
	jwt.RegisteredClaims
}

// GetRegisteredClaims returns the standard claims that will be set automatically
func (claims *Claims) GetRegisteredClaims() *jwt.RegisteredClaims {
	// MUST return pointer to registered claims of this struct
	return &claims.RegisteredClaims
}

// Validate validates a token
func (s *AuthService) Validate(ctx context.Context, in *pb.ValidationRequest) (*pb.ValidationResult, error) {
	valid, token, err := s.Authenticator.Validate(in.GetToken(), &Claims{})
	if err != nil {
		log.Println(err)
		return &pb.ValidationResult{Valid: false}, status.Error(codes.Internal, "Failed to validate token")
	}
	if claims, ok := token.Claims.(*Claims); ok && valid {
		log.Printf("valid authentication claims: %v", claims)
		return &pb.ValidationResult{Valid: true}, nil
	}
	return &pb.ValidationResult{Valid: false}, nil
}

// Login logs in a user
func (s *AuthService) Login(ctx context.Context, in *pb.LoginRequest) (*pb.AuthToken, error) {
	user, err := s.Database.GetUserByEmail(in.GetEmail())
	if err != nil {
		log.Println(err)
		return nil, status.Error(codes.NotFound, "no such user")
	}
	if !auth.CheckPasswordHash(in.GetPassword(), user.HashedPassword) {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	// authenticated
	token, err := s.Authenticator.SignJwtClaims(&Claims{
		UserEmail: user.Email,
	})
	if err != nil {
		log.Println(err)
		return nil, status.Error(codes.Internal, "error while signing token")
	}

	expirationTime := time.Now().Add(s.Authenticator.ExpiresAfter)
	return &pb.AuthToken{
		Token:   token,
		Email:   user.Email,
		Expires: timestamppb.New(expirationTime),
	}, nil
}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	authenticator := auth.Authenticator{
		ExpiresAfter: 100 * time.Second,
		Issuer:       "issuer@example.org",
		Audience:     "example.org",
	}

	keyConfig := auth.AuthenticatorKeyConfig{Generate: true}
	if err := authenticator.SetupKeys(&keyConfig); err != nil {
		log.Fatalf("failed to setup keys: %v", err)
	}

	service := AuthService{
		Authenticator: &authenticator,
		Database: &userDatabase{
			users: make(map[string]*User),
		},
	}

	server := grpc.NewServer()
	pb.RegisterAuthServer(server, &service)

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

```

### Example: Reflection

```proto
// examples/reflect/reflect.proto

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

```

For more examples, see `examples/`.

#### Development

##### Tooling

Before you get started, make sure you have installed the following tools:

    $ python3 -m pip install pre-commit bump2version invoke
    $ go install golang.org/x/tools/cmd/goimports
    $ go install golang.org/x/lint/golint
    $ go install github.com/fzipp/gocyclo

It is advised to install the git commit hooks to enforce code checks:

```bash
inv install-hooks
```

To check if all checks pass:

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

To compile, you can use the provided utility:

```bash
inv compile-protos
```
