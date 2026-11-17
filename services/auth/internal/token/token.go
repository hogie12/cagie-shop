// Package token issues access tokens (signed JWTs) and refresh tokens (opaque random strings).
package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Issuer, Audience and the role names are a contract with the gateway,
// which rejects tokens that do not match. Keep them in sync.
const (
	Issuer       = "cagie-auth"
	Audience     = "cagie-shop"
	AccessTTL    = 15 * time.Minute
	RefreshTTL   = 7 * 24 * time.Hour
	RoleCustomer = "customer"
	RoleAdmin    = "admin"
)

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type Signer struct{ key *rsa.PrivateKey }

// NewSigner parses an RSA private key in PEM form (PKCS#1 or PKCS#8).
func NewSigner(pem []byte) (*Signer, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(pem)
	if err != nil {
		return nil, err
	}
	return &Signer{key: key}, nil
}

func (s *Signer) Issue(userID, role string, now time.Time) (string, error) {
	c := claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Audience:  jwt.ClaimStrings{Audience},
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodRS256, c).SignedString(s.key)
}

// NewRefresh returns a random token for the client and the hash to store.
func NewRefresh() (plain, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	return plain, HashRefresh(plain), nil
}

func HashRefresh(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
