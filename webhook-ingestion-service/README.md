# Webhook Ingestion Service (Golang)

## Kontekst

Zewnętrzny provider (np. “Payments Provider”) wysyła webhooki o zdarzeniach: payment_succeeded, payment_failed, refund_created. Webhooki mogą przyjść:
	•	wielokrotnie (duplikaty),
	•	w złej kolejności,
	•	z opóźnieniem,
	•	równolegle (burst).

Twoim zadaniem jest zbudować microservice, który przyjmie webhook, zweryfikuje go, zapisze i przetworzy dokładnie raz (w praktyce: at-least-once delivery + idempotent processing).

⸻

## Wymagania funkcjonalne

1) HTTP API

POST /webhooks/provider
	•	Body: JSON event
	•	Headers:
	•	X-Event-Id (unikalny ID eventu)
	•	X-Signature (HMAC SHA256 payloadu, secret z env)
	•	X-Event-Timestamp (unix seconds) — do ochrony przed replay

Zwraca:
	•	202 Accepted jeśli event przyjęty do kolejki/processing
	•	400 jeśli payload niepoprawny
	•	401 jeśli signature błędna
	•	409 jeśli event już był przyjęty (duplikat)

GET /events/{id}
	•	Zwraca status eventu:
	•	received | processed | failed
	•	attempts
	•	last_error
	•	processed_at

GET /healthz i GET /readyz
	•	readyz powinno zależeć od dostępności storage (np. sqlite/postgres) oraz “czy worker działa”.

⸻

## Wymagania niefunkcjonalne (to jest “senior core”)

2) Idempotency + dedup
	•	Event identyfikowany przez X-Event-Id.
	•	Jeśli event był już received/processed → nie przetwarzaj drugi raz.
	•	Jeśli event jest in-flight → też nie duplikuj pracy.

3) Storage

Minimum: SQLite (łatwe do uruchomienia) albo Postgres (bonus).
Tabela/relacja events:
	•	id, type, payload, status, attempts, next_retry_at, last_error, created_at, updated_at, processed_at
Wymóg: unikalny constraint na id.

4) Worker + retry
	•	Background worker pobiera eventy received/failed z next_retry_at <= now.
	•	Retry z exponential backoff (np. 1s, 2s, 4s… cap 1m) + jitter (bonus).
	•	Max attempts np. 10 → status failed.
	•	Processing ma być cancelable przez context.

5) Concurrency safety
	•	Obsłuż burst: 1000 webhooków w krótkim czasie.
	•	Worker nie może “wziąć” tego samego joba w 2 instancjach (jeśli uruchomisz 2 procesy).
	•	SQLite: zrobisz to transakcyjnie.
	•	Postgres: SELECT … FOR UPDATE SKIP LOCKED (bonus, bardzo senior).

6) Observability
	•	structured logs z request_id + event_id
	•	minimalne metryki (Prometheus endpoint /metrics jako bonus) albo chociaż liczniki w logach:
	•	accepted_total
	•	deduped_total
	•	processed_total
	•	failed_total
	•	retries_total
	•	sensowna separacja warstw: httpapi / service / store / worker.

7) Graceful shutdown
	•	HTTP przestaje przyjmować nowe requesty
	•	worker kończy aktualny event (albo bezpiecznie przerywa) w timeout
	•	flush logów

⸻

Payload example

{
  "type": "payment_succeeded",
  "data": {
    "payment_id": "pay_123",
    "amount": 4999,
    "currency": "USD",
    "customer_id": "cus_9"
  }
}

Processing może być “dummy”, np. zapis do tabeli payments lub zapis do loga. Ważne są: idempotency i retry.

