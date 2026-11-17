package shared

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Broker tests need RABBITMQ_TEST_URL pointing at a throwaway vhost (make test-setup creates "test").
// Relay tests also need TEST_DATABASE_URL.

func brokerURL(t *testing.T) string {
	t.Helper()
	u := os.Getenv("RABBITMQ_TEST_URL")
	if u == "" {
		t.Skip("RABBITMQ_TEST_URL not set")
	}
	return u
}

const testRetryDelay = 150 * time.Millisecond

// resetBroker deletes everything this package declares so each test starts clean.
func resetBroker(t *testing.T, url string) {
	t.Helper()
	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ch, _ := conn.Channel()
	names := []string{QueueDeadLetters}
	for _, q := range queues {
		names = append(names, q.name, q.name+".retry")
	}
	for _, n := range names {
		_, _ = ch.QueueDelete(n, false, false, false)
		ch, _ = reopen(conn, ch)
	}
	for _, ex := range []string{ExchangeEvents, ExchangeDLX} {
		_ = ch.ExchangeDelete(ex, false, false)
		ch, _ = reopen(conn, ch)
	}
}

// A failed delete closes the channel, so open a fresh one when needed.
func reopen(conn *amqp.Connection, ch *amqp.Channel) (*amqp.Channel, error) {
	if !ch.IsClosed() {
		return ch, nil
	}
	return conn.Channel()
}

func publisher(url string) *Publisher {
	p := NewPublisher(url)
	p.RetryDelay = testRetryDelay
	return p
}

func startConsumer(t *testing.T, url string, h Handler) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	c := &Consumer{URL: url, Queue: QueueInventory, Handler: h, RetryDelay: testRetryDelay}
	go c.Run(ctx)
}

func publishOrderEvent(t *testing.T, p *Publisher) Envelope {
	t.Helper()
	env, err := NewEnvelope(RouteOrderCreated, OrderCreated{OrderID: "o1", UserID: "u1", Items: []OrderItem{{ProductID: 1, Quantity: 2}}})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(env)
	if err := p.Publish(context.Background(), ExchangeEvents, RouteOrderCreated, body, nil); err != nil {
		t.Fatal(err)
	}
	return env
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func deadLetterCount(t *testing.T, url string) int {
	t.Helper()
	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ch, _ := conn.Channel()
	q, err := ch.QueueInspect(QueueDeadLetters)
	if err != nil {
		t.Fatal(err)
	}
	return q.Messages
}

func TestPublishAndConsume(t *testing.T) {
	url := brokerURL(t)
	resetBroker(t, url)
	got := make(chan Envelope, 1)
	startConsumer(t, url, func(_ context.Context, e Envelope) error { got <- e; return nil })
	p := publisher(url)
	defer p.Close()
	sent := publishOrderEvent(t, p)
	select {
	case e := <-got:
		var oc OrderCreated
		if err := e.Decode(&oc); err != nil || e.EventID != sent.EventID || oc.OrderID != "o1" || oc.Items[0].Quantity != 2 {
			t.Fatalf("wrong event: %+v %+v %v", e, oc, err)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("no message received")
	}
}

func TestUnroutableMessageIsAnError(t *testing.T) {
	url := brokerURL(t)
	resetBroker(t, url)
	p := publisher(url)
	defer p.Close()
	if err := p.Publish(context.Background(), ExchangeEvents, "nobody.listens", []byte(`{}`), nil); err == nil {
		t.Fatal("a message no queue wants must not look like a success")
	}
}

func TestQueueExistsBeforeConsumerEverStarts(t *testing.T) {
	url := brokerURL(t)
	resetBroker(t, url)
	p := publisher(url)
	defer p.Close()
	publishOrderEvent(t, p) // no consumer has ever run, the publisher declared the queue

	got := make(chan struct{}, 1)
	startConsumer(t, url, func(context.Context, Envelope) error { got <- struct{}{}; return nil })
	select {
	case <-got:
	case <-time.After(8 * time.Second):
		t.Fatal("event published before the consumer started was lost")
	}
}

func TestTransientFailureIsRetried(t *testing.T) {
	url := brokerURL(t)
	resetBroker(t, url)
	var calls atomic.Int32
	startConsumer(t, url, func(context.Context, Envelope) error {
		if calls.Add(1) < MaxAttempts {
			return errors.New("database busy")
		}
		return nil
	})
	p := publisher(url)
	defer p.Close()
	publishOrderEvent(t, p)
	waitFor(t, "third attempt", func() bool { return calls.Load() == MaxAttempts })
	time.Sleep(400 * time.Millisecond)
	if calls.Load() != MaxAttempts || deadLetterCount(t, url) != 0 {
		t.Fatalf("calls=%d deadLetters=%d", calls.Load(), deadLetterCount(t, url))
	}
}

func TestMessageIsDeadLetteredAfterMaxAttempts(t *testing.T) {
	url := brokerURL(t)
	resetBroker(t, url)
	var calls atomic.Int32
	startConsumer(t, url, func(context.Context, Envelope) error { calls.Add(1); return errors.New("always broken") })
	p := publisher(url)
	defer p.Close()
	publishOrderEvent(t, p)
	waitFor(t, "dead letter", func() bool { return deadLetterCount(t, url) == 1 })
	if calls.Load() != MaxAttempts {
		t.Fatalf("handler ran %d times, want %d", calls.Load(), MaxAttempts)
	}
}

func TestPermanentErrorGoesStraightToDeadLetters(t *testing.T) {
	url := brokerURL(t)
	resetBroker(t, url)
	var calls atomic.Int32
	startConsumer(t, url, func(context.Context, Envelope) error { calls.Add(1); return ErrPermanent })
	p := publisher(url)
	defer p.Close()
	publishOrderEvent(t, p)
	waitFor(t, "dead letter", func() bool { return deadLetterCount(t, url) == 1 })
	if calls.Load() != 1 {
		t.Fatalf("handler ran %d times, want 1", calls.Load())
	}
}

func TestUnreadableMessageIsDeadLettered(t *testing.T) {
	url := brokerURL(t)
	resetBroker(t, url)
	var calls atomic.Int32
	startConsumer(t, url, func(context.Context, Envelope) error { calls.Add(1); return nil })
	p := publisher(url)
	defer p.Close()
	if err := p.Publish(context.Background(), ExchangeEvents, RouteOrderCreated, []byte(`not json`), nil); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "dead letter", func() bool { return deadLetterCount(t, url) == 1 })
	if calls.Load() != 0 {
		t.Fatal("handler must not see unreadable messages")
	}
}

func TestHandlerSeesTheAttemptNumber(t *testing.T) {
	url := brokerURL(t)
	resetBroker(t, url)
	seen := make(chan [2]any, MaxAttempts+1)
	startConsumer(t, url, func(ctx context.Context, _ Envelope) error {
		seen <- [2]any{Attempt(ctx), IsLastAttempt(ctx)}
		return errors.New("fail until dead-lettered")
	})
	p := publisher(url)
	defer p.Close()
	publishOrderEvent(t, p)
	waitFor(t, "dead letter", func() bool { return deadLetterCount(t, url) == 1 })
	close(seen)
	var got [][2]any
	for s := range seen {
		got = append(got, s)
	}
	want := [][2]any{{0, false}, {1, false}, {2, true}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestPanickingHandlerIsTreatedAsFailure(t *testing.T) {
	url := brokerURL(t)
	resetBroker(t, url)
	var calls atomic.Int32
	startConsumer(t, url, func(context.Context, Envelope) error {
		if calls.Add(1) == 1 {
			panic("boom")
		}
		return nil
	})
	p := publisher(url)
	defer p.Close()
	publishOrderEvent(t, p)
	waitFor(t, "retry after panic", func() bool { return calls.Load() == 2 })
}

type recordingExecer struct{ args []any }

func (r *recordingExecer) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	r.args = args
	return pgconn.CommandTag{}, nil
}

func TestEnqueueStoresAFullEnvelope(t *testing.T) {
	rec := &recordingExecer{}
	if err := Enqueue(context.Background(), rec, RouteStockReserved, StockReserved{OrderID: "o9"}); err != nil {
		t.Fatal(err)
	}
	var env Envelope
	if err := json.Unmarshal([]byte(rec.args[2].(string)), &env); err != nil {
		t.Fatal(err)
	}
	if env.EventID != rec.args[0] || env.Type != RouteStockReserved || rec.args[1] != RouteStockReserved || string(env.Data) != `{"orderId":"o9"}` {
		t.Fatalf("envelope = %+v args=%v", env, rec.args)
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS outbox`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, OutboxSchema); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestRelayPublishesThenMarksRowsOnce(t *testing.T) {
	url := brokerURL(t)
	pool := testPool(t)
	resetBroker(t, url)
	ctx := context.Background()
	if err := Enqueue(ctx, pool, RouteOrderCreated, OrderCreated{OrderID: "o1", UserID: "u1", Items: []OrderItem{{ProductID: 1, Quantity: 1}}}); err != nil {
		t.Fatal(err)
	}
	got := make(chan Envelope, 4)
	startConsumer(t, url, func(_ context.Context, e Envelope) error { got <- e; return nil })

	p := publisher(url)
	defer p.Close()
	relay := &Relay{Pool: pool, Pub: p}
	if n, err := relay.Flush(ctx, 10); err != nil || n != 1 {
		t.Fatalf("flush: n=%d err=%v", n, err)
	}
	select {
	case <-got:
	case <-time.After(8 * time.Second):
		t.Fatal("relay did not deliver the event")
	}
	if n, err := relay.Flush(ctx, 10); err != nil || n != 0 {
		t.Fatalf("second flush must find nothing: n=%d err=%v", n, err)
	}
	var unpublished int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE published_at IS NULL`).Scan(&unpublished)
	if unpublished != 0 {
		t.Fatalf("%d rows still unpublished", unpublished)
	}
}

func TestRelayKeepsRowsWhenBrokerIsDown(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	if err := Enqueue(ctx, pool, RouteOrderCreated, OrderCreated{OrderID: "o1"}); err != nil {
		t.Fatal(err)
	}
	down := NewPublisher("amqp://nobody:nopass@127.0.0.1:1/")
	relay := &Relay{Pool: pool, Pub: down}
	n, err := relay.Flush(ctx, 10)
	if err == nil || n != 0 {
		t.Fatalf("want an error and 0 sent, got n=%d err=%v", n, err)
	}
	var unpublished int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE published_at IS NULL`).Scan(&unpublished)
	if unpublished != 1 {
		t.Fatalf("the event must wait in the outbox, unpublished=%d", unpublished)
	}
}

func TestTraceContextTravelsInTheEnvelope(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	ctx, span := tp.Tracer("t").Start(context.Background(), "place order")
	defer span.End()

	rec := &recordingExecer{}
	if err := Enqueue(ctx, rec, RouteOrderCreated, OrderCreated{OrderID: "o1"}); err != nil {
		t.Fatal(err)
	}
	var env Envelope
	if err := json.Unmarshal([]byte(rec.args[2].(string)), &env); err != nil {
		t.Fatal(err)
	}
	if env.Trace["traceparent"] == "" {
		t.Fatalf("no traceparent in envelope: %+v", env.Trace)
	}

	// The consumer side, possibly much later and in another process.
	_, consumer := startFromEnvelope(context.Background(), "process", trace.SpanKindConsumer, env.EventID, env.Trace)
	defer consumer.End()
	if consumer.SpanContext().TraceID() != span.SpanContext().TraceID() {
		t.Fatal("the consumer must continue the producer's trace")
	}
}
