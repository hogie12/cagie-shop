// Package consumer turns broker messages into calls on the store.
package consumer

import (
	"context"
	"fmt"

	shared "cagie-shop/shared"
)

type Processor interface {
	ProcessPayment(ctx context.Context, req shared.PaymentRequested, lastAttempt bool) error
}

// PaymentRequested handles payment.requested. Messages that can never be valid
// are permanent failures; anything else is retried.
func PaymentRequested(p Processor) shared.Handler {
	return func(ctx context.Context, env shared.Envelope) error {
		if env.Type != shared.RoutePaymentRequested {
			return fmt.Errorf("%w: unexpected event type %q", shared.ErrPermanent, env.Type)
		}
		var req shared.PaymentRequested
		if err := env.Decode(&req); err != nil {
			return fmt.Errorf("%w: %v", shared.ErrPermanent, err)
		}
		if req.OrderID == "" || req.AmountIDR <= 0 {
			return fmt.Errorf("%w: payment request without order id or positive amount", shared.ErrPermanent)
		}
		return p.ProcessPayment(ctx, req, shared.IsLastAttempt(ctx))
	}
}
