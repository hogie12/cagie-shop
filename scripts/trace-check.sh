#!/bin/sh
# Places one order and shows that a single trace covers every service it
# touched. Needs `make up-obs`. Uses 3 of the gateway's 5-per-minute logins.
set -eu
cd "$(dirname "$0")/.."
set -a; . ./.env; set +a

G=localhost:8080
J='Content-Type: application/json'
JAEGER=localhost:16686
PRODUCT=1 # Butter Croissant
py() { python3 -c "import sys,json;d=json.load(sys.stdin);print($1)"; }

email="trace-$(date +%s)@example.com"; pw=$(openssl rand -hex 12)
curl -s -o /dev/null -X POST $G/auth/register -H "$J" -d "{\"email\":\"$email\",\"password\":\"$pw\"}"
token=$(curl -s -X POST $G/auth/login -H "$J" -d "{\"email\":\"$email\",\"password\":\"$pw\"}" | py "d['accessToken']")
admin=$(curl -s -X POST $G/auth/login -H "$J" -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}" | py "d['accessToken']")
before=$(curl -s $G/inventory/$PRODUCT | py "d['quantity']")

# The gateway answers with X-Request-Id, which is the trace id.
headers=$(mktemp)
order=$(curl -s -D "$headers" -X POST $G/orders -H "$J" -H "Authorization: Bearer $token" -d "{\"items\":[{\"productId\":$PRODUCT,\"quantity\":1}]}" | py "d['id']")
trace=$(grep -i '^x-request-id:' "$headers" | tr -d '\r' | awk '{print $2}'); rm -f "$headers"
echo "order $order  trace $trace"

for _ in $(seq 1 30); do
  status=$(curl -s $G/orders/$order -H "Authorization: Bearer $token" | py "d['status']")
  [ "$status" = confirmed ] || [ "$status" = cancelled ] && break
  sleep 0.5
done
echo "order status: $status"
curl -s -o /dev/null -X PUT $G/inventory/$PRODUCT -H "$J" -H "Authorization: Bearer $admin" -d "{\"quantity\":$before}"

sleep 6 # services export spans in batches every few seconds
curl -s "$JAEGER/api/traces/$trace" | python3 -c '
import sys, json
data = json.load(sys.stdin)["data"]
if not data:
    sys.exit("trace not found in Jaeger")
t = data[0]
names = {p: v["serviceName"] for p, v in t["processes"].items()}
spans = sorted(t["spans"], key=lambda s: s["startTime"])
start = spans[0]["startTime"]
print(f"{len(spans)} spans from {len(set(names.values()))} services: {sorted(set(names.values()))}")
for s in spans:
    if s["operationName"].startswith(("SELECT", "INSERT", "UPDATE", "query", "prepare", "connect", "acquire", "begin", "commit", "rollback", "batch")):
        continue  # database spans hidden here; Jaeger shows them all
    offset = (s["startTime"] - start) / 1000
    service = names[s["processID"]]
    op = s["operationName"]
    print(f"  +{offset:7.1f} ms  {service:<9} {op}")
'
echo "open: http://$JAEGER/trace/$trace"
