package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io/ioutil"

	"github.com/golang-jwt/jwt/v4"
	"github.com/lestrrat-go/jwx/jwk"
)

// RSAKeyPair is an RSA key pair
type RSAKeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

// GenerateRSAKeyPair generates an RSA key pair
func GenerateRSAKeyPair() (*RSAKeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	publicKey := privateKey.Public()
	publicKeyRSA, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("expected key type rsa.PublicKey, but got %T", publicKey)
	}
	return &RSAKeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKeyRSA,
	}, nil
}

// ParseSigningKeyFromPEMData parses a private RSA signing key from PEM data
func ParseSigningKeyFromPEMData(keyData []byte) (*rsa.PrivateKey, error) {
	return jwt.ParseRSAPrivateKeyFromPEM(keyData)
}

// ParseSigningKeyFromPEMFile parses a private RSA signing key from a PEM file
func ParseSigningKeyFromPEMFile(path string) (*rsa.PrivateKey, error) {
	keyData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %v", path, err)
	}
	key, err := ParseSigningKeyFromPEMData(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s as private PEM signing key: %v", path, err)
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
