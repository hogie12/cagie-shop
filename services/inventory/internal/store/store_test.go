package store_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"cagie-shop/inventory/internal/db"
	"cagie-shop/inventory/internal/migrate"
	"cagie-shop/inventory/internal/store"
	shared "cagie-shop/shared"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Runs against a real Postgres. Set TEST_DATABASE_URL to a throwaway database.
type fixture struct {
	ctx  context.Context
	pool *pgxpool.Pool
	q    *db.Queries
	s    *store.Store
	run  string // makes order ids unique per test run
}

func setup(t *testing.T) *fixture {
	t.Helper()
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
	t.Cleanup(pool.Close)
	return &fixture{ctx: ctx, pool: pool, q: db.New(pool), s: store.New(pool), run: fmt.Sprint(time.Now().UnixNano())}
}

func (f *fixture) stock(t *testing.T, product int64, qty int32) {
	t.Helper()
	if _, err := f.q.SetStock(f.ctx, db.SetStockParams{ProductID: product, Quantity: qty}); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) left(t *testing.T, product int64) int32 {
	t.Helper()
	s, err := f.q.GetStock(f.ctx, product)
	if err != nil {
		t.Fatal(err)
	}
	return s.Quantity
}

func (f *fixture) events(t *testing.T, key, orderID string) int {
	t.Helper()
	var n int
	err := f.pool.QueryRow(f.ctx, `SELECT count(*) FROM outbox WHERE routing_key = $1 AND payload->'data'->>'orderId' = $2`, key, orderID).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func order(id string, items ...shared.OrderItem) shared.OrderCreated {
	return shared.OrderCreated{OrderID: id, UserID: "u", Items: items}
}

func TestReserveSuccessAnnouncesStockReserved(t *testing.T) {
	f := setup(t)
	f.stock(t, 910001, 5)
	f.stock(t, 910002, 2)
	o := order("ok-"+f.run, shared.OrderItem{ProductID: 910001, Quantity: 2}, shared.OrderItem{ProductID: 910002, Quantity: 1})
	if err := f.s.ReserveOrder(f.ctx, "evt-ok-"+f.run, o); err != nil {
		t.Fatal(err)
	}
	if f.left(t, 910001) != 3 || f.left(t, 910002) != 1 {
		t.Fatalf("stock = %d, %d", f.left(t, 910001), f.left(t, 910002))
	}
	if f.events(t, shared.RouteStockReserved, o.OrderID) != 1 {
		t.Fatal("expected one stock.reserved event")
	}
	var rows int
	_ = f.pool.QueryRow(f.ctx, `SELECT count(*) FROM reservations WHERE order_id = $1`, o.OrderID).Scan(&rows)
	if rows != 2 {
		t.Fatalf("reservation rows = %d", rows)
	}
}

func TestDuplicateEventChangesNothing(t *testing.T) {
	f := setup(t)
	f.stock(t, 910011, 5)
	o := order("dup-"+f.run, shared.OrderItem{ProductID: 910011, Quantity: 2})
	for i := 0; i < 3; i++ {
		if err := f.s.ReserveOrder(f.ctx, "evt-dup-"+f.run, o); err != nil {
			t.Fatal(err)
		}
	}
	if f.left(t, 910011) != 3 {
		t.Fatalf("stock = %d, want 3 (reserved once)", f.left(t, 910011))
	}
	if f.events(t, shared.RouteStockReserved, o.OrderID) != 1 {
		t.Fatal("duplicate deliveries must not announce twice")
	}
}

func TestRejectionIsAllOrNothing(t *testing.T) {
	f := setup(t)
	f.stock(t, 910021, 5)
	f.stock(t, 910022, 1)
	o := order("rej-"+f.run, shared.OrderItem{ProductID: 910021, Quantity: 2}, shared.OrderItem{ProductID: 910022, Quantity: 3})
	for i := 0; i < 2; i++ { // second delivery is a duplicate of a rejected event
		if err := f.s.ReserveOrder(f.ctx, "evt-rej-"+f.run, o); err != nil {
			t.Fatal(err)
		}
	}
	if f.left(t, 910021) != 5 || f.left(t, 910022) != 1 {
		t.Fatalf("a rejected order must change no stock, got %d and %d", f.left(t, 910021), f.left(t, 910022))
	}
	if f.events(t, shared.RouteStockRejected, o.OrderID) != 1 || f.events(t, shared.RouteStockReserved, o.OrderID) != 0 {
		t.Fatal("expected exactly one stock.rejected and no stock.reserved")
	}
	var rows int
	_ = f.pool.QueryRow(f.ctx, `SELECT count(*) FROM reservations WHERE order_id = $1`, o.OrderID).Scan(&rows)
	if rows != 0 {
		t.Fatalf("rejected order left %d reservation rows", rows)
	}
}

func TestUnknownProductIsRejected(t *testing.T) {
	f := setup(t)
	o := order("unk-"+f.run, shared.OrderItem{ProductID: 919999, Quantity: 1})
	if err := f.s.ReserveOrder(f.ctx, "evt-unk-"+f.run, o); err != nil {
		t.Fatal(err)
	}
	if f.events(t, shared.RouteStockRejected, o.OrderID) != 1 {
		t.Fatal("expected stock.rejected for an unstocked product")
	}
}

func TestCompetingOrdersNeverOversell(t *testing.T) {
	f := setup(t)
	f.stock(t, 910031, 3)
	const buyers = 12
	var wg sync.WaitGroup
	for i := 0; i < buyers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := fmt.Sprintf("race-%s-%d", f.run, i)
			if err := f.s.ReserveOrder(f.ctx, "evt-"+id, order(id, shared.OrderItem{ProductID: 910031, Quantity: 1})); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	var reserved, rejected int
	for i := 0; i < buyers; i++ {
		id := fmt.Sprintf("race-%s-%d", f.run, i)
		reserved += f.events(t, shared.RouteStockReserved, id)
		rejected += f.events(t, shared.RouteStockRejected, id)
	}
	if reserved != 3 || rejected != buyers-3 || f.left(t, 910031) != 0 {
		t.Fatalf("reserved=%d rejected=%d left=%d, want 3, %d, 0", reserved, rejected, buyers-3, f.left(t, 910031))
	}
}

func TestReleaseOrderPutsStockBackExactlyOnce(t *testing.T) {
	f := setup(t)
	f.stock(t, 910041, 5)
	f.stock(t, 910042, 4)
	o := "rel-" + f.run
	items := []shared.OrderItem{{ProductID: 910041, Quantity: 3}, {ProductID: 910042, Quantity: 2}}
	if err := f.s.ReserveOrder(f.ctx, "evt-res-"+f.run, shared.OrderCreated{OrderID: o, Items: items}); err != nil {
		t.Fatal(err)
	}
	if f.left(t, 910041) != 2 || f.left(t, 910042) != 2 {
		t.Fatal("setup: stock should be reserved")
	}
	for i := 0; i < 3; i++ { // the same payment.failed event, delivered three times
		if err := f.s.ReleaseOrder(f.ctx, "evt-fail-"+f.run, o); err != nil {
			t.Fatal(err)
		}
	}
	if f.left(t, 910041) != 5 || f.left(t, 910042) != 4 {
		t.Fatalf("stock = %d, %d, want 5, 4 (released once)", f.left(t, 910041), f.left(t, 910042))
	}
	// A different event for the same order must not release again either.
	if err := f.s.ReleaseOrder(f.ctx, "evt-fail-again-"+f.run, o); err != nil {
		t.Fatal(err)
	}
	if f.left(t, 910041) != 5 || f.left(t, 910042) != 4 {
		t.Fatal("a reservation may be released only once")
	}
}

func TestReleaseOfAnUnknownOrderChangesNothing(t *testing.T) {
	f := setup(t)
	f.stock(t, 910051, 7)
	if err := f.s.ReleaseOrder(f.ctx, "evt-none-"+f.run, "no-such-order-"+f.run); err != nil {
		t.Fatal(err)
	}
	if f.left(t, 910051) != 7 {
		t.Fatal("nothing should change")
	}
}
