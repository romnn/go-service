package main

import (
	"context"
	"fmt"

	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
	gogrpcservice "github.com/romnnn/go-grpc-service"

	"github.com/romnnn/flags4urfavecli/flags"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

// Version will be injected at build time
var Version string = "Unknown"

// BuildTime will be injected at build time
var BuildTime string = ""

var server SampleServer

// SampleServer ...
type SampleServer struct {
	gogrpcservice.Service

	connected bool
}

// Shutdown ...
func (s *SampleServer) Shutdown() {
	s.Service.GracefulStop()
	// Do any additional shutdown here
}

func (s *SampleServer) greetingHandler(c echo.Context) error {
	return c.JSONPretty(http.StatusOK, map[string]string{"message": "welcome"}, "  ")
}

func (s *SampleServer) setupRouter() *echo.Echo {
	echoServer := echo.New()
	echoServer.GET("/", s.greetingHandler)
	return echoServer
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

	name := "sample service"

	app := &cli.App{
		Name:  name,
		Usage: "serves as an example",
		Flags: cliFlags,
		Action: func(cliCtx *cli.Context) error {
			server = SampleServer{
				Service: gogrpcservice.Service{
					Name:      name,
					Version:   Version,
					BuildTime: BuildTime,
					PostBootstrapHook: func(bs *gogrpcservice.Service) error {
						log.Info("<your app name> (c) <your name>")
						return nil
					},
					ConnectHook: func(bs *gogrpcservice.Service) error {
						server.connected = true
						return nil
					},
				},
			}
			port := fmt.Sprintf(":%d", cliCtx.Int("port"))
			listener, err := net.Listen("tcp", port)
			if err != nil {
				return fmt.Errorf("failed to listen: %v", err)
			}

			if err := server.Service.BootstrapHTTP(context.Background(), cliCtx, server.setupRouter(), nil); err != nil {
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
func (s *SampleServer) Serve(ctx *cli.Context, listener net.Listener) error {

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

	if err := s.Service.HTTPServer.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
	log.Info("closing socket")
	listener.Close()
	return nil
}
