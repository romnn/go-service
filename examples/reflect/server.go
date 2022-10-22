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
