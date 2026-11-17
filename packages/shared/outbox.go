package shared

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxSchema is the table each service creates in its own database (in a
// migration). Events are inserted in the same transaction as the business
// change, so "data saved" and "event will be sent" cannot disagree.
const OutboxSchema = `
CREATE TABLE outbox (
    id           BIGSERIAL   PRIMARY KEY,
    event_id     TEXT        NOT NULL UNIQUE,
    routing_key  TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);
CREATE INDEX outbox_unpublished_idx ON outbox (id) WHERE published_at IS NULL;`

// Execer is satisfied by pgx.Tx and *pgxpool.Pool.
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Enqueue writes an event to the outbox. Pass the transaction that holds the business change.
func Enqueue(ctx context.Context, db Execer, routingKey string, data any) error {
	env, err := NewEnvelope(routingKey, data)
	if err != nil {
		return err
	}
	env.Trace = map[string]string{}
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(env.Trace))
	payload, err := jsonString(env)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO outbox (event_id, routing_key, payload) VALUES ($1, $2, $3)`, env.EventID, routingKey, payload)
	return err
}

// Relay publishes outbox rows to the broker. A row is marked published only
// after the broker confirmed it. If the process dies between confirm and
// commit, the row is sent again, which consumers handle through the event id.
// Run one relay per service: several would work (SKIP LOCKED) but could reorder events.
type Relay struct {
	Pool     *pgxpool.Pool
	Pub      *Publisher
	Interval time.Duration
	Batch    int
}

func (r *Relay) Run(ctx context.Context) {
	interval, batch := r.Interval, r.Batch
	if interval <= 0 {
		interval = time.Second
	}
	if batch <= 0 {
		batch = 50
	}
	for ctx.Err() == nil {
		n, err := r.Flush(ctx, batch)
		if err != nil && ctx.Err() == nil {
			log.Printf("outbox relay: %v", err)
		}
		if err == nil && n == batch {
			continue // more rows are waiting
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

// Flush publishes up to batch pending rows and returns how many were sent.
func (r *Relay) Flush(ctx context.Context, batch int) (int, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit

	rows, err := tx.Query(ctx, `SELECT id, routing_key, payload::text FROM outbox
		WHERE published_at IS NULL ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED`, batch)
	if err != nil {
		return 0, err
	}
	type pending struct {
		id      int64
		key     string
		payload string
	}
	var todo []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.key, &p.payload); err != nil {
			rows.Close()
			return 0, err
		}
		todo = append(todo, p)
	}
	rows.Close()
	if rows.Err() != nil {
		return 0, rows.Err()
	}

	var sent []int64
	var pubErr error
	for _, p := range todo {
		pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		pubCtx, span := startFromPayload(pubCtx, "publish "+p.key, trace.SpanKindProducer, p.payload)
		pubErr = r.Pub.Publish(pubCtx, ExchangeEvents, p.key, []byte(p.payload), nil)
		endSpan(span, pubErr)
		cancel()
		if pubErr != nil {
			break
		}
		sent = append(sent, p.id)
	}
	if len(sent) > 0 {
		if _, err := tx.Exec(ctx, `UPDATE outbox SET published_at = now() WHERE id = ANY($1)`, sent); err != nil {
			return 0, err
		}
		if err := tx.Commit(ctx); err != nil {
			return 0, err
		}
	}
	return len(sent), pubErr
}

const tracerName = "cagie-shop/shared"

// startFromPayload starts a span that continues the trace stored in an
// envelope's JSON (if any).
func startFromPayload(ctx context.Context, name string, kind trace.SpanKind, payload string) (context.Context, trace.Span) {
	var env struct {
		EventID string            `json:"eventId"`
		Trace   map[string]string `json:"trace"`
	}
	_ = json.Unmarshal([]byte(payload), &env)
	return startFromEnvelope(ctx, name, kind, env.EventID, env.Trace)
}

func startFromEnvelope(ctx context.Context, name string, kind trace.SpanKind, eventID string, carrier map[string]string) (context.Context, trace.Span) {
	parent := otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(carrier))
	return otel.Tracer(tracerName).Start(parent, name,
		trace.WithSpanKind(kind),
		trace.WithAttributes(attribute.String("messaging.system", "rabbitmq"), attribute.String("messaging.message.id", eventID)))
}

func endSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}
