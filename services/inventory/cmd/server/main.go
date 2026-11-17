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

	"cagie-shop/inventory/internal/consumer"
	"cagie-shop/inventory/internal/db"
	"cagie-shop/inventory/internal/handler"
	"cagie-shop/inventory/internal/migrate"
	"cagie-shop/inventory/internal/store"
	shared "cagie-shop/shared"
	"cagie-shop/shared/obs"
	"github.com/gin-gonic/gin"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Fatal("RABBITMQ_URL is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := obs.Setup(ctx, "inventory")
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

	// Outbox relay: publishes the events this service saved. Consumer: reacts to orders.
	pub := shared.NewPublisher(rabbitURL)
	defer pub.Close()
	go (&shared.Relay{Pool: pool, Pub: pub}).Run(ctx)
	go (&shared.Consumer{URL: rabbitURL, Queue: shared.QueueInventory, Handler: consumer.Events(store.New(pool))}).Run(ctx)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(obs.Middleware("inventory"), obs.RequestLog(), gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) {
		pingCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(pingCtx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	handler.NewStock(db.New(pool)).Routes(r)

	srv := &http.Server{Addr: ":" + port, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("inventory listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

// flush sends any spans still buffered before the process exits.
func flush(shutdown func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = shutdown(ctx)
}
