// Package domain holds the few types shared between the handler and the store.
package domain

const (
	StatusPending         = "pending"          // waiting for stock
	StatusAwaitingPayment = "awaiting_payment" // stock held, waiting for payment
	StatusConfirmed       = "confirmed"
	StatusCancelled       = "cancelled"
)

// Item is one order line with the name and price copied from the catalog.
type Item struct {
	ProductID    int64
	Name         string
	UnitPriceIDR int64
	Quantity     int32
}
