package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
	"github.com/lestrrat-go/jwx/jwk"
)

// Authenticator provides convenient methods for signing and validating JWT claims
type Authenticator struct {
	Issuer        string
	Audience      string
	ExpireSeconds int64

	SignKey *rsa.PrivateKey
	JwkSet  jwk.Set
}

// Claims defines the interface that custom JWT claim types must implement
type Claims interface {
	jwt.Claims
	GetRegisteredClaims() *jwt.RegisteredClaims
}

// Validate checks a token if it is valid (e.g. has not expired)
func (auth *Authenticator) Validate(tokenString string, claims Claims) (bool, *jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("expecting JWT header kid to be string, but got %T", t.Header["kid"])
		}

		alg, ok := t.Header["alg"].(string)
		if !ok {
			return nil, fmt.Errorf("expecting JWT header alg to be string, but got %T", t.Header["alg"])
		}

		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("expected RSA signing method, but got %v", alg)
		}

		if matchingKey, ok := auth.JwkSet.LookupKeyID(kid); ok {
			var key rsa.PublicKey
			err := matchingKey.Raw(&key)
			return &key, err
		}
		return nil, fmt.Errorf("unable to find key with id %q", kid)
	})
	if err != nil {
		return false, nil, err
	}
	return token.Valid, token, nil
}

// SetupKeys loads or generates keys from the config
func (auth *Authenticator) SetupKeys(config *AuthenticatorKeyConfig) error {
	var errs = make([]error, 4)
	if config.Jwks != "" {
		auth.JwkSet, errs[0] = ParseJwkSet([]byte(config.Jwks))
	}
	if config.JwksFile != "" {
		auth.JwkSet, errs[1] = LoadJwkSetFromFile(config.JwksFile)
	}
	if config.Key != "" {
		auth.SignKey, errs[2] = ParseSigningKeyFromPEMData([]byte(config.Key))
	}
	if config.KeyFile != "" {
		auth.SignKey, errs[3] = ParseSigningKeyFromPEMFile(config.KeyFile)
	}
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	if auth.SignKey == nil || auth.JwkSet == nil {
		if !config.Generate {
			return errors.New("missing signing key or jwk set and --generate disabled")
		}
		keyPair, err := GenerateRSAKeyPair()
		if err != nil {
			return fmt.Errorf("failed to generate RSA key pair: %v", err)
		}
		auth.SignKey = keyPair.PrivateKey
		jwkSet, err := ToJwks(keyPair.PublicKey)
		if err != nil {
			return fmt.Errorf("failed to generate JWK set: %v", err)
		}
		auth.JwkSet = jwkSet
	}
	return nil
}
