// Package consumer turns events from the broker into order state changes.
package consumer

import (
	"context"
	"fmt"
	"log"

	shared "cagie-shop/shared"
)

type Orders interface {
	MarkAwaitingPayment(ctx context.Context, id string) (bool, error)
	ConfirmOrder(ctx context.Context, id string) (bool, error)
	CancelOrder(ctx context.Context, id, reason string) (bool, error)
}

// Events moves an order through its life:
//
//	stock.reserved    pending -> awaiting_payment (and asks Payment to charge)
//	stock.rejected    -> cancelled
//	payment.succeeded awaiting_payment -> confirmed
//	payment.failed    -> cancelled
//
// Every transition is guarded by the current status, so duplicates and late
// messages are acknowledged without changing anything.
func Events(o Orders) shared.Handler {
	return func(ctx context.Context, env shared.Envelope) error {
		switch env.Type {
		case shared.RouteStockReserved:
			var e shared.StockReserved
			if err := decode(env, &e, func() bool { return e.OrderID != "" }); err != nil {
				return err
			}
			changed, err := o.MarkAwaitingPayment(ctx, e.OrderID)
			logIgnored(changed, "request payment", e.OrderID, env.EventID)
			return err
		case shared.RouteStockRejected:
			var e shared.StockRejected
			if err := decode(env, &e, func() bool { return e.OrderID != "" }); err != nil {
				return err
			}
			changed, err := o.CancelOrder(ctx, e.OrderID, e.Reason)
			logIgnored(changed, "cancel", e.OrderID, env.EventID)
			return err
		case shared.RoutePaymentSucceeded:
			var e shared.PaymentSucceeded
			if err := decode(env, &e, func() bool { return e.OrderID != "" }); err != nil {
				return err
			}
			changed, err := o.ConfirmOrder(ctx, e.OrderID)
			logIgnored(changed, "confirm", e.OrderID, env.EventID)
			return err
		case shared.RoutePaymentFailed:
			var e shared.PaymentFailed
			if err := decode(env, &e, func() bool { return e.OrderID != "" }); err != nil {
				return err
			}
			changed, err := o.CancelOrder(ctx, e.OrderID, "payment failed: "+e.Reason)
			logIgnored(changed, "cancel", e.OrderID, env.EventID)
			return err
		}
		return fmt.Errorf("%w: unexpected event type %q", shared.ErrPermanent, env.Type)
	}
}

func decode(env shared.Envelope, v any, valid func() bool) error {
	if err := env.Decode(v); err != nil || !valid() {
		return fmt.Errorf("%w: bad %s payload", shared.ErrPermanent, env.Type)
	}
	return nil
}

func logIgnored(changed bool, action, orderID, eventID string) {
	if !changed {
		log.Printf("order %s: %s ignored (already handled or unknown), event %s", orderID, action, eventID)
	}
}
