package store_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"cagie-shop/payment/internal/migrate"
	"cagie-shop/payment/internal/provider"
	"cagie-shop/payment/internal/store"
	shared "cagie-shop/shared"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Runs against a real Postgres. Set TEST_DATABASE_URL to a throwaway database.
type fakeProvider struct {
	result provider.Result
	err    error
	calls  int
}

func (f *fakeProvider) Charge(context.Context, string, int64) (provider.Result, error) {
	f.calls++
	return f.result, f.err
}

type fixture struct {
	ctx  context.Context
	pool *pgxpool.Pool
	prov *fakeProvider
	s    *store.Store
	run  string
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
	prov := &fakeProvider{result: provider.Result{Approved: true}}
	return &fixture{ctx: ctx, pool: pool, prov: prov, s: store.New(pool, prov), run: fmt.Sprint(time.Now().UnixNano())}
}

func (f *fixture) req(name string) shared.PaymentRequested {
	return shared.PaymentRequested{OrderID: name + "-" + f.run, AmountIDR: 56000}
}

func (f *fixture) events(t *testing.T, key, orderID string) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(f.ctx, `SELECT count(*) FROM outbox WHERE routing_key = $1 AND payload->'data'->>'orderId' = $2`, key, orderID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *fixture) payment(t *testing.T, orderID string) (status, reason string, found bool) {
	t.Helper()
	var r *string
	err := f.pool.QueryRow(f.ctx, `SELECT status, reason FROM payments WHERE order_id = $1`, orderID).Scan(&status, &r)
	if err != nil {
		return "", "", false
	}
	if r != nil {
		reason = *r
	}
	return status, reason, true
}

func TestApprovedPaymentAnnouncesSuccess(t *testing.T) {
	f := setup(t)
	r := f.req("ok")
	if err := f.s.ProcessPayment(f.ctx, r, false); err != nil {
		t.Fatal(err)
	}
	if st, _, ok := f.payment(t, r.OrderID); !ok || st != "succeeded" {
		t.Fatalf("status=%q found=%v", st, ok)
	}
	if f.events(t, shared.RoutePaymentSucceeded, r.OrderID) != 1 || f.events(t, shared.RoutePaymentFailed, r.OrderID) != 0 {
		t.Fatal("expected exactly one payment.succeeded")
	}
}

func TestDeclinedPaymentIsFinalAndAnnouncesFailure(t *testing.T) {
	f := setup(t)
	f.prov.result = provider.Result{Approved: false, Reason: "card declined"}
	r := f.req("decline")
	if err := f.s.ProcessPayment(f.ctx, r, false); err != nil {
		t.Fatalf("a decline is an answer, not an error: %v", err)
	}
	st, reason, _ := f.payment(t, r.OrderID)
	if st != "failed" || reason != "card declined" || f.events(t, shared.RoutePaymentFailed, r.OrderID) != 1 {
		t.Fatalf("status=%q reason=%q", st, reason)
	}
}

func TestDuplicateRequestChargesAndAnnouncesOnce(t *testing.T) {
	f := setup(t)
	r := f.req("dup")
	for i := 0; i < 3; i++ {
		if err := f.s.ProcessPayment(f.ctx, r, false); err != nil {
			t.Fatal(err)
		}
	}
	if f.prov.calls != 1 {
		t.Fatalf("provider called %d times, want 1", f.prov.calls)
	}
	if f.events(t, shared.RoutePaymentSucceeded, r.OrderID) != 1 {
		t.Fatal("duplicates must not announce twice")
	}
}

func TestOutageIsRetriedUntilTheLastAttempt(t *testing.T) {
	f := setup(t)
	f.prov.err = provider.ErrUnavailable
	r := f.req("outage")
	if err := f.s.ProcessPayment(f.ctx, r, false); !errors.Is(err, provider.ErrUnavailable) {
		t.Fatalf("an early outage must be returned for retry, got %v", err)
	}
	if _, _, found := f.payment(t, r.OrderID); found || f.events(t, shared.RoutePaymentFailed, r.OrderID) != 0 {
		t.Fatal("nothing may be decided while retries remain")
	}

	if err := f.s.ProcessPayment(f.ctx, r, true); err != nil {
		t.Fatalf("on the last attempt the outage becomes a decision: %v", err)
	}
	st, reason, _ := f.payment(t, r.OrderID)
	if st != "failed" || reason != "payment provider unavailable" || f.events(t, shared.RoutePaymentFailed, r.OrderID) != 1 {
		t.Fatalf("status=%q reason=%q", st, reason)
	}
}

func TestRecoveryDuringRetriesSucceeds(t *testing.T) {
	f := setup(t)
	r := f.req("recover")
	f.prov.err = provider.ErrUnavailable
	_ = f.s.ProcessPayment(f.ctx, r, false)
	f.prov.err = nil // the provider comes back before the next attempt
	if err := f.s.ProcessPayment(f.ctx, r, false); err != nil {
		t.Fatal(err)
	}
	if f.events(t, shared.RoutePaymentSucceeded, r.OrderID) != 1 {
		t.Fatal("expected payment.succeeded after recovery")
	}
}
