package ghinstallation

import (
	"crypto/rsa"

	jwt "github.com/golang-jwt/jwt/v4"
)

// Signer is a JWT token signer. This is a wrapper around [jwt.SigningMethod] with predetermined
// key material.
type Signer interface {
	// Sign signs the given claims and returns a JWT token string, as specified
	// by [jwt.Token.SignedString]
	Sign(claims jwt.Claims) (string, error)
}

// RSASigner signs JWT tokens using RSA keys.
type RSASigner struct {
	method *jwt.SigningMethodRSA
	key    *rsa.PrivateKey
}

func NewRSASigner(method *jwt.SigningMethodRSA, key *rsa.PrivateKey) *RSASigner {
	return &RSASigner{
		method: method,
		key:    key,
	}
}

// Sign signs the JWT claims with the RSA key.
func (s *RSASigner) Sign(claims jwt.Claims) (string, error) {
	return jwt.NewWithClaims(s.method, claims).SignedString(s.key)
}
