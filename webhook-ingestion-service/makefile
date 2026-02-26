APP_NAME=webhook-ingestion
DB_URL=postgres://app:app@localhost:5432/webhookdb?sslmode=disable

.PHONY: up down logs psql migrate reset test run

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f postgres

psql:
	psql "$(DB_URL)"

migrate:
	docker compose exec -T postgres psql -U app -d webhookdb -f /migrations/001_create_events.sql
	
reset:
	docker compose down -v
	docker compose up -d
	$(MAKE) migrate

test:
	go test ./...

run:
	DB_URL="postgres://app:app@localhost:5432/webhookdb?sslmode=disable" \
	WEBHOOK_SECRET="dev-secret" \
	go run ./cmd/api