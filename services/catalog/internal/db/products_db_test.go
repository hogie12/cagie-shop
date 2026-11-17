package db_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"cagie-shop/catalog/internal/db"
	"cagie-shop/catalog/internal/migrate"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Runs against a real Postgres. Set TEST_DATABASE_URL to a throwaway database.
func TestProductLifecycle(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	if err := migrate.Up(dsn); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	q := db.New(pool)

	p, err := q.CreateProduct(ctx, db.CreateProductParams{Name: "Test Tart", Category: "pastry", PriceIdr: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.CreateProduct(ctx, db.CreateProductParams{Name: "Bad", Category: "car"}); err == nil {
		t.Fatal("category CHECK constraint should reject 'car'")
	}

	flowers, err := q.ListProducts(ctx, pgtype.Text{String: "flower", Valid: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range flowers {
		if f.Category != "flower" {
			t.Fatalf("filter leaked %q", f.Category)
		}
	}

	if _, err := q.UpdateProduct(ctx, db.UpdateProductParams{ID: p.ID, Name: "Renamed", Category: "pastry", PriceIdr: 2000}); err != nil {
		t.Fatal(err)
	}
	got, _ := q.GetProduct(ctx, p.ID)
	if got.Name != "Renamed" || got.PriceIdr != 2000 {
		t.Fatalf("update not applied: %+v", got)
	}

	if n, err := q.DeactivateProduct(ctx, p.ID); err != nil || n != 1 {
		t.Fatalf("deactivate: n=%d err=%v", n, err)
	}
	if _, err := q.GetProduct(ctx, p.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("want ErrNoRows after deactivate, got %v", err)
	}
}
