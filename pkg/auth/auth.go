package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/lestrrat-go/jwx/jwk"
)

// Authenticator ...
type Authenticator struct {
	Issuer   string
	Audience string

	SignKey       *rsa.PrivateKey
	JwkSet        jwk.Set
	ExpireSeconds int64
}

// Claims ...
type Claims interface {
	jwt.Claims
	GetStandardClaims() *jwt.StandardClaims
}

// Validate checks a token if it is valid (e.g. has not expired)
func (a *Authenticator) Validate(token string, claims Claims) (bool, *jwt.Token, error) {
	validatedToken, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		keyID, ok := t.Header["kid"].(string)
		if !ok {
			return nil, errors.New("expecting JWT header to have kid of type string")
		}

		keyAlg, ok := t.Header["alg"].(string)
		if !ok {
			return nil, errors.New("expecting JWT header to have alg of type string")
		}

		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", keyAlg)
		}

		if matchingKey, ok := a.JwkSet.LookupKeyID(keyID); ok {
			var key rsa.PublicKey
			err := matchingKey.Raw(&key)
			return &key, err
		}
		return nil, fmt.Errorf("unable to find key %q", keyID)
	})
	if err != nil {
		return false, nil, err
	}
	return validatedToken.Valid, validatedToken, nil
}

// Login logs in by signing the provided claim
func (a *Authenticator) Login(claims Claims) (string, int64, error) {
	token, err := a.SignJwtClaims(claims)
	return token, a.ExpireSeconds, err
}

// SetupKeys loads keys from files, environment variables or generates a pair of keys
func (a *Authenticator) SetupKeys(config *AuthenticatorKeyConfig) error {
	var errs = make([]error, 4)
	if config.Jwks != "" {
		a.JwkSet, errs[0] = ParseJwkSet([]byte(config.Jwks))
	}
	if config.JwksFile != "" {
		a.JwkSet, errs[1] = LoadJwkSetFromFile(config.JwksFile)
	}
	if config.Key != "" {
		a.SignKey, errs[2] = ParseSigningKey([]byte(config.Key))
	}
	if config.KeyFile != "" {
		a.SignKey, errs[3] = LoadSigningKeyFromFile(config.KeyFile)
	}
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	if a.SignKey == nil || a.JwkSet == nil {
		if !config.Generate {
			return errors.New("missing signing key or jwk set and --generate disabled")
		}
		// var err error
		// a.SignKey, _, a.JwkSet, _, err = GenerateKeys()

		keyPair, err := GenerateRSAKeyPair()
		if err != nil {
			return fmt.Errorf("failed to generate RSA key pair: %v", err)
		}
		a.SignKey = keyPair.PrivateKey
		jwkSet, err := ToJwks(keyPair.PublicKey)
		if err != nil {
			return fmt.Errorf("failed to generate JWK set: %v", err)
		}
		a.JwkSet = jwkSet
	}
	return nil
}
