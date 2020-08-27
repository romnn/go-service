package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/dgrijalva/jwt-go"
	"github.com/lestrrat-go/jwx/jwk"
)

// GenerateKeys generates private and public keys for jwt validation
func GenerateKeys() (*rsa.PrivateKey, []byte, *jwk.Set, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	pKey, ok := key.Public().(*rsa.PublicKey)
	if !ok {
		return nil, nil, nil, nil, errors.New("unexpected key type. expected public RSA key")
	}
	jwkJSON, err := ToJWKS(pKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	jwkSet := new(jwk.Set)
	if err := jwkSet.UnmarshalJSON(jwkJSON); err != nil {
		return nil, nil, nil, nil, err
	}
	return key, ToPEM(key), jwkSet, jwkJSON, nil
}

// ParseSigningKey ...
func ParseSigningKey(keyData []byte) (*rsa.PrivateKey, error) {
	return jwt.ParseRSAPrivateKeyFromPEM(keyData)
}

// LoadSigningKeyFromFile ...
func LoadSigningKeyFromFile(privateKeyFile string) (*rsa.PrivateKey, error) {
	data, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read %s: %v", privateKeyFile, err)
	}
	key, err := ParseSigningKey(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse private key PEM file at %s: %v", privateKeyFile, err)
	}
	return key, nil
}

// ParseJwkSet ...
func ParseJwkSet(jwkSetData []byte) (*jwk.Set, error) {
	jwkSet := new(jwk.Set)
	if err := jwkSet.UnmarshalJSON(jwkSetData); err != nil {
		return nil, err
	}
	return jwkSet, nil
}

// LoadJwkSetFromFile ...
func LoadJwkSetFromFile(jwkSetFile string) (*jwk.Set, error) {
	jwksData, err := ioutil.ReadFile(jwkSetFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to read %s: %v", jwkSetFile, err)
	}
	jwkSet, err := ParseJwkSet(jwksData)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse jwk set at %s: %v", jwkSetFile, err)
	}
	return jwkSet, nil
}
