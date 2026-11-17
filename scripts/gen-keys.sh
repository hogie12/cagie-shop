#!/bin/sh
# Generates the RSA key pair used to sign (auth) and verify (gateway) JWTs.
# Dev only: the private key is world-readable so the non-root container user can read it.
# In production the private key lives in a secret store (Kubernetes Secret later).
set -eu
dir="$(cd "$(dirname "$0")/.." && pwd)/.keys"
mkdir -p "$dir"
if [ -f "$dir/jwt_private.pem" ]; then
  echo "keys already exist in $dir (delete them to regenerate)"; exit 0
fi
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "$dir/jwt_private.pem"
openssl pkey -in "$dir/jwt_private.pem" -pubout -out "$dir/jwt_public.pem"
chmod 644 "$dir/jwt_private.pem" "$dir/jwt_public.pem"
echo "created $dir/jwt_private.pem and jwt_public.pem"
