package auth

import (
	"crypto/rand"
	"crypto/rsa"
	// "errors"
	"fmt"
	"io/ioutil"
	"reflect"

	"github.com/dgrijalva/jwt-go"
	"github.com/lestrrat-go/jwx/jwk"
)

// KeyPair is an RSA key pair
type KeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	// PublicKey  []byte
	// PublicKeyPEM []byte
	// JWKSet       jwk.Set
	// JWKJson      []byte
}

// GenerateRSAKeyPair generates an RSA key pair
func GenerateRSAKeyPair() (*KeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	publicKey, ok := privateKey.Public().(*rsa.PublicKey)
	if !ok {
		typ := reflect.TypeOf(privateKey.Public())
		return nil, fmt.Errorf("unexpected key type (expected rsa.PublicKey, got %v)", typ)
	}
	// jwkJSON, err := ToJWKS(pKey)
	// if err != nil {
	// 	return nil, err
	// }
	// jwkSet, err := jwk.Parse(jwkJSON)
	// if err != nil {
	// 	return nil, err
	// }
	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		// PublicKeyPEM: ToPEM(key),
		// JWKSet:       jwkSet,
		// JWKJson:      jwkJSON,
	}, nil
}

// ParseSigningKey ...
func ParseSigningKey(keyData []byte) (*rsa.PrivateKey, error) {
	return jwt.ParseRSAPrivateKeyFromPEM(keyData)
}

// LoadSigningKeyFromFile ...
func LoadSigningKeyFromFile(privateKeyFile string) (*rsa.PrivateKey, error) {
	data, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", privateKeyFile, err)
	}
	key, err := ParseSigningKey(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key PEM file at %s: %v", privateKeyFile, err)
	}
	return key, nil
}

// ParseJwkSet ...
func ParseJwkSet(jwkSetData []byte) (jwk.Set, error) {
	return jwk.Parse(jwkSetData)
}

// LoadJwkSetFromFile ...
func LoadJwkSetFromFile(jwkSetFile string) (jwk.Set, error) {
	jwksData, err := ioutil.ReadFile(jwkSetFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", jwkSetFile, err)
	}
	jwkSet, err := ParseJwkSet(jwksData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jwk set at %s: %v", jwkSetFile, err)
	}
	return jwkSet, nil
}
