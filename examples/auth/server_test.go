package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	pb "github.com/romnn/go-service/examples/auth/gen"
	"github.com/romnn/go-service/pkg/auth"
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
	service *AuthService
	server  *grpc.Server
	client  pb.AuthClient
}

func (test *test) setup(t *testing.T) *test {
	var err error
	t.Parallel()

	authenticator := auth.Authenticator{
		ExpireSeconds: 100,
		Issuer:        "mock-issuer",
		Audience:      "mock-audience",
	}

	keyConfig := auth.AuthenticatorKeyConfig{Generate: true}
	if err := authenticator.SetupKeys(&keyConfig); err != nil {
		t.Fatalf("failed to setup keys: %v", err)
	}

	test.service = &AuthService{
		Authenticator: &authenticator,
		Database: &userDatabase{
			users: make(map[string]*User),
		},
	}

	test.server = grpc.NewServer()
	pb.RegisterAuthServer(test.server, test.service)

	listener := bufconn.Listen(bufSize)
	go func() {
		if err := test.server.Serve(listener); err != nil {
			t.Fatalf("failed to serve: %v", err)
		}
	}()

	test.conn, err = grpc.Dial(
		"bufnet",
		grpc.WithDialer(dailerFor(listener)),
		grpc.WithInsecure(),
		grpc.WithTimeout(20*time.Second),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("failed to dial grpc service: %v", err)
	}

	test.client = pb.NewAuthClient(test.conn)
	return test
}

func (test *test) teardown() {
	if test.conn != nil {
		_ = test.conn.Close()
	}
	if test.server != nil {
		test.server.GracefulStop()
	}
}

func TestLoginMissingUser(t *testing.T) {
	test := new(test).setup(t)
	defer test.teardown()

	assertLoginFails(t, test.client, &pb.LoginRequest{
		Email:    "test@example.com",
		Password: "secret",
	})
}

func TestLoginExisingUser(t *testing.T) {
	test := new(test).setup(t)
	defer test.teardown()

	// add user
	password := "secret"
	user := User{
		Email:          "test@example.com",
		HashedPassword: auth.MustHashPassword(password),
	}
	test.service.Database.AddUser(&user)

	assertSuccessfulLogin(t, test.client, &pb.LoginRequest{
		Email:    user.Email,
		Password: password,
	})

	// login fails for wrong password
	assertLoginFails(t, test.client, &pb.LoginRequest{
		Email:    user.Email,
		Password: password + "'",
	})
	assertLoginFails(t, test.client, &pb.LoginRequest{
		Email:    user.Email + " ",
		Password: password,
	})
	assertLoginFails(t, test.client, &pb.LoginRequest{
		Email:    user.Email,
		Password: "",
	})
	assertLoginFails(t, test.client, &pb.LoginRequest{
		Email:    "",
		Password: password,
	})

	assertLoginFails(t, test.client, &pb.LoginRequest{
		Email:    user.Email,
		Password: "sEcReT", // password is case sensitive
	})

	// remove user again
	if _, err := test.service.Database.RemoveUserByEmail(user.Email); err != nil {
		t.Errorf("failed to remove user: %v", err)
	}

	assertLoginFails(t, test.client, &pb.LoginRequest{
		Email:    user.Email,
		Password: password,
	})
}

func TestValidatesExistingUser(t *testing.T) {
	test := new(test).setup(t)
	defer test.teardown()

	// add user
	password := "secret"
	user := User{
		Email:          "test@example.com",
		HashedPassword: auth.MustHashPassword(password),
	}
	test.service.Database.AddUser(&user)

	// get user token
	response, err := assertSuccessfulLogin(t, test.client, &pb.LoginRequest{
		Email:    user.Email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("failed to login valid user: %v", err)
	}
	assertIsValidToken(t, test.client, &pb.ValidationRequest{Token: response.Token})

	// delete the valid user
	if _, err := test.service.Database.RemoveUserByEmail(user.Email); err != nil {
		t.Fatalf("failed to remove user: %v", err)
	}

	// the token is still valid
	validationReq := pb.ValidationRequest{Token: response.Token}
	assertIsValidToken(t, test.client, &validationReq)

	// the token is invalid after it expires
	expiration := time.Duration(test.service.Authenticator.ExpireSeconds+200) * time.Second
	at(time.Now().Add(expiration), func() {
		assertIsInvalidToken(t, test.client, &validationReq)
	})
}

func TestValidationFailsForBadToken(t *testing.T) {
	test := new(test).setup(t)
	defer test.teardown()

	badToken := "12.adbs."
	validationReq := pb.ValidationRequest{Token: badToken}
	if _, err := test.client.Validate(context.Background(), &validationReq); err == nil {
		t.Fatalf("did not return error for malformed jwt token %q", badToken)
	}
}

// Overrides time value for tests, restores afterwards
// ref: https://github.com/dgrijalva/jwt-go/blob/master/example_test.go#L81
func at(t time.Time, f func()) {
	jwt.TimeFunc = func() time.Time { return t }
	f()
	jwt.TimeFunc = time.Now
}

func assertSuccessfulLogin(t *testing.T, client pb.AuthClient, login *pb.LoginRequest) (*pb.AuthToken, error) {
	result, err := client.Login(context.Background(), login)
	if err != nil {
		t.Fatalf("login for user %v failed unexpectedly: %v", login, err)
	}
	return result, err
}

func assertLoginFails(t *testing.T, client pb.AuthClient, login *pb.LoginRequest) {
	result, err := client.Login(context.Background(), login)
	if err == nil {
		t.Fatalf("login for user %v with token %q succeeded unexpectedly", login.GetEmail(), result.Token)
	}
}

func assertIsValidToken(t *testing.T, client pb.AuthClient, req *pb.ValidationRequest) {
	valid, err := client.Validate(context.Background(), req)
	if err != nil || (valid != nil && !valid.Valid) {
		t.Fatalf("validation for token %q failed unexpectedly: %v", req.Token, err)
	}
}

func assertIsInvalidToken(t *testing.T, client pb.AuthClient, req *pb.ValidationRequest) {
	valid, err := client.Validate(context.Background(), req)
	if err == nil || (valid != nil && valid.Valid) {
		t.Fatalf("validation for token %q succeeded unexpectedly", req.Token)
	}
}
