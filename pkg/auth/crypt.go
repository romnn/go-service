package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/lestrrat-go/jwx/jwk"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword creates a cryptograhic hash of a password
func HashPassword(password string) (string, error) {
	// bcrypt.MaxCost takes very very very long
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost+4)
	return string(bytes), err
}

// MustHashPassword creates a cryptographic hash of a password or panics
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
	jwkJSON, err := ToJwksJSON(pub)
	if err != nil {
		return nil, err
	}
	jwkSet, err := jwk.Parse(jwkJSON)
	if err != nil {
		return nil, err
	}
	return jwkSet, nil
}

// ToJwksJSON converts a RSA public key to a JSON encoded JWK set
func ToJwksJSON(pub *rsa.PublicKey) ([]byte, error) {
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

// SignJwtClaims signs JWT claims using RS256 and returns the token string
func (auth *Authenticator) SignJwtClaims(claims Claims) (string, error) {
	expirationTime := time.Now().Add(time.Duration(auth.ExpireSeconds) * time.Second)

	// set structured JWT claims set
	// https://pkg.go.dev/github.com/golang-jwt/jwt/v4#RegisteredClaims
	// https://datatracker.ietf.org/doc/html/rfc7519#section-4.1
	reg := claims.GetRegisteredClaims()
	reg.ExpiresAt = jwt.NewNumericDate(expirationTime)
	reg.Issuer = auth.Issuer
	reg.Audience = jwt.ClaimStrings([]string{auth.Audience})

	// create the token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "0"
	token.Header["alg"] = "RS256"

	// sign the token
	return token.SignedString(auth.SignKey)
}
