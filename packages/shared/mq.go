package shared

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeEvents   = "shop.events"
	ExchangeDLX      = "shop.dlx"
	QueueDeadLetters = "shop.dead-letters"

	// One queue per service, named after the service that reads it.
	QueueInventory = "inventory.events"
	QueueOrder     = "order.events"
	QueuePayment   = "payment.events"

	// MaxAttempts is how many times a message is tried before it is dead-lettered.
	MaxAttempts = 3

	DefaultRetryDelay = 5 * time.Second
)

type queueSpec struct {
	name string
	keys []string
}

// Every consumer queue is declared here, by every service, on every connect.
// If only the consumer declared its queue, an event published before that
// consumer's first start would find no queue and be dropped.
var queues = []queueSpec{
	{QueueInventory, []string{RouteOrderCreated, RoutePaymentFailed}},
	{QueueOrder, []string{RouteStockReserved, RouteStockRejected, RoutePaymentSucceeded, RoutePaymentFailed}},
	{QueuePayment, []string{RoutePaymentRequested}},
}

// DeclareTopology creates the exchanges, queues and bindings (idempotent).
// Each queue gets a ".retry" twin: messages wait there for retryDelay, then
// expire back into the main queue. Messages that are rejected for good go to
// the shop.dead-letters queue through the dead-letter exchange.
func DeclareTopology(ch *amqp.Channel, retryDelay time.Duration) error {
	if retryDelay <= 0 {
		retryDelay = DefaultRetryDelay
	}
	for _, ex := range []string{ExchangeEvents, ExchangeDLX} {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			return err
		}
	}
	if _, err := ch.QueueDeclare(QueueDeadLetters, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(QueueDeadLetters, "#", ExchangeDLX, false, nil); err != nil {
		return err
	}
	for _, q := range queues {
		if _, err := ch.QueueDeclare(q.name, true, false, false, false, amqp.Table{"x-dead-letter-exchange": ExchangeDLX}); err != nil {
			return err
		}
		for _, key := range q.keys {
			if err := ch.QueueBind(q.name, key, ExchangeEvents, false, nil); err != nil {
				return err
			}
		}
		retry := amqp.Table{
			"x-message-ttl":             retryDelay.Milliseconds(),
			"x-dead-letter-exchange":    "", // default exchange: routes by queue name
			"x-dead-letter-routing-key": q.name,
		}
		if _, err := ch.QueueDeclare(q.name+".retry", true, false, false, false, retry); err != nil {
			return err
		}
	}
	return nil
}

// Publisher sends persistent messages and waits for the broker's confirmation.
// It connects lazily and reconnects on the next call after any failure.
type Publisher struct {
	URL        string
	RetryDelay time.Duration

	mu      sync.Mutex
	conn    *amqp.Connection
	ch      *amqp.Channel
	returns chan amqp.Return
}

func NewPublisher(url string) *Publisher { return &Publisher{URL: url} }

func (p *Publisher) connect() error {
	conn, err := amqp.DialConfig(p.URL, amqp.Config{Dial: amqp.DefaultDial(5 * time.Second), Heartbeat: 10 * time.Second})
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}
	if err := ch.Confirm(false); err != nil {
		conn.Close()
		return err
	}
	if err := DeclareTopology(ch, p.RetryDelay); err != nil {
		conn.Close()
		return err
	}
	p.conn, p.ch, p.returns = conn, ch, ch.NotifyReturn(make(chan amqp.Return, 16))
	return nil
}

func (p *Publisher) reset() {
	if p.conn != nil {
		p.conn.Close()
	}
	p.conn, p.ch, p.returns = nil, nil, nil
}

// Publish returns nil only once the broker has confirmed it stored the message
// on at least one queue. An unroutable message is an error, not a silent drop.
func (p *Publisher) Publish(ctx context.Context, exchange, key string, body []byte, headers amqp.Table) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ch == nil {
		if err := p.connect(); err != nil {
			return fmt.Errorf("connect to broker: %w", err)
		}
	}
	conf, err := p.ch.PublishWithDeferredConfirmWithContext(ctx, exchange, key, true, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent, ContentType: "application/json", Body: body, Headers: headers,
	})
	if err != nil {
		p.reset()
		return err
	}
	acked, err := conf.WaitContext(ctx)
	if err != nil || !acked {
		p.reset()
		return fmt.Errorf("publish not confirmed: %v", err)
	}
	// The broker sends basic.return before the confirm, so a return is already waiting.
	select {
	case r := <-p.returns:
		return fmt.Errorf("message unroutable (%s) for key %q", r.ReplyText, key)
	default:
		return nil
	}
}

func (p *Publisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reset()
}
