// Package jwtverify checks access tokens with the auth service's public key.
package jwtverify

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Issuer, Audience and the role names are a contract with the auth service.
// Keep them in sync with services/auth/internal/token.
const (
	Issuer       = "cagie-auth"
	Audience     = "cagie-shop"
	RoleCustomer = "customer"
	RoleAdmin    = "admin"
)

type Claims struct {
	UserID string
	Role   string
}

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type Verifier struct {
	parser *jwt.Parser
	key    any
}

// New parses an RSA public key in PEM form. Only RS256 is accepted, which
// blocks "alg: none" and the HS256-with-public-key confusion attack.
func New(pem []byte) (*Verifier, error) {
	key, err := jwt.ParseRSAPublicKeyFromPEM(pem)
	if err != nil {
		return nil, err
	}
	return &Verifier{
		key: key,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithIssuer(Issuer),
			jwt.WithAudience(Audience),
			jwt.WithExpirationRequired(),
			jwt.WithLeeway(10*time.Second), // clock skew between hosts
		),
	}, nil
}

func (v *Verifier) Verify(raw string) (Claims, error) {
	var c claims
	if _, err := v.parser.ParseWithClaims(raw, &c, func(*jwt.Token) (any, error) { return v.key, nil }); err != nil {
		return Claims{}, err
	}
	if c.Subject == "" || (c.Role != RoleCustomer && c.Role != RoleAdmin) {
		return Claims{}, errors.New("token is missing subject or has an unknown role")
	}
	return Claims{UserID: c.Subject, Role: c.Role}, nil
}
