#!/bin/sh
# One-time setup on the server: writes deploy/.env with strong random
# passwords and creates the JWT key pair. Secrets are made here and never
# leave the server.
#
#   sudo ./setup.sh <shop domain> <api domain> <your email>
#   sudo ./setup.sh shop.example.com api.example.com you@example.com
set -eu
cd "$(dirname "$0")"

if [ $# -ne 3 ]; then
  echo "usage: sudo ./setup.sh <shop domain> <api domain> <email>" >&2
  exit 1
fi
if [ "$(id -u)" -ne 0 ]; then
  echo "run with sudo (the key file must be owned by the container user)" >&2
  exit 1
fi
if [ -f .env ]; then
  echo ".env already exists; delete it first if you really want new secrets" >&2
  exit 1
fi

rand() { openssl rand -hex 24; }
admin_password=$(rand)

umask 077
cat > .env <<EOF
SHOP_DOMAIN=$1
API_DOMAIN=$2
ACME_EMAIL=$3
POSTGRES_USER=cagie
POSTGRES_PASSWORD=$(rand)
RABBITMQ_USER=cagie
RABBITMQ_PASSWORD=$(rand)
ADMIN_EMAIL=$3
ADMIN_PASSWORD=$admin_password
FAKE_PROVIDER_CONTROLS=true
IMAGE_TAG=latest
EOF

mkdir -p keys
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out keys/jwt_private.pem 2>/dev/null
openssl pkey -in keys/jwt_private.pem -pubout -out keys/jwt_public.pem
# The services run as user 10001 inside their containers.
chown 10001:10001 keys/jwt_private.pem keys/jwt_public.pem
chmod 400 keys/jwt_private.pem
chmod 444 keys/jwt_public.pem

echo "Wrote deploy/.env and deploy/keys/."
echo
echo "Admin login for the shop (shown once, save it in your password manager):"
echo "  email:    $3"
echo "  password: $admin_password"
