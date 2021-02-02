## go-grpc-service

[![Build Status](https://github.com/romnn/go-grpc-service/workflows/test/badge.svg)](https://github.com/romnn/go-grpc-service/actions)
[![GitHub](https://img.shields.io/github/license/romnn/go-grpc-service)](https://github.com/romnn/go-grpc-service)
[![GoDoc](https://godoc.org/github.com/romnn/go-grpc-service?status.svg)](https://godoc.org/github.com/romnn/go-grpc-service)  [![Test Coverage](https://codecov.io/gh/romnn/go-grpc-service/branch/master/graph/badge.svg)](https://codecov.io/gh/romnn/go-grpc-service)
[![Release](https://img.shields.io/github/release/romnn/go-grpc-service)](https://github.com/romnn/go-grpc-service/releases/latest)

Optionated base service for gRPC (and HTTP) server implementations in `golang` using `google.golang.org/grpc`.

The main additional benefits of this base service include:

- Centralized and extensible hook system for configuring services
- Support for gRPC method reflection, e.g. custom proto method options are injected to the method context via interceptors (see the sample service)
- Unified logging and setup
- Boilerplate code for dialing other gRPC services provided
- Abstraction layer around health checking for gRPC and HTTP



#### Usage as a library

```golang
import "github.com/romnn/go-grpc-service"
```

For a full example, check out the sample service in `examples/`:

```bash
# check out the sample GRPC service
go run github.com/romnn/go-grpc-service/examples/sample-grpc-service --port 8080
# check out the sample HTTP service
go run github.com/romnn/go-grpc-service/examples/sample-http-service --port 8080
# check out the sample JWT authentication GRPC service
go run github.com/romnn/go-grpc-service/examples/sample-auth-service --port 8080 --generate
```

#### References

- Set the logging format for HTTP services:
    ```golang
    s.Service.SetLogFormat(&log.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
    ```


#### Development

######  Prerequisites

Before you get started, make sure you have installed the following tools::

    $ python3 -m pip install -U cookiecutter>=1.4.0
    $ python3 -m pip install pre-commit bump2version invoke ruamel.yaml halo
    $ go get -u golang.org/x/tools/cmd/goimports
    $ go get -u golang.org/x/lint/golint
    $ go get -u github.com/fzipp/gocyclo
    $ go get -u github.com/mitchellh/gox  # if you want to test building on different architectures

**Remember**: To be able to excecute the tools downloaded with `go get`, 
make sure to include `$GOPATH/bin` in your `$PATH`.
If `echo $GOPATH` does not give you a path make sure to run
(`export GOPATH="$HOME/go"` to set it). In order for your changes to persist, 
do not forget to add these to your shells `.bashrc`.

With the tools in place, it is strongly advised to install the git commit hooks to make sure checks are passing in CI:
```bash
invoke install-hooks
```

You can check if all checks pass at any time:
```bash
invoke pre-commit
```

Note for Maintainers: After merging changes, tag your commits with a new version and push to GitHub to create a release:
```bash
bump2version (major | minor | patch)
git push --follow-tags
```

If you want to (re-)generate the sample grpc service, make sure to install `protoc`, `protoc-gen-go` and `protoc-gen-go-grpc`.
You can then use the provided script:
```bash
apt install -y protobuf-compiler
go install google.golang.org/protobuf/cmd/protoc-gen-go
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc
invoke compile-proto
```

#### Note

This project is still in the alpha stage and should not be considered production ready.
