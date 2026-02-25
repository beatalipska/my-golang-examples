package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"webhook-ingestion-service/internal/model"
)

type EventRepo struct {
	db *sql.DB
}

func NewEventRepo(db *sql.DB) *EventRepo {
	return &EventRepo{db: db}
}

// Returns (created=true) if inserted, (created=false) if duplicate
func (r *EventRepo) InsertReceived(ctx context.Context, id string, eventType string, payload json.RawMessage) (bool, error) {
	const q = `
INSERT INTO events (id, type, payload, status, attempts, next_retry_at)
VALUES ($1, $2, $3, 'received', 0, now())
ON CONFLICT (id) DO NOTHING;
`
	res, err := r.db.ExecContext(ctx, q, id, eventType, payload)
	if err != nil {
		return false, err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return ra == 1, nil
}

func (r *EventRepo) GetByID(ctx context.Context, id string) (model.Event, error) {
	const q = `
SELECT id, type, status, attempts, next_retry_at, last_error, created_at, updated_at, processed_at
FROM events
WHERE id = $1;
`
	var e model.Event
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&e.ID,
		&e.Type,
		&e.Status,
		&e.Attempts,
		&e.NextRetryAt,
		&e.LastError,
		&e.CreatedAt,
		&e.UpdatedAt,
		&e.ProcessedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Event{}, model.ErrNotFound
		}
		return model.Event{}, err
	}
	return e, nil
}

// Optional: helpful for readyz/healthz
func (r *EventRepo) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	return r.db.PingContext(ctx)
}
