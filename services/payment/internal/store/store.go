// Package store holds the payment decision logic.
package store

import (
	"context"
	"errors"

	"cagie-shop/payment/internal/db"
	"cagie-shop/payment/internal/provider"
	shared "cagie-shop/shared"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Provider interface {
	Charge(ctx context.Context, idempotencyKey string, amountIDR int64) (provider.Result, error)
}

type Store struct {
	pool     *pgxpool.Pool
	provider Provider
}

func New(pool *pgxpool.Pool, p Provider) *Store { return &Store{pool: pool, provider: p} }

const reasonUnavailable = "payment provider unavailable"

// ProcessPayment handles payment.requested and always ends in a decision:
// payment.succeeded or payment.failed.
//
// A decline is final. An outage returns an error so the message is retried;
// on the last attempt it becomes payment.failed instead, because dead-lettering
// would leave the order waiting and its stock held forever.
func (s *Store) ProcessPayment(ctx context.Context, req shared.PaymentRequested, lastAttempt bool) error {
	q := db.New(s.pool)
	if _, err := q.GetPaymentByOrder(ctx, req.OrderID); err == nil {
		return nil // already decided, nothing to charge or announce again
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	// The provider call happens outside any database transaction. If we crash
	// right after it, the retry charges again with the same idempotency key, so
	// the provider answers the same and no money moves twice.
	res, err := s.provider.Charge(ctx, req.OrderID, req.AmountIDR)
	if err != nil {
		if !lastAttempt {
			return err
		}
		res = provider.Result{Approved: false, Reason: reasonUnavailable}
	}
	return s.record(ctx, req, res)
}

// record saves the payment and its event in one transaction.
func (s *Store) record(ctx context.Context, req shared.PaymentRequested, res provider.Result) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit

	status, reason := "succeeded", pgtype.Text{}
	if !res.Approved {
		status, reason = "failed", pgtype.Text{String: res.Reason, Valid: true}
	}
	n, err := db.New(tx).CreatePayment(ctx, db.CreatePaymentParams{
		OrderID: req.OrderID, AmountIdr: req.AmountIDR, Status: status, Reason: reason,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return nil // lost a race with another delivery of the same request
	}
	if res.Approved {
		err = shared.Enqueue(ctx, tx, shared.RoutePaymentSucceeded, shared.PaymentSucceeded{OrderID: req.OrderID})
	} else {
		err = shared.Enqueue(ctx, tx, shared.RoutePaymentFailed, shared.PaymentFailed{OrderID: req.OrderID, Reason: res.Reason})
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
