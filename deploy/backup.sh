#!/bin/sh
# Dumps each service database to deploy/backups/ and keeps 7 days.
# Run daily from cron (see deploy/README.md). Copy the folder off the server
# now and then: a backup on the same disk does not survive losing the disk.
set -eu
cd "$(dirname "$0")"
. ./.env
mkdir -p backups
stamp=$(date +%Y-%m-%d)
for db in auth catalog inventory orders payment; do
  docker compose -f docker-compose.prod.yml --env-file .env exec -T postgres \
    pg_dump -U "$POSTGRES_USER" -Fc "$db" > "backups/$db-$stamp.dump"
done
find backups -name '*.dump' -mtime +7 -delete
echo "backups written for $stamp"
