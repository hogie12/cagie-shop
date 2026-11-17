package consumer

import (
	"context"
	"errors"
	"testing"

	shared "cagie-shop/shared"
)

type fakeReserver struct {
	eventID  string
	order    shared.OrderCreated
	released string
	err      error
	calls    int
}

func (f *fakeReserver) ReserveOrder(_ context.Context, id string, o shared.OrderCreated) error {
	f.calls++
	f.eventID, f.order = id, o
	return f.err
}

func (f *fakeReserver) ReleaseOrder(_ context.Context, id, orderID string) error {
	f.calls++
	f.eventID, f.released = id, orderID
	return f.err
}

func envelope(t *testing.T, key string, data any) shared.Envelope {
	t.Helper()
	e, err := shared.NewEnvelope(key, data)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestValidEventReachesTheStoreWithItsEventID(t *testing.T) {
	f := &fakeReserver{}
	e := envelope(t, shared.RouteOrderCreated, shared.OrderCreated{OrderID: "o1", Items: []shared.OrderItem{{ProductID: 1, Quantity: 2}}})
	if err := Events(f)(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if f.eventID != e.EventID || f.order.OrderID != "o1" {
		t.Fatalf("got %q %+v", f.eventID, f.order)
	}
}

func TestPaymentFailedReleasesTheOrdersStock(t *testing.T) {
	f := &fakeReserver{}
	e := envelope(t, shared.RoutePaymentFailed, shared.PaymentFailed{OrderID: "o7", Reason: "card declined"})
	if err := Events(f)(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if f.released != "o7" || f.eventID != e.EventID {
		t.Fatalf("released=%q eventID=%q", f.released, f.eventID)
	}
}

func TestInvalidEventsAreFailedPermanently(t *testing.T) {
	bad := map[string]shared.Envelope{
		"wrong type":          envelope(t, shared.RouteStockReserved, shared.StockReserved{OrderID: "o"}),
		"failed, no order id": envelope(t, shared.RoutePaymentFailed, shared.PaymentFailed{Reason: "x"}),
		"no items":            envelope(t, shared.RouteOrderCreated, shared.OrderCreated{OrderID: "o"}),
		"no order id":         envelope(t, shared.RouteOrderCreated, shared.OrderCreated{Items: []shared.OrderItem{{ProductID: 1, Quantity: 1}}}),
		"zero qty":            envelope(t, shared.RouteOrderCreated, shared.OrderCreated{OrderID: "o", Items: []shared.OrderItem{{ProductID: 1, Quantity: 0}}}),
		"bad json":            {EventID: "x", Type: shared.RouteOrderCreated, Data: []byte(`"just a string"`)},
	}
	for name, e := range bad {
		f := &fakeReserver{}
		err := Events(f)(context.Background(), e)
		if !errors.Is(err, shared.ErrPermanent) || f.calls != 0 {
			t.Errorf("%s: err=%v calls=%d, want a permanent error and no store call", name, err, f.calls)
		}
	}
}

func TestStoreErrorsAreRetried(t *testing.T) {
	f := &fakeReserver{err: errors.New("db down")}
	e := envelope(t, shared.RouteOrderCreated, shared.OrderCreated{OrderID: "o", Items: []shared.OrderItem{{ProductID: 1, Quantity: 1}}})
	err := Events(f)(context.Background(), e)
	if err == nil || errors.Is(err, shared.ErrPermanent) {
		t.Fatalf("a database error should be retryable, got %v", err)
	}
}
