package token

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func newKey(t *testing.T) (*Signer, *rsa.PublicKey) {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSigner(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}))
	if err != nil {
		t.Fatal(err)
	}
	return s, &k.PublicKey
}

func TestIssuedTokenVerifiesWithPublicKey(t *testing.T) {
	s, pub := newKey(t)
	raw, err := s.Issue("user-1", RoleAdmin, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var c claims
	tok, err := jwt.ParseWithClaims(raw, &c, func(*jwt.Token) (any, error) { return pub, nil },
		jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer(Issuer), jwt.WithAudience(Audience))
	if err != nil || !tok.Valid {
		t.Fatalf("token should verify: %v", err)
	}
	if c.Subject != "user-1" || c.Role != RoleAdmin {
		t.Fatalf("wrong claims: %+v", c)
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	s, pub := newKey(t)
	raw, _ := s.Issue("u", RoleCustomer, time.Now().Add(-time.Hour))
	_, err := jwt.Parse(raw, func(*jwt.Token) (any, error) { return pub, nil }, jwt.WithValidMethods([]string{"RS256"}))
	if err == nil {
		t.Fatal("expired token accepted")
	}
}

func TestRefreshTokensAreUniqueAndHashed(t *testing.T) {
	p1, h1, _ := NewRefresh()
	p2, _, _ := NewRefresh()
	if p1 == p2 || h1 == p1 || HashRefresh(p1) != h1 {
		t.Fatal("refresh token generation is wrong")
	}
}
