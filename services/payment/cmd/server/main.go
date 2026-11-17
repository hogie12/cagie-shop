package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cagie-shop/payment/internal/consumer"
	"cagie-shop/payment/internal/handler"
	"cagie-shop/payment/internal/migrate"
	"cagie-shop/payment/internal/provider"
	"cagie-shop/payment/internal/store"
	shared "cagie-shop/shared"
	"cagie-shop/shared/obs"
	"github.com/gin-gonic/gin"
)

func main() {
	dsn := mustEnv("DATABASE_URL")
	rabbitURL := mustEnv("RABBITMQ_URL")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := obs.Setup(ctx, "payment")
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

	fake := provider.NewFake()
	pub := shared.NewPublisher(rabbitURL)
	defer pub.Close()
	go (&shared.Relay{Pool: pool, Pub: pub}).Run(ctx)
	go (&shared.Consumer{URL: rabbitURL, Queue: shared.QueuePayment, Handler: consumer.PaymentRequested(store.New(pool, fake))}).Run(ctx)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(obs.Middleware("payment"), obs.RequestLog(), gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) {
		pingCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	if os.Getenv("FAKE_PROVIDER_CONTROLS") == "true" {
		log.Print("WARNING: fake provider controls are enabled")
		handler.Routes(r, fake)
	}

	srv := &http.Server{Addr: ":" + port, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("payment listening on :%s", port)
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

// flush sends any spans still buffered before the process exits.
func flush(shutdown func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = shutdown(ctx)
}
