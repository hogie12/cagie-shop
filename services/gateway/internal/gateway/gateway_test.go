package gateway

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"cagie-shop/gateway/internal/jwtverify"
	"cagie-shop/gateway/internal/ratelimit"
	"github.com/golang-jwt/jwt/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"golang.org/x/time/rate"
)

// upstream records the last request it received.
type upstream struct {
	srv  *httptest.Server
	mu   sync.Mutex
	path string
	hdr  http.Header
	body string
}

func newUpstream(t *testing.T) *upstream {
	u := &upstream{}
	u.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		u.mu.Lock()
		u.path, u.hdr, u.body = r.URL.Path, r.Header.Clone(), string(b)
		u.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(u.srv.Close)
	return u
}

func (u *upstream) last() (string, http.Header) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.path, u.hdr
}

type env struct {
	h         http.Handler
	auth      *upstream
	catalog   *upstream
	inventory *upstream
	orders    *upstream
	payment   *upstream
	key       *rsa.PrivateKey
	pubPEM    []byte
}

func setup(t *testing.T, mutate ...func(*Config)) *env {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pubDER, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	v, err := jwtverify.New(pubPEM)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{auth: newUpstream(t), catalog: newUpstream(t), inventory: newUpstream(t), orders: newUpstream(t), payment: newUpstream(t), key: key, pubPEM: pubPEM}
	au, _ := url.Parse(e.auth.srv.URL)
	cu, _ := url.Parse(e.catalog.srv.URL)
	iu, _ := url.Parse(e.inventory.srv.URL)
	ou, _ := url.Parse(e.orders.srv.URL)
	pu, _ := url.Parse(e.payment.srv.URL)
	cfg := Config{
		Verifier: v, AuthURL: au, CatalogURL: cu, InventoryURL: iu, OrderURL: ou, PaymentURL: pu,
		AllowedOrigins: []string{"https://shop.example.com"},
		General:        ratelimit.New(rate.Limit(1000), 1000),
		Strict:         ratelimit.New(rate.Limit(1000), 1000),
	}
	for _, m := range mutate {
		m(&cfg)
	}
	e.h = New(cfg)
	return e
}

type tokenOpts struct {
	role, sub, iss, aud string
	exp                 time.Time
}

func (e *env) token(t *testing.T, o tokenOpts) string {
	t.Helper()
	if o.sub == "" {
		o.sub = "user-1"
	}
	if o.iss == "" {
		o.iss = jwtverify.Issuer
	}
	if o.aud == "" {
		o.aud = jwtverify.Audience
	}
	if o.exp.IsZero() {
		o.exp = time.Now().Add(10 * time.Minute)
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": o.sub, "role": o.role, "iss": o.iss, "aud": []string{o.aud},
		"iat": time.Now().Unix(), "exp": o.exp.Unix(),
	}).SignedString(e.key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func (e *env) do(method, path, body string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "203.0.113.7:5555"
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	e.h.ServeHTTP(closeNotifyRecorder{w}, req)
	return w
}

// The reverse proxy asks its writer for CloseNotify, which a real HTTP server
// provides but httptest.ResponseRecorder does not.
type closeNotifyRecorder struct{ *httptest.ResponseRecorder }

func (closeNotifyRecorder) CloseNotify() <-chan bool { return make(chan bool) }

func bearer(tok string) map[string]string { return map[string]string{"Authorization": "Bearer " + tok} }

func TestPublicRoutesNeedNoToken(t *testing.T) {
	e := setup(t)
	if w := e.do("GET", "/products", "", nil); w.Code != 200 {
		t.Fatalf("GET /products: %d", w.Code)
	}
	if p, _ := e.catalog.last(); p != "/products" {
		t.Fatalf("upstream path = %q", p)
	}
	if w := e.do("GET", "/products/3", "", nil); w.Code != 200 {
		t.Fatalf("GET /products/3: %d", w.Code)
	}
}

func TestProductWritesRequireAdmin(t *testing.T) {
	e := setup(t)
	body := `{"name":"x"}`
	if w := e.do("POST", "/products", body, nil); w.Code != 401 {
		t.Fatalf("no token: want 401, got %d", w.Code)
	}
	cust := e.token(t, tokenOpts{role: "customer"})
	for _, m := range []string{"POST /products", "PUT /products/1", "DELETE /products/1"} {
		parts := strings.SplitN(m, " ", 2)
		if w := e.do(parts[0], parts[1], body, bearer(cust)); w.Code != 403 {
			t.Fatalf("customer %s: want 403, got %d", m, w.Code)
		}
	}
	admin := e.token(t, tokenOpts{role: "admin", sub: "admin-9"})
	if w := e.do("POST", "/products", body, bearer(admin)); w.Code != 200 {
		t.Fatalf("admin: want 200, got %d", w.Code)
	}
	_, h := e.catalog.last()
	if h.Get(HeaderUserID) != "admin-9" || h.Get(HeaderUserRole) != "admin" {
		t.Fatalf("identity headers wrong: %v", h)
	}
}

func TestBadTokensAreAllRejected(t *testing.T) {
	e := setup(t)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	forged, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "u", "role": "admin", "iss": jwtverify.Issuer, "aud": []string{jwtverify.Audience},
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(other)
	algNone, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "u", "role": "admin", "iss": jwtverify.Issuer, "aud": []string{jwtverify.Audience},
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	// Classic confusion attack: sign with HS256 using the public key as the secret.
	hs256, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "u", "role": "admin", "iss": jwtverify.Issuer, "aud": []string{jwtverify.Audience},
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(e.pubPEM)
	noExp, _ := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "u", "role": "admin", "iss": jwtverify.Issuer, "aud": []string{jwtverify.Audience},
	}).SignedString(e.key)

	cases := map[string]string{
		"signed by another key": forged,
		"alg none":              algNone,
		"hs256 with public key": hs256,
		"no expiry":             noExp,
		"expired":               e.token(t, tokenOpts{role: "admin", exp: time.Now().Add(-time.Hour)}),
		"wrong issuer":          e.token(t, tokenOpts{role: "admin", iss: "evil"}),
		"wrong audience":        e.token(t, tokenOpts{role: "admin", aud: "other-app"}),
		"unknown role":          e.token(t, tokenOpts{role: "superuser"}),
		"garbage":               "not.a.jwt",
	}
	for name, tok := range cases {
		if w := e.do("POST", "/products", `{}`, bearer(tok)); w.Code != 401 {
			t.Errorf("%s: want 401, got %d", name, w.Code)
		}
	}
	if w := e.do("POST", "/products", `{}`, map[string]string{"Authorization": "Basic abc"}); w.Code != 401 {
		t.Errorf("non-bearer scheme: want 401, got %d", w.Code)
	}
}

func TestClientCannotSpoofIdentityHeaders(t *testing.T) {
	e := setup(t)
	spoof := map[string]string{
		HeaderUserID: "evil", HeaderUserRole: "admin", "Authorization": "Bearer garbage",
	}
	// Public route: spoofed headers must not reach the service.
	e.do("GET", "/products", "", spoof)
	_, h := e.catalog.last()
	if h.Get(HeaderUserID) != "" || h.Get(HeaderUserRole) != "" || h.Get("Authorization") != "" {
		t.Fatalf("spoofed headers reached upstream: %v", h)
	}
	// Customer token plus a claim of admin in a header is still just a customer.
	cust := e.token(t, tokenOpts{role: "customer"})
	if w := e.do("POST", "/products", `{}`, map[string]string{
		"Authorization": "Bearer " + cust, HeaderUserRole: "admin",
	}); w.Code != 403 {
		t.Fatalf("want 403, got %d", w.Code)
	}
	// Valid admin token: identity comes from the token, not from a spoofed header.
	admin := e.token(t, tokenOpts{role: "admin", sub: "real-admin"})
	e.do("POST", "/products", `{}`, map[string]string{
		"Authorization": "Bearer " + admin, HeaderUserID: "evil",
	})
	_, h = e.catalog.last()
	if h.Get(HeaderUserID) != "real-admin" || h.Get("Authorization") != "" {
		t.Fatalf("headers wrong: %v", h)
	}
}

func TestAuthRoutesStripPrefixAndForwardBody(t *testing.T) {
	e := setup(t)
	if w := e.do("POST", "/auth/login", `{"email":"a@b.co"}`, nil); w.Code != 200 {
		t.Fatalf("login: %d", w.Code)
	}
	p, _ := e.auth.last()
	if p != "/login" || e.auth.body != `{"email":"a@b.co"}` {
		t.Fatalf("upstream got path=%q body=%q", p, e.auth.body)
	}
	if w := e.do("GET", "/auth/login", "", nil); w.Code != 404 {
		t.Fatalf("GET /auth/login: want 404, got %d", w.Code)
	}
	if w := e.do("GET", "/internal/secret", "", nil); w.Code != 404 {
		t.Fatalf("unknown path: want 404, got %d", w.Code)
	}
}

func TestStrictRateLimitOnLoginIgnoresForwardedFor(t *testing.T) {
	e := setup(t, func(c *Config) { c.Strict = ratelimit.New(rate.Every(time.Hour), 2) })
	codes := []int{}
	for _, ip := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
		w := e.do("POST", "/auth/login", `{}`, map[string]string{"X-Forwarded-For": ip})
		codes = append(codes, w.Code)
	}
	if codes[0] != 200 || codes[1] != 200 || codes[2] != 429 {
		t.Fatalf("codes = %v, want [200 200 429] (X-Forwarded-For must not reset the limit)", codes)
	}
	// Other routes are not affected by the strict limiter.
	if w := e.do("GET", "/products", "", nil); w.Code != 200 {
		t.Fatalf("products: %d", w.Code)
	}
}

func TestCORS(t *testing.T) {
	e := setup(t)
	w := e.do("OPTIONS", "/products", "", map[string]string{"Origin": "https://shop.example.com"})
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") != "https://shop.example.com" ||
		w.Header().Get("Access-Control-Allow-Credentials") != "true" ||
		!strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatalf("preflight failed: %d %v", w.Code, w.Header())
	}
	w = e.do("GET", "/products", "", map[string]string{"Origin": "https://evil.example.com"})
	if w.Header().Get("Access-Control-Allow-Origin") != "" || w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("disallowed origin must not get CORS headers")
	}
	w = e.do("OPTIONS", "/products", "", map[string]string{"Origin": "https://evil.example.com"})
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("disallowed preflight must not get CORS headers")
	}
}

func TestUpstreamDownReturns502(t *testing.T) {
	e := setup(t)
	e.catalog.srv.Close()
	w := e.do("GET", "/products", "", nil)
	if w.Code != http.StatusBadGateway || !strings.Contains(w.Body.String(), "upstream unavailable") {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
}

func TestRequestIDIsGeneratedAndForwarded(t *testing.T) {
	e := setup(t)
	w := e.do("GET", "/products", "", map[string]string{HeaderRequestID: "client-chosen"})
	id := w.Header().Get(HeaderRequestID)
	_, h := e.catalog.last()
	if id == "" || id == "client-chosen" || h.Get(HeaderRequestID) != id {
		t.Fatalf("response id %q, upstream id %q", id, h.Get(HeaderRequestID))
	}
}

func TestOrdersNeedAnyValidToken(t *testing.T) {
	e := setup(t)
	if w := e.do("POST", "/orders", `{}`, nil); w.Code != 401 {
		t.Fatalf("no token: want 401, got %d", w.Code)
	}
	if w := e.do("GET", "/orders", "", bearer("garbage")); w.Code != 401 {
		t.Fatalf("bad token: want 401, got %d", w.Code)
	}
	cust := e.token(t, tokenOpts{role: "customer", sub: "ana"})
	for _, m := range []string{"POST /orders", "GET /orders", "GET /orders/abc"} {
		parts := strings.SplitN(m, " ", 2)
		if w := e.do(parts[0], parts[1], `{}`, bearer(cust)); w.Code != 200 {
			t.Fatalf("%s: want 200, got %d", m, w.Code)
		}
	}
	_, h := e.orders.last()
	if h.Get(HeaderUserID) != "ana" || h.Get(HeaderUserRole) != "customer" {
		t.Fatalf("identity headers wrong: %v", h)
	}
}

func TestInventoryReadIsPublicWriteIsAdminOnly(t *testing.T) {
	e := setup(t)
	if w := e.do("GET", "/inventory/1", "", nil); w.Code != 200 {
		t.Fatalf("public read: %d", w.Code)
	}
	if w := e.do("PUT", "/inventory/1", `{"quantity":5}`, nil); w.Code != 401 {
		t.Fatalf("no token: want 401, got %d", w.Code)
	}
	if w := e.do("PUT", "/inventory/1", `{"quantity":5}`, bearer(e.token(t, tokenOpts{role: "customer"}))); w.Code != 403 {
		t.Fatalf("customer: want 403, got %d", w.Code)
	}
	if w := e.do("PUT", "/inventory/1", `{"quantity":5}`, bearer(e.token(t, tokenOpts{role: "admin"}))); w.Code != 200 {
		t.Fatalf("admin: want 200, got %d", w.Code)
	}
}

func TestInternalRoutesAreNeverReachableFromOutside(t *testing.T) {
	e := setup(t)
	admin := bearer(e.token(t, tokenOpts{role: "admin"}))
	for _, path := range []string{"/internal/reserve/1", "/internal/release/1", "/internal/anything"} {
		for _, hdr := range []map[string]string{nil, admin} {
			if w := e.do("POST", path, `{"quantity":1}`, hdr); w.Code != 404 {
				t.Errorf("%s: want 404, got %d", path, w.Code)
			}
		}
	}
	if p, _ := e.inventory.last(); p != "" {
		t.Fatalf("inventory service was reached at %q", p)
	}
}

func TestProviderControlsAreAdminOnly(t *testing.T) {
	e := setup(t)
	for _, m := range []string{"GET", "PUT"} {
		if w := e.do(m, "/payments/provider", `{"mode":"down"}`, nil); w.Code != 401 {
			t.Errorf("%s no token: want 401, got %d", m, w.Code)
		}
		if w := e.do(m, "/payments/provider", `{"mode":"down"}`, bearer(e.token(t, tokenOpts{role: "customer"}))); w.Code != 403 {
			t.Errorf("%s customer: want 403, got %d", m, w.Code)
		}
	}
	if p, _ := e.payment.last(); p != "" {
		t.Fatalf("payment service was reached at %q without an admin token", p)
	}
	if w := e.do("PUT", "/payments/provider", `{"mode":"down"}`, bearer(e.token(t, tokenOpts{role: "admin", sub: "boss"}))); w.Code != 200 {
		t.Fatalf("admin: want 200, got %d", w.Code)
	}
	p, h := e.payment.last()
	if p != "/payments/provider" || h.Get(HeaderUserRole) != "admin" {
		t.Fatalf("upstream got path=%q role=%q", p, h.Get(HeaderUserRole))
	}
}

func TestNothingElseOfPaymentIsPublic(t *testing.T) {
	e := setup(t)
	admin := bearer(e.token(t, tokenOpts{role: "admin"}))
	for _, path := range []string{"/payments", "/payments/1", "/payments/provider/extra"} {
		for _, hdr := range []map[string]string{nil, admin} {
			if w := e.do("GET", path, "", hdr); w.Code != 404 {
				t.Errorf("%s: want 404, got %d", path, w.Code)
			}
		}
	}
}

func TestRefreshCookieReachesOnlyAuth(t *testing.T) {
	e := setup(t)
	cookie := map[string]string{"Cookie": "cagie_refresh=secret-value"}
	e.do("POST", "/auth/refresh", "", cookie)
	if _, h := e.auth.last(); h.Get("Cookie") != "cagie_refresh=secret-value" {
		t.Fatalf("auth must receive the cookie, got %q", h.Get("Cookie"))
	}
	e.do("GET", "/products", "", cookie)
	if _, h := e.catalog.last(); h.Get("Cookie") != "" {
		t.Fatalf("catalog must not receive cookies, got %q", h.Get("Cookie"))
	}
	cust := e.token(t, tokenOpts{role: "customer"})
	e.do("GET", "/orders", "", map[string]string{"Authorization": "Bearer " + cust, "Cookie": "cagie_refresh=secret-value"})
	if _, h := e.orders.last(); h.Get("Cookie") != "" {
		t.Fatalf("orders must not receive cookies, got %q", h.Get("Cookie"))
	}
}

func TestTracesStartAtTheGateway(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	defer otel.SetTracerProvider(noop.NewTracerProvider())

	e := setup(t)
	const fake = "00-11111111111111111111111111111111-2222222222222222-01"
	w := e.do("GET", "/products", "", map[string]string{"traceparent": fake})
	_, h := e.catalog.last()
	got := h.Get("Traceparent")
	if got == "" || got == fake || strings.Contains(got, "11111111111111111111111111111111") {
		t.Fatalf("upstream traceparent = %q: the gateway must start a new trace", got)
	}
	// The request id the customer sees is the trace id the services see.
	if traceID := strings.Split(got, "-")[1]; w.Header().Get(HeaderRequestID) != traceID {
		t.Fatalf("request id %q != trace id %q", w.Header().Get(HeaderRequestID), traceID)
	}
}
