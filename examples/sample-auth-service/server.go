package main

import (
	"context"
	"fmt"

	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/dgrijalva/jwt-go"
	gogrpcservice "github.com/romnnn/go-grpc-service"
	"github.com/romnnn/go-grpc-service/auth"
	pb "github.com/romnnn/go-grpc-service/gen/sample-services"

	"github.com/romnnn/flags4urfavecli/flags"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Version will be injected at build time
var Version string = "Unknown"

// BuildTime will be injected at build time
var BuildTime string = ""

var server AuthServer

// User ...
type User struct {
	ID             string
	Username       string
	Email          string
	HashedPassword string
}

// UserMgmtBackend ...
type UserMgmtBackend interface {
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	AddUser(ctx context.Context, user *User) (*User, error)
	RemoveUserByEmail(ctx context.Context, email string) (*User, error)
}

// AuthServer ...
type AuthServer struct {
	gogrpcservice.Service
	pb.UnimplementedAuthenticationServer
	Authenticator *auth.Authenticator
	UserBackend   UserMgmtBackend
}

// Shutdown ...
func (s *AuthServer) Shutdown() {
	s.Service.GracefulStop()
	// Do any additional shutdown here
}

// MyClaims ...
type MyClaims struct {
	UserID string `json:"userid"`
	jwt.StandardClaims
}

// GetStandardClaims ...
func (claims *MyClaims) GetStandardClaims() *jwt.StandardClaims {
	// the authenticator will use this method to get the standard claims that will be set based on the config
	// note that it is important to return a pointer to the current claims' standard claims and not any ones
	return &claims.StandardClaims
}

// Validate checks a token if it is valid (e.g. has not expired)
func (s *AuthServer) Validate(ctx context.Context, in *pb.TokenValidationRequest) (*pb.TokenValidationResult, error) {
	valid, token, err := s.Authenticator.Validate(in.GetToken(), &MyClaims{})
	if err != nil {
		log.Error(err)
		return &pb.TokenValidationResult{Valid: false}, status.Error(codes.Internal, "Failed to validate token")
	}
	if claims, ok := token.Claims.(*MyClaims); ok && valid {
		log.Infof("valid authentication claims: %v", claims)
		return &pb.TokenValidationResult{Valid: true}, nil
	}
	return &pb.TokenValidationResult{Valid: false}, nil
}

// Login logs in a user
func (s *AuthServer) Login(ctx context.Context, in *pb.UserLoginRequest) (*pb.AuthenticationToken, error) {
	user, err := s.UserBackend.GetUserByEmail(ctx, in.GetEmail())
	if err != nil {
		log.Error(err)
		return nil, status.Error(codes.NotFound, "no such user")
	}
	if !auth.CheckPasswordHash(in.GetPassword(), user.HashedPassword) {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	// authenticated
	token, expireSeconds, err := s.Authenticator.Login(&MyClaims{
		UserID: user.ID,
	})
	if err != nil {
		log.Error(err)
		return nil, status.Error(codes.Internal, "error while signing token")
	}
	return &pb.AuthenticationToken{
		Token:      token,
		Email:      user.Email,
		UserId:     user.ID,
		Expiration: expireSeconds,
	}, nil
}

func main() {
	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdown
		server.Shutdown()
	}()

	cliFlags := []cli.Flag{
		&flags.LogLevelFlag,
		&cli.IntFlag{
			Name:    "port",
			Value:   80,
			Aliases: []string{"p"},
			EnvVars: []string{"PORT"},
			Usage:   "service port",
		},
	}

	// Add the default set of CLI flags for the authenticator
	// Of course, you can also define your own flags and manually populate `Authenticator` and `AuthenticatorKeyConfig`
	cliFlags = append(cliFlags, auth.DefaultCLIFlags(&auth.DefaultCLIFlagsOptions{
		Issuer:   "issuer@example.org",
		Audience: "example.org",
	})...)

	name := "sample authentication service"

	app := &cli.App{
		Name:  name,
		Usage: "serves as an example",
		Flags: cliFlags,
		Action: func(cliCtx *cli.Context) error {
			server = AuthServer{
				Service: gogrpcservice.Service{
					Name:      name,
					Version:   Version,
					BuildTime: BuildTime,
					PostBootstrapHook: func(bs *gogrpcservice.Service) error {
						log.Info("<your app name> (c) <your name>")
						return nil
					},
				},
				Authenticator: &auth.Authenticator{
					ExpireSeconds: int64(cliCtx.Int("expire-sec")),
					Issuer:        cliCtx.String("issuer"),
					Audience:      cliCtx.String("audience"),
				},
			}
			port := fmt.Sprintf(":%d", cliCtx.Int("port"))
			listener, err := net.Listen("tcp", port)
			if err != nil {
				return fmt.Errorf("failed to listen: %v", err)
			}

			if err := server.Authenticator.SetupKeys(auth.AuthenticatorKeyConfig{}.Parse(cliCtx)); err != nil {
				return err
			}
			if err := server.Service.BootstrapGrpc(context.Background(), cliCtx, nil); err != nil {
				return err
			}
			return server.Serve(cliCtx, listener)
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// Serve starts the service
func (s *AuthServer) Serve(ctx *cli.Context, listener net.Listener) error {

	go func() {
		log.Info("connecting...")
		if err := server.Service.Connect(ctx); err != nil {
			log.Error(err)
			s.Shutdown()
		}
		s.Service.Ready = true
		s.Service.SetHealthy(true)
		log.Infof("%s ready at %s", s.Service.Name, listener.Addr())
	}()

	pb.RegisterAuthenticationServer(s.Service.GrpcServer, s)
	if err := server.Service.ServeGrpc(listener); err != nil {
		return err
	}
	log.Info("closing socket")
	listener.Close()
	return nil
}
