# Webhook Ingestion Service (Go + Postgres)

Minimal webhook ingestion pipeline with:
- HMAC-SHA256 signature verification (hex) + timestamp window (5m)
- Idempotent ingest (`ON CONFLICT DO NOTHING`)
- Atomic claiming for processing (`FOR UPDATE SKIP LOCKED`)
- Retry scheduling with exponential backoff + full jitter
- Background worker + graceful shutdown
- `healthz` (liveness) and `readyz` (DB readiness)
- Debug endpoint: `POST /process/once` (optional, for local demos)

## Architecture (high level)

Provider -> HTTP API -> Postgres (events)
Worker polls Postgres, claims due events atomically, processes, marks processed/failed.

## Local dev

### 1) Start Postgres
```bash
make up
```

### 2) Run migrations
```bash
make migrate
```
## 3) Run API
```bash
make run
```
Expected:
	•	GET /healthz -> 200 OK
	•	GET /readyz -> 200 ready (DB reachable)

## Send a webhook (HMAC hex)
```bash
SECRET="dev-secret"
TS=$(date +%s)
BODY='{"type":"payment_succeeded","data":{"payment_id":"pay_123","amount":4999,"currency":"USD"}}'

SIG=$(printf "%s.%s" "$TS" "$BODY" | openssl dgst -sha256 -hmac "$SECRET" -binary | xxd -p -c 256)

curl -i -X POST http://localhost:8080/webhooks/provider \
  -H "Content-Type: application/json" \
  -H "X-Event-Id: evt_123" \
  -H "X-Event-Timestamp: $TS" \
  -H "X-Signature: $SIG" \
  --data "$BODY"
```

## Check event status

```bash
curl -s http://localhost:8080/events/evt_123 | jq
```
## Processing

The background worker runs automatically. For manual processing (debug):
```bash
curl -i -X POST http://localhost:8080/process/once

Response headers:
	•	X-Processed: 1 processed one event
	•	X-Processed: 0 no due events
```
# Notes
	•	healthz is a liveness probe (does not require DB)
	•	readyz checks DB connectivity
	•	idempotency is based on unique events.id (provider event id)
