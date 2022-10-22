package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/lestrrat-go/jwx/jwk"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword creates a cryptograhic hash of a password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// MustHashPassword creates a cryptographic hash of a password
//
// Panics if the password cannot be hashed.
func MustHashPassword(pw string) string {
	hashed, err := HashPassword(pw)
	if err != nil {
		panic(err)
	}
	return hashed
}

// CheckPasswordHash compares a password against its hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// JWK encodes a JSON web key
type JWK struct {
	KID       string `json:"kid"`
	Algorithm string `json:"alg"`
	E         string `json:"e"`
	KTY       string `json:"kty"`
	N         string `json:"n"`
}

// ToJwks converts a RSA public key to a JWK set
func ToJwks(pub *rsa.PublicKey) (jwk.Set, error) {
	jwkJSON, err := ToJwksJson(pub)
	if err != nil {
		return nil, err
	}
	jwkSet, err := jwk.Parse(jwkJSON)
	if err != nil {
		return nil, err
	}
	return jwkSet, nil
}

// ToJwksJson converts a RSA public key to a JSON encoded JWK set
func ToJwksJson(pub *rsa.PublicKey) ([]byte, error) {
	// See https://github.com/golang/crypto/blob/master/acme/jws.go#L90
	// https://tools.ietf.org/html/rfc7518#section-6.3.1
	n := pub.N
	e := big.NewInt(int64(pub.E))
	// Field order is important.
	// See https://tools.ietf.org/html/rfc7638#section-3.3 for details.
	return json.Marshal(JWK{
		KID:       "0",
		Algorithm: "RS256",
		E:         base64.RawURLEncoding.EncodeToString(e.Bytes()),
		KTY:       "RSA",
		N:         base64.RawURLEncoding.EncodeToString(n.Bytes()),
	})
}

// ToPEM converts a RSA private key into PEM format
func ToPEM(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)
}

// SignJwtClaims signs JWT claims
func (a *Authenticator) SignJwtClaims(claims Claims) (string, error) {
	expirationTime := time.Now().Add(time.Duration(a.ExpireSeconds) * time.Second)
	// add metadata to the the JWT claims, which should include some user ID
	sc := claims.GetStandardClaims()
	// expiry time is expressed as unix milliseconds
	sc.ExpiresAt = expirationTime.Unix()
	sc.Issuer = a.Issuer
	sc.Audience = a.Audience

	// create the token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "0"
	token.Header["alg"] = "RS256"

	return token.SignedString(a.SignKey)
}
