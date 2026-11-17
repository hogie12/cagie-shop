package consumer

import (
	"context"
	"errors"
	"testing"

	shared "cagie-shop/shared"
)

type fakeOrders struct {
	awaiting, confirmed, cancelled []string
	reason                         string
	changed                        bool
	err                            error
}

func (f *fakeOrders) MarkAwaitingPayment(_ context.Context, id string) (bool, error) {
	f.awaiting = append(f.awaiting, id)
	return f.changed, f.err
}
func (f *fakeOrders) ConfirmOrder(_ context.Context, id string) (bool, error) {
	f.confirmed = append(f.confirmed, id)
	return f.changed, f.err
}
func (f *fakeOrders) CancelOrder(_ context.Context, id, reason string) (bool, error) {
	f.cancelled = append(f.cancelled, id)
	f.reason = reason
	return f.changed, f.err
}

func envelope(t *testing.T, key string, data any) shared.Envelope {
	t.Helper()
	e, err := shared.NewEnvelope(key, data)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func handle(t *testing.T, f *fakeOrders, key string, data any) error {
	t.Helper()
	return Events(f)(context.Background(), envelope(t, key, data))
}

func TestEachEventMovesTheOrderTheRightWay(t *testing.T) {
	f := &fakeOrders{changed: true}
	steps := []struct {
		key  string
		data any
	}{
		{shared.RouteStockReserved, shared.StockReserved{OrderID: "a"}},
		{shared.RoutePaymentSucceeded, shared.PaymentSucceeded{OrderID: "b"}},
		{shared.RouteStockRejected, shared.StockRejected{OrderID: "c", Reason: "insufficient stock for product 4"}},
	}
	for _, s := range steps {
		if err := handle(t, f, s.key, s.data); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.awaiting) != 1 || f.awaiting[0] != "a" || len(f.confirmed) != 1 || f.confirmed[0] != "b" ||
		len(f.cancelled) != 1 || f.cancelled[0] != "c" || f.reason != "insufficient stock for product 4" {
		t.Fatalf("%+v", f)
	}
}

func TestPaymentFailureCancelsWithTheReason(t *testing.T) {
	f := &fakeOrders{changed: true}
	if err := handle(t, f, shared.RoutePaymentFailed, shared.PaymentFailed{OrderID: "o1", Reason: "card declined"}); err != nil {
		t.Fatal(err)
	}
	if len(f.cancelled) != 1 || f.reason != "payment failed: card declined" {
		t.Fatalf("%+v", f)
	}
}

func TestAlreadyHandledEventsAreNotErrors(t *testing.T) {
	f := &fakeOrders{changed: false} // the store says nothing changed
	for key, data := range map[string]any{
		shared.RouteStockReserved:    shared.StockReserved{OrderID: "o"},
		shared.RoutePaymentSucceeded: shared.PaymentSucceeded{OrderID: "o"},
		shared.RoutePaymentFailed:    shared.PaymentFailed{OrderID: "o"},
	} {
		if err := handle(t, f, key, data); err != nil {
			t.Errorf("%s: a duplicate must be acknowledged quietly, got %v", key, err)
		}
	}
}

func TestBadMessagesFailPermanently(t *testing.T) {
	bad := map[string]shared.Envelope{
		"unknown type":    envelope(t, shared.RouteOrderCreated, shared.OrderCreated{OrderID: "o"}),
		"reserved no id":  envelope(t, shared.RouteStockReserved, shared.StockReserved{}),
		"rejected no id":  envelope(t, shared.RouteStockRejected, shared.StockRejected{Reason: "x"}),
		"succeeded no id": envelope(t, shared.RoutePaymentSucceeded, shared.PaymentSucceeded{}),
		"failed no id":    envelope(t, shared.RoutePaymentFailed, shared.PaymentFailed{Reason: "x"}),
		"wrong data":      {EventID: "x", Type: shared.RouteStockReserved, Data: []byte(`"oops"`)},
	}
	for name, e := range bad {
		f := &fakeOrders{}
		if err := Events(f)(context.Background(), e); !errors.Is(err, shared.ErrPermanent) ||
			len(f.awaiting)+len(f.confirmed)+len(f.cancelled) != 0 {
			t.Errorf("%s: err=%v", name, err)
		}
	}
}

func TestDatabaseErrorsAreRetried(t *testing.T) {
	f := &fakeOrders{err: errors.New("db down")}
	err := handle(t, f, shared.RoutePaymentSucceeded, shared.PaymentSucceeded{OrderID: "o1"})
	if err == nil || errors.Is(err, shared.ErrPermanent) {
		t.Fatalf("want a retryable error, got %v", err)
	}
}
