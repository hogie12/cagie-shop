#!/bin/sh
# Runs every Go test package against real Postgres and RabbitMQ.
# CI sets PG_URL and RABBITMQ_URL; on a laptop they come from .env (make up first).
set -eu
cd "$(dirname "$0")/.."
if [ -z "${PG_URL:-}" ]; then
  set -a; . ./.env; set +a
  PG_URL="postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@localhost:5432"
  RABBITMQ_URL="amqp://$RABBITMQ_USER:$RABBITMQ_PASSWORD@localhost:5672"
fi

run() { # module dir, test database (empty for none)
  echo "== $1"
  if [ -n "$2" ]; then
    (cd "$1" && TEST_DATABASE_URL="$PG_URL/$2?sslmode=disable" RABBITMQ_TEST_URL="$RABBITMQ_URL/test" go test ./... -count=1)
  else
    (cd "$1" && go test ./... -count=1)
  fi
}

run packages/shared shared_test
run services/catalog catalog_test
run services/auth auth_test
run services/gateway ""
run services/inventory inventory_test
run services/order orders_test
run services/payment payment_test
