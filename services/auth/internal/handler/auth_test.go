package handler

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"cagie-shop/auth/internal/db"
	"cagie-shop/auth/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeStore struct {
	users  map[string]db.User // by email
	tokens map[string]db.RefreshToken
}

func newFake() *fakeStore {
	return &fakeStore{users: map[string]db.User{}, tokens: map[string]db.RefreshToken{}}
}

func (f *fakeStore) CreateUser(_ context.Context, a db.CreateUserParams) (db.User, error) {
	if _, dup := f.users[a.Email]; dup {
		return db.User{}, &pgconn.PgError{Code: "23505"}
	}
	u := db.User{ID: "id-" + strconv.Itoa(len(f.users)+1), Email: a.Email, PasswordHash: a.PasswordHash, Role: a.Role}
	f.users[a.Email] = u
	return u, nil
}
func (f *fakeStore) GetUserByEmail(_ context.Context, e string) (db.User, error) {
	if u, ok := f.users[e]; ok {
		return u, nil
	}
	return db.User{}, pgx.ErrNoRows
}
func (f *fakeStore) GetUserByID(_ context.Context, id string) (db.User, error) {
	for _, u := range f.users {
		if u.ID == id {
			return u, nil
		}
	}
	return db.User{}, pgx.ErrNoRows
}
func (f *fakeStore) CreateRefreshToken(_ context.Context, a db.CreateRefreshTokenParams) error {
	f.tokens[a.TokenHash] = db.RefreshToken{UserID: a.UserID, TokenHash: a.TokenHash, ExpiresAt: a.ExpiresAt}
	return nil
}
func (f *fakeStore) GetRefreshToken(_ context.Context, h string) (db.RefreshToken, error) {
	if t, ok := f.tokens[h]; ok {
		return t, nil
	}
	return db.RefreshToken{}, pgx.ErrNoRows
}
func (f *fakeStore) RevokeRefreshToken(_ context.Context, h string) (int64, error) {
	t, ok := f.tokens[h]
	if !ok || t.RevokedAt.Valid {
		return 0, nil
	}
	t.RevokedAt.Valid = true
	f.tokens[h] = t
	return 1, nil
}
func (f *fakeStore) RevokeAllUserTokens(_ context.Context, uid string) error {
	for h, t := range f.tokens {
		if t.UserID == uid {
			t.RevokedAt.Valid = true
			f.tokens[h] = t
		}
	}
	return nil
}

type env struct {
	r     *gin.Engine
	store *fakeStore
	now   time.Time
}

func setup(t *testing.T) *env {
	t.Helper()
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	signer, err := token.NewSigner(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}))
	if err != nil {
		t.Fatal(err)
	}
	e := &env{store: newFake(), now: time.Now()}
	h, err := NewAuth(e.store, signer, func() time.Time { return e.now }, true)
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	e.r = gin.New()
	h.Routes(e.r)
	return e
}

func (e *env) post(path, body string) *httptest.ResponseRecorder {
	return e.postWithCookie(path, body, "")
}

// postWithCookie sends the refresh cookie the way a browser would.
func (e *env) postWithCookie(path, body, refresh string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if refresh != "" {
		req.AddCookie(&http.Cookie{Name: RefreshCookie, Value: refresh})
	}
	w := httptest.NewRecorder()
	e.r.ServeHTTP(w, req)
	return w
}

type session struct {
	access  string
	refresh string
	cookie  *http.Cookie
}

func sessionFrom(t *testing.T, w *httptest.ResponseRecorder) session {
	t.Helper()
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if _, leaked := body["refreshToken"]; leaked {
		t.Fatal("the refresh token must never be in the response body")
	}
	s := session{}
	s.access, _ = body["accessToken"].(string)
	for _, c := range w.Result().Cookies() {
		if c.Name == RefreshCookie {
			s.refresh, s.cookie = c.Value, c
		}
	}
	return s
}

const creds = `{"email":"Ana@Example.com","password":"s3cret-pass"}`

func (e *env) loginTokens(t *testing.T) session {
	t.Helper()
	if w := e.post("/register", creds); w.Code != 201 {
		t.Fatalf("register: %d %s", w.Code, w.Body)
	}
	w := e.post("/login", creds)
	if w.Code != 200 {
		t.Fatalf("login: %d %s", w.Code, w.Body)
	}
	return sessionFrom(t, w)
}

func TestRegisterValidationAndDuplicate(t *testing.T) {
	e := setup(t)
	for name, body := range map[string]string{
		"bad email":      `{"email":"nope","password":"s3cret-pass"}`,
		"short password": `{"email":"a@b.co","password":"short"}`,
	} {
		if w := e.post("/register", body); w.Code != 400 {
			t.Errorf("%s: want 400, got %d", name, w.Code)
		}
	}
	if w := e.post("/register", creds); w.Code != 201 || strings.Contains(w.Body.String(), "password") {
		t.Fatalf("register: %d %s", w.Code, w.Body)
	}
	if w := e.post("/register", `{"email":"ana@example.com","password":"another-pass"}`); w.Code != 409 {
		t.Fatalf("email differing only by case must conflict, got %d", w.Code)
	}
}

func TestRegisterAlwaysCreatesCustomer(t *testing.T) {
	e := setup(t)
	e.post("/register", `{"email":"a@b.co","password":"s3cret-pass","role":"admin"}`)
	if got := e.store.users["a@b.co"].Role; got != "customer" {
		t.Fatalf("role = %q, want customer", got)
	}
}

func TestLoginFailuresLookIdentical(t *testing.T) {
	e := setup(t)
	e.post("/register", creds)
	wrongPass := e.post("/login", `{"email":"ana@example.com","password":"wrong-pass1"}`)
	noUser := e.post("/login", `{"email":"ghost@example.com","password":"wrong-pass1"}`)
	if wrongPass.Code != 401 || noUser.Code != 401 || wrongPass.Body.String() != noUser.Body.String() {
		t.Fatalf("responses differ: %d %s vs %d %s", wrongPass.Code, wrongPass.Body, noUser.Code, noUser.Body)
	}
}

func TestLoginSetsAProtectedRefreshCookie(t *testing.T) {
	e := setup(t)
	s := e.loginTokens(t)
	if s.access == "" || s.refresh == "" {
		t.Fatalf("access=%q refresh=%q", s.access, s.refresh)
	}
	c := s.cookie
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/auth" || c.MaxAge <= 0 {
		t.Fatalf("cookie flags wrong: %+v", c)
	}
}

func TestRefreshRotatesAndDetectsReuse(t *testing.T) {
	e := setup(t)
	first := e.loginTokens(t)

	w := e.postWithCookie("/refresh", "", first.refresh)
	if w.Code != 200 {
		t.Fatalf("refresh: %d %s", w.Code, w.Body)
	}
	second := sessionFrom(t, w)
	if second.refresh == "" || second.refresh == first.refresh || second.access == "" {
		t.Fatal("refresh must return a new access token and rotate the cookie")
	}

	// Replaying the old token is a theft signal: it fails and kills the new token too.
	if w := e.postWithCookie("/refresh", "", first.refresh); w.Code != 401 {
		t.Fatalf("reuse should be 401, got %d", w.Code)
	}
	if w := e.postWithCookie("/refresh", "", second.refresh); w.Code != 401 {
		t.Fatalf("new token should be revoked after reuse, got %d", w.Code)
	}
}

func TestRefreshRejectsMissingExpiredAndUnknown(t *testing.T) {
	e := setup(t)
	s := e.loginTokens(t)
	if w := e.post("/refresh", ""); w.Code != 401 {
		t.Fatalf("no cookie: %d", w.Code)
	}
	w := e.postWithCookie("/refresh", "", "unknown")
	if w.Code != 401 {
		t.Fatalf("unknown: %d", w.Code)
	}
	if c := sessionFrom(t, w).cookie; c == nil || c.MaxAge >= 0 {
		t.Fatal("a rejected refresh should clear the cookie")
	}
	e.now = e.now.Add(token.RefreshTTL + time.Minute)
	if w := e.postWithCookie("/refresh", "", s.refresh); w.Code != 401 {
		t.Fatalf("expired: %d", w.Code)
	}
}

func TestLogoutRevokesTokenAndClearsCookie(t *testing.T) {
	e := setup(t)
	s := e.loginTokens(t)
	w := e.postWithCookie("/logout", "", s.refresh)
	if w.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", w.Code)
	}
	if c := sessionFrom(t, w).cookie; c == nil || c.MaxAge >= 0 {
		t.Fatal("logout should clear the cookie")
	}
	if w := e.postWithCookie("/refresh", "", s.refresh); w.Code != 401 {
		t.Fatalf("token should be dead after logout, got %d", w.Code)
	}
	if w := e.post("/logout", ""); w.Code != http.StatusNoContent {
		t.Fatalf("logout without a session is still 204, got %d", w.Code)
	}
}
