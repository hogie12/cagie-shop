// Package store holds the inventory's event-driven stock logic.
package store

import (
	"context"
	"fmt"
	"sort"

	"cagie-shop/inventory/internal/db"
	shared "cagie-shop/shared"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// ReserveOrder handles an order.created event. Either every line is reserved
// and stock.reserved is announced, or nothing changes and stock.rejected is
// announced. Handling the same event twice is harmless.
func (s *Store) ReserveOrder(ctx context.Context, eventID string, o shared.OrderCreated) error {
	lines := mergeSorted(o.Items)
	reason, err := s.tryReserve(ctx, eventID, o.OrderID, lines)
	if err != nil || reason == "" {
		return err
	}
	return s.reject(ctx, eventID, o.OrderID, reason)
}

// tryReserve returns "" when the event is done (reserved, or already handled)
// and a rejection reason when the order cannot be fulfilled.
func (s *Store) tryReserve(ctx context.Context, eventID, orderID string, lines []shared.OrderItem) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit
	q := db.New(tx)

	fresh, err := q.InsertInbox(ctx, eventID)
	if err != nil {
		return "", err
	}
	if fresh == 0 {
		return "", nil // duplicate delivery, the first one already did the work
	}
	// Lines are sorted by product id, so two orders competing for the same
	// products lock rows in the same order and cannot deadlock each other.
	for _, l := range lines {
		n, err := q.ReserveStock(ctx, db.ReserveStockParams{ProductID: l.ProductID, Quantity: l.Quantity})
		if err != nil {
			return "", err
		}
		if n == 0 {
			if _, err := q.GetStock(ctx, l.ProductID); err == pgx.ErrNoRows {
				return fmt.Sprintf("product %d is not stocked", l.ProductID), nil
			} else if err != nil {
				return "", err
			}
			return fmt.Sprintf("insufficient stock for product %d", l.ProductID), nil // deferred rollback undoes earlier lines
		}
		if err := q.CreateReservation(ctx, db.CreateReservationParams{OrderID: orderID, ProductID: l.ProductID, Quantity: l.Quantity}); err != nil {
			return "", err
		}
	}
	if err := shared.Enqueue(ctx, tx, shared.RouteStockReserved, shared.StockReserved{OrderID: orderID}); err != nil {
		return "", err
	}
	return "", tx.Commit(ctx)
}

func (s *Store) reject(ctx context.Context, eventID, orderID, reason string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit
	fresh, err := db.New(tx).InsertInbox(ctx, eventID)
	if err != nil {
		return err
	}
	if fresh == 0 {
		return nil
	}
	if err := shared.Enqueue(ctx, tx, shared.RouteStockRejected, shared.StockRejected{OrderID: orderID, Reason: reason}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ReleaseOrder undoes ReserveOrder after a payment.failed event (the saga's
// compensation step). Each reservation is released at most once, and handling
// the same event twice is harmless.
func (s *Store) ReleaseOrder(ctx context.Context, eventID, orderID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit
	q := db.New(tx)

	fresh, err := q.InsertInbox(ctx, eventID)
	if err != nil {
		return err
	}
	if fresh == 0 {
		return nil
	}
	released, err := q.ReleaseReservations(ctx, orderID)
	if err != nil {
		return err
	}
	// Sorted like ReserveOrder so concurrent updates lock rows in the same order.
	sort.Slice(released, func(i, j int) bool { return released[i].ProductID < released[j].ProductID })
	for _, r := range released {
		if _, err := q.ReleaseStock(ctx, db.ReleaseStockParams{ProductID: r.ProductID, Quantity: r.Quantity}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// mergeSorted adds up repeated products and orders lines by product id.
func mergeSorted(items []shared.OrderItem) []shared.OrderItem {
	sum := map[int64]int32{}
	for _, it := range items {
		sum[it.ProductID] += it.Quantity
	}
	out := make([]shared.OrderItem, 0, len(sum))
	for id, q := range sum {
		out = append(out, shared.OrderItem{ProductID: id, Quantity: q})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProductID < out[j].ProductID })
	return out
}
