# Cagie Pastry & Flower Shop

[![CI](https://github.com/hogie12/cagie-shop/actions/workflows/ci.yml/badge.svg)](https://github.com/hogie12/cagie-shop/actions/workflows/ci.yml)

A learning project: an e-commerce backend split into microservices, with a web frontend.

## Stack
- Backend: Go, one module per service, `go.work` workspace
- Data: PostgreSQL, one database per service
- Events: RabbitMQ
- Frontend: Next.js 16 (App Router) + Tailwind 4 + motion.dev, in `apps/web` (Vercel later)
- Run: Docker Compose (VPS), Kubernetes with kind (laptop)

## Services (planned)
auth, catalog, gateway, inventory, order, payment (all done)

## Run the infrastructure
```bash
cp .env.example .env   # then set real passwords
make up                # Postgres + RabbitMQ
make psql              # list the per-service databases
```
RabbitMQ web UI: http://localhost:15672 (user and password from `.env`).

## Catalog service (internal, reached through the gateway at /products)
`GET /products[?category=pastry|flower]`, `GET /products/{id}`, `POST /products`, `PUT /products/{id}`, `DELETE /products/{id}` (soft delete), `GET /healthz`.
Writes are admin-only at the gateway, and Catalog checks `X-User-Role` again as a second layer. It is not published to the host.
```bash
make sqlc          # regenerate DB code after editing queries/ or migrations/
make test-catalog  # handler tests + real-Postgres test
```

## Auth service (internal, reached through the gateway at /auth/*)
`POST /register`, `/login`, `/refresh`, `/logout`, `GET /healthz`. Access token: RS256 JWT, 15 min. Refresh token: opaque, 7 days, stored hashed, rotated on every use.
First run: `make keys` (creates `.keys/`, gitignored), then `make up`. The first admin comes from `ADMIN_EMAIL` / `ADMIN_PASSWORD` in `.env`.

## Inventory service (internal)
Public through the gateway: `GET /inventory/{productId}`, `PUT /inventory/{productId}` (admin). Stock is reserved by reacting to `order.created` events, not by REST calls.
For each order it reserves every line in one transaction (all or nothing), then announces `stock.reserved` or `stock.rejected`. A processed-events table (`inbox`) makes repeated deliveries harmless.
Seed stock uses product ids 1-6 from the catalog seed. Inventory does not check that a product exists in Catalog (separate databases).

## Order service (internal)
`POST /orders` `{"items":[{"productId":1,"quantity":2}]}` returns **202** with the order as `pending` and a `Location` header. `GET /orders/{id}` shows how it progresses: `pending` (waiting for stock) → `awaiting_payment` (stock held) → `confirmed`, or `cancelled` with a `cancelReason`. Also `GET /orders` (own orders). `GET /orders/{id}` is for the owner or an admin.
Prices come from Catalog at order time and are copied into the order. The order and its `order.created` event are saved in one transaction (outbox).

## Payment service (internal)
No public API. It reacts to `payment.requested` (sent by Order once stock is held), charges a **fake provider**, and announces `payment.succeeded` or `payment.failed`. One payment per order (unique `order_id`), so a repeated request cannot charge or announce twice.
The fake provider has three modes, switched by an admin with `PUT /payments/provider` `{"mode":"succeed|decline|down"}` (only served when `FAKE_PROVIDER_CONTROLS=true`, the Compose default):
- `decline`: a final answer (card declined) -> `payment.failed` at once.
- `down`: an outage. The consumer retries; on the last attempt the payment becomes `payment.failed` ("payment provider unavailable") so stock is not held forever.
When payment fails, Inventory puts the reserved stock back (the saga's compensation step).

## Events (RabbitMQ)
Exchange `shop.events` (topic). Every message is `{eventId, type, occurredAt, data}`; contracts live in `packages/shared/events.go`.
```
order.created ──> Inventory ──stock.reserved──> Order ──payment.requested──> Payment
                     │  └─stock.rejected──> Order (cancel)                     │
                     └<──── payment.failed (release stock) ─────────────────────┤
                                  Order <── payment.succeeded / payment.failed ─┘
```
Queues: `inventory.events`, `order.events`, `payment.events`, plus a `.retry` twin each and `shop.dead-letters`.
| Concern | How it is handled |
|---|---|
| Crash between saving and publishing | Transactional outbox: the event is saved with the data, a relay publishes it after the broker confirms |
| Broker down | Events wait in the outbox; publisher and consumers reconnect with backoff |
| Same message twice | Inventory records processed `eventId`s in the same transaction as the stock change; Order only changes a `pending` order |
| Failing message | Retried 3 times via a 5-second delay queue, then moved to `shop.dead-letters` |
| Unreadable or invalid message | Straight to `shop.dead-letters` |
| A decision that must always be made | A handler can ask `IsLastAttempt(ctx)` and turn "still failing" into a business outcome (Payment does this for outages) |
| Unroutable message | Publish fails (mandatory flag), so the outbox row stays and is retried |
Look at it in the RabbitMQ UI: http://localhost:15672 (Queues tab).

### Payment demo (five scenarios, about a minute)
```bash
./scripts/demo-payment.sh   # normal, declined card, outage that recovers, outage that never ends, duplicate message
```

### Failure demos (run them, then read the decisions log)
```bash
docker compose stop rabbitmq     # place an order: you still get 202; start rabbitmq: it confirms
docker compose stop inventory    # place an order: it stays pending; start inventory: it confirms
```
For duplicates, publish the same `order.created` envelope several times through the RabbitMQ UI (Exchanges, `shop.events`, Publish message): stock changes once.

### Try it (login first, see the Auth section)
```bash
TOKEN=...   # accessToken from POST /auth/login
curl -X POST localhost:8080/orders -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"items":[{"productId":1,"quantity":2},{"productId":4,"quantity":1}]}'   # note the id
curl localhost:8080/orders/ORDER_ID -H "Authorization: Bearer $TOKEN"
curl localhost:8080/inventory/1
```

## Gateway (port 8080, the only published service)
`/auth/{register,login,refresh,logout}` go to Auth. `GET /products[/id]` and `GET /inventory/{id}` are public. `POST/PUT/DELETE /products` and `PUT /inventory/{id}` need an admin JWT. `/orders*` needs any valid JWT. `GET/PUT /payments/provider` is admin-only. `/internal/*` is never routed (no service has internal REST routes any more).
It verifies tokens with the public key, removes any client-sent `X-User-*` and `Authorization` headers, then adds `X-User-Id` / `X-User-Role` from the verified token. Also: per-IP rate limits (strict on login/register), CORS allowlist from `CORS_ALLOWED_ORIGINS`, 1 MB body limit, a request ID on every request, and a 502 when a service is down.
```bash
make test-setup   # once: creates the throwaway RabbitMQ vhost used by tests
make test         # all services and the shared library (needs `make up` running)
```

## Web app (apps/web)
```bash
make up                      # backend first
cp apps/web/.env.example apps/web/.env.local
make web                     # http://localhost:3000
make test-web                # lint, types, unit tests
make test-e2e                # Playwright: browse, buy, declined card (uses the running stack)
```
Pages: `/` (shop, server-rendered and cached about a minute), `/cart`, `/login`, `/orders`, `/orders/[id]` (live status, polls every second), `/admin` (stock, prices, products, payment provider mode).
Sign-in: the access token stays in memory; the refresh token is an HttpOnly cookie set by Auth on `/auth`. In production the shop and API must share one site (for example `shop.example.com` and `api.example.com`) and run with `COOKIE_SECURE=true`.
Photos: Unsplash, credited in `apps/web/public/images/CREDITS.md`. Store copy on the home page is sample text, and `[YOUR ADDRESS]`-style placeholders need real details.

## Logs and tracing
Every service writes JSON logs (`log/slog`) with `service`, and `trace_id` / `span_id` on request and event lines. Tracing uses OpenTelemetry; Jaeger shows the traces.
```bash
make build      # images one at a time (parallel builds can exhaust Docker's memory)
make up-obs     # everything + Jaeger, services export traces to it
./scripts/trace-check.sh   # places an order and lists every service in its trace
```
Jaeger UI: http://localhost:16686. The `X-Request-Id` the gateway returns is the trace id, so a customer's reference finds the trace and every related log line.
How the trace crosses RabbitMQ: the trace context is saved inside the event envelope when the event enters the outbox, and each consumer continues it.
Plain `make up` leaves tracing off (no exporter), but logs still share a trace id per request.

## CI (GitHub Actions)
`.github/workflows/ci.yml` runs on every push and pull request:
- **Go tests**: all packages against real Postgres and RabbitMQ (`scripts/ci-test-go.sh`, also runnable on a laptop after `make up`).
- **Web checks**: lint, typecheck, unit tests, production build.
- **End-to-end**: the whole stack in Docker plus the Playwright tests.
- **Images** (pushes to `main` only, after all tests pass): `ghcr.io/hogie12/cagie-shop-<service>`, tagged with the commit SHA and `latest`, for linux/amd64.

CI makes its own throwaway `.env` and keys each run (`scripts/ci-env.sh`); no repository secrets are needed.

## Deployment
Production runs on one VPS with Docker Compose behind Caddy (automatic HTTPS), and the web app on Vercel. Step-by-step guide: [deploy/README.md](deploy/README.md).

## Decisions log
| # | Decision | Why |
|---|----------|-----|
| 1 | One database per service | Services never read each other's tables, so they can change and fail independently. Data is shared through APIs and events only. |
| 2 | Events (RabbitMQ) for the order flow | Order, stock and payment steps are decoupled. If one service is down, messages wait in the queue. |
| 3 | Go for the backend | Small binaries and low memory, which suits a 2 GB VPS. Common language for cloud tooling. |
| 4 | Shared module holds event types only | Services agree on message shapes without sharing business code. |
| 5 | Memory limits on containers | Target VPS has 2 GB RAM. Measured idle: Postgres ~29 MB, RabbitMQ ~105 MB. |
| 6 | Kubernetes only on the laptop | 2 GB is too small for a Kubernetes control plane plus the services. |
| 7 | sqlc + pgx instead of an ORM | Go has no Prisma equivalent. SQL stays visible and the generated code is type-safe. |
| 8 | Prices as integer rupiah | Floats cause rounding errors with money. |
| 9 | DTOs separate from DB structs | A schema change does not silently change the public API. |
| 10 | Soft delete (`active=false`) | Old orders can still reference a product that is no longer sold. |
| 11 | Service migrates itself at startup | Simple for one instance. With many replicas, run migrations as a separate job (revisit in the Kubernetes stage). |
| 12 | Go 1.26 base image | goose v3.30.1 requires Go 1.26. Measured: catalog uses ~10 MB RAM, image ~45 MB. |
| 13 | RS256 (key pair) instead of a shared secret | Only Auth can sign tokens. The gateway holds just the public key, so a compromised gateway cannot mint tokens. |
| 14 | Short access token + rotating refresh token | A stolen access token expires in 15 minutes. A replayed refresh token revokes the whole session family. |
| 15 | Refresh tokens stored as SHA-256 hashes | A database leak does not give attackers usable tokens. |
| 16 | argon2id at OWASP minimum cost (19 MiB) | Strong hashing, but every concurrent login costs that RAM on a 2 GB VPS. |
| 17 | Same error for unknown email and wrong password, plus a dummy hash | Stops attackers from discovering which emails are registered (message and timing). |
| 18 | Token contract (issuer, audience, role names) duplicated in auth and gateway | Services share a contract, not code, and each Docker build stays self-contained. |
| 19 | Gateway verifies JWTs locally | No call to Auth per request, so browsing keeps working if Auth is down. Cost: a token stays valid until it expires (15 min) even after logout. |
| 20 | Gateway deletes client `X-User-*` headers on every request | Services trust these headers, so a client must never be able to set them. |
| 21 | Only RS256 accepted | Blocks `alg: none` and HS256-with-public-key attacks. Tested. |
| 22 | Only the gateway is published; Catalog and Auth are internal | Nothing can bypass the gateway's checks from outside. Catalog does not re-check roles yet (known gap, revisit with Kubernetes network policies). |
| 23 | Rate limiter ignores `X-Forwarded-For` for now | Otherwise a client could fake an IP. When Caddy is added in front, its address must be configured as a trusted proxy. |
| 24 | ~~Inventory's reserve/release under `/internal`~~ (replaced in stage 5 by events) | Internal APIs and public APIs are different things. Only services inside the network may change stock directly. |
| 25 | Services re-check `X-User-*` headers (Catalog writes, Inventory writes, Order) | Defense in depth: a request that skips the gateway is still refused. Still relies on the network keeping outsiders away. |
| 26 | Prices come from Catalog at order time and are copied into the order | Never trust client prices, and later price changes must not rewrite history. |
| 27 | Reserve is one atomic `UPDATE ... WHERE quantity >= n` | A read-then-write check would oversell under concurrency. |
| 28 | ~~No retries and no idempotency in stage 4~~ (shown on purpose, fixed in stage 5) | A timeout is ambiguous (did it happen or not?). Retrying would double-reserve. Shown on purpose, fixed in stage 5. |
| 29 | Order stores a copy of name and price per line | Order history stays correct even if the product changes or is removed. |
| 30 | ~~Chaos env flag~~ (removed in stage 5: stopping a container reproduces failures for real) | |
| 31 | Order returns 202 and the client reads the result later | The order is not decided yet; claiming success before stock is reserved would be a lie. The frontend will poll. |
| 32 | Transactional outbox | Saving data and publishing an event are two systems; doing both separately can lose the event. One transaction cannot. |
| 33 | At-least-once delivery plus idempotent consumers | Exactly-once does not exist. Inventory records event ids in the same transaction as the stock change. |
| 34 | Inventory reserves a whole order in one local transaction, rows locked in product-id order | All or nothing, and two orders cannot deadlock each other. |
| 35 | Queues are declared by every service on every connect | If only the consumer declared its queue, an event published before its first start would be dropped. Tested. |
| 36 | Publisher uses confirms and the mandatory flag | "Sent" means the broker stored it; a message nobody wants is an error, not a silent drop. |
| 37 | Retry through a TTL queue, then a dead-letter queue | Transient failures heal themselves, poison messages cannot loop forever, and nothing is lost. |
| 38 | `packages/shared` is a real library; Order and Inventory build from the repo root | One implementation of the tricky messaging code. Cost: those two Dockerfiles copy `packages/shared` and use a root `.dockerignore`. |
| 39 | Classic durable queues, one relay per service | Light on RAM for a 2 GB VPS. Several relays could reorder events; revisit if a service needs to scale out. |
| 40 | Order owns the price and sends `payment.requested` with the amount | Inventory never needs to know prices, and the status change and the request happen in one transaction. |
| 41 | A decline is an answer, an outage is an error | Declines are final and become `payment.failed`. Outages are retried, because the money may still be collectable. |
| 42 | On the last retry an outage becomes `payment.failed` | Dead-lettering would leave the order waiting and its stock held forever. A saga must always reach a decision. |
| 43 | Compensation: `payment.failed` makes Inventory release the stock | A distributed transaction has no rollback; each step needs an explicit undo. Releases are marked per reservation, so they happen once. |
| 44 | The provider call happens outside a database transaction, with an idempotency key | Never hold a transaction open over a network call. If we crash after charging, the retry gets the same answer and no money moves twice. |
| 45 | Every status change is guarded by the current status | Duplicates and late messages change nothing, whatever order they arrive in. |
| 46 | Fake provider controls only exist when `FAKE_PROVIDER_CONTROLS=true` | A teaching switch that can break payments must never exist in a real deployment. |
| 47 | Queues renamed to `inventory.events`, `order.events`, `payment.events` | Names should say who reads the queue, not one event it once carried. The old empty queues were deleted by hand, because declaring new ones does not remove old ones. |

| 48 | Refresh token in an HttpOnly, SameSite=Lax cookie; access token only in memory | Scripts on the page (including injected ones) can never read the refresh token. SameSite blocks cross-site use of the cookie. |
| 49 | Gateway forwards cookies only to Auth; CORS allows credentials for exact allowlisted origins | Least privilege: no other service ever sees the refresh token. |
| 50 | The browser runs one refresh at a time | Refresh tokens are single-use. Two parallel refreshes (React runs effects twice in development) look like theft and revoke every session. |
| 51 | Shop page rendered on the server with `'use cache'` + `cacheLife('minutes')` | Fast first load and part of the prefetched shell. Cost: shown stock may be a few minutes old; the order flow decides real stock and price. |
| 52 | Cart in localStorage, prices there are display-only | No account needed to shop; the order service always prices from the catalog. |
| 53 | Order page polls every second until decided (gives up after 2 minutes) | Simple and enough for a saga that settles in seconds. Server-sent events would be the next step. |
| 54 | motion.dev with `MotionConfig reducedMotion="user"`; no three.js | Animations respect the visitor's reduce-motion setting. 3D would add weight with no product models and clash with the flat illustrations. |
| 55 | Unsplash photos self-hosted; photo paths stored in the catalog | No hotlinking, works offline in dev, and admins can change a product's photo path. |
| 56 | No TanStack Query | One polling hook did not justify a data library. |
| 57 | Next 16 rules followed: `useSearchParams`/`useParams` inside Suspense, `error.tsx` uses `retry` | Read from the docs bundled with the installed Next version, not from memory. |

| 58 | OpenTelemetry + Jaeger, set up once in `packages/shared/obs` | One vendor-neutral standard; all six services trace and log the same way. |
| 59 | The gateway drops client trace headers and starts every trace | Outsiders cannot join or fake traces. The request id equals the trace id. |
| 60 | Trace context travels inside the event envelope | The outbox publishes later from another goroutine, so the request context is gone by then; the envelope keeps it, with no schema change. |
| 61 | Jaeger is an optional Compose profile, in-memory, 256 MB limit | Useful locally; on a 2 GB VPS it only runs when needed. Traces are lost on restart. |
| 62 | `log/slog` JSON logs with trace ids | Standard library, machine-readable, and every line links to its trace. |
| 63 | Images build one at a time (`make build`) | Six parallel Go builds with the OpenTelemetry libraries exhausted Docker's 4 GB. |

| 64 | CI starts Postgres and RabbitMQ from the same Compose file as a laptop, with random throwaway passwords | No credentials in the repo, and CI behaves like local development. |
| 65 | Images are built in CI and pushed to GHCR; the VPS only pulls | A 2 GB VPS cannot build the Go images. Each image is tagged with its commit SHA, so a deploy can roll back to an exact build. |
| 66 | Dockerfiles cross-compile and run as numeric user 10001 | Any target CPU builds without emulation; the final image has no RUN step and still never runs as root. |
| 67 | Action versions pinned exactly, skipping releases under two weeks old | Reproducible runs and a short buffer against a bad release. |

| 68 | Caddy in front, the only published container, with a fixed IP the gateway trusts for `X-Forwarded-For` | Automatic HTTPS. Each visitor keeps their own rate limit behind the proxy, and nobody else can fake an IP. |
| 69 | Secrets and JWT keys generated on the server by `deploy/setup.sh` | They never exist in git, CI or a laptop. The private key is readable only by the container user. |
| 70 | Production images pinned by tag in `deploy/.env` (`latest` or `sha-<commit>`) | One command updates or rolls back to an exact CI build. |
| 71 | RabbitMQ gets a volume and a fixed hostname in production | Queued messages survive restarts; RabbitMQ keys its data by node name. |

### Known gaps after stage 9
- No metrics or dashboards (Prometheus/Grafana) and no alerts, for example on dead-lettered messages.
- The web app does not trace in the browser; traces start at the gateway.

### Known gaps after stage 7
- Admin changes reach the cached shop page within about a minute (no on-demand revalidation yet).
- Product photos are file paths; there is no upload yet.
- Not deployed yet (VPS + Vercel stage).

### Known gaps after stage 6
- No refunds: a payment that succeeded cannot be undone (for example if the order is cancelled afterwards).
- No payment timeout: an order waiting for a payment that never arrives stays `awaiting_payment`.
- An order stuck in `pending` (for example after a dead-lettered message) is never cleaned up. One from stage 4 still exists in the database.
- Dead-lettered messages are only visible in the RabbitMQ UI; nothing alerts on them yet (stage 9).
- Stage 6 builds needed several minutes on a busy machine; the Docker image builds run in parallel and compete for CPU.
