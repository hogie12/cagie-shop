# Docker Desktop's CLI is not always on PATH on macOS.
export PATH := /Applications/Docker.app/Contents/Resources/bin:$(PATH)

.PHONY: up down logs ps psql

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

ps:
	docker compose ps

psql:
	docker compose exec postgres sh -c 'psql -P pager=off -U "$$POSTGRES_USER" -l'

.PHONY: sqlc test-catalog
sqlc:
	cd services/catalog && sqlc generate

# Needs `make up` first; uses the catalog_test database.
test-catalog:
	set -a && . ./.env && set +a && cd services/catalog && \
	TEST_DATABASE_URL="postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@localhost:5432/catalog_test?sslmode=disable" go test ./... -count=1

.PHONY: keys test-auth
keys:
	./scripts/gen-keys.sh

test-auth:
	set -a && . ./.env && set +a && cd services/auth && \
	TEST_DATABASE_URL="postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@localhost:5432/auth_test?sslmode=disable" go test ./... -count=1

.PHONY: test-gateway test
test-gateway:
	cd services/gateway && go test ./... -count=1

test: test-catalog test-auth test-gateway test-inventory test-order test-shared test-payment

.PHONY: test-inventory
test-inventory:
	set -a && . ./.env && set +a && cd services/inventory && \
	TEST_DATABASE_URL="postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@localhost:5432/inventory_test?sslmode=disable" go test ./... -count=1

.PHONY: test-order
test-order:
	set -a && . ./.env && set +a && cd services/order && \
	TEST_DATABASE_URL="postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@localhost:5432/orders_test?sslmode=disable" go test ./... -count=1

# Creates the throwaway RabbitMQ vhost the messaging tests use (safe to repeat).
.PHONY: test-setup test-shared
test-setup:
	docker compose exec -T rabbitmq sh -c 'rabbitmqctl add_vhost test >/dev/null 2>&1; rabbitmqctl set_permissions -p test "$$RABBITMQ_DEFAULT_USER" ".*" ".*" ".*"'

test-shared:
	set -a && . ./.env && set +a && cd packages/shared && \
	RABBITMQ_TEST_URL="amqp://$$RABBITMQ_USER:$$RABBITMQ_PASSWORD@localhost:5672/test" \
	TEST_DATABASE_URL="postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@localhost:5432/shared_test?sslmode=disable" go test ./... -count=1

.PHONY: test-payment
test-payment:
	set -a && . ./.env && set +a && cd services/payment && \
	TEST_DATABASE_URL="postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@localhost:5432/payment_test?sslmode=disable" go test ./... -count=1

# Web app (Next.js). Needs `make up` for the backend.
.PHONY: web test-web test-e2e
web:
	cd apps/web && pnpm dev

test-web:
	cd apps/web && pnpm lint && pnpm exec tsc --noEmit && pnpm test

test-e2e:
	cd apps/web && pnpm test:e2e

# Everything plus Jaeger, with every service sending traces to it.
# UI: http://localhost:16686
.PHONY: up-obs
up-obs:
	OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318 docker compose --profile obs up -d

# Builds the Go images one at a time. Building all six in parallel can run
# Docker out of memory (the OpenTelemetry libraries make each build heavier).
SERVICES := catalog auth gateway inventory order payment
.PHONY: build
build:
	for s in $(SERVICES); do docker compose build $$s || exit 1; done
