package db_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"cagie-shop/inventory/internal/db"
	"cagie-shop/inventory/internal/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Runs against a real Postgres. Set TEST_DATABASE_URL to a throwaway database.
func TestConcurrentReservationsNeverOversell(t *testing.T) {
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

	const product, stock, buyers = 900001, 10, 40
	if _, err := q.SetStock(ctx, db.SetStockParams{ProductID: product, Quantity: stock}); err != nil {
		t.Fatal(err)
	}

	var won atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < buyers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n, err := q.ReserveStock(ctx, db.ReserveStockParams{ProductID: product, Quantity: 1})
			if err != nil {
				t.Error(err)
				return
			}
			won.Add(int32(n))
		}()
	}
	wg.Wait()

	left, _ := q.GetStock(ctx, product)
	if won.Load() != stock || left.Quantity != 0 {
		t.Fatalf("%d buyers got stock (want %d), %d left (want 0)", won.Load(), stock, left.Quantity)
	}

	if n, _ := q.ReleaseStock(ctx, db.ReleaseStockParams{ProductID: product, Quantity: 4}); n != 1 {
		t.Fatal("release should affect one row")
	}
	if n, _ := q.ReleaseStock(ctx, db.ReleaseStockParams{ProductID: 999999999, Quantity: 1}); n != 0 {
		t.Fatal("release of unknown product should affect zero rows")
	}
	left, _ = q.GetStock(ctx, product)
	if left.Quantity != 4 {
		t.Fatalf("after release: %d, want 4", left.Quantity)
	}
}
