#!/bin/sh
# Walks through the payment saga on the running stack (make up first).
# Needs: curl, python3, docker. Admin login comes from .env; the demo customer
# gets a throwaway password generated on each run, nothing is stored.
set -eu
cd "$(dirname "$0")/.."
export PATH="/Applications/Docker.app/Contents/Resources/bin:$PATH"
set -a; . ./.env; set +a

G=localhost:8080
J='Content-Type: application/json'
PRODUCT=2   # Chocolate Eclair (the price comes from the catalog)
json() { python3 -c "import sys,json;d=json.load(sys.stdin);print(d$1)"; }

login() { curl -s -X POST $G/auth/login -H "$J" -d "{\"email\":\"$1\",\"password\":\"$2\"}" | json "['accessToken']"; }
mode()  { curl -s -o /dev/null -X PUT $G/payments/provider -H "$J" -H "Authorization: Bearer $ADMIN" -d "{\"mode\":\"$1\"}"; echo "provider mode -> $1"; }
stock() { curl -s $G/inventory/$PRODUCT | json "['quantity']"; }
status(){ curl -s $G/orders/$1 -H "Authorization: Bearer $CUSTOMER" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d['status']+(' ('+d['cancelReason']+')' if d.get('cancelReason') else ''))"; }
place() { curl -s -X POST $G/orders -H "$J" -H "Authorization: Bearer $CUSTOMER" -d "{\"items\":[{\"productId\":$PRODUCT,\"quantity\":$1}]}" | json "['id']"; }
sql()   { docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$1" -tAc "$2"; }

# Prints each new status once, with the seconds since the order was placed.
watch_order() {
  id=$1; want=$2; last=""; start=$(date +%s)
  for _ in $(seq 1 120); do
    s=$(status "$id")
    if [ "$s" != "$last" ]; then echo "  +$(( $(date +%s) - start ))s  $s"; last=$s; fi
    case "$s" in "$want"*) return 0 ;; esac
    sleep 0.5
  done
  echo "  gave up waiting for '$want'"; return 1
}

ADMIN=$(login "$ADMIN_EMAIL" "$ADMIN_PASSWORD")
PW=$(openssl rand -hex 12); EMAIL="demo-$(date +%s)@example.com"
curl -s -o /dev/null -X POST $G/auth/register -H "$J" -d "{\"email\":\"$EMAIL\",\"password\":\"$PW\"}"
CUSTOMER=$(login "$EMAIL" "$PW")
reset() {
  mode succeed >/dev/null
  curl -s -o /dev/null -X PUT $G/inventory/$PRODUCT -H "$J" -H "Authorization: Bearer $ADMIN" -d '{"quantity":20}'
}
trap reset EXIT
reset

echo; echo "=== A. Normal order"
before=$(stock); id=$(place 2); watch_order "$id" confirmed
echo "  stock: $before -> $(stock)   (2 eclairs sold)"
CONFIRMED=$id

echo; echo "=== B. Card declined: order is cancelled and the stock comes back"
reset; mode decline; before=$(stock); id=$(place 3); watch_order "$id" cancelled; sleep 1
echo "  stock: $before -> $(stock)   (reserved, then released)"

echo; echo "=== C. Provider outage that ends during the retries: order still succeeds"
reset; mode down; id=$(place 1)
sleep 3; echo "  (3s: first attempt failed, retrying; bringing the provider back)"; mode succeed
watch_order "$id" confirmed

echo; echo "=== D. Provider outage that never ends: order is cancelled, not stuck"
reset; mode down; before=$(stock); id=$(place 1); watch_order "$id" cancelled; sleep 1
echo "  stock: $before -> $(stock)   (released)"

echo; echo "=== E. The same payment.requested delivered 3 more times"
reset
payload=$(sql orders "select payload from outbox where routing_key='payment.requested' and payload->'data'->>'orderId'='$CONFIRMED'")
body=$(python3 -c 'import json,sys;print(json.dumps({"properties":{"delivery_mode":2},"routing_key":"payment.requested","payload":sys.argv[1],"payload_encoding":"string"}))' "$payload")
for _ in 1 2 3; do curl -s -u "$RABBITMQ_USER:$RABBITMQ_PASSWORD" -X POST localhost:15672/api/exchanges/%2F/shop.events/publish -H "$J" -d "$body" >/dev/null; done
sleep 4
echo "  payments for that order:        $(sql payment "select count(*) from payments where order_id='$CONFIRMED'")"
echo "  payment.succeeded events:       $(sql payment "select count(*) from outbox where routing_key='payment.succeeded' and payload->'data'->>'orderId'='$CONFIRMED'")"
echo "  order is still:                 $(status "$CONFIRMED")"
echo; echo "dead-letter queue size: $(docker compose exec -T rabbitmq rabbitmqctl list_queues name messages 2>/dev/null | awk '$1=="shop.dead-letters"{print $2}')"
