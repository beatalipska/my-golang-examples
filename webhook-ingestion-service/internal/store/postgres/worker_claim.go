package postgres

import (
	"context"
	"database/sql"
	"time"

	"webhook-ingestion-service/internal/model"
	"webhook-ingestion-service/internal/task"
)

// ClaimNextDue atomically claims ONE due event and marks it as processing.
// Returns (event, true, nil) if claimed; (zero, false, nil) if none due.
func (r *EventRepo) ClaimNextDue(ctx context.Context) (task.ClaimedEvent, bool, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		// default isolation is fine; row locks do the heavy lifting
	})
	if err != nil {
		return task.ClaimedEvent{}, false, err
	}
	defer func() { _ = tx.Rollback() }()

	const selectQ = `
SELECT id, type, payload, attempts
FROM events
WHERE status IN ('received','failed')
  AND next_retry_at <= now()
ORDER BY created_at
FOR UPDATE SKIP LOCKED
LIMIT 1;
`
	var e task.ClaimedEvent
	err = tx.QueryRowContext(ctx, selectQ).Scan(&e.ID, &e.Type, &e.Payload, &e.Attempts)
	if err == sql.ErrNoRows {
		_ = tx.Commit()
		return task.ClaimedEvent{}, false, nil
	}
	if err != nil {
		return task.ClaimedEvent{}, false, err
	}

	const updateQ = `
UPDATE events
SET status = 'processing',
    attempts = attempts + 1
WHERE id = $1
RETURNING attempts;
`
	if _, err := tx.ExecContext(ctx, updateQ, e.ID); err != nil {
		return task.ClaimedEvent{}, false, err
	}

	if err := tx.Commit(); err != nil {
		return task.ClaimedEvent{}, false, err
	}
	return e, true, nil
}

func (r *EventRepo) MarkProcessed(ctx context.Context, id string) error {
	const q = `
UPDATE events
SET status = 'processed',
    processed_at = now(),
    last_error = NULL
WHERE id = $1;
`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *EventRepo) MarkFailed(ctx context.Context, id string, lastErr string, nextRetryAt time.Time) error {
	const q = `
UPDATE events
SET status = 'failed',
    last_error = $2,
    next_retry_at = $3
WHERE id = $1;
`
	_, err := r.db.ExecContext(ctx, q, id, lastErr, nextRetryAt.UTC())
	return err
}

// (Optional helper) expose status for tests/debug
func (r *EventRepo) GetStatus(ctx context.Context, id string) (model.EventStatus, int, error) {
	const q = `SELECT status, attempts FROM events WHERE id = $1;`
	var st model.EventStatus
	var attempts int
	err := r.db.QueryRowContext(ctx, q, id).Scan(&st, &attempts)
	if err == sql.ErrNoRows {
		return "", 0, model.ErrNotFound
	}
	return st, attempts, err
}
