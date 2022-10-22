package auth

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

  "github.com/golang-jwt/jwt/v4"
	// "github.com/dgrijalva/jwt-go"
)

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
	t.Parallel()

	test.authenticator = &Authenticator{
		ExpireSeconds: 100,
		Issuer:        "mock-issuer",
		Audience:      "mock-audience",
	}
	keyConfig := AuthenticatorKeyConfig{Generate: true}
	if err := test.authenticator.SetupKeys(&keyConfig); err != nil {
		t.Fatalf("failed to setup keys: %v", err)
	}
	return test
}

func TestValidatesValidToken(t *testing.T) {
	test := new(test).setup(t)

	userID := "123"
	tokenString, expireSeconds, err := test.authenticator.Login(&testClaims{
		UserID: userID,
	})
	if err != nil {
		t.Errorf("failed to sign token: %v", err)
	}
	valid, token, err := test.authenticator.Validate(tokenString, &testClaims{})
	if err != nil {
		t.Errorf("failed to validate token: %v", err)
	}
	claims, ok := token.Claims.(*testClaims)
	if !ok {
		t.Errorf("expected testClaims, but got %T", token.Claims)
	}
	if claims.UserID != userID {
		t.Errorf("expected signed user ID to be %q but got %q", userID, claims.UserID)
	}
	if !valid || !token.Valid {
		t.Error("expected signed user token to be valid")
	}

	// Check that the token no longer validates after it expires
	expiration := time.Duration(expireSeconds+200) * time.Second
	at(time.Now().Add(expiration), func() {
		valid, token, err := test.authenticator.Validate(tokenString, &testClaims{})
		if err == nil {
			t.Error("expected error when validating expired user token")
		}
		if valid || token != nil {
			t.Errorf("unexpected valid=%t and token=%v on expired user token", valid, token)
		}
	})
}

func TestValidateInvalidTokenFails(t *testing.T) {
	test := new(test).setup(t)

	valid, token, err = test.authenticator.Validate("invalid-token", &testClaims{})
	if err == nil {
		t.Error("expected error when validating an invalid user token")
	}
	if valid || token != nil {
		t.Errorf("unexpected valid=%t and token=%v on invalid user token", valid, token)
	}
}

// Override time value for tests and restore after
// ref: https://github.com/dgrijalva/jwt-go/blob/master/example_test.go#L81
func at(t time.Time, f func()) {
	jwt.TimeFunc = func() time.Time { return t }
	f()
	jwt.TimeFunc = time.Now
}
