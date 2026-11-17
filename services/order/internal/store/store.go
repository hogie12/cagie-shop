// Package store is the Postgres implementation of the handler's Store.
package store

import (
	"context"
	"errors"

	"cagie-shop/order/internal/db"
	"cagie-shop/order/internal/domain"
	shared "cagie-shop/shared"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool, q: db.New(pool)} }

// CreateOrder saves the order, its items and the order.created event in one
// transaction. Either all three exist or none do, so an order can never be
// saved without its announcement (or announced without being saved).
func (s *Store) CreateOrder(ctx context.Context, userID string, total int64, items []domain.Item) (db.Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Order{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit
	q := s.q.WithTx(tx)
	o, err := q.CreateOrder(ctx, db.CreateOrderParams{UserID: userID, TotalIdr: total})
	if err != nil {
		return db.Order{}, err
	}
	event := shared.OrderCreated{OrderID: o.ID, UserID: userID}
	for _, it := range items {
		err := q.CreateOrderItem(ctx, db.CreateOrderItemParams{
			OrderID: o.ID, ProductID: it.ProductID, Name: it.Name, UnitPriceIdr: it.UnitPriceIDR, Quantity: it.Quantity,
		})
		if err != nil {
			return db.Order{}, err
		}
		event.Items = append(event.Items, shared.OrderItem{ProductID: it.ProductID, Quantity: it.Quantity})
	}
	if err := shared.Enqueue(ctx, tx, shared.RouteOrderCreated, event); err != nil {
		return db.Order{}, err
	}
	return o, tx.Commit(ctx)
}

// MarkAwaitingPayment moves a pending order to awaiting_payment and, in the same
// transaction, queues the payment.requested event with the order's total.
// It reports false when the order was not pending (duplicate or unknown).
func (s *Store) MarkAwaitingPayment(ctx context.Context, id string) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit
	total, err := s.q.WithTx(tx).MarkAwaitingPayment(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := shared.Enqueue(ctx, tx, shared.RoutePaymentRequested, shared.PaymentRequested{OrderID: id, AmountIDR: total}); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

// ConfirmOrder and CancelOrder report whether the order actually changed.
// False means it was already decided (a duplicate or late message) or is unknown.
func (s *Store) ConfirmOrder(ctx context.Context, id string) (bool, error) {
	n, err := s.q.ConfirmOrder(ctx, id)
	return n > 0, err
}

func (s *Store) CancelOrder(ctx context.Context, id, reason string) (bool, error) {
	n, err := s.q.CancelOrder(ctx, db.CancelOrderParams{ID: id, CancelReason: pgtype.Text{String: reason, Valid: true}})
	return n > 0, err
}

func (s *Store) GetOrder(ctx context.Context, id string) (db.Order, error) {
	return s.q.GetOrder(ctx, id)
}

func (s *Store) ListOrders(ctx context.Context, userID string) ([]db.Order, error) {
	return s.q.ListOrdersByUser(ctx, userID)
}

func (s *Store) ListItems(ctx context.Context, orderIDs []string) ([]db.OrderItem, error) {
	return s.q.ListItemsForOrders(ctx, orderIDs)
}
