package auth

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
)

const (
	parallel = true
)

func setUpAuthenticator(t *testing.T) (*Authenticator, error) {
	authenticator := &Authenticator{
		ExpireSeconds: 100,
		Issuer:        "mock-issuer",
		Audience:      "mock-audience",
	}
	if err := authenticator.SetupKeys(&AuthenticatorKeyConfig{Generate: true}); err != nil {
		return nil, fmt.Errorf("failed to setup keys: %v", err)
	}
	return authenticator, nil
}

type testClaims struct {
	UserID string `json:"user_id"`
	jwt.StandardClaims
}

func (claims *testClaims) GetStandardClaims() *jwt.StandardClaims {
	return &claims.StandardClaims
}

type test struct {
	authenticator *Authenticator
}

func (test *test) setup(t *testing.T) *test {
	var err error
	if parallel {
		t.Parallel()
	}
	// This wil disable the application logger
	log.SetOutput(ioutil.Discard)

	test.authenticator, err = setUpAuthenticator(t)
	if err != nil {
		t.Fatalf("failed to setup the authentication service: %v", err)
	}
	return test
}

func TestValidation(t *testing.T) {
	test := new(test).setup(t)
	// Sign a logged in user token and make sure the token validates
	userID := "123"
	userToken, expireSeconds, err := test.authenticator.Login(&testClaims{
		UserID: userID,
	})
	if err != nil {
		t.Errorf("failed to log in user and sign token: %v", err)
	}
	valid, token, err := test.authenticator.Validate(userToken, &testClaims{})
	if err != nil {
		t.Errorf("unexpected error while validating user token: %v", err)
	}
	claims, ok := token.Claims.(*testClaims)
	if !ok {
		t.Errorf("expected signed user token to unmarshal to struct of type testClaims, but got %T", token.Claims)
	}
	if claims.UserID != userID {
		t.Errorf("expected signed user ID to be %q but got %q", userID, claims.UserID)
	}
	if !valid || !token.Valid {
		t.Error("expected signed user token to be valid")
	}

	// Check that the token no longer validates after it expires
	at(time.Now().Add(time.Duration(expireSeconds+200)*time.Second), func() {
		valid, token, err := test.authenticator.Validate(userToken, &testClaims{})
		if err == nil {
			t.Error("expected error when validating expired user token")
		}
		if valid || token != nil {
			t.Errorf("unexpected valid=%t and token=%v on expired user token", valid, token)
		}
	})

	// Make sure some random token does not valdidate
	valid, token, err = test.authenticator.Validate("this-is-fake", &testClaims{})
	if err == nil {
		t.Error("expected error when validating an invalid user token")
	}
	if valid || token != nil {
		t.Errorf("unexpected valid=%t and token=%v on invalid user token", valid, token)
	}
}

// Override time value for tests.  Restore default value after.
// Source: https://github.com/dgrijalva/jwt-go/blob/master/example_test.go#L81
func at(t time.Time, f func()) {
	jwt.TimeFunc = func() time.Time { return t }
	f()
	jwt.TimeFunc = time.Now
}
