package store_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"cagie-shop/order/internal/domain"
	"cagie-shop/order/internal/migrate"
	"cagie-shop/order/internal/store"
	shared "cagie-shop/shared"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Runs against a real Postgres. Set TEST_DATABASE_URL to a throwaway database.
func setup(t *testing.T) (context.Context, *pgxpool.Pool, *store.Store) {
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
	return ctx, pool, store.New(pool)
}

func TestCreateOrderWritesTheOrderCreatedEventInTheSameTransaction(t *testing.T) {
	ctx, pool, s := setup(t)
	o, err := s.CreateOrder(ctx, "store-test-user", 56000, []domain.Item{{ProductID: 1, Name: "Croissant", UnitPriceIDR: 28000, Quantity: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if o.Status != domain.StatusPending {
		t.Fatalf("new order status = %s", o.Status)
	}
	var payload []byte
	var published bool
	err = pool.QueryRow(ctx, `SELECT payload, published_at IS NOT NULL FROM outbox WHERE routing_key = $1 AND payload->'data'->>'orderId' = $2`,
		shared.RouteOrderCreated, o.ID).Scan(&payload, &published)
	if err != nil {
		t.Fatalf("no outbox row for the order: %v", err)
	}
	var env shared.Envelope
	var oc shared.OrderCreated
	if err := json.Unmarshal(payload, &env); err != nil || env.Decode(&oc) != nil {
		t.Fatal("outbox payload is not a valid envelope")
	}
	if published || oc.UserID != "store-test-user" || len(oc.Items) != 1 || oc.Items[0].Quantity != 2 {
		t.Fatalf("published=%v event=%+v", published, oc)
	}
}

func TestFailedCreateLeavesNoOrderAndNoEvent(t *testing.T) {
	ctx, pool, s := setup(t)
	var ordersBefore, eventsBefore int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE user_id = 'atomic-user'`).Scan(&ordersBefore)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM outbox`).Scan(&eventsBefore)

	// The same product twice breaks UNIQUE(order_id, product_id): everything rolls back.
	_, err := s.CreateOrder(ctx, "atomic-user", 1, []domain.Item{
		{ProductID: 7, Name: "A", UnitPriceIDR: 1, Quantity: 1},
		{ProductID: 7, Name: "A", UnitPriceIDR: 1, Quantity: 1},
	})
	if err == nil {
		t.Fatal("duplicate product line should fail")
	}
	var ordersAfter, eventsAfter int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE user_id = 'atomic-user'`).Scan(&ordersAfter)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM outbox`).Scan(&eventsAfter)
	if ordersAfter != ordersBefore || eventsAfter != eventsBefore {
		t.Fatalf("orders %d→%d, events %d→%d: a failed transaction must leave nothing", ordersBefore, ordersAfter, eventsBefore, eventsAfter)
	}
}

func (f *fixture) paymentRequests(t *testing.T, orderID string) (n int, amount int64) {
	t.Helper()
	err := f.pool.QueryRow(f.ctx, `SELECT count(*), coalesce(max((payload->'data'->>'amountIdr')::bigint), 0) FROM outbox
		WHERE routing_key = $1 AND payload->'data'->>'orderId' = $2`, shared.RoutePaymentRequested, orderID).Scan(&n, &amount)
	if err != nil {
		t.Fatal(err)
	}
	return n, amount
}

type fixture struct {
	ctx  context.Context
	pool *pgxpool.Pool
	s    *store.Store
}

func newFixture(t *testing.T) *fixture {
	ctx, pool, s := setup(t)
	return &fixture{ctx: ctx, pool: pool, s: s}
}

var oneItem = []domain.Item{{ProductID: 1, Name: "Croissant", UnitPriceIDR: 28000, Quantity: 2}}

func TestAwaitingPaymentRequestsPaymentExactlyOnce(t *testing.T) {
	f := newFixture(t)
	o, _ := f.s.CreateOrder(f.ctx, "pay-user", 56000, oneItem)
	for i := 0; i < 3; i++ { // stock.reserved delivered three times
		changed, err := f.s.MarkAwaitingPayment(f.ctx, o.ID)
		if err != nil || changed != (i == 0) {
			t.Fatalf("delivery %d: changed=%v err=%v", i, changed, err)
		}
	}
	if n, amount := f.paymentRequests(t, o.ID); n != 1 || amount != 56000 {
		t.Fatalf("payment requests = %d for %d, want 1 for 56000", n, amount)
	}
	got, _ := f.s.GetOrder(f.ctx, o.ID)
	if got.Status != domain.StatusAwaitingPayment {
		t.Fatalf("status = %s", got.Status)
	}
	if changed, _ := f.s.MarkAwaitingPayment(f.ctx, "no-such-order"); changed {
		t.Fatal("unknown order cannot await payment")
	}
}

func TestConfirmNeedsAPaymentRequestFirst(t *testing.T) {
	f := newFixture(t)
	o, _ := f.s.CreateOrder(f.ctx, "confirm-user", 56000, oneItem)
	if ok, _ := f.s.ConfirmOrder(f.ctx, o.ID); ok {
		t.Fatal("a pending order has no payment yet and must not be confirmed")
	}
	_, _ = f.s.MarkAwaitingPayment(f.ctx, o.ID)
	if ok, err := f.s.ConfirmOrder(f.ctx, o.ID); err != nil || !ok {
		t.Fatalf("confirm after payment request: ok=%v err=%v", ok, err)
	}
	if ok, _ := f.s.ConfirmOrder(f.ctx, o.ID); ok {
		t.Fatal("a second confirm must change nothing")
	}
	if ok, _ := f.s.CancelOrder(f.ctx, o.ID, "late payment.failed"); ok {
		t.Fatal("a confirmed order must not be cancelled by a late message")
	}
	if got, _ := f.s.GetOrder(f.ctx, o.ID); got.Status != domain.StatusConfirmed || got.CancelReason.Valid {
		t.Fatalf("order = %+v", got)
	}
}

func TestCancelWorksFromPendingAndAwaitingPayment(t *testing.T) {
	f := newFixture(t)
	a, _ := f.s.CreateOrder(f.ctx, "cancel-user", 1, oneItem)
	if ok, err := f.s.CancelOrder(f.ctx, a.ID, "insufficient stock for product 1"); err != nil || !ok {
		t.Fatalf("cancel pending: ok=%v err=%v", ok, err)
	}
	if got, _ := f.s.GetOrder(f.ctx, a.ID); got.Status != domain.StatusCancelled || got.CancelReason.String != "insufficient stock for product 1" {
		t.Fatalf("order = %+v", got)
	}
	if changed, _ := f.s.MarkAwaitingPayment(f.ctx, a.ID); changed {
		t.Fatal("a cancelled order must not ask for payment")
	}

	b, _ := f.s.CreateOrder(f.ctx, "cancel-user", 1, oneItem)
	_, _ = f.s.MarkAwaitingPayment(f.ctx, b.ID)
	if ok, err := f.s.CancelOrder(f.ctx, b.ID, "payment failed: card declined"); err != nil || !ok {
		t.Fatalf("cancel awaiting payment: ok=%v err=%v", ok, err)
	}
	if ok, _ := f.s.ConfirmOrder(f.ctx, b.ID); ok {
		t.Fatal("a cancelled order must not be confirmed")
	}
}
