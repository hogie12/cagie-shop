#!/bin/sh
# Pulls the images CI built and (re)starts the stack.
#
#   ./deploy.sh               # the images tag already in .env (latest at first)
#   ./deploy.sh <full sha>    # an exact build, e.g. to roll back (git rev-parse HEAD)
#   ./deploy.sh latest        # back to the newest build
set -eu
cd "$(dirname "$0")"
[ -f .env ] || { echo "no .env: run sudo ./setup.sh first" >&2; exit 1; }

if [ $# -ge 1 ]; then
  tag=latest
  [ "$1" = latest ] || tag="sha-$1"   # CI tags each image sha-<full commit sha>
  sed -i "s/^IMAGE_TAG=.*/IMAGE_TAG=$tag/" .env
  echo "Using images tagged $tag"
fi

compose="docker compose -f docker-compose.prod.yml --env-file .env"
$compose pull
$compose up -d --remove-orphans
docker image prune -f >/dev/null
$compose ps --format 'table {{.Service}}\t{{.Status}}'
