// Package gateway is the only public entrance. It checks tokens, enforces roles,
// rate limits, and forwards requests to the services behind it.
package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"strings"
	"time"

	"cagie-shop/gateway/internal/jwtverify"
	"cagie-shop/gateway/internal/ratelimit"
	"cagie-shop/shared/obs"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// Headers the gateway sets for services behind it. Clients can never set
// them: they are deleted from every incoming request before forwarding.
const (
	HeaderUserID    = "X-User-Id"
	HeaderUserRole  = "X-User-Role"
	HeaderRequestID = "X-Request-Id"
)

const maxBodyBytes = 1 << 20

type Config struct {
	Verifier       *jwtverify.Verifier
	AuthURL        *url.URL
	CatalogURL     *url.URL
	InventoryURL   *url.URL
	OrderURL       *url.URL
	PaymentURL     *url.URL
	AllowedOrigins []string
	General        *ratelimit.Limiter // every request, per IP
	Strict         *ratelimit.Limiter // login and register, per IP
	// TrustedProxies are the reverse proxies (Caddy in production) allowed to
	// tell the gateway the visitor's real IP via X-Forwarded-For. Empty means
	// trust nobody: the TCP peer is the client.
	TrustedProxies []string
}

type ctxKey int

const (
	claimsKey ctxKey = iota
	requestIDKey
)

func New(cfg Config) (*gin.Engine, error) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// X-Forwarded-For is believed only when the request comes from a trusted
	// proxy. Anyone else cannot fake an IP to dodge the rate limit, and behind
	// Caddy each visitor still gets their own limit instead of sharing Caddy's.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	// Traces start here: a client's own trace headers are dropped first, so
	// outsiders cannot join or fake traces inside the shop.
	r.Use(dropClientTrace(), obs.Middleware("gateway"), requestID(), obs.RequestLog(), gin.Recovery(),
		cors(cfg.AllowedOrigins), limitBody(), rateLimit(cfg.General))

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	authProxy := proxy(cfg.AuthURL, "/auth", true)
	auth := r.Group("/auth")
	auth.POST("/register", rateLimit(cfg.Strict), authProxy)
	auth.POST("/login", rateLimit(cfg.Strict), authProxy)
	auth.POST("/refresh", authProxy)
	auth.POST("/logout", authProxy)

	catalogProxy := proxy(cfg.CatalogURL, "", false)
	r.GET("/products", catalogProxy)
	r.GET("/products/:id", catalogProxy)
	admin := r.Group("/products", authenticate(cfg.Verifier), requireRole(jwtverify.RoleAdmin))
	admin.POST("", catalogProxy)
	admin.PUT("/:id", catalogProxy)
	admin.DELETE("/:id", catalogProxy)

	// Inventory: anyone can read stock, only admins change it. The service's
	// /internal/* routes are never registered here, so outsiders cannot reach them.
	inventoryProxy := proxy(cfg.InventoryURL, "", false)
	r.GET("/inventory/:id", inventoryProxy)
	r.PUT("/inventory/:id", authenticate(cfg.Verifier), requireRole(jwtverify.RoleAdmin), inventoryProxy)

	// Orders: any logged-in user, the order service filters by the user id header.
	orderProxy := proxy(cfg.OrderURL, "", false)
	orders := r.Group("/orders", authenticate(cfg.Verifier))
	orders.POST("", orderProxy)
	orders.GET("", orderProxy)
	orders.GET("/:id", orderProxy)

	// Payment has no public API. Admins can switch the fake provider's mode
	// (the payment service only serves this when FAKE_PROVIDER_CONTROLS=true).
	paymentProxy := proxy(cfg.PaymentURL, "", false)
	providerAdmin := r.Group("/payments/provider", authenticate(cfg.Verifier), requireRole(jwtverify.RoleAdmin))
	providerAdmin.GET("", paymentProxy)
	providerAdmin.PUT("", paymentProxy)

	return r, nil
}

func dropClientTrace() gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, h := range []string{"Traceparent", "Tracestate", "Baggage"} {
			c.Request.Header.Del(h)
		}
		c.Next()
	}
}

// requestID uses the trace id, so the id a customer reports matches the
// trace in Jaeger and every service's logs. Never taken from the client.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := ""
		if sc := trace.SpanContextFromContext(c.Request.Context()); sc.IsValid() {
			id = sc.TraceID().String()
		} else {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDKey, id))
		c.Header(HeaderRequestID, id)
		c.Next()
	}
}

func cors(allowed []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && slices.Contains(allowed, origin) {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			// Lets the web app send the refresh cookie to /auth. Safe only
			// because the origin is matched exactly against the allowlist.
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Add("Vary", "Origin")
			if c.Request.Method == http.MethodOptions {
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				h.Set("Access-Control-Max-Age", "600")
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
		}
		c.Next()
	}
}

func limitBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		c.Next()
	}
}

func rateLimit(l *ratelimit.Limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if l != nil && !l.Allow(c.ClientIP()) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}

func authenticate(v *jwtverify.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || raw == "" {
			unauthorized(c)
			return
		}
		claims, err := v.Verify(raw)
		if err != nil {
			unauthorized(c)
			return
		}
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), claimsKey, claims))
		c.Set("claims", claims)
		c.Next()
	}
}

// unauthorized gives the same answer for a missing, malformed, forged or expired token.
func unauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", `Bearer realm="cagie-shop"`)
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
}

func requireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := c.Request.Context().Value(claimsKey).(jwtverify.Claims)
		if !ok || claims.Role != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}

// proxy forwards to target. The client's identity headers and Authorization
// header are always removed; identity headers are re-added only from a
// verified token, so services behind the gateway can trust them. Cookies
// (the refresh token) reach only the auth service.
func proxy(target *url.URL, stripPrefix string, keepCookies bool) gin.HandlerFunc {
	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.URL.Path = "/" + strings.TrimPrefix(strings.TrimPrefix(pr.In.URL.Path, stripPrefix), "/")
			pr.Out.URL.RawPath = ""
			pr.SetXForwarded()
			h := pr.Out.Header
			h.Del(HeaderUserID)
			h.Del(HeaderUserRole)
			h.Del("Authorization")
			if !keepCookies {
				h.Del("Cookie")
			}
			if claims, ok := pr.In.Context().Value(claimsKey).(jwtverify.Claims); ok {
				h.Set(HeaderUserID, claims.UserID)
				h.Set(HeaderUserRole, claims.Role)
			}
			if id, ok := pr.In.Context().Value(requestIDKey).(string); ok {
				h.Set(HeaderRequestID, id)
			}
		},
		// otelhttp adds the traceparent header, so services continue the trace.
		Transport: otelhttp.NewTransport(&http.Transport{
			DialContext:           (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
			ResponseHeaderTimeout: 10 * time.Second,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       60 * time.Second,
		}),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.ErrorContext(r.Context(), "upstream error", "upstream", target.Host, "error", err.Error())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"upstream unavailable"}`))
		},
	}
	return func(c *gin.Context) { rp.ServeHTTP(c.Writer, c.Request) }
}
