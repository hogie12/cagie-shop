package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cagie-shop/gateway/internal/gateway"
	"cagie-shop/gateway/internal/jwtverify"
	"cagie-shop/gateway/internal/ratelimit"
	"cagie-shop/shared/obs"
	"golang.org/x/time/rate"
)

func main() {
	pem, err := os.ReadFile(mustEnv("JWT_PUBLIC_KEY_FILE"))
	if err != nil {
		log.Fatalf("read public key: %v", err)
	}
	verifier, err := jwtverify.New(pem)
	if err != nil {
		log.Fatalf("parse public key: %v", err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := obs.Setup(ctx, "gateway")
	if err != nil {
		log.Fatalf("tracing: %v", err)
	}
	defer flush(shutdownTracing)

	general := ratelimit.New(rate.Limit(20), 40)           // 20 requests/second per IP
	strict := ratelimit.New(rate.Every(12*time.Second), 5) // about 5 login/register attempts per minute per IP
	go general.RunSweeper(ctx)
	go strict.RunSweeper(ctx)

	h, err := gateway.New(gateway.Config{
		Verifier:       verifier,
		AuthURL:        mustURL("AUTH_URL"),
		CatalogURL:     mustURL("CATALOG_URL"),
		InventoryURL:   mustURL("INVENTORY_URL"),
		OrderURL:       mustURL("ORDER_URL"),
		PaymentURL:     mustURL("PAYMENT_URL"),
		AllowedOrigins: splitList(os.Getenv("CORS_ALLOWED_ORIGINS")),
		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES")),
		General:        general,
		Strict:         strict,
	})
	if err != nil {
		log.Fatalf("gateway config: %v", err)
	}
	srv := &http.Server{
		Addr: ":" + port, Handler: h,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	go func() {
		log.Printf("gateway listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func mustEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("%s is required", name)
	}
	return v
}

func mustURL(name string) *url.URL {
	u, err := url.Parse(mustEnv(name))
	if err != nil || u.Scheme == "" || u.Host == "" {
		log.Fatalf("%s must be a full URL like http://host:port", name)
	}
	return u
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// flush sends any spans still buffered before the process exits.
func flush(shutdown func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = shutdown(ctx)
}
