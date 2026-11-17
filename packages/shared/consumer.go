package shared

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ErrPermanent marks a failure that retrying cannot fix. The message goes
// straight to the dead-letter queue.
var ErrPermanent = errors.New("permanent failure")

type Handler func(ctx context.Context, env Envelope) error

const attemptHeader = "x-attempt"

type attemptKey struct{}

// Attempt is how many times this message was tried before (0 on the first delivery).
func Attempt(ctx context.Context) int {
	n, _ := ctx.Value(attemptKey{}).(int)
	return n
}

// IsLastAttempt is true when a failure now would send the message to the
// dead-letter queue. A handler can use it to turn "still failing" into a
// proper business outcome instead of leaving the work undone.
func IsLastAttempt(ctx context.Context) bool { return Attempt(ctx) >= MaxAttempts-1 }

// Consumer processes one queue. Delivery is at-least-once, so handlers must
// be idempotent. A message is acknowledged only after its handler succeeds.
type Consumer struct {
	URL        string
	Queue      string
	Handler    Handler
	RetryDelay time.Duration
}

// Run consumes until ctx ends, reconnecting with backoff when the connection drops.
func (c *Consumer) Run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		started := time.Now()
		err := c.session(ctx)
		if ctx.Err() != nil {
			return
		}
		if time.Since(started) > time.Minute {
			backoff = time.Second // it ran fine for a while, so start the backoff over
		}
		log.Printf("consumer %s: %v, reconnecting in %s", c.Queue, err, backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, 30*time.Second)
	}
}

func (c *Consumer) session(ctx context.Context) error {
	conn, err := amqp.DialConfig(c.URL, amqp.Config{Dial: amqp.DefaultDial(5 * time.Second), Heartbeat: 10 * time.Second})
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	if err := DeclareTopology(ch, c.RetryDelay); err != nil {
		return err
	}
	if err := ch.Qos(10, 0, false); err != nil {
		return err
	}
	msgs, err := ch.Consume(c.Queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	retry := NewPublisher(c.URL)
	retry.RetryDelay = c.RetryDelay
	defer retry.Close()
	log.Printf("consumer %s: ready", c.Queue)

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return errors.New("connection closed")
			}
			c.handle(ctx, retry, d)
		}
	}
}

func (c *Consumer) handle(ctx context.Context, retry *Publisher, d amqp.Delivery) {
	var env Envelope
	if err := json.Unmarshal(d.Body, &env); err != nil || env.EventID == "" {
		log.Printf("consumer %s: unreadable message, dead-lettering", c.Queue)
		_ = d.Nack(false, false)
		return
	}
	// Continue the producer's trace: one order is one trace across services.
	spanCtx, span := startFromEnvelope(ctx, "process "+env.Type, trace.SpanKindConsumer, env.EventID, env.Trace)
	span.SetAttributes(attribute.Int("messaging.attempt", attemptOf(d.Headers)))
	err := c.safeCall(context.WithValue(spanCtx, attemptKey{}, attemptOf(d.Headers)), env)
	endSpan(span, err)
	switch {
	case err == nil:
		_ = d.Ack(false)
	case errors.Is(err, ErrPermanent):
		log.Printf("consumer %s: event %s failed permanently: %v", c.Queue, env.EventID, err)
		_ = d.Nack(false, false)
	default:
		attempt := attemptOf(d.Headers) + 1
		if attempt >= MaxAttempts {
			log.Printf("consumer %s: event %s failed %d times, dead-lettering: %v", c.Queue, env.EventID, attempt, err)
			_ = d.Nack(false, false)
			return
		}
		log.Printf("consumer %s: event %s failed (attempt %d), retrying: %v", c.Queue, env.EventID, attempt, err)
		pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		// Hand the message to the retry queue first and acknowledge afterwards,
		// so a crash in between duplicates the message instead of losing it.
		if perr := retry.Publish(pubCtx, "", c.Queue+".retry", d.Body, amqp.Table{attemptHeader: int32(attempt)}); perr != nil {
			_ = d.Nack(false, true)
			return
		}
		_ = d.Ack(false)
	}
}

func (c *Consumer) safeCall(ctx context.Context, env Envelope) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("handler panic: %v", r)
		}
	}()
	return c.Handler(ctx, env)
}

func attemptOf(h amqp.Table) int {
	switch v := h[attemptHeader].(type) {
	case int32:
		return int(v)
	case int64:
		return int(v)
	}
	return 0
}
