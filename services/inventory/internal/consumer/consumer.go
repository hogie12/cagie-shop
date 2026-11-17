// Package consumer turns broker messages into calls on the store.
package consumer

import (
	"context"
	"fmt"

	shared "cagie-shop/shared"
)

type Stock interface {
	ReserveOrder(ctx context.Context, eventID string, o shared.OrderCreated) error
	ReleaseOrder(ctx context.Context, eventID, orderID string) error
}

// Events handles order.created (reserve stock) and payment.failed (give it
// back). A message that can never be valid is a permanent failure
// (dead-lettered at once); anything else is retried.
func Events(s Stock) shared.Handler {
	return func(ctx context.Context, env shared.Envelope) error {
		switch env.Type {
		case shared.RouteOrderCreated:
			return orderCreated(ctx, s, env)
		case shared.RoutePaymentFailed:
			var e shared.PaymentFailed
			if err := env.Decode(&e); err != nil || e.OrderID == "" {
				return fmt.Errorf("%w: bad payment.failed payload", shared.ErrPermanent)
			}
			return s.ReleaseOrder(ctx, env.EventID, e.OrderID)
		}
		return fmt.Errorf("%w: unexpected event type %q", shared.ErrPermanent, env.Type)
	}
}

func orderCreated(ctx context.Context, s Stock, env shared.Envelope) error {
	var oc shared.OrderCreated
	if err := env.Decode(&oc); err != nil {
		return fmt.Errorf("%w: %v", shared.ErrPermanent, err)
	}
	if oc.OrderID == "" || len(oc.Items) == 0 {
		return fmt.Errorf("%w: order without id or items", shared.ErrPermanent)
	}
	for _, it := range oc.Items {
		if it.ProductID <= 0 || it.Quantity <= 0 {
			return fmt.Errorf("%w: invalid line %+v", shared.ErrPermanent, it)
		}
	}
	return s.ReserveOrder(ctx, env.EventID, oc)
}
