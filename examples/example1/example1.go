package main

import (
	"fmt"

	gogrpcservice "github.com/romnnn/go-grpc-service"
)

func run() string {
	return gogrpcservice.Shout("This is an example")
}

func main() {
	fmt.Println(run())
}
