package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cagie-shop/auth/internal/db"
	"cagie-shop/auth/internal/handler"
	"cagie-shop/auth/internal/migrate"
	"cagie-shop/auth/internal/password"
	"cagie-shop/auth/internal/token"
	"cagie-shop/shared/obs"
	"github.com/gin-gonic/gin"
)

func main() {
	dsn := mustEnv("DATABASE_URL")
	keyPEM, err := os.ReadFile(mustEnv("JWT_PRIVATE_KEY_FILE"))
	if err != nil {
		log.Fatalf("read private key: %v", err)
	}
	signer, err := token.NewSigner(keyPEM)
	if err != nil {
		log.Fatalf("parse private key: %v", err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := obs.Setup(ctx, "auth")
	if err != nil {
		log.Fatalf("tracing: %v", err)
	}
	defer flush(shutdownTracing)

	pool, err := obs.NewPool(ctx, dsn)
	if err != nil {
		log.Fatalf("db pool: %v", err)
	}
	defer pool.Close()
	if err := migrate.Up(dsn); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	store := db.New(pool)
	if err := bootstrapAdmin(ctx, store); err != nil {
		log.Fatalf("bootstrap admin: %v", err)
	}

	auth, err := handler.NewAuth(store, signer, time.Now, os.Getenv("COOKIE_SECURE") == "true")
	if err != nil {
		log.Fatalf("handler: %v", err)
	}
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	// Request bodies hold passwords; the request log records only method, path and status.
	r.Use(obs.Middleware("auth"), obs.RequestLog(), gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) {
		pingCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	auth.Routes(r)

	srv := &http.Server{Addr: ":" + port, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("auth listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

// bootstrapAdmin creates the first admin from env vars when no admin exists.
func bootstrapAdmin(ctx context.Context, store *db.Queries) error {
	email, pass := os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD")
	if email == "" || pass == "" {
		return nil
	}
	n, err := store.CountAdmins(ctx)
	if err != nil || n > 0 {
		return err
	}
	hash, err := password.Hash(pass)
	if err != nil {
		return err
	}
	_, err = store.CreateUser(ctx, db.CreateUserParams{
		Email: strings.ToLower(strings.TrimSpace(email)), PasswordHash: hash, Role: token.RoleAdmin,
	})
	if err == nil {
		log.Printf("created initial admin account")
	}
	return err
}

func mustEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("%s is required", name)
	}
	return v
}

// flush sends any spans still buffered before the process exits.
func flush(shutdown func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = shutdown(ctx)
}
