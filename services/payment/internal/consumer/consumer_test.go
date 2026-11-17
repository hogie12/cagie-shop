package consumer

import (
	"context"
	"errors"
	"testing"

	shared "cagie-shop/shared"
)

type fakeProcessor struct {
	req   shared.PaymentRequested
	last  bool
	err   error
	calls int
}

func (f *fakeProcessor) ProcessPayment(_ context.Context, r shared.PaymentRequested, last bool) error {
	f.calls++
	f.req, f.last = r, last
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

func TestValidRequestReachesTheStore(t *testing.T) {
	f := &fakeProcessor{}
	e := envelope(t, shared.RoutePaymentRequested, shared.PaymentRequested{OrderID: "o1", AmountIDR: 56000})
	if err := PaymentRequested(f)(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if f.req.OrderID != "o1" || f.req.AmountIDR != 56000 || f.last {
		t.Fatalf("%+v last=%v", f.req, f.last)
	}
}

func TestInvalidRequestsFailPermanently(t *testing.T) {
	bad := map[string]shared.Envelope{
		"wrong type":  envelope(t, shared.RouteStockReserved, shared.StockReserved{OrderID: "o"}),
		"no order id": envelope(t, shared.RoutePaymentRequested, shared.PaymentRequested{AmountIDR: 5}),
		"zero amount": envelope(t, shared.RoutePaymentRequested, shared.PaymentRequested{OrderID: "o"}),
		"negative":    envelope(t, shared.RoutePaymentRequested, shared.PaymentRequested{OrderID: "o", AmountIDR: -5}),
		"bad data":    {EventID: "x", Type: shared.RoutePaymentRequested, Data: []byte(`"oops"`)},
	}
	for name, e := range bad {
		f := &fakeProcessor{}
		if err := PaymentRequested(f)(context.Background(), e); !errors.Is(err, shared.ErrPermanent) || f.calls != 0 {
			t.Errorf("%s: err=%v calls=%d", name, err, f.calls)
		}
	}
}

func TestStoreErrorsAreRetried(t *testing.T) {
	f := &fakeProcessor{err: errors.New("provider down")}
	e := envelope(t, shared.RoutePaymentRequested, shared.PaymentRequested{OrderID: "o", AmountIDR: 5})
	if err := PaymentRequested(f)(context.Background(), e); err == nil || errors.Is(err, shared.ErrPermanent) {
		t.Fatalf("want a retryable error, got %v", err)
	}
}
