// Package shared is the platform library: event contracts, the RabbitMQ
// publisher and consumer, and the transactional outbox. Services talk to each
// other only through the events defined here.
package shared

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Routing keys on the shop.events exchange.
const (
	RouteOrderCreated     = "order.created"
	RouteStockReserved    = "stock.reserved"
	RouteStockRejected    = "stock.rejected"
	RoutePaymentRequested = "payment.requested"
	RoutePaymentSucceeded = "payment.succeeded"
	RoutePaymentFailed    = "payment.failed"
)

type OrderItem struct {
	ProductID int64 `json:"productId"`
	Quantity  int32 `json:"quantity"`
}

type OrderCreated struct {
	OrderID string      `json:"orderId"`
	UserID  string      `json:"userId"`
	Items   []OrderItem `json:"items"`
}

type StockReserved struct {
	OrderID string `json:"orderId"`
}

type StockRejected struct {
	OrderID string `json:"orderId"`
	Reason  string `json:"reason"`
}

// PaymentRequested is sent by Order once stock is held. Order owns the price,
// so the amount travels with the request.
type PaymentRequested struct {
	OrderID   string `json:"orderId"`
	AmountIDR int64  `json:"amountIdr"`
}

type PaymentSucceeded struct {
	OrderID string `json:"orderId"`
}

type PaymentFailed struct {
	OrderID string `json:"orderId"`
	Reason  string `json:"reason"`
}

// Envelope wraps every message. EventID stays the same if the message is
// published again, which is how consumers recognise duplicates.
type Envelope struct {
	EventID    string          `json:"eventId"`
	Type       string          `json:"type"`
	OccurredAt time.Time       `json:"occurredAt"`
	Data       json.RawMessage `json:"data"`
	// Trace carries the W3C trace context (traceparent) of the request that
	// caused this event, so the consumer continues the same trace even when
	// the outbox relay publishes it seconds later.
	Trace map[string]string `json:"trace,omitempty"`
}

func NewEnvelope(routingKey string, data any) (Envelope, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{EventID: uuid.NewString(), Type: routingKey, OccurredAt: time.Now().UTC(), Data: raw}, nil
}

func (e Envelope) Decode(v any) error { return json.Unmarshal(e.Data, v) }
