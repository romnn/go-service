package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/romnnn/go-grpc-service/auth"
	pb "github.com/romnnn/go-grpc-service/gen/sample-services"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const (
	parallel = false
	bufSize  = 1024 * 1024
)

type DialerFunc = func(string, time.Duration) (net.Conn, error)

func dailerFor(listener *bufconn.Listener) DialerFunc {
	return func(string, time.Duration) (net.Conn, error) {
		return listener.Dial()
	}
}

func setUpAuthServer(t *testing.T, listener *bufconn.Listener) (*AuthServer, error) {
	server := &AuthServer{
		Authenticator: &auth.Authenticator{
			ExpireSeconds: 100,
			Issuer:        "mock-issuer",
			Audience:      "mock-audience",
		},
		UserBackend: &MockUserMgmtBackend{},
	}
	if err := server.Authenticator.SetupKeys(&auth.AuthenticatorKeyConfig{Generate: true}); err != nil {
		return nil, fmt.Errorf("failed to setup keys: %v", err)
	}
	if err := server.Service.BootstrapGrpc(nil, nil); err != nil {
		return nil, fmt.Errorf("failed to setup grpc server: %v", err)
	}

	go func() {
		pb.RegisterAuthenticationServer(server.Service.GrpcServer, server)
		if err := server.ServeGrpc(listener); err != nil {
			t.Fatalf("failed to serve auth service: %v", err)
		}
	}()

	return server, nil
}

func teardownServer(server interface{ Shutdown() }) {
	server.Shutdown()
}

// UserMgmtBackend ...
type MockUserMgmtBackend struct {
	users []*User
}

func (um *MockUserMgmtBackend) AddUser(ctx context.Context, user *User) (*User, error) {
	um.users = append(um.users, user)
	return user, nil
}

func (um *MockUserMgmtBackend) RemoveUserByEmail(ctx context.Context, email string) (*User, error) {
	var filtered []*User
	var removed *User
	for _, user := range um.users {
		if email != user.Email {
			filtered = append(filtered, user)
		} else {
			removed = user
		}
	}
	um.users = filtered
	return removed, nil
}

func (um *MockUserMgmtBackend) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	for _, user := range um.users {
		if email == user.Email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

type test struct {
	authEndpoint *grpc.ClientConn
	authServer   *AuthServer
	authClient   pb.AuthenticationClient
}

func (test *test) setup(t *testing.T) *test {
	var err error
	if parallel {
		t.Parallel()
	}
	// This wil disable the application logger
	log.SetOutput(ioutil.Discard)

	authListener := bufconn.Listen(bufSize)
	test.authServer, err = setUpAuthServer(t, authListener)
	if err != nil {
		t.Fatalf("failed to setup the authentication service: %v", err)
		return test
	}

	// Create endpoints
	test.authEndpoint, err = grpc.DialContext(context.Background(), "bufnet", grpc.WithDialer(dailerFor(authListener)), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatalf("failed to dial bufnet: %v", err)
		return test
	}

	test.authClient = pb.NewAuthenticationClient(test.authEndpoint)
	return test
}

func (test *test) teardown() {
	_ = test.authEndpoint.Close()
	test.authServer.Shutdown()
}

func mustHashPassword(pw string) string {
	hashed, err := auth.HashPassword(pw)
	if err != nil {
		panic(err)
	}
	return hashed
}

func TestLogin(t *testing.T) {
	test := new(test).setup(t)
	defer test.teardown()

	// Invalid login because no such user has been added
	assertLoginFails(t, test.authClient, &pb.UserLoginRequest{
		Email:    "test@example.com",
		Password: "secret",
	})

	// Add the user from to make sure login succeeds now
	user1Password := "secret"
	user1, _ := test.authServer.UserBackend.AddUser(context.Background(), &User{
		Email:          "test@example.com",
		HashedPassword: mustHashPassword(user1Password),
	})

	assertSuccessfulLogin(t, test.authClient, &pb.UserLoginRequest{
		Email:    user1.Email,
		Password: user1Password,
	})

	// Invalid user login because of typos
	assertLoginFails(t, test.authClient, &pb.UserLoginRequest{
		Email:    user1.Email,
		Password: user1Password + "'",
	})
	assertLoginFails(t, test.authClient, &pb.UserLoginRequest{
		Email:    user1.Email + " ",
		Password: user1Password,
	})

	// Remove user1 to make sure login fails again
	if _, err := test.authServer.UserBackend.RemoveUserByEmail(context.Background(), user1.Email); err != nil {
		t.Errorf("failed to remove user: %v", err)
	}

	assertLoginFails(t, test.authClient, &pb.UserLoginRequest{
		Email:    user1.Email,
		Password: user1Password,
	})

	// Sanity check empty values
	assertLoginFails(t, test.authClient, &pb.UserLoginRequest{
		Email:    user1.Email,
		Password: "",
	})
	assertLoginFails(t, test.authClient, &pb.UserLoginRequest{
		Email:    "",
		Password: user1Password,
	})
	assertLoginFails(t, test.authClient, &pb.UserLoginRequest{
		Email:    "",
		Password: "",
	})
}

func TestValidation(t *testing.T) {
	test := new(test).setup(t)
	defer test.teardown()
	email := "test@example.com"
	pw := "secret"
	addedUser, _ := test.authServer.UserBackend.AddUser(context.Background(), &User{
		Email:          email,
		HashedPassword: mustHashPassword(pw),
	})

	// get the token from a valid user
	response, err := assertSuccessfulLogin(t, test.authClient, &pb.UserLoginRequest{
		Email:    email,
		Password: pw,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertIsValidToken(t, test.authClient, &pb.TokenValidationRequest{Token: response.Token})

	// delete the valid user and make sure the token is valid until it expires
	if _, err := test.authServer.UserBackend.RemoveUserByEmail(context.Background(), addedUser.Email); err != nil {
		t.Fatalf("Failed to remove user: %v", err)
	}
	assertIsValidToken(t, test.authClient, &pb.TokenValidationRequest{Token: response.Token})
	at(time.Now().Add(time.Duration(test.authServer.Authenticator.ExpireSeconds+200)*time.Second), func() {
		assertIsInvalidToken(t, test.authClient, &pb.TokenValidationRequest{Token: response.Token})
	})

	// test for a malformed token (invalid format)
	badToken := "12.adbs."
	if _, err := test.authClient.Validate(context.Background(), &pb.TokenValidationRequest{Token: badToken}); err == nil {
		t.Fatalf("Bad jwt \"%s\"did not cause internal parse error", badToken)
	}
}

// Override time value for tests.  Restore default value after.
// Source: https://github.com/dgrijalva/jwt-go/blob/master/example_test.go#L81
func at(t time.Time, f func()) {
	jwt.TimeFunc = func() time.Time { return t }
	f()
	jwt.TimeFunc = time.Now
}

func assertSuccessfulLogin(t *testing.T, client pb.AuthenticationClient, login *pb.UserLoginRequest) (*pb.AuthenticationToken, error) {
	loginResult, err := client.Login(context.Background(), login)
	if err != nil {
		t.Fatalf("Login for user %v failed unexpectedly: %v", login, err)
	}
	return loginResult, err
}

func assertLoginFails(t *testing.T, client pb.AuthenticationClient, login *pb.UserLoginRequest) {
	loginResult, err := client.Login(context.Background(), login)
	if err == nil {
		t.Fatalf("Login for user %v succeeded unexpectedly with token %v", login.GetEmail(), loginResult.Token)
	}
}

func assertIsValidToken(t *testing.T, client pb.AuthenticationClient, request *pb.TokenValidationRequest) {
	valid, err := client.Validate(context.Background(), request)
	if err != nil || (valid != nil && !valid.Valid) {
		t.Fatalf("Validation for token %s yielded invalid unexpectedly: %v", request.Token, err)
	}
}

func assertIsInvalidToken(t *testing.T, client pb.AuthenticationClient, request *pb.TokenValidationRequest) {
	valid, err := client.Validate(context.Background(), request)
	if err == nil || (valid != nil && valid.Valid) {
		t.Fatalf("Validation for token %s yielded valid unexpectedly", request.Token)
	}
}
