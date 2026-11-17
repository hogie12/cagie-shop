#!/bin/sh
# Writes a throwaway .env with random passwords for CI. Never use it for real.
set -eu
cd "$(dirname "$0")/.."
if [ -f .env ] && [ "${CI:-}" != "true" ]; then
  echo "refusing to overwrite .env outside CI" >&2; exit 1
fi
rand() { openssl rand -hex 16; }
cat > .env <<ENV
POSTGRES_USER=cagie
POSTGRES_PASSWORD=$(rand)
RABBITMQ_USER=cagie
RABBITMQ_PASSWORD=$(rand)
ADMIN_EMAIL=admin@cagie.local
ADMIN_PASSWORD=$(rand)
CORS_ALLOWED_ORIGINS=http://localhost:3000
COOKIE_SECURE=false
FAKE_PROVIDER_CONTROLS=true
ENV
