package main

import (
	"context"
	"fmt"
	"log"

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
	return &pb.AuthToken{
		Token:      token,
		Email:      user.Email,
		Expiration: s.Authenticator.ExpireSeconds,
	}, nil
}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	authenticator := auth.Authenticator{
		ExpireSeconds: 100,
		Issuer:        "issuer@example.org",
		Audience:      "example.org",
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
